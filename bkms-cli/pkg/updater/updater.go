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

// Package updater checks for and applies bkms-cli updates.
package updater

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
)

const (
	// cliTagPrefix identifies bkms-cli releases in the monorepo.
	cliTagPrefix = "bkms-cli/"
	// maxBinarySize bounds memory use while go-selfupdate verifies a release.
	maxBinarySize = 512 * 1024 * 1024
)

// Build pipelines inject either an owner/repository slug or an HTTP(S) latest
// directory URL, so the binary needs no runtime update configuration file.
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

// provider checks for and applies updates from one release source.
type provider interface {
	Check(ctx context.Context) (Info, error)
	Update(ctx context.Context) (Info, error)
}

// Check reports whether this binary has a newer release.
func Check(ctx context.Context) (Info, error) {
	instance, err := newProvider()
	if err != nil {
		return Info{}, err
	}
	return instance.Check(ctx)
}

// Update installs a newer release when one is available.
func Update(ctx context.Context) (Info, error) {
	instance, err := newProvider()
	if err != nil {
		return Info{}, err
	}
	return instance.Update(ctx)
}

// newProvider creates the provider configured for this binary's update source.
func newProvider() (provider, error) {
	source := strings.TrimSpace(updateSource)
	if source == "" {
		return nil, errors.Wrap(ErrUpdateNotConfigured, "update source is empty")
	}

	lowerSource := strings.ToLower(source)
	if strings.HasPrefix(lowerSource, "http://") || strings.HasPrefix(lowerSource, "https://") {
		return newRepoProvider(source)
	}
	return newGitHubProvider(source)
}

// platformAssetName returns the complete release asset name for a platform.
func platformAssetName(goos, goarch, version string) (string, error) {
	if goos != "linux" && goos != "darwin" && goos != "windows" {
		return "", errors.Errorf("unsupported update platform %s/%s", goos, goarch)
	}
	if goarch != "amd64" && goarch != "arm64" {
		return "", errors.Errorf("unsupported update platform %s/%s", goos, goarch)
	}

	name := fmt.Sprintf("bkms-cli-%s-%s-v%s", goos, goarch, version)
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
		return nil, errors.Wrap(ErrInvalidVersion, "version is empty")
	}

	normalized := strings.TrimPrefix(value, cliTagPrefix)
	parsed, err := semver.StrictNewVersion(strings.TrimPrefix(normalized, "v"))
	if err != nil {
		return nil, errors.Wrapf(ErrInvalidVersion, "version %q: %v", value, err)
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
