package updater

import (
	"context"
	"errors"
	"io"
	"runtime"
	"strings"
	"time"

	selfupdate "github.com/creativeprojects/go-selfupdate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GitHub provider", func() {
	providerWithRelease := func(tag string, assets ...selfupdate.SourceAsset) *githubProvider {
		config, err := githubUpdaterConfig()
		Expect(err).NotTo(HaveOccurred())
		config.Source = githubTestSource{releases: []selfupdate.SourceRelease{
			githubTestRelease{tag: tag, assets: assets},
		}}

		instance, err := selfupdate.NewUpdater(config)
		Expect(err).NotTo(HaveOccurred())
		return &githubProvider{
			updater:    instance,
			repository: selfupdate.ParseSlug(githubRepository),
		}
	}

	currentAssetName := func() string {
		name, err := platformAssetName(runtime.GOOS, runtime.GOARCH)
		Expect(err).NotTo(HaveOccurred())
		return name
	}

	It("uses the release checksum file", func() {
		config, err := githubUpdaterConfig()
		Expect(err).NotTo(HaveOccurred())
		validator, ok := config.Validator.(*selfupdate.ChecksumValidator)
		Expect(ok).To(BeTrue())
		Expect(validator.UniqueFilename).To(Equal(checksumFilename))
	})

	It("uses the configured GitHub repository", func() {
		originalRepository := githubRepository
		githubRepository = "example/bkms-cli"
		DeferCleanup(func() { githubRepository = originalRepository })

		provider, err := newGitHubProvider()
		Expect(err).NotTo(HaveOccurred())
		owner, repository, err := provider.repository.GetSlug()
		Expect(err).NotTo(HaveOccurred())
		Expect(owner).To(Equal("example"))
		Expect(repository).To(Equal("bkms-cli"))
	})

	It("configures a limit for GitHub asset downloads", func() {
		config, err := githubUpdaterConfig()
		Expect(err).NotTo(HaveOccurred())
		source, ok := config.Source.(maxBytesSource)
		Expect(ok).To(BeTrue())
		Expect(source.limit).To(Equal(int64(maxBinarySize)))
	})

	It("selects only the exact bkms-cli platform asset", func() {
		assetName := currentAssetName()
		provider := providerWithRelease("v1.3.0",
			githubTestAsset{id: 1, name: strings.Replace(assetName, "bkms-cli", "bkms-server", 1)},
			githubTestAsset{id: 2, name: assetName + ".bak"},
			githubTestAsset{id: 3, name: assetName, size: 1024},
			githubTestAsset{id: 4, name: checksumFilename},
		)

		info, release, err := provider.detect(context.Background(), "v1.2.0")
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Available).To(BeTrue())
		Expect(release.AssetName).To(Equal(assetName))
	})

	It("rejects a release without the checksum file", func() {
		provider := providerWithRelease("v1.3.0",
			githubTestAsset{id: 1, name: currentAssetName(), size: 1024},
		)

		_, err := provider.Check(context.Background(), "v1.2.0")
		Expect(errors.Is(err, selfupdate.ErrValidationAssetNotFound)).To(BeTrue())
	})

	It("ignores a release whose tag is not SemVer", func() {
		provider := providerWithRelease("v20260101",
			githubTestAsset{id: 1, name: currentAssetName(), size: 1024},
			githubTestAsset{id: 2, name: checksumFilename},
		)

		_, err := provider.Check(context.Background(), "v1.2.0")
		Expect(errors.Is(err, ErrNoRelease)).To(BeTrue())
	})

	It("rejects an oversized release before downloading it", func() {
		provider := providerWithRelease("v1.3.0",
			githubTestAsset{id: 1, name: currentAssetName(), size: int(maxBinarySize + 1)},
			githubTestAsset{id: 2, name: checksumFilename},
		)

		_, err := provider.Update(context.Background(), "v1.2.0")
		Expect(errors.Is(err, ErrBinaryTooLarge)).To(BeTrue())
	})

	It("limits the actual downloaded asset", func() {
		assetName := currentAssetName()
		source := githubTestSource{
			releases: []selfupdate.SourceRelease{githubTestRelease{
				tag: "v1.3.0",
				assets: []selfupdate.SourceAsset{
					githubTestAsset{id: 1, name: assetName, size: 4},
					githubTestAsset{id: 2, name: checksumFilename},
				},
			}},
			downloads: map[int64]string{1: "oversized"},
		}
		config, err := githubUpdaterConfig()
		Expect(err).NotTo(HaveOccurred())
		config.Source = maxBytesSource{Source: source, limit: 4}
		instance, err := selfupdate.NewUpdater(config)
		Expect(err).NotTo(HaveOccurred())
		provider := &githubProvider{
			updater:    instance,
			repository: selfupdate.ParseSlug(githubRepository),
		}

		_, err = provider.Update(context.Background(), "v1.2.0")
		Expect(errors.Is(err, ErrBinaryTooLarge)).To(BeTrue())
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
		return nil, errors.New("unexpected GitHub asset download")
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
