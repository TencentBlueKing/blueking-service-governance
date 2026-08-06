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

package migrate_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render/migrate"
)

var _ = Describe("DraftSet YAML round-trip", func() {
	// 该测试只验证 yaml 序列化/反序列化保形：六种 Kind 各放一条 entry，
	// 经过 SaveDrafts → LoadDrafts 应当完全相等。
	// 不依赖任何 DB；纯文件 IO。
	It("preserves all six kinds verbatim through save/load", func() {
		original := &migrate.DraftSet{
			ComponentDefDefaultValues: []migrate.ComponentDefDefaultValueDraft{{
				Base: migrate.Base{
					Original:  `{{ .BKMS.ENV.X }}`,
					Converted: `${{env.X}}`,
				},
				Name:         "ResourceLimits",
				Version:      "v1.0.0",
				PropertyName: "cpu",
			}},
			AppModelComponentProperties: []migrate.AppModelComponentPropertyDraft{{
				Base: migrate.Base{
					Original:  `{{ .bkmsAppName }}`,
					Converted: `${{env.BKMS_APP_NAME}}`,
					Labels: migrate.Labels{
						AppName:       "demo-app",
						WorkspaceName: "default-ws",
						ComponentName: "polaris-abc12",
					},
				},
				AppID:         "app-1",
				ComponentName: "polaris-abc12",
				PropertyKey:   "instanceKey",
			}},
			WorkspaceComponentProperties: []migrate.WorkspaceComponentPropertyDraft{{
				Base: migrate.Base{
					Original:  `${{X}}`,
					Converted: `${{env.X}}`,
					Labels: migrate.Labels{
						WorkspaceName:     "ws-2",
						WorkspaceDisabled: true,
						ComponentName:     "limits-zzz99",
					},
				},
				ID:          "507f1f77bcf86cd799439011",
				PropertyKey: "memory",
			}},
			AppModelTafFileContents: []migrate.AppModelTafFileContentDraft{{
				Base: migrate.Base{
					Original:  `<taf>{{ .BKMS.ENV.A }}</taf>`,
					Converted: `<taf>${{env.A}}</taf>`,
				},
				AppID: "app-2",
			}},
			AppConfigFileTafs: []migrate.AppConfigFileTafDraft{{
				Base: migrate.Base{
					Original: `${{X}}`,
					// Converted 留空 + Error 非空，覆盖失败条目的存储路径
					Error: "needs manual migration",
				},
				ID:      "507f1f77bcf86cd799439012",
				Overlay: false,
			}, {
				Base: migrate.Base{
					Original:  `${{Y}}`,
					Converted: `${{env.Y}}`,
				},
				ID:      "507f1f77bcf86cd799439013",
				Overlay: true,
			}},
			AppConfigFileVersionTafs: []migrate.AppConfigFileVersionTafDraft{{
				Base: migrate.Base{
					Original:  `${{Z}}`,
					Converted: `${{env.Z}}`,
				},
				ID:      "507f1f77bcf86cd799439014",
				Overlay: true,
			}},
		}

		path := filepath.Join(GinkgoT().TempDir(), "drafts.yaml")
		Expect(migrate.SaveDrafts(path, original)).To(Succeed())

		loaded, err := migrate.LoadDrafts(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded).To(Equal(original))
	})

	It("returns empty DraftSet when file is missing", func() {
		// LoadDrafts 在文件不存在时不应报错，方便首次 apply 走 happy path。
		ds, err := migrate.LoadDrafts(filepath.Join(GinkgoT().TempDir(), "nope.yaml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(ds).To(Equal(&migrate.DraftSet{}))
	})
})

var _ = Describe("RunConvert", func() {
	It("skips text without templates", func() {
		got := migrate.RunConvert("hello world")
		Expect(got.Skip).To(BeTrue())
		Expect(got.Err).NotTo(HaveOccurred())
		Expect(got.Converted).To(BeEmpty())
	})

	It("skips when conversion is a no-op (already namespaced)", func() {
		// 已经是 ${{env.X}}，Convert 不会改写，触发 Skip 分支
		got := migrate.RunConvert(`${{env.X}}`)
		Expect(got.Skip).To(BeTrue())
		Expect(got.Converted).To(BeEmpty())
	})

	It("returns converted text on success", func() {
		got := migrate.RunConvert(`{{ .BKMS.ENV.X }}`)
		Expect(got.Skip).To(BeFalse())
		Expect(got.Err).NotTo(HaveOccurred())
		Expect(got.Converted).To(Equal(`${{env.X}}`))
	})

	It("surfaces convert errors", func() {
		// {{ if }} 是不支持的控制流，Convert 会回 ErrNeedsManual
		got := migrate.RunConvert(`{{ if .x }}y{{ end }}`)
		Expect(got.Skip).To(BeFalse())
		Expect(got.Err).To(HaveOccurred())
		Expect(got.Converted).To(BeEmpty())
	})
})
