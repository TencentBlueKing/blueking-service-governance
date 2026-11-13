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

// Package cliusage 为 bkms-cli 请求输出事件日志，设计见 design_notes/cli_api_usage.md。
package cliusage

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
)

const (
	// HeaderDeviceID 设备 ID 请求头
	HeaderDeviceID = "X-Bkms-Device-Id"

	// VersionDev 源码构建
	VersionDev = "dev"
	// VersionInvalid 版本号非法
	VersionInvalid = "invalid"
	// VersionLegacy v1.0.1、v1.0.2，未声明 UA
	VersionLegacy = "<1.0.3"
	// DeviceIDInvalid 设备 ID 格式非法
	DeviceIDInvalid = "invalid"

	cliUserAgentPrefix   = "bkms-cli/"
	restyUserAgentPrefix = "go-resty/"
	// maxVersionLen 版本号最大长度
	maxVersionLen = 64
)

var (
	// platformPattern 匹配 "(<os>/<arch>)"
	platformPattern = regexp.MustCompile(`^\(([a-z0-9_]+)/([a-z0-9_]+)\)$`)
	// deviceIDPattern 匹配 HMAC-SHA256 十六进制串
	deviceIDPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// Client bkms-cli 客户端信息
type Client struct {
	Version  string
	OS       string
	Arch     string
	DeviceID string
}

// Identify 识别 bkms-cli 请求。v1.0.1、v1.0.2 按 resty 默认 UA + Bearer 认证识别。
func Identify(r *http.Request) (Client, bool) {
	var client Client
	ua := r.UserAgent()
	switch {
	case strings.HasPrefix(ua, cliUserAgentPrefix):
		client = parseCLIUserAgent(strings.TrimPrefix(ua, cliUserAgentPrefix))
	case strings.HasPrefix(ua, restyUserAgentPrefix) && hasBearerToken(r):
		client = Client{Version: VersionLegacy}
	default:
		return Client{}, false
	}
	client.DeviceID = normalizeDeviceID(r.Header.Get(HeaderDeviceID))
	return client, true
}

// parseCLIUserAgent 解析 "1.0.7 (linux/amd64)"
func parseCLIUserAgent(s string) Client {
	version, platform, _ := strings.Cut(s, " ")
	client := Client{Version: normalizeVersion(version)}
	if m := platformPattern.FindStringSubmatch(platform); m != nil {
		client.OS, client.Arch = m[1], m[2]
	}
	return client
}

func normalizeVersion(version string) string {
	if version == VersionDev {
		return VersionDev
	}
	if len(version) > maxVersionLen {
		return VersionInvalid
	}
	if _, err := semver.StrictNewVersion(version); err != nil {
		return VersionInvalid
	}
	return version
}

func normalizeDeviceID(deviceID string) string {
	if deviceID == "" {
		return ""
	}
	if !deviceIDPattern.MatchString(deviceID) {
		return DeviceIDInvalid
	}
	return deviceID
}

func hasBearerToken(r *http.Request) bool {
	parts := strings.Fields(r.Header.Get("Authorization"))
	return len(parts) == 2 && parts[0] == "Bearer"
}
