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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

var _ = Describe("AppCfgFileDefService — Env View", func() {
	var f *defServiceFixture

	BeforeEach(func() {
		f = setupDefServiceFixture()
	})

	AfterEach(func() {
		f.DiApp.RequireStop()
	})

	Context("GetEnvFileDetail", func() {
		It("should return default file for unified config", func() {
			result := f.createFrameworkFile("values.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			detail, err := f.Svc.GetEnvFileDetail(f.Ctx, def, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(detail.DefaultFile).NotTo(BeNil())
			Expect(detail.DisplayFile).NotTo(BeNil())
			Expect(detail.DisplayFile.ID).To(Equal(detail.DefaultFile.ID))
			Expect(detail.DisplayFile.ID).To(Equal(result.ID))
			Expect(detail.HasEnvInstance).To(BeFalse())
			Expect(detail.EditableContentField).To(Equal("content"))
		})

		It("should return nil display file for framework overlay strategy without env instance", func() {
			result := f.createFrameworkFile("values.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			detail, err := f.Svc.GetEnvFileDetail(f.Ctx, def, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(detail.DefaultFile).NotTo(BeNil())
			Expect(detail.DisplayFile).To(BeNil())
			Expect(detail.HasEnvInstance).To(BeFalse())
			Expect(detail.EditableContentField).To(Equal("none"))
		})

		It("should return default file for plain overwrite strategy without env instance", func() {
			result := f.createPlainFile("app.conf", "/data", "k=v")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			mounted := []string{"prod"}
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			detail, err := f.Svc.GetEnvFileDetail(f.Ctx, def, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(detail.DefaultFile).NotTo(BeNil())
			Expect(detail.DisplayFile).NotTo(BeNil())
			Expect(detail.DisplayFile.ID).To(Equal(detail.DefaultFile.ID))
			Expect(detail.HasEnvInstance).To(BeFalse())
			Expect(detail.EditableContentField).To(Equal("content"))
		})

		It("should return env file when env instance exists", func() {
			result := f.createFrameworkFile("values.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			overlay := "env: prod\n"
			created, _, isNewFile, err := f.Svc.PrepareEnvContentUpdate(f.Ctx, def, "prod", overlay, "editor")
			Expect(err).NotTo(HaveOccurred())
			Expect(isNewFile).To(BeTrue())
			created, err = f.Svc.CreateFileWithVersion(f.Ctx, *created, def.Name, "prod overlay", "editor")
			Expect(err).NotTo(HaveOccurred())

			detail, err := f.Svc.GetEnvFileDetail(f.Ctx, def, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(detail.DisplayFile).NotTo(BeNil())
			Expect(detail.DisplayFile.ID).To(Equal(created.ID))
			Expect(detail.HasEnvInstance).To(BeTrue())
			Expect(detail.EditableContentField).To(Equal("overlayContent"))
			Expect(detail.BaseContentInfo).NotTo(BeNil())
			Expect(detail.BaseContentInfo.HolderID).To(Equal(result.ID))
			Expect(detail.BaseContentInfo.HolderName).To(Equal("values.yaml"))
			Expect(detail.BaseContentInfo.HolderContentSourceType).To(Equal("local"))
			Expect(detail.BaseContentInfo.IsFromAnotherFile).To(BeTrue())
			Expect(detail.BaseContentInfo.Content).To(ContainSubstring("key: value"))
		})

		It("should return error when plain file is not effective for env", func() {
			result := f.createPlainFile("app.conf", "/data", "k=v")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			mounted := []string{"prod"}
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Svc.GetEnvFileDetail(f.Ctx, def, "staging")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("not effective"))
		})

		It("should reject reading unmounted env even in unified config mode", func() {
			result := f.createPlainFile("unified-limited.conf", "/data", "k=v")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			// 保持统一配置，但限定只挂载 prod
			mounted := []string{"prod"}
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			// staging 不在挂载范围，即使是统一配置也应拒绝
			_, err = f.Svc.GetEnvFileDetail(f.Ctx, def, "staging")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("not effective"))

			// prod 在挂载范围内，应返回默认文件
			detail, err := f.Svc.GetEnvFileDetail(f.Ctx, def, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(detail.DisplayFile).NotTo(BeNil())
			Expect(detail.DisplayFile.ID).To(Equal(result.ID))
		})

		It("should return default file when querying default env name", func() {
			result := f.createFrameworkFile("values.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			detail, err := f.Svc.GetEnvFileDetail(f.Ctx, def, appcfg.EnvNameDefault)
			Expect(err).NotTo(HaveOccurred())
			Expect(detail.DefaultFile).NotTo(BeNil())
			Expect(detail.DisplayFile).NotTo(BeNil())
			Expect(detail.DisplayFile.ID).To(Equal(result.ID))
			Expect(detail.HasEnvInstance).To(BeFalse())
			Expect(detail.EditableContentField).To(Equal("content"))
		})
	})

	Context("GetMountPreview", func() {
		It("should include framework defs for any env", func() {
			f.createFrameworkFile("values.yaml")

			items, err := f.Svc.GetMountPreview(f.Ctx, f.AppID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
			Expect(items[0].Def.Name).To(Equal("values.yaml"))
			Expect(items[0].ContentSource).To(Equal(appcfg.AppConfigFileTypeNormal))
			Expect(items[0].HasEnvFile).To(BeFalse())
		})

		It("should include plain defs only for mounted envs", func() {
			result := f.createPlainFile("app.conf", "/data", "k=v")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			mounted := []string{"prod"}
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			items, err := f.Svc.GetMountPreview(f.Ctx, f.AppID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
			Expect(items[0].Def.Name).To(Equal("app.conf"))

			items, err = f.Svc.GetMountPreview(f.Ctx, f.AppID, "staging")
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(BeEmpty())
		})

		It("should mark HasEnvFile when env instance exists", func() {
			result := f.createFrameworkFile("values.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			overlay := "env: prod\n"
			created, _, _, err := f.Svc.PrepareEnvContentUpdate(f.Ctx, def, "prod", overlay, "editor")
			Expect(err).NotTo(HaveOccurred())
			_, err = f.Svc.CreateFileWithVersion(f.Ctx, *created, def.Name, "prod overlay", "editor")
			Expect(err).NotTo(HaveOccurred())

			items, err := f.Svc.GetMountPreview(f.Ctx, f.AppID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
			Expect(items[0].HasEnvFile).To(BeTrue())
		})

		It("should show overlay content source for framework env instance", func() {
			result := f.createFrameworkFile("values.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			overlay := "env: prod\n"
			created, _, _, err := f.Svc.PrepareEnvContentUpdate(f.Ctx, def, "prod", overlay, "editor")
			Expect(err).NotTo(HaveOccurred())
			_, err = f.Svc.CreateFileWithVersion(f.Ctx, *created, def.Name, "prod overlay", "editor")
			Expect(err).NotTo(HaveOccurred())

			items, err := f.Svc.GetMountPreview(f.Ctx, f.AppID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
			Expect(items[0].HasEnvFile).To(BeTrue())
			Expect(items[0].ContentSource).To(Equal(appcfg.AppConfigFileTypeOverlay))
		})

		It("should show overwrite content source for plain env instance", func() {
			result := f.createPlainFile("app.conf", "/data", "k=v")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			mounted := []string{"prod"}
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			created, _, _, err := f.Svc.PrepareEnvContentUpdate(f.Ctx, def, "prod", "key=prod", "editor")
			Expect(err).NotTo(HaveOccurred())
			_, err = f.Svc.CreateFileWithVersion(f.Ctx, *created, def.Name, "plain prod overwrite", "editor")
			Expect(err).NotTo(HaveOccurred())

			items, err := f.Svc.GetMountPreview(f.Ctx, f.AppID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
			Expect(items[0].HasEnvFile).To(BeTrue())
			Expect(items[0].ContentSource).To(Equal(appcfg.AppConfigFileType("overwrite")))
		})
	})
})
