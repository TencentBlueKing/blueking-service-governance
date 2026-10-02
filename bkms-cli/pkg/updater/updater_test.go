/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/version"
)

var _ = Describe("Updater", func() {
	BeforeEach(func() {
		GinkgoT().Setenv("BKMS_CLI_INSTALL_SOURCE", "")
		oldVersion, oldConfig := version.Version, config.G
		DeferCleanup(func() { version.Version, config.G = oldVersion, oldConfig })
		version.Version = "1.2.0"
		config.G = &config.Config{}
	})

	DescribeTable("parses current versions",
		func(value, expected string) {
			v, err := parseVersion(value)
			if expected == "" {
				Expect(errors.Is(err, ErrInvalidVersion)).To(BeTrue())
				return
			}
			Expect(err).NotTo(HaveOccurred())
			Expect(v.String()).To(Equal(expected))
		},
		Entry("tag", " bkms-cli/v1.2.3\n", "1.2.3"),
		Entry("pseudo", "v1.2.4-0.20260910000000-abcdef123456", "1.2.4-0.20260910000000-abcdef123456"),
		Entry("other product", "bkms-server/v1.2.3", ""),
	)

	Describe("static version and release downloads", func() {
		var c *client
		var server *httptest.Server
		var responses map[string][]byte
		var requests []string
		const latestPath = "/bkms-cli/latest.txt"
		const releasePath = "/generic/bkms/public-assets/bkms-cli/releases/v1.3.0/"

		BeforeEach(func() {
			requests = nil
			responses = map[string][]byte{latestPath: []byte("1.3.0\n")}
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.URL.RequestURI())
				if r.URL.Path == "/oversize" {
					w.(http.Flusher).Flush()
					_, _ = w.Write([]byte("too large"))
					return
				}
				data, ok := responses[r.URL.Path]
				if !ok {
					http.NotFound(w, r)
					return
				}
				_, _ = w.Write(data)
			}))
			DeferCleanup(server.Close)
			config.G.Update = config.UpdateSource{
				LatestVersionURL:    server.URL + latestPath,
				DownloadURLTemplate: server.URL + "/generic/bkms/public-assets/bkms-cli/releases/v{version}/{archive}",
			}
			var err error
			c, err = newClient()
			Expect(err).NotTo(HaveOccurred())
		})

		It("guides npm installations without downloading release assets", func() {
			GinkgoT().Setenv("BKMS_CLI_INSTALL_SOURCE", "npm")
			responses[latestPath] = []byte("1.4.2\n")
			info, err := Update(context.Background(), false)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.LatestVersion).To(Equal("1.4.2"))
			Expect(info.UpgradeCommand).To(Equal("npm i -g " + npmPackageName + "@latest"))
			Expect(requests).To(Equal([]string{latestPath}))
		})

		It("offers a Go install for development builds without replacing them", func() {
			version.Version = version.DevelopmentVersion
			info, err := Update(context.Background(), false)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.CurrentVersion).To(Equal("dev"))
			Expect(info.Available).To(BeTrue())
			Expect(info.UpgradeCommand).To(Equal(
				"go install " + goModulePath + "@latest"))
			Expect(requests).To(Equal([]string{latestPath}))
		})

		It("allows force to install a release over a development build", func() {
			version.Version = version.DevelopmentVersion
			_, err := Update(context.Background(), true)
			Expect(errors.Is(err, ErrNoRelease)).To(BeTrue())
			Expect(requests).To(Equal([]string{latestPath, releasePath + "checksums.txt"}))
		})

		DescribeTable("compares stable versions",
			func(latest string, available bool) {
				responses[latestPath] = []byte(latest)
				info, err := c.check(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(info).To(Equal(Info{
					CurrentVersion: "1.2.0",
					LatestVersion:  strings.TrimSpace(latest),
					Available:      available,
				}))
				Expect(requests).To(Equal([]string{latestPath}))
			},
			Entry("same", "1.2.0", false),
			Entry("older", "1.1.0", false),
			Entry("newer with CRLF", "1.3.0\r\n", true),
		)

		DescribeTable("rejects invalid stable pointers",
			func(latest string) {
				responses[latestPath] = []byte(latest)
				_, err := c.check(context.Background())
				Expect(errors.Is(err, ErrInvalidVersion)).To(BeTrue())
			},
			Entry("prerelease", "1.3.0-rc.1"),
			Entry("metadata", "1.3.0+build"),
			Entry("HTML", "<html>error</html>"),
		)

		It("reports a missing version file without falling back to another source", func() {
			delete(responses, latestPath)
			_, err := c.check(context.Background())
			Expect(errors.Is(err, ErrNoRelease)).To(BeTrue())
		})

		It("limits downloads without Content-Length", func() {
			_, err := c.download(context.Background(), server.URL+"/oversize", 4)
			Expect(errors.Is(err, ErrDownloadTooLarge)).To(BeTrue())
		})

		It("limits the version file size", func() {
			responses[latestPath] = []byte(strings.Repeat("1", maxVersionSize+1))
			_, err := c.check(context.Background())
			Expect(errors.Is(err, ErrDownloadTooLarge)).To(BeTrue())
		})

		DescribeTable("validates configured distribution assets before replacing the executable",
			func(scenario string, succeeds bool) {
				binary := []byte("new executable")
				archive := releaseArchive(binary)
				asset := archiveName("1.3.0")
				checksum := fmt.Sprintf("%x  %s\n", sha256.Sum256(archive), asset)
				if scenario == "wrong hash" {
					checksum = fmt.Sprintf("%064d  %s\n", 0, asset)
				}
				responses[releasePath+"checksums.txt"] = []byte(checksum)
				if scenario != "missing archive" {
					responses[releasePath+asset] = archive
				}
				executable := filepath.Join(GinkgoT().TempDir(), "bkms-cli")
				Expect(os.WriteFile(executable, []byte("old executable"), 0o755)).To(Succeed())

				err := c.install(context.Background(), "1.3.0", executable)
				installed, readErr := os.ReadFile(executable)
				Expect(readErr).NotTo(HaveOccurred())
				if succeeds {
					Expect(err).NotTo(HaveOccurred())
					Expect(installed).To(Equal(binary))
					Expect(requests).To(Equal([]string{
						"/generic/bkms/public-assets/bkms-cli/releases/v1.3.0/checksums.txt",
						"/generic/bkms/public-assets/bkms-cli/releases/v1.3.0/" + asset,
					}))
				} else {
					Expect(err).To(HaveOccurred())
					Expect(string(installed)).To(Equal("old executable"))
				}
			},
			Entry("verified asset", "valid", true),
			Entry("mismatched hash", "wrong hash", false),
			Entry("missing archive", "missing archive", false),
		)
	})
})

func releaseArchive(binary []byte) []byte {
	var buf bytes.Buffer
	if runtime.GOOS == "windows" {
		archive := zip.NewWriter(&buf)
		file, err := archive.Create("bkms-cli.exe")
		Expect(err).NotTo(HaveOccurred())
		_, err = file.Write(binary)
		Expect(err).NotTo(HaveOccurred())
		Expect(archive.Close()).To(Succeed())
	} else {
		compressed := gzip.NewWriter(&buf)
		archive := tar.NewWriter(compressed)
		Expect(archive.WriteHeader(&tar.Header{Name: "bkms-cli", Mode: 0o755, Size: int64(len(binary))})).To(Succeed())
		_, err := archive.Write(binary)
		Expect(err).NotTo(HaveOccurred())
		Expect(archive.Close()).To(Succeed())
		Expect(compressed.Close()).To(Succeed())
	}
	return buf.Bytes()
}
