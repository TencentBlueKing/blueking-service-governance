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

var _ = Describe("AppCfgFileDefService — Env Instance", func() {
	var f *defServiceFixture

	BeforeEach(func() {
		f = setupDefServiceFixture()
	})

	AfterEach(func() {
		f.DiApp.RequireStop()
	})

	Context("FindEnvInstance", func() {
		It("should return nil when no env instance exists", func() {
			result := f.createFrameworkFile("values.yaml")

			found, err := f.Svc.FindEnvInstance(f.Ctx, result.Def.ID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeNil())
		})

		It("should return the env instance when it exists", func() {
			result := f.createFrameworkFile("values.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			// 切换到独立配置并创建环境实例
			isUnified := false
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			overlay := "env: prod\n"
			prepared, _, _, err := f.Svc.PrepareEnvContentUpdate(f.Ctx, def, "prod", overlay, "editor")
			Expect(err).NotTo(HaveOccurred())
			created, err := f.Svc.CreateFileWithVersion(f.Ctx, *prepared, def.Name, "prod overlay", "editor")
			Expect(err).NotTo(HaveOccurred())

			found, err := f.Svc.FindEnvInstance(f.Ctx, def.ID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).NotTo(BeNil())
			Expect(found.ID).To(Equal(created.ID))
		})
	})

	Context("CreateEnvInstance", func() {
		It("should create a framework overlay env instance", func() {
			result := f.createFrameworkFile("values.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			defaultFileWithDef, err := f.Svc.GetDefaultFileWithDef(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())

			overlay := "env: prod\n"
			created, err := f.Svc.CreateEnvInstance(f.Ctx, *defaultFileWithDef, appcfg.CreateEnvInstanceParams{
				EnvName:        "prod",
				OverlayContent: &overlay,
				Operator:       "editor",
				Description:    "create prod overlay",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(created.EnvName).To(Equal("prod"))
			Expect(created.Type).To(Equal(appcfg.AppConfigFileTypeOverlay))
			Expect(created.OverlayContent).NotTo(BeNil())
			Expect(*created.OverlayContent).To(Equal(overlay))
			Expect(created.BaseAppConfigFileID).NotTo(BeNil())
			Expect(*created.BaseAppConfigFileID).To(Equal(result.ID))
		})

		It("should create a plain overwrite env instance", func() {
			result := f.createPlainFile("app.conf", "/data", "key=default")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			// 挂载 prod 环境
			mounted := []string{"prod"}
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			defaultFileWithDef, err := f.Svc.GetDefaultFileWithDef(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())

			content := "key=prod"
			created, err := f.Svc.CreateEnvInstance(f.Ctx, *defaultFileWithDef, appcfg.CreateEnvInstanceParams{
				EnvName:     "prod",
				Content:     &content,
				Operator:    "editor",
				Description: "create prod overwrite",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(created.EnvName).To(Equal("prod"))
			Expect(created.Type).To(Equal(appcfg.AppConfigFileTypeNormal))
			Expect(created.Content).NotTo(BeNil())
			Expect(*created.Content).To(Equal(content))
		})

		It("should reject creating plain env instance for unmounted env", func() {
			result := f.createPlainFile("app.conf", "/data", "key=default")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			// 只挂载 prod，不挂载 staging
			mounted := []string{"prod"}
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			defaultFileWithDef, err := f.Svc.GetDefaultFileWithDef(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())

			content := "key=staging"
			_, err = f.Svc.CreateEnvInstance(f.Ctx, *defaultFileWithDef, appcfg.CreateEnvInstanceParams{
				EnvName:     "staging",
				Content:     &content,
				Operator:    "editor",
				Description: "should fail",
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
		})

		It("should reject creating framework overlay without overlayContent", func() {
			result := f.createFrameworkFile("values.yaml")

			defaultFileWithDef, err := f.Svc.GetDefaultFileWithDef(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Svc.CreateEnvInstance(f.Ctx, *defaultFileWithDef, appcfg.CreateEnvInstanceParams{
				EnvName:  "prod",
				Operator: "editor",
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
		})
	})

	Context("ResetEnvInstanceToDefault", func() {
		It("should delete the env instance and its versions", func() {
			result := f.createFrameworkFile("values.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			// 切到独立配置
			isUnified := false
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			// 创建环境实例
			overlay := "env: prod\n"
			prepared, _, _, err := f.Svc.PrepareEnvContentUpdate(f.Ctx, def, "prod", overlay, "editor")
			Expect(err).NotTo(HaveOccurred())
			created, err := f.Svc.CreateFileWithVersion(f.Ctx, *prepared, def.Name, "prod overlay", "editor")
			Expect(err).NotTo(HaveOccurred())

			// Reset
			err = f.Svc.ResetEnvInstanceToDefault(f.Ctx, def, "prod")
			Expect(err).NotTo(HaveOccurred())

			// 环境实例应已被删除
			_, err = f.FileStore.GetByID(f.Ctx, created.ID)
			Expect(err).To(HaveOccurred())

			// 默认文件仍存在
			_, err = f.FileStore.GetByID(f.Ctx, result.ID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject reset when config is unified", func() {
			result := f.createFrameworkFile("values.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			// 默认就是统一配置
			err = f.Svc.ResetEnvInstanceToDefault(f.Ctx, def, "prod")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrResetToDefaultRequiresIndependentConfig)).To(BeTrue())
		})

		It("should succeed even when no env instance exists", func() {
			result := f.createFrameworkFile("values.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			// 没有创建实例就直接 reset，应该幂等成功
			err = f.Svc.ResetEnvInstanceToDefault(f.Ctx, def, "prod")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("CleanupPlainEnvInstancesByEnv", func() {
		It("should delete plain env instances and remove env from mountedEnvNames", func() {
			result := f.createPlainFile("app.conf", "/data", "k=v")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			// 挂载到 prod + staging，切换到独立配置
			isUnified := false
			mounted := []string{"prod", "staging"}
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			// 为 prod 创建环境实例
			prepared, _, _, err := f.Svc.PrepareEnvContentUpdate(f.Ctx, def, "prod", "key=prod", "editor")
			Expect(err).NotTo(HaveOccurred())
			prodFile, err := f.Svc.CreateFileWithVersion(f.Ctx, *prepared, def.Name, "prod file", "editor")
			Expect(err).NotTo(HaveOccurred())

			// 清理 prod 环境
			err = f.Svc.CleanupPlainEnvInstancesByEnv(f.Ctx, f.AppID, "prod")
			Expect(err).NotTo(HaveOccurred())

			// prod 实例应已被删除
			_, err = f.FileStore.GetByID(f.Ctx, prodFile.ID)
			Expect(err).To(HaveOccurred())

			// mountedEnvNames 应移除 prod，只剩 staging
			updatedDef, err := f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedDef.EnvConfigMode.MountedEnvNames).To(Equal([]string{"staging"}))
		})

		It("should not affect framework defs", func() {
			f.createFrameworkFile("values.yaml")
			f.createPlainFile("app.conf", "/data", "k=v")

			// 清理不存在的环境应幂等成功，且不影响 framework
			err := f.Svc.CleanupPlainEnvInstancesByEnv(f.Ctx, f.AppID, "prod")
			Expect(err).NotTo(HaveOccurred())

			// framework def 不应被删除
			defs, err := f.DefStore.ListByApp(f.Ctx, f.AppID)
			Expect(err).NotTo(HaveOccurred())
			Expect(defs).To(HaveLen(2))
		})

		It("should no-op when no plain defs exist", func() {
			f.createFrameworkFile("values.yaml")

			err := f.Svc.CleanupPlainEnvInstancesByEnv(f.Ctx, f.AppID, "prod")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
