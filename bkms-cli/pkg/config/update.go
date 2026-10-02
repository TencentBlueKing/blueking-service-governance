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

package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

const (
	DefaultUpdateLatestURL = "https://raw.githubusercontent.com/TencentBlueKing/blueking-service-governance/" +
		"main/bkms-cli/latest.txt"
	DefaultUpdateDownloadURLTemplate = "https://github.com/TencentBlueKing/blueking-service-governance/" +
		"releases/download/bkms-cli%2Fv{version}/{archive}"
)

// UpdateSource describes one distribution channel. Both URLs are configured together.
type UpdateSource struct {
	LatestVersionURL    string `yaml:"latestVersionUrl"`
	DownloadURLTemplate string `yaml:"downloadUrlTemplate"`
}

// IsZero keeps an unconfigured source out of the YAML file.
func (s UpdateSource) IsZero() bool {
	return s.LatestVersionURL == "" && s.DownloadURLTemplate == ""
}

// Validate checks URLs and the supported template placeholders.
func (s UpdateSource) Validate() error {
	if s.LatestVersionURL == "" || s.DownloadURLTemplate == "" {
		return errors.New("update latest URL and download URL template must be configured together")
	}
	if err := validateUpdateURL(s.LatestVersionURL); err != nil {
		return errors.Wrap(err, "update.latestVersionUrl")
	}
	if !strings.Contains(s.DownloadURLTemplate, "{archive}") {
		return errors.New("update.downloadUrlTemplate must contain {archive}")
	}
	return errors.Wrap(validateUpdateURL(s.DownloadURL("1.0.0", "checksums.txt")), "update.downloadUrlTemplate")
}

func validateUpdateURL(value string) error {
	if strings.ContainsAny(value, "{} \t\r\n") {
		return errors.New("URL contains whitespace or unsupported placeholders")
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return err
	}
	if (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Hostname() == "" || parsed.Fragment != "" {
		return errors.New("expected an absolute HTTP(S) URL without a fragment")
	}
	return nil
}

// DownloadURL expands the archive or checksums.txt URL for one version.
func (s UpdateSource) DownloadURL(version, archive string) string {
	return strings.NewReplacer("{version}", version, "{archive}", archive).Replace(s.DownloadURLTemplate)
}

// UpdateSource returns the configured pair, or the official defaults when both are absent.
func (c *Config) UpdateSource() (UpdateSource, error) {
	source := UpdateSource{}
	if c != nil {
		source = c.Update
	}
	source.LatestVersionURL = strings.TrimSpace(source.LatestVersionURL)
	source.DownloadURLTemplate = strings.TrimSpace(source.DownloadURLTemplate)
	if source.IsZero() {
		source = UpdateSource{
			LatestVersionURL:    DefaultUpdateLatestURL,
			DownloadURLTemplate: DefaultUpdateDownloadURLTemplate,
		}
	}
	return source, source.Validate()
}

// String displays only explicit overrides, leaving built-in defaults out of the stored configuration.
func (s UpdateSource) String() string {
	if s.IsZero() {
		return ""
	}
	return fmt.Sprintf("update:\n  latestVersionUrl: %s\n  downloadUrlTemplate: %s\n",
		s.LatestVersionURL, s.DownloadURLTemplate)
}
