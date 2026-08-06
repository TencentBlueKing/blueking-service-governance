// Package updater checks for and applies bkms-cli updates.
package updater

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Masterminds/semver/v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/version"
)

const (
	sourceGitHub = "github"
	sourceRepo   = "repo"
	// cliTagPrefix identifies bkms-cli releases in the monorepo.
	cliTagPrefix = "bkms-cli/"
	// maxBinarySize bounds memory use while go-selfupdate verifies a release.
	maxBinarySize = 512 * 1024 * 1024
)

var (
	// Build pipelines inject these values with -ldflags so one binary has exactly
	// one update source and does not need a runtime update configuration file.
	updateSource     = ""
	repoBaseURL      = ""
	githubRepository = ""
)

var (
	// ErrUpdateNotConfigured indicates that the binary has no usable update source.
	ErrUpdateNotConfigured = errors.New("self-update is not configured for this build")
	// ErrInvalidVersion indicates that the current or remote version is not valid SemVer.
	ErrInvalidVersion = errors.New("invalid update version")
	// ErrNoRelease indicates that no release asset matches the current platform.
	ErrNoRelease = errors.New("no compatible release found")
	// ErrBinaryTooLarge indicates that a release asset exceeds the supported size.
	ErrBinaryTooLarge = errors.New("update binary exceeds size limit")
)

// Info describes the result of an update check.
type Info struct {
	CurrentVersion string
	LatestVersion  string
	Available      bool
}

// provider checks for and applies updates from one release source.
type provider interface {
	Check(ctx context.Context, current string) (Info, error)
	Update(ctx context.Context, current string) (Info, error)
}

// Check reports whether this binary has a newer release.
func Check(ctx context.Context) (Info, error) {
	instance, err := newProvider()
	if err != nil {
		return Info{}, err
	}
	return instance.Check(ctx, version.Version)
}

// Update installs a newer release when one is available.
func Update(ctx context.Context) (Info, error) {
	instance, err := newProvider()
	if err != nil {
		return Info{}, err
	}
	return instance.Update(ctx, version.Version)
}

// newProvider creates the provider configured for this binary's update source.
func newProvider() (provider, error) {
	switch strings.ToLower(strings.TrimSpace(updateSource)) {
	case sourceGitHub:
		return newGitHubProvider()
	case sourceRepo:
		return newRepoProvider(repoBaseURL)
	default:
		return nil, fmt.Errorf("%w: update source %q", ErrUpdateNotConfigured, updateSource)
	}
}

// platformAssetName is the shared release contract for both update sources.
// Exact names keep bkms-cli assets distinct from other products in the same release.
func platformAssetName(goos, goarch string) (string, error) {
	if goos != "linux" && goos != "darwin" && goos != "windows" {
		return "", fmt.Errorf("unsupported update platform %s/%s", goos, goarch)
	}
	if goarch != "amd64" && goarch != "arm64" {
		return "", fmt.Errorf("unsupported update platform %s/%s", goos, goarch)
	}

	name := fmt.Sprintf("bkms-cli-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name, nil
}

// parseVersion accepts the optional bkms-cli tag and "v" prefixes but otherwise
// requires strict SemVer so comparison behavior stays deterministic.
func parseVersion(value string) (*semver.Version, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("%w: version is empty", ErrInvalidVersion)
	}

	normalized := strings.TrimPrefix(value, cliTagPrefix)
	parsed, err := semver.StrictNewVersion(strings.TrimPrefix(normalized, "v"))
	if err != nil {
		return nil, fmt.Errorf("%w %q: %v", ErrInvalidVersion, value, err)
	}
	return parsed, nil
}

// buildInfo reports an update only when the source version is strictly newer.
func buildInfo(current, latest *semver.Version) Info {
	return Info{
		CurrentVersion: current.String(),
		LatestVersion:  latest.String(),
		Available:      latest.GreaterThan(current),
	}
}

func validateBinarySize(size int64) error {
	if size > maxBinarySize {
		return fmt.Errorf("%w: %d bytes exceeds %d-byte limit", ErrBinaryTooLarge, size, maxBinarySize)
	}
	return nil
}

func normalizeBinarySizeError(err error) error {
	var sizeError *http.MaxBytesError
	if errors.As(err, &sizeError) {
		return fmt.Errorf("%w: exceeds %d-byte limit", ErrBinaryTooLarge, sizeError.Limit)
	}
	return err
}
