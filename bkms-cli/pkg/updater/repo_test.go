package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Repo provider", func() {
	DescribeTable("builds the platform artifact name",
		func(goos, goarch, expected string) {
			name, err := platformAssetName(goos, goarch)
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal(expected))
		},
		Entry("Linux AMD64", "linux", "amd64", "bkms-cli-linux-amd64"),
		Entry("Linux ARM64", "linux", "arm64", "bkms-cli-linux-arm64"),
		Entry("macOS AMD64", "darwin", "amd64", "bkms-cli-darwin-amd64"),
		Entry("macOS ARM64", "darwin", "arm64", "bkms-cli-darwin-arm64"),
		Entry("Windows AMD64", "windows", "amd64", "bkms-cli-windows-amd64.exe"),
		Entry("Windows ARM64", "windows", "arm64", "bkms-cli-windows-arm64.exe"),
	)

	It("rejects unsupported platforms", func() {
		_, err := platformAssetName("freebsd", "amd64")
		Expect(err).To(MatchError("unsupported update platform freebsd/amd64"))
	})

	Describe("configuration", func() {
		It("rejects an empty repository URL as missing configuration", func() {
			_, err := newRepoProvider("  ")
			Expect(errors.Is(err, ErrUpdateNotConfigured)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("repo base URL is empty")))
		})

		It("accepts an HTTPS repository URL", func() {
			provider, err := newRepoProvider("https://repo.example.com/latest/")
			Expect(err).NotTo(HaveOccurred())
			Expect(provider.baseURL).To(Equal("https://repo.example.com/latest/"))
		})
	})

	Describe("flat artifact repository", func() {
		var (
			server        *httptest.Server
			binary        []byte
			checksum      string
			versionValue  string
			includeHash   bool
			requestLock   sync.Mutex
			assetRequests int
			assetPath     string
			cacheHeaders  []string
		)

		BeforeEach(func() {
			binary = []byte("new bkms-cli binary")
			hash := sha256.Sum256(binary)
			checksum = hex.EncodeToString(hash[:])
			versionValue = "v1.3.0\n"
			includeHash = true
			assetRequests = 0
			cacheHeaders = nil
			assetName, err := platformAssetName(runtime.GOOS, runtime.GOARCH)
			Expect(err).NotTo(HaveOccurred())
			assetPath = "/" + assetName

			server = httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				requestLock.Lock()
				cacheHeaders = append(cacheHeaders, request.Header.Get("Cache-Control"))
				requestLock.Unlock()

				switch request.URL.Path {
				case "/version":
					_, _ = response.Write([]byte(versionValue))
				case assetPath:
					requestLock.Lock()
					assetRequests++
					requestLock.Unlock()
					if includeHash {
						response.Header().Set(checksumHeader, checksum)
					}
					_, _ = response.Write(binary)
				default:
					http.NotFound(response, request)
				}
			}))
		})

		AfterEach(func() {
			server.Close()
		})

		providerForTarget := func(targetPath string) *repoProvider {
			provider, err := newRepoProvider(server.URL)
			Expect(err).NotTo(HaveOccurred())
			provider.targetPath = targetPath
			return provider
		}

		It("checks the plain-text version without downloading the binary", func() {
			info, err := providerForTarget("").Check(context.Background(), "v1.2.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(info).To(Equal(Info{
				CurrentVersion: "1.2.0",
				LatestVersion:  "1.3.0",
				Available:      true,
			}))

			requestLock.Lock()
			defer requestLock.Unlock()
			Expect(assetRequests).To(BeZero())
			Expect(cacheHeaders).To(Equal([]string{"no-cache"}))
		})

		It("downloads, verifies, and replaces the current binary", func() {
			targetPath := filepath.Join(GinkgoT().TempDir(), "bkms-cli")
			Expect(os.WriteFile(targetPath, []byte("old binary"), 0o755)).To(Succeed())

			info, err := providerForTarget(targetPath).Update(context.Background(), "v1.2.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Available).To(BeTrue())
			updated, err := os.ReadFile(targetPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated).To(Equal(binary))

			requestLock.Lock()
			defer requestLock.Unlock()
			Expect(assetRequests).To(Equal(1))
			Expect(cacheHeaders).To(Equal([]string{"no-cache", "no-cache"}))
		})

		It("does not download or downgrade when the remote version is older", func() {
			versionValue = "v1.1.0\n"
			targetPath := filepath.Join(GinkgoT().TempDir(), "bkms-cli")
			oldBinary := []byte("current binary")
			Expect(os.WriteFile(targetPath, oldBinary, 0o755)).To(Succeed())

			info, err := providerForTarget(targetPath).Update(context.Background(), "v1.2.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Available).To(BeFalse())
			current, readErr := os.ReadFile(targetPath)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(current).To(Equal(oldBinary))

			requestLock.Lock()
			defer requestLock.Unlock()
			Expect(assetRequests).To(BeZero())
			Expect(cacheHeaders).To(Equal([]string{"no-cache"}))
		})

		It("does not replace the binary when the checksum header is missing", func() {
			includeHash = false
			targetPath := filepath.Join(GinkgoT().TempDir(), "bkms-cli")
			oldBinary := []byte("old binary")
			Expect(os.WriteFile(targetPath, oldBinary, 0o755)).To(Succeed())

			_, err := providerForTarget(targetPath).Update(context.Background(), "v1.2.0")
			Expect(errors.Is(err, ErrChecksumMissing)).To(BeTrue())
			current, readErr := os.ReadFile(targetPath)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(current).To(Equal(oldBinary))
		})

		It("does not replace the binary when the checksum is wrong", func() {
			wrongHash := sha256.Sum256([]byte("different binary"))
			checksum = hex.EncodeToString(wrongHash[:])
			targetPath := filepath.Join(GinkgoT().TempDir(), "bkms-cli")
			oldBinary := []byte("old binary")
			Expect(os.WriteFile(targetPath, oldBinary, 0o755)).To(Succeed())

			_, err := providerForTarget(targetPath).Update(context.Background(), "v1.2.0")
			Expect(err).To(HaveOccurred())
			current, readErr := os.ReadFile(targetPath)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(current).To(Equal(oldBinary))
		})
	})
})
