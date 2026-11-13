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

package cliusage_test

import (
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/cliusage"
)

var _ = Describe("Identify", func() {
	deviceID := strings.Repeat("ab", 32)

	newRequest := func(userAgent, authorization, device string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("User-Agent", userAgent)
		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}
		if device != "" {
			req.Header.Set(cliusage.HeaderDeviceID, device)
		}
		return req
	}

	DescribeTable("identifies bkms-cli requests",
		func(userAgent, authorization, device string, expected cliusage.Client) {
			client, ok := cliusage.Identify(newRequest(userAgent, authorization, device))
			Expect(ok).To(BeTrue())
			Expect(client).To(Equal(expected))
		},
		Entry("release version with platform and device id",
			"bkms-cli/1.0.7 (linux/amd64)", "Bearer t", deviceID,
			cliusage.Client{Version: "1.0.7", OS: "linux", Arch: "amd64", DeviceID: deviceID}),
		Entry("pre-release version",
			"bkms-cli/1.1.0-rc.1 (darwin/arm64)", "", "",
			cliusage.Client{Version: "1.1.0-rc.1", OS: "darwin", Arch: "arm64"}),
		Entry("development build",
			"bkms-cli/dev (windows/amd64)", "", "",
			cliusage.Client{Version: cliusage.VersionDev, OS: "windows", Arch: "amd64"}),
		Entry("invalid version",
			"bkms-cli/v1.0 (linux/amd64)", "", "",
			cliusage.Client{Version: cliusage.VersionInvalid, OS: "linux", Arch: "amd64"}),
		Entry("too long version",
			"bkms-cli/1.0.0-"+strings.Repeat("a", 64), "", "",
			cliusage.Client{Version: cliusage.VersionInvalid}),
		Entry("malformed platform",
			"bkms-cli/1.0.7 (Linux; x86_64)", "", "",
			cliusage.Client{Version: "1.0.7"}),
		Entry("invalid device id",
			"bkms-cli/1.0.7 (linux/amd64)", "", "not-a-device-id",
			cliusage.Client{Version: "1.0.7", OS: "linux", Arch: "amd64", DeviceID: cliusage.DeviceIDInvalid}),
		Entry("uppercase device id",
			"bkms-cli/1.0.7 (linux/amd64)", "", strings.ToUpper(deviceID),
			cliusage.Client{Version: "1.0.7", OS: "linux", Arch: "amd64", DeviceID: cliusage.DeviceIDInvalid}),
		Entry("legacy cli with resty default user agent and bearer token",
			"go-resty/2.17.0 (https://github.com/go-resty/resty)", "Bearer t", "",
			cliusage.Client{Version: cliusage.VersionLegacy}),
	)

	DescribeTable("ignores other requests",
		func(userAgent, authorization string) {
			_, ok := cliusage.Identify(newRequest(userAgent, authorization, ""))
			Expect(ok).To(BeFalse())
		},
		Entry("browser", "Mozilla/5.0", ""),
		Entry("browser with bearer token", "Mozilla/5.0", "Bearer t"),
		Entry("resty without bearer token", "go-resty/2.17.0 (https://github.com/go-resty/resty)", ""),
		Entry("resty with non-bearer authorization", "go-resty/2.17.0", "Basic dXNlcjpwYXNz"),
		Entry("empty user agent", "", "Bearer t"),
	)
})
