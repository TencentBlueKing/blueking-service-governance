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

// Package updater installs verified releases from the configured distribution endpoints.
package updater

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	selfupdate "github.com/creativeprojects/go-selfupdate"
	binaryupdate "github.com/creativeprojects/go-selfupdate/update"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/version"
)

const (
	npmPackageName = "@blueking/bkms-cli"
	goModulePath   = "github.com/TencentBlueKing/blueking-service-governance/bkms-cli"

	cliTagPrefix     = "bkms-cli/"
	installSourceEnv = "BKMS_CLI_INSTALL_SOURCE"
	npmInstallSource = "npm"
	checksumFilename = "checksums.txt"

	checkTimeout    = 15 * time.Second
	downloadTimeout = 5 * time.Minute
	maxVersionSize  = 128
	maxBinarySize   = 512 * 1024 * 1024
	maxChecksumSize = 1024 * 1024
)

var (
	// ErrUpdateNotConfigured indicates an invalid update source configuration.
	ErrUpdateNotConfigured = errors.New("invalid update source configuration")
	// ErrInvalidVersion indicates an invalid current or published version.
	ErrInvalidVersion = errors.New("invalid update version")
	// ErrNoRelease indicates that the version file or a release asset was not found.
	ErrNoRelease = errors.New("no compatible release found")
	// ErrDownloadTooLarge indicates a download exceeds its size limit.
	ErrDownloadTooLarge = errors.New("update download exceeds size limit")
)

// Info describes the result of an update check.
type Info struct {
	CurrentVersion string
	LatestVersion  string
	Available      bool
	UpgradeCommand string
}

func upgradeCommand() string {
	if os.Getenv(installSourceEnv) == npmInstallSource {
		return "npm i -g " + npmPackageName + "@latest"
	}
	// Local source builds also get a Go installation command.
	info, _ := debug.ReadBuildInfo()
	if version.Version == version.DevelopmentVersion ||
		(info != nil && info.Main.Version != "" && info.Main.Version != "(devel)") {
		return "go install " + goModulePath + "@latest"
	}
	return ""
}

// Check reads the stable CLI version without using the GitHub API.
func Check(ctx context.Context) (Info, error) {
	c, err := newClient()
	if err != nil {
		return Info{}, err
	}
	return c.check(ctx)
}

// Update installs a newer release after checksum verification.
// Package-managed installations only return upgrade instructions unless force is set.
func Update(ctx context.Context, force bool) (Info, error) {
	c, err := newClient()
	if err != nil {
		return Info{}, err
	}
	info, err := c.check(ctx)
	if err != nil || !info.Available {
		return info, err
	}
	if info.UpgradeCommand != "" && !force {
		return info, nil
	}
	executable, err := selfupdate.ExecutablePath()
	if err != nil {
		return info, errors.Wrap(err, "locate executable")
	}
	return info, c.install(ctx, info.LatestVersion, executable)
}

type client struct {
	source     config.UpdateSource
	httpClient *http.Client
}

func newClient() (*client, error) {
	source, err := config.G.UpdateSource()
	if err != nil {
		return nil, errors.Wrapf(ErrUpdateNotConfigured, "invalid update configuration: %v", err)
	}
	return &client{source: source, httpClient: &http.Client{Timeout: downloadTimeout}}, nil
}

func (c *client) check(ctx context.Context) (Info, error) {
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	current, err := parseVersion(version.Version)
	if err != nil && version.Version != version.DevelopmentVersion {
		return Info{}, errors.Wrap(err, "parse current version")
	}
	data, err := c.download(ctx, c.source.LatestVersionURL, maxVersionSize)
	if err != nil {
		return Info{}, err
	}
	latest, err := semver.StrictNewVersion(strings.TrimSpace(string(data)))
	if err != nil {
		return Info{}, errors.Wrapf(ErrInvalidVersion, "invalid latest.txt: %v", err)
	}
	if latest.Prerelease() != "" || latest.Metadata() != "" {
		return Info{}, errors.Wrap(ErrInvalidVersion, "latest.txt must contain a stable version")
	}
	return Info{
		CurrentVersion: version.Version,
		LatestVersion:  latest.String(),
		Available:      current == nil || latest.GreaterThan(current),
		UpgradeCommand: upgradeCommand(),
	}, nil
}

func archiveName(releaseVersion string) string {
	extension := "tar.gz"
	if runtime.GOOS == "windows" {
		extension = "zip"
	}
	return fmt.Sprintf("bkms-cli_%s_%s_%s.%s", releaseVersion, runtime.GOOS, runtime.GOARCH, extension)
}

func (c *client) install(ctx context.Context, releaseVersion, executable string) error {
	asset := archiveName(releaseVersion)
	checksums, err := c.download(ctx, c.source.DownloadURL(releaseVersion, checksumFilename), maxChecksumSize)
	if err != nil {
		return err
	}
	data, err := c.download(ctx, c.source.DownloadURL(releaseVersion, asset), maxBinarySize)
	if err != nil {
		return err
	}
	validator := selfupdate.ChecksumValidator{UniqueFilename: checksumFilename}
	if err = validator.Validate(asset, data, checksums); err != nil {
		return errors.Wrap(err, "verify release checksum")
	}
	binary, err := selfupdate.DecompressCommand(
		bytes.NewReader(data), asset, "bkms-cli", runtime.GOOS, runtime.GOARCH,
	)
	if err != nil {
		return errors.Wrap(err, "extract release binary")
	}
	if closer, ok := binary.(io.Closer); ok {
		defer closer.Close()
	}
	limited := http.MaxBytesReader(nil, io.NopCloser(binary), maxBinarySize)
	return errors.Wrap(binaryupdate.Apply(limited, binaryupdate.Options{TargetPath: executable}), "replace executable")
}

func (c *client) download(ctx context.Context, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", version.UserAgent())
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "download %s", url)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.Wrapf(ErrNoRelease, "download %s: HTTP 404", url)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}
	if resp.ContentLength > limit {
		return nil, errors.Wrapf(ErrDownloadTooLarge, "%s exceeds %d bytes", url, limit)
	}
	body := http.MaxBytesReader(nil, resp.Body, limit)
	data, err := io.ReadAll(body)
	var sizeError *http.MaxBytesError
	if errors.As(err, &sizeError) {
		return nil, errors.Wrapf(ErrDownloadTooLarge, "%s exceeds %d bytes", url, limit)
	}
	return data, errors.Wrapf(err, "read %s", url)
}

func parseVersion(value string) (*semver.Version, error) {
	normalized := strings.TrimPrefix(strings.TrimSpace(value), cliTagPrefix)
	parsed, err := semver.StrictNewVersion(strings.TrimPrefix(normalized, "v"))
	if err != nil {
		return nil, errors.Wrapf(ErrInvalidVersion, "version %q: %v", value, err)
	}
	return parsed, nil
}
