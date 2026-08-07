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
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	binaryupdate "github.com/creativeprojects/go-selfupdate/update"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/version"
)

const (
	// versionFilename points at the latest published bkms-cli Git tag in the flat repository.
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
type repoProvider struct {
	baseURL string
	// targetPath is empty in production, which makes go-selfupdate replace the
	// running executable; tests set it to a temporary file.
	targetPath string
}

// newRepoProvider validates the latest-directory URL.
func newRepoProvider(baseURL string) (*repoProvider, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, errors.Wrap(ErrUpdateNotConfigured, "repo base URL is empty")
	}

	parsedURL, err := url.ParseRequestURI(baseURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, errors.Wrapf(ErrUpdateNotConfigured, "invalid repo update URL %q", baseURL)
	}
	// The repository address is fixed at build time, and internal builds may
	// intentionally use an HTTP endpoint.
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, errors.Wrapf(
			ErrUpdateNotConfigured,
			"unsupported repo update URL scheme %q",
			parsedURL.Scheme,
		)
	}

	return &repoProvider{baseURL: parsedURL.String()}, nil
}

// Check fetches only the small version file and never downloads the binary.
func (p *repoProvider) Check(ctx context.Context) (Info, error) {
	currentVersion, err := parseVersion(version.Version)
	if err != nil {
		return Info{}, errors.Wrap(err, "parse current version")
	}

	latestVersion, err := p.fetchLatestVersion(ctx)
	if err != nil {
		return Info{}, err
	}
	return buildInfo(currentVersion, latestVersion), nil
}

// Update downloads, verifies, and atomically replaces the executable when the
// repository advertises a newer version.
func (p *repoProvider) Update(ctx context.Context) (Info, error) {
	info, err := p.Check(ctx)
	if err != nil || !info.Available {
		return info, err
	}

	body, checksum, err := p.fetchBinary(ctx, info.LatestVersion)
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
			return info, errors.Wrapf(err, "apply repo update; rollback failed: %v", rollbackErr)
		}
		return info, errors.Wrap(err, "apply repo update")
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
		return nil, errors.Wrap(err, "read repo version")
	}
	if len(content) > maxVersionSize {
		return nil, errors.Wrapf(ErrInvalidVersion, "repo version file exceeds %d bytes", maxVersionSize)
	}
	latestVersion, err := parseVersion(string(content))
	if err != nil {
		return nil, errors.Wrap(err, "parse repo version")
	}
	return latestVersion, nil
}

// fetchBinary returns the versioned binary advertised by the latest version
// file, together with its checksum and a size-limited response body.
func (p *repoProvider) fetchBinary(ctx context.Context, version string) (io.ReadCloser, []byte, error) {
	assetName, err := platformAssetName(runtime.GOOS, runtime.GOARCH, version)
	if err != nil {
		return nil, nil, err
	}
	response, err := p.fetchFile(ctx, assetName)
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
		return nil, errors.Wrap(err, "build repo file URL")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, http.NoBody)
	if err != nil {
		return nil, errors.Wrap(err, "create repo file request")
	}
	request.Header.Set("Cache-Control", "no-cache")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch repo file %q", filename)
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return nil, errors.Errorf(
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
		return nil, errors.Wrap(ErrChecksumInvalid, "expected a hexadecimal SHA256 value")
	}
	return checksum, nil
}
