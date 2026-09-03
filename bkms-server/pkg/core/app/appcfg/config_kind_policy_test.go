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

package appcfg_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

var _ = Describe("FrameworkPolicy", func() {
	var policy appcfg.ConfigKindPolicy

	BeforeEach(func() {
		policy = appcfg.FrameworkPolicy{}
	})

	Describe("ValidateContent", func() {
		It("should accept empty content", func() {
			Expect(policy.ValidateContent("", appcfg.FileFormatYAML)).To(Succeed())
		})

		It("should accept valid YAML", func() {
			Expect(policy.ValidateContent("foo: bar\nlist:\n  - a\n  - b", appcfg.FileFormatYAML)).To(Succeed())
		})

		It("should reject invalid YAML", func() {
			err := policy.ValidateContent("foo: [invalid", appcfg.FileFormatYAML)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("YAML"))
		})

		It("should validate as YAML regardless of format parameter", func() {
			Expect(policy.ValidateContent("key: value", "unknown_format")).To(Succeed())
		})
	})

	Describe("GetEnvInstanceStrategy", func() {
		It("should return overlay strategy", func() {
			Expect(policy.GetEnvInstanceStrategy()).To(Equal(appcfg.EnvInstanceStrategyOverlay))
		})
	})

	Describe("IsAlwaysMount", func() {
		It("should always mount to all environments", func() {
			def := &appcfg.AppConfigFileDef{AppID: "app1", Name: "cfg.yaml"}
			Expect(policy.IsAlwaysMount(def, "prod")).To(BeTrue())
			Expect(policy.IsAlwaysMount(def, "dev")).To(BeTrue())
			Expect(policy.IsAlwaysMount(def, "")).To(BeTrue())
		})
	})

	Describe("AllowMountDirUpdate", func() {
		It("should not allow mount dir update", func() {
			Expect(policy.AllowMountDirUpdate()).To(BeFalse())
		})
	})
})

var _ = Describe("DefaultPolicies", func() {
	It("should contain framework policy", func() {
		p, ok := appcfg.DefaultPolicies[appcfg.ConfigKindFramework]
		Expect(ok).To(BeTrue())
		Expect(p).To(BeAssignableToTypeOf(appcfg.FrameworkPolicy{}))
	})
})
