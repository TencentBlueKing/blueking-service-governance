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

package cluster_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
)

const platformToken = "platform-token" //nolint:gosec // G101: 测试用的假 Token

// useBCSConfig 在当前用例内使用指定的全局配置与 Token 解析函数，用例结束后恢复
func useBCSConfig(baseURL string, allowWorkspaceClusterTokens bool, resolver cluster.TokenResolver) {
	origin := config.G
	config.G = &config.Config{BCS: config.BCSConfig{
		BaseUrl:                     baseURL,
		Token:                       platformToken,
		AllowWorkspaceClusterTokens: allowWorkspaceClusterTokens,
	}}
	cluster.SetTokenResolver(resolver)
	DeferCleanup(func() {
		config.G = origin
		cluster.SetTokenResolver(nil)
	})
}

var _ = Describe("NewConfig", func() {
	It("does not carry the platform token when workspace cluster tokens are enabled", func() {
		useBCSConfig("https://bcs.example.com", true, nil)
		Expect(cluster.NewConfig("BCS-K8S-00001").Rest.BearerToken).To(BeEmpty())
	})

	It("carries the platform token when workspace cluster tokens are disabled", func() {
		useBCSConfig("https://bcs.example.com", false, nil)
		Expect(cluster.NewConfig("BCS-K8S-00001").Rest.BearerToken).To(Equal(platformToken))
	})
})
