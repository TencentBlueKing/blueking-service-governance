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
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	selfupdate "github.com/creativeprojects/go-selfupdate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/version"
)

func useCurrentVersion(value string) {
	original := version.Version
	version.Version = value
	DeferCleanup(func() { version.Version = original })
}

func useUpdateSource(value string) {
	original := updateSource
	updateSource = value
	DeferCleanup(func() { updateSource = original })
}

func testAssetName(semver string) string {
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("bkms-cli_%s_%s_%s.%s", semver, runtime.GOOS, runtime.GOARCH, ext)
}

func clientWithSource(source selfupdate.Source) *client {
	instance, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: checksumFilename},
		Filters:   []string{assetFilter()},
	})
	Expect(err).NotTo(HaveOccurred())
	return &client{updater: instance, repository: selfupdate.ParseSlug("example/bkms-cli")}
}

func clientWithRelease(tag string, assets ...selfupdate.SourceAsset) *client {
	return clientWithSource(githubTestSource{
		releases: []selfupdate.SourceRelease{githubTestRelease{tag: tag, assets: assets}},
	})
}

var _ = Describe("Updater", func() {
	Describe("configuration", func() {
		It("rejects builds without an update source", func() {
			useUpdateSource("")
			_, err := newClient()
			Expect(errors.Is(err, ErrUpdateNotConfigured)).To(BeTrue())
		})

		It("accepts an owner/repository slug", func() {
			useUpdateSource("example/bkms-cli")
			c, err := newClient()
			Expect(err).NotTo(HaveOccurred())
			owner, repository, err := c.repository.GetSlug()
			Expect(err).NotTo(HaveOccurred())
			Expect(owner).To(Equal("example"))
			Expect(repository).To(Equal("bkms-cli"))
		})

		DescribeTable(
			"detects npm install paths",
			func(p string, want bool) {
				Expect(pathLooksLikeNPMInstall(p)).To(Equal(want))
			},
			Entry("node_modules binary", "/usr/local/lib/node_modules/@blueking/bkms-cli/bin/bkms-cli", true),
			Entry(
				"windows node_modules",
				`C:\Users\me\AppData\Roaming\npm\node_modules\@blueking\bkms-cli\bin\bkms-cli.exe`,
				true,
			),
			Entry("plain install", "/usr/local/bin/bkms-cli", false),
		)
	})

	Describe("version parsing", func() {
		DescribeTable("compares SemVer",
			func(currentValue, latestValue string, available bool) {
				current, err := parseVersion(currentValue)
				Expect(err).NotTo(HaveOccurred())
				latest, err := parseVersion(latestValue)
				Expect(err).NotTo(HaveOccurred())
				Expect(latest.GreaterThan(current)).To(Equal(available))
			},
			Entry("newer", "1.2.3", "1.3.0", true),
			Entry("same", "1.2.3", "v1.2.3", false),
			Entry("older", "1.3.0", "1.2.3", false),
			Entry("prerelease of current", "1.2.3", "1.2.3-fix.1", false),
		)

		It("rejects an empty version", func() {
			_, err := parseVersion("")
			Expect(errors.Is(err, ErrInvalidVersion)).To(BeTrue())
		})

		DescribeTable("rejects non-SemVer",
			func(value string) {
				_, err := parseVersion(value)
				Expect(errors.Is(err, ErrInvalidVersion)).To(BeTrue())
			},
			Entry("date tag", "v20260101"),
			Entry("partial", "v1.2"),
			Entry("leading zero", "v01.2.3"),
			Entry("other product", "bkms-server/v1.2.3"),
		)

		It("accepts bkms-cli tags and trims whitespace", func() {
			v, err := parseVersion("  bkms-cli/v1.2.3\n")
			Expect(err).NotTo(HaveOccurred())
			Expect(v.String()).To(Equal("1.2.3"))
		})
	})

	Describe("binary size", func() {
		It("accepts unknown or max size", func() {
			Expect(validateBinarySize(-1)).To(Succeed())
			Expect(validateBinarySize(maxBinarySize)).To(Succeed())
		})

		It("rejects oversized assets", func() {
			Expect(errors.Is(validateBinarySize(maxBinarySize+1), ErrBinaryTooLarge)).To(BeTrue())
		})

		It("normalizes MaxBytesError", func() {
			err := normalizeBinarySizeError(&http.MaxBytesError{Limit: maxBinarySize})
			Expect(errors.Is(err, ErrBinaryTooLarge)).To(BeTrue())
		})
	})

	Describe("GitHub releases", func() {
		BeforeEach(func() {
			useCurrentVersion("1.2.0")
		})

		It("selects the platform archive and requires checksums.txt", func() {
			asset := testAssetName("1.3.0")
			c := clientWithRelease("bkms-cli/v1.3.0",
				githubTestAsset{id: 1, name: strings.Replace(asset, "bkms-cli", "bkms-server", 1)},
				githubTestAsset{id: 2, name: asset, size: 1024},
				githubTestAsset{id: 3, name: checksumFilename},
			)
			info, release, err := c.detect(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(info).To(Equal(Info{
				CurrentVersion: "1.2.0",
				LatestVersion:  "1.3.0",
				Available:      true,
			}))
			Expect(release.AssetName).To(Equal(asset))
		})

		It("rejects a release without checksums.txt", func() {
			c := clientWithRelease("v1.3.0",
				githubTestAsset{id: 1, name: testAssetName("1.3.0"), size: 1024},
			)
			_, _, err := c.detect(context.Background())
			Expect(errors.Is(err, selfupdate.ErrValidationAssetNotFound)).To(BeTrue())
		})

		It("ignores non-SemVer tags", func() {
			c := clientWithRelease("v20260101",
				githubTestAsset{id: 1, name: testAssetName("1.3.0"), size: 1024},
				githubTestAsset{id: 2, name: checksumFilename},
			)
			_, _, err := c.detect(context.Background())
			Expect(errors.Is(err, ErrNoRelease)).To(BeTrue())
		})

		It("rejects an oversized release before download", func() {
			c := clientWithRelease("v1.3.0",
				githubTestAsset{id: 1, name: testAssetName("1.3.0"), size: int(maxBinarySize + 1)},
				githubTestAsset{id: 2, name: checksumFilename},
			)
			info, release, err := c.detect(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Available).To(BeTrue())
			Expect(errors.Is(validateBinarySize(int64(release.AssetByteSize)), ErrBinaryTooLarge)).To(BeTrue())
		})

		It("limits the downloaded asset body", func() {
			asset := testAssetName("1.3.0")
			c := clientWithSource(maxBytesSource{
				Source: githubTestSource{
					releases: []selfupdate.SourceRelease{githubTestRelease{
						tag: "v1.3.0",
						assets: []selfupdate.SourceAsset{
							githubTestAsset{id: 1, name: asset, size: 4},
							githubTestAsset{id: 2, name: checksumFilename},
						},
					}},
					downloads: map[int64]string{1: "oversized"},
				},
				limit: 4,
			})
			info, release, err := c.detect(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Available).To(BeTrue())
			executable, err := selfupdate.ExecutablePath()
			Expect(err).NotTo(HaveOccurred())
			err = normalizeBinarySizeError(c.updater.UpdateTo(context.Background(), release, executable))
			Expect(errors.Is(err, ErrBinaryTooLarge)).To(BeTrue())
		})
	})
})

type githubTestSource struct {
	releases  []selfupdate.SourceRelease
	downloads map[int64]string
}

func (s githubTestSource) ListReleases(
	context.Context,
	selfupdate.Repository,
) ([]selfupdate.SourceRelease, error) {
	return s.releases, nil
}

func (s githubTestSource) DownloadReleaseAsset(
	_ context.Context,
	_ *selfupdate.Release,
	assetID int64,
) (io.ReadCloser, error) {
	content, ok := s.downloads[assetID]
	if !ok {
		return nil, errors.New("unexpected asset download")
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

type githubTestRelease struct {
	tag    string
	assets []selfupdate.SourceAsset
}

func (githubTestRelease) GetID() int64                          { return 1 }
func (r githubTestRelease) GetTagName() string                  { return r.tag }
func (githubTestRelease) GetDraft() bool                        { return false }
func (githubTestRelease) GetPrerelease() bool                   { return false }
func (githubTestRelease) GetPublishedAt() time.Time             { return time.Time{} }
func (githubTestRelease) GetReleaseNotes() string               { return "" }
func (githubTestRelease) GetName() string                       { return "bkms-cli" }
func (githubTestRelease) GetURL() string                        { return "https://example.com/release" }
func (r githubTestRelease) GetAssets() []selfupdate.SourceAsset { return r.assets }

type githubTestAsset struct {
	id   int64
	name string
	size int
}

func (a githubTestAsset) GetID() int64                  { return a.id }
func (a githubTestAsset) GetName() string               { return a.name }
func (a githubTestAsset) GetSize() int                  { return a.size }
func (a githubTestAsset) GetBrowserDownloadURL() string { return "https://example.com/" + a.name }
