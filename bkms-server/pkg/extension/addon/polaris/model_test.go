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

package polaris_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
)

var _ = Describe("PolarisConfig", func() {
	Describe("IsAvailableInEnv", func() {
		Context("when ScopeType is empty (default, treated as environment)", func() {
			It("should return true only for environments in ScopeEnvNames", func() {
				// 空类型归入 environment 分支：仅对 ScopeEnvNames 内的环境生效
				config := &polaris.PolarisConfig{
					ScopeType:     "",
					ScopeEnvNames: []string{"dev"},
				}
				Expect(config.IsAvailableInEnv("dev")).To(BeTrue())
				Expect(config.IsAvailableInEnv("any-env")).To(BeFalse())
			})

			It("should return false when ScopeEnvNames is empty", func() {
				config := &polaris.PolarisConfig{ScopeType: ""}
				Expect(config.IsAvailableInEnv("any-env")).To(BeFalse())
			})
		})

		Context("when ScopeType is environment", func() {
			It("should return true only for specified environments", func() {
				config := &polaris.PolarisConfig{
					ScopeType:     component.ScopeTypeEnvironment,
					ScopeEnvNames: []string{"dev", "staging"},
				}
				Expect(config.IsAvailableInEnv("dev")).To(BeTrue())
				Expect(config.IsAvailableInEnv("staging")).To(BeTrue())
				Expect(config.IsAvailableInEnv("production")).To(BeFalse())
			})
		})
	})
})
