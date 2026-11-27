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

package clustertoken_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
)

func TestClusterToken(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cluster Token Suite")
}

var _ = BeforeSuite(func() {
	if err := testutil.SetUpGlobalDatabase(); err != nil {
		panic("failed to set up global database: " + err.Error())
	}
})

var _ = AfterSuite(func() {
	if err := testutil.TeardownGlobalDatabase(); err != nil {
		panic("failed to teardown global database: " + err.Error())
	}
})

// setBCSConfig 在当前用例内替换 config.G.BCS，用例结束后恢复
func setBCSConfig(cfg config.BCSConfig) {
	origin := config.G.BCS
	config.G.BCS = cfg
	DeferCleanup(func() { config.G.BCS = origin })
}

// bcsConfig 返回平台 Token 为 platform-token 的 BCS 配置
func bcsConfig(allowWorkspaceClusterTokens bool) config.BCSConfig {
	return config.BCSConfig{
		Token:                       "platform-token", //nolint:gosec // G101: 测试用的假 Token
		AllowWorkspaceClusterTokens: allowWorkspaceClusterTokens,
	}
}

// fakeBCS 返回固定集群列表的 BCS 云 API 客户端
type fakeBCS struct {
	*bcs.StubApiClient
	clusters []bcs.Cluster
}

func (f *fakeBCS) ListClustersByProject(context.Context, string) ([]bcs.Cluster, error) {
	return f.clusters, nil
}
