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

package accesspoint_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/accesspoint"
)

var _ = Describe("OperatorValues", func() {
	It("should point the operator to the first client host of the access point", func() {
		ap := &accesspoint.AccessPoint{
			ID:          "ap-id",
			Kind:        accesspoint.KindWorkspace,
			ClientHosts: []string{"10.0.0.1", "10.0.0.2"},
		}

		values := accesspoint.GetOperatorValues(ap, "BCS-K8S-26458", "bcs-project-id")

		Expect(values).To(Equal(map[string]any{
			"args": []any{
				map[string]any{"name": "polaris_api_type", "value": "HTTP_ADMIN"},
			},
			"env": map[string]any{
				"BK_BCS_clusterId":           "BCS-K8S-26458",
				"BK_projectId":               "bcs-project-id",
				"BK_BCS_polaris_api_path":    "10.0.0.1:8090",
				"BK_BCS_polaris_api_version": "v2",
				"BK_BCS_polaris_auth_mode":   "X-Polaris-Token",
			},
			"authProxy": map[string]any{"bcsPolarisPlatformID": ""},
		}))
	})

	It("should bracket an IPv6 client host", func() {
		ap := &accesspoint.AccessPoint{ClientHosts: []string{"fd00::1"}}

		values := accesspoint.GetOperatorValues(ap, "BCS-K8S-00001", "bcs-project-id")

		Expect(values).To(HaveKeyWithValue("env", HaveKeyWithValue("BK_BCS_polaris_api_path", "[fd00::1]:8090")))
	})

	It("should never contain the access point token", func() {
		ap := &accesspoint.AccessPoint{ClientHosts: []string{"polaris.example.com"}, Token: "secret-token-value"}

		values := accesspoint.GetOperatorValues(ap, "BCS-K8S-00001", "bcs-project-id")

		data, err := json.Marshal(values)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).NotTo(ContainSubstring("secret-token-value"))
	})
})
