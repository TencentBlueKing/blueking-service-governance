package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	binaryupdate "github.com/creativeprojects/go-selfupdate/update"
)

const (
	// versionFilename points at the latest published SemVer in the flat repository.
	versionFilename = "version"
	// checksumHeader is supplied by the artifact repository for binary downloads.
	checksumHeader = "X-Checksum-Sha256"
	// maxVersionSize prevents an unexpected response from being read without a bound.
	maxVersionSize = 1024
)

var (
	// ErrChecksumMissing indicates that the artifact response has no SHA256 checksum header.
	ErrChecksumMissing = errors.New("update checksum header is missing")
	// ErrChecksumInvalid indicates that the artifact response contains an invalid SHA256 checksum.
	ErrChecksumInvalid = errors.New("update checksum header is invalid")
)

// repoProvider reads a flat latest directory instead of release metadata.
// targetPath is empty in production, which makes go-selfupdate replace the
// running executable; tests set it to a temporary file.
type repoProvider struct {
	baseURL    string
	assetName  string
	targetPath string
}

// newRepoProvider validates the latest-directory URL and resolves the exact
// platform asset name once for subsequent checks and updates.
func newRepoProvider(baseURL string) (*repoProvider, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("%w: repo base URL is empty", ErrUpdateNotConfigured)
	}

	parsedURL, err := url.ParseRequestURI(baseURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("%w: invalid repo update URL %q", ErrUpdateNotConfigured, baseURL)
	}
	// The repository address is fixed at build time, and internal builds may
	// intentionally use an HTTP endpoint.
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf(
			"%w: unsupported repo update URL scheme %q",
			ErrUpdateNotConfigured,
			parsedURL.Scheme,
		)
	}

	assetName, err := platformAssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, err
	}

	return &repoProvider{
		baseURL:   parsedURL.String(),
		assetName: assetName,
	}, nil
}

// Check fetches only the small version file and never downloads the binary.
func (p *repoProvider) Check(ctx context.Context, current string) (Info, error) {
	currentVersion, err := parseVersion(current)
	if err != nil {
		return Info{}, fmt.Errorf("parse current version: %w", err)
	}

	latestVersion, err := p.fetchLatestVersion(ctx)
	if err != nil {
		return Info{}, err
	}
	return buildInfo(currentVersion, latestVersion), nil
}

// Update downloads, verifies, and atomically replaces the executable when the
// repository advertises a newer version.
func (p *repoProvider) Update(ctx context.Context, current string) (Info, error) {
	info, err := p.Check(ctx, current)
	if err != nil || !info.Available {
		return info, err
	}

	body, checksum, err := p.fetchBinary(ctx)
	if err != nil {
		return info, err
	}
	defer body.Close()

	if err := binaryupdate.Apply(body, binaryupdate.Options{
		TargetPath: p.targetPath,
		Checksum:   checksum,
	}); err != nil {
		err = normalizeBinarySizeError(err)
		// A rollback failure leaves the executable path inconsistent and therefore
		// needs to be visible separately from the original replacement error.
		if rollbackErr := binaryupdate.RollbackError(err); rollbackErr != nil {
			return info, fmt.Errorf("apply repo update: %w; rollback failed: %v", err, rollbackErr)
		}
		return info, fmt.Errorf("apply repo update: %w", err)
	}
	return info, nil
}

// fetchLatestVersion reads and validates the repository's plain-text version file.
func (p *repoProvider) fetchLatestVersion(ctx context.Context) (*semver.Version, error) {
	response, err := p.fetchFile(ctx, versionFilename)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	// Read one extra byte so an oversized response can be distinguished from a
	// valid response exactly at the configured limit.
	content, err := io.ReadAll(io.LimitReader(response.Body, maxVersionSize+1))
	if err != nil {
		return nil, fmt.Errorf("read repo version: %w", err)
	}
	if len(content) > maxVersionSize {
		return nil, fmt.Errorf("%w: repo version file exceeds %d bytes", ErrInvalidVersion, maxVersionSize)
	}
	latestVersion, err := parseVersion(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse repo version: %w", err)
	}
	return latestVersion, nil
}

// fetchBinary returns a size-limited binary and its advertised checksum.
func (p *repoProvider) fetchBinary(ctx context.Context) (io.ReadCloser, []byte, error) {
	response, err := p.fetchFile(ctx, p.assetName)
	if err != nil {
		return nil, nil, err
	}
	if sizeErr := validateBinarySize(response.ContentLength); sizeErr != nil {
		_ = response.Body.Close()
		return nil, nil, sizeErr
	}

	checksum, err := decodeChecksum(response.Header.Get(checksumHeader))
	if err != nil {
		_ = response.Body.Close()
		return nil, nil, err
	}
	return http.MaxBytesReader(nil, response.Body, maxBinarySize), checksum, nil
}

// fetchFile sends a no-cache GET request to a file in the repository's latest
// directory and returns only successful responses. The caller closes the body.
func (p *repoProvider) fetchFile(ctx context.Context, filename string) (*http.Response, error) {
	fileURL, err := url.JoinPath(p.baseURL, filename)
	if err != nil {
		return nil, fmt.Errorf("build repo file URL: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create repo file request: %w", err)
	}
	request.Header.Set("Cache-Control", "no-cache")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch repo file %q: %w", filename, err)
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return nil, fmt.Errorf(
			"fetch repo file %q: unexpected HTTP status %s",
			filename,
			response.Status,
		)
	}
	return response, nil
}

// decodeChecksum converts the repository header into the bytes expected by Apply.
func decodeChecksum(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, ErrChecksumMissing
	}
	checksum, err := hex.DecodeString(value)
	if err != nil || len(checksum) != sha256.Size {
		return nil, fmt.Errorf("%w: expected a hexadecimal SHA256 value", ErrChecksumInvalid)
	}
	return checksum, nil
}
