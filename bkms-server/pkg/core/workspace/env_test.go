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

package workspace_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
)

var _ = Describe("BuildDefaultEnvs", func() {
	var envs []envmodel.Environment

	BeforeEach(func() {
		envs = workspace.BuildDefaultEnvs("alice", "ws-1", "proj-1")
	})

	It("returns exactly 3 default environments", func() {
		Expect(envs).To(HaveLen(3))
	})

	DescribeTable(
		"each default env has the expected attributes",
		func(index int, name, displayName, envType string) {
			env := envs[index]
			Expect(env.Name).To(Equal(name))
			Expect(env.DisplayName).To(Equal(displayName))
			Expect(env.Type).To(Equal(envType))
			Expect(env.WorkspaceID).To(Equal("ws-1"))
			Expect(env.Cluster.ProjectCode).To(Equal("proj-1"))
			Expect(env.Creator).To(Equal("alice"))
		},
		Entry("test env", 0, "test", "测试环境", string(bkmsenv.TypeTest)),
		Entry("staging env", 1, "staging", "预发布环境", string(bkmsenv.TypeStaging)),
		Entry("production env", 2, "production", "正式环境", string(bkmsenv.TypeProduction)),
	)
})
