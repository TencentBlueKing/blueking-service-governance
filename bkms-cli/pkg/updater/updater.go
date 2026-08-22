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

// Package updater checks for and applies bkms-cli updates from GitHub Releases.
package updater

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	selfupdate "github.com/creativeprojects/go-selfupdate"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/version"
)

const (
	cliTagPrefix     = "bkms-cli/"
	checksumFilename = "checksums.txt"
	maxBinarySize    = 512 * 1024 * 1024
)

// updateSource is injected at build time as owner/repository.
var updateSource = ""

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

// InstalledViaNPM reports whether the running binary lives under an npm package tree.
func InstalledViaNPM() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		resolved = exe
	}
	return pathLooksLikeNPMInstall(resolved)
}

func pathLooksLikeNPMInstall(p string) bool {
	normalized := strings.ReplaceAll(filepath.ToSlash(p), `\`, "/")
	return strings.Contains(normalized, "/node_modules/")
}

// Check reports whether this binary has a newer GitHub release.
func Check(ctx context.Context) (Info, error) {
	c, err := newClient()
	if err != nil {
		return Info{}, err
	}
	info, _, err := c.detect(ctx)
	return info, err
}

// Update installs a newer GitHub release when one is available.
func Update(ctx context.Context) (Info, error) {
	c, err := newClient()
	if err != nil {
		return Info{}, err
	}
	info, release, err := c.detect(ctx)
	if err != nil || !info.Available {
		return info, err
	}
	if err = validateBinarySize(int64(release.AssetByteSize)); err != nil {
		return info, err
	}

	executable, err := selfupdate.ExecutablePath()
	if err != nil {
		return info, errors.Wrap(err, "locate executable")
	}
	if err := c.updater.UpdateTo(ctx, release, executable); err != nil {
		return info, errors.Wrap(normalizeBinarySizeError(err), "apply update")
	}
	return info, nil
}

type client struct {
	updater    *selfupdate.Updater
	repository selfupdate.Repository
}

func newClient() (*client, error) {
	source := strings.TrimSpace(updateSource)
	if source == "" {
		return nil, errors.Wrap(ErrUpdateNotConfigured, "update source is empty")
	}
	slug := selfupdate.ParseSlug(source)
	if _, _, err := slug.GetSlug(); err != nil {
		return nil, errors.Wrapf(ErrUpdateNotConfigured, "invalid GitHub repository %q", source)
	}

	ghSource, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return nil, errors.Wrap(err, "create GitHub source")
	}
	instance, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:    maxBytesSource{Source: ghSource, limit: maxBinarySize},
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: checksumFilename},
		Filters:   []string{assetFilter()},
	})
	if err != nil {
		return nil, errors.Wrap(err, "create updater")
	}
	return &client{updater: instance, repository: slug}, nil
}

func assetFilter() string {
	ext := `tar\.gz`
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf(`^bkms-cli_.+_%s_%s\.%s$`, runtime.GOOS, runtime.GOARCH, ext)
}

func (c *client) detect(ctx context.Context) (Info, *selfupdate.Release, error) {
	current, err := parseVersion(version.Version)
	if err != nil {
		return Info{}, nil, errors.Wrap(err, "parse current version")
	}
	release, found, err := c.updater.DetectLatest(ctx, c.repository)
	if err != nil {
		return Info{}, nil, errors.Wrap(err, "detect latest release")
	}
	if !found {
		return Info{}, nil, ErrNoRelease
	}
	latest, err := parseVersion(release.Version())
	if err != nil {
		return Info{}, nil, errors.Wrap(err, "parse latest version")
	}
	return Info{
		CurrentVersion: current.String(),
		LatestVersion:  latest.String(),
		Available:      latest.GreaterThan(current),
	}, release, nil
}

// maxBytesSource limits downloaded release assets before go-selfupdate buffers them.
type maxBytesSource struct {
	selfupdate.Source
	limit int64
}

func (s maxBytesSource) DownloadReleaseAsset(
	ctx context.Context,
	release *selfupdate.Release,
	assetID int64,
) (io.ReadCloser, error) {
	body, err := s.Source.DownloadReleaseAsset(ctx, release, assetID)
	if err != nil {
		return nil, err
	}
	return http.MaxBytesReader(nil, body, s.limit), nil
}

func parseVersion(value string) (*semver.Version, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.Wrap(ErrInvalidVersion, "version is empty")
	}
	normalized := strings.TrimPrefix(value, cliTagPrefix)
	parsed, err := semver.StrictNewVersion(strings.TrimPrefix(normalized, "v"))
	if err != nil {
		return nil, errors.Wrapf(ErrInvalidVersion, "version %q: %v", value, err)
	}
	return parsed, nil
}

func validateBinarySize(size int64) error {
	if size > maxBinarySize {
		return errors.Wrapf(ErrBinaryTooLarge, "%d bytes exceeds %d-byte limit", size, maxBinarySize)
	}
	return nil
}

func normalizeBinarySizeError(err error) error {
	var sizeError *http.MaxBytesError
	if errors.As(err, &sizeError) {
		return errors.Wrapf(ErrBinaryTooLarge, "exceeds %d-byte limit", sizeError.Limit)
	}
	return err
}
