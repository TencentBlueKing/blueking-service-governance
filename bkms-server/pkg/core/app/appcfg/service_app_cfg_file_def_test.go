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

var _ = Describe("AppCfgFileDefService — Create / Update / Delete", func() {
	var f *defServiceFixture

	BeforeEach(func() {
		f = setupDefServiceFixture()
	})

	AfterEach(func() {
		f.DiApp.RequireStop()
	})

	Context("Create", func() {
		It("should create def, file and initial version", func() {
			result := f.createFrameworkFile("values.yaml")

			Expect(result.Def).NotTo(BeNil())
			Expect(result.Def.Name).To(Equal("values.yaml"))
			Expect(result.Def.ConfigKind).To(Equal(appcfg.ConfigKindFramework))
			Expect(result.Def.EnvConfigMode.IsUnifiedConfig).To(BeTrue())
			Expect(result.DefID).To(Equal(result.Def.ID))
			Expect(result.CurrentVersion).To(Equal(int64(1)))

			gotDef, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotDef.Name).To(Equal("values.yaml"))

			gotFile, err := f.FileStore.GetByID(f.Ctx, result.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotFile.DefID).To(Equal(result.Def.ID))
		})

		It("should reject empty config kind", func() {
			content := "key: value"
			_, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "legacy-values.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				Description:       "legacy init",
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("config kind is required"))
		})

		It("should reject invalid YAML content", func() {
			content := "invalid: [yaml"
			_, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "bad.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				ConfigKind:        appcfg.ConfigKindFramework,
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("content validation"))
		})

		It("should accept non-YAML content for plain kind", func() {
			content := "this is { not yaml ["
			result, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
				AppID:             f.AppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "plain.conf",
				MountDir:          "/etc/app",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
				ConfigKind:        appcfg.ConfigKindPlain,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Def.ConfigKind).To(Equal(appcfg.ConfigKindPlain))
		})
	})

	Context("UpdateAppCfgFileDef", func() {
		It("should update name", func() {
			result := f.createFrameworkFile("old-name.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			newName := "new-name.yaml"
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				Name:     &newName,
				Operator: "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			updated, err := f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Name).To(Equal("new-name.yaml"))
		})

		It("should reject mountDir update for framework kind", func() {
			result := f.createFrameworkFile("fw.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			newDir := "/new/path"
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				MountDir: &newDir,
				Operator: "editor",
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("mountDir"))
		})

		It("should reject mountDir update for unsupported config kind", func() {
			defID, err := f.DefStore.Add(f.Ctx, appcfg.AppConfigFileDef{
				AppID:      f.AppID,
				Name:       "legacy.yaml",
				ConfigKind: "",
				MountDir:   "/old/path",
				Creator:    "tester",
				EnvConfigMode: appcfg.EnvConfigMode{
					IsUnifiedConfig: true,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			def, err := f.DefStore.GetByID(f.Ctx, defID)
			Expect(err).NotTo(HaveOccurred())

			newDir := "/new/path"
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				MountDir: &newDir,
				Operator: "editor",
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, appcfg.ErrInvalidConfigSpec)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("unsupported config kind"))

			unchanged, err := f.DefStore.GetByID(f.Ctx, defID)
			Expect(err).NotTo(HaveOccurred())
			Expect(unchanged.MountDir).To(Equal("/old/path"))
		})

		It("should allow mountDir update for plain kind", func() {
			result := f.createPlainFile("editable.conf", "/old/path", "content")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			newDir := "/new/path"
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				MountDir: &newDir,
				Operator: "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			updated, err := f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.MountDir).To(Equal("/new/path"))
		})

		It("should reject update when def is nil", func() {
			err := f.Svc.UpdateAppCfgFileDef(f.Ctx, nil, appcfg.FileDefUpdate{
				Operator: "editor",
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("def is required"))
		})

		It("should clean up removed env instances when shrinking mountedEnvNames", func() {
			result := f.createPlainFile("shrink.conf", "/data", "k=v")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			// 挂载 prod + staging，切到独立配置
			isUnified := false
			mounted := []string{"prod", "staging"}
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				MountedEnvNames: &mounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			// 为 staging 创建环境实例
			prepared, _, _, err := f.Svc.PrepareEnvContentUpdate(f.Ctx, def, "staging", "key=staging", "editor")
			Expect(err).NotTo(HaveOccurred())
			stagingFile, err := f.Svc.CreateFileWithVersion(f.Ctx, *prepared, def.Name, "staging file", "editor")
			Expect(err).NotTo(HaveOccurred())

			// 缩减 mountedEnvNames 到只有 prod
			def, err = f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			newMounted := []string{"prod"}
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				MountedEnvNames: &newMounted,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			// staging 实例应已被清理
			_, err = f.FileStore.GetByID(f.Ctx, stagingFile.ID)
			Expect(err).To(HaveOccurred())

			// 默认文件仍存在
			_, err = f.FileStore.GetByID(f.Ctx, result.ID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should clean up env instances when switching to unified config", func() {
			result := f.createFrameworkFile("env-test.yaml")
			def, err := f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			// 先切换到独立配置
			isUnified := false
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			// 创建一个环境实例
			envContent := "env: prod"
			envFile := appcfg.AppConfigFile{
				DefID:   def.ID,
				AppID:   f.AppID,
				EnvName: "prod",
				Type:    appcfg.AppConfigFileTypeNormal,
				VersionedContent: appcfg.VersionedContent{
					ContentSourceType: appcfg.ContentSourceTypeLocal,
					Format:            appcfg.FileFormatYAML,
					Content:           &envContent,
				},
				Creator:        "tester",
				Updater:        "tester",
				CurrentVersion: 1,
			}
			envFileID, err := f.FileStore.Add(f.Ctx, envFile)
			Expect(err).NotTo(HaveOccurred())

			// 切回统一配置
			def, err = f.DefStore.GetByID(f.Ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			isUnified = true
			err = f.Svc.UpdateAppCfgFileDef(f.Ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			// 环境实例应已被删除
			_, err = f.FileStore.GetByID(f.Ctx, envFileID)
			Expect(err).To(HaveOccurred())

			// 默认实例仍然存在
			defaultFile, err := f.FileStore.GetByID(f.Ctx, result.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(defaultFile.EnvName).To(Equal(appcfg.EnvNameDefault))
		})
	})

	Context("DeleteFile (cascade)", func() {
		It("should delete default file along with its def and sibling env instances", func() {
			result := f.createFrameworkFile("cascade.yaml")

			// 添加一个环境实例
			envContent := "env: staging"
			envFile := appcfg.AppConfigFile{
				DefID:   result.Def.ID,
				AppID:   f.AppID,
				EnvName: "staging",
				Type:    appcfg.AppConfigFileTypeNormal,
				VersionedContent: appcfg.VersionedContent{
					ContentSourceType: appcfg.ContentSourceTypeLocal,
					Format:            appcfg.FileFormatYAML,
					Content:           &envContent,
				},
				Creator:        "tester",
				Updater:        "tester",
				CurrentVersion: 1,
			}
			envFileID, err := f.FileStore.Add(f.Ctx, envFile)
			Expect(err).NotTo(HaveOccurred())

			// 删除默认文件
			_, err = f.Svc.DeleteFile(f.Ctx, f.AppID, result.ID)
			Expect(err).NotTo(HaveOccurred())

			// 默认文件已删除
			_, err = f.FileStore.GetByID(f.Ctx, result.ID)
			Expect(err).To(HaveOccurred())

			// 环境实例已被级联删除
			_, err = f.FileStore.GetByID(f.Ctx, envFileID)
			Expect(err).To(HaveOccurred())

			// def 已被删除
			_, err = f.DefStore.GetByID(f.Ctx, result.Def.ID)
			Expect(err).To(HaveOccurred())
		})

		It("should only delete the env instance when deleting a non-default file", func() {
			result := f.createFrameworkFile("partial.yaml")
			defID := result.Def.ID

			envContent := "env: prod"
			envFile := appcfg.AppConfigFile{
				DefID:   defID,
				AppID:   f.AppID,
				EnvName: "prod",
				Type:    appcfg.AppConfigFileTypeNormal,
				VersionedContent: appcfg.VersionedContent{
					ContentSourceType: appcfg.ContentSourceTypeLocal,
					Format:            appcfg.FileFormatYAML,
					Content:           &envContent,
				},
				Creator:        "tester",
				Updater:        "tester",
				CurrentVersion: 1,
			}
			envFileID, err := f.FileStore.Add(f.Ctx, envFile)
			Expect(err).NotTo(HaveOccurred())

			// 仅删除环境实例
			_, err = f.Svc.DeleteFile(f.Ctx, f.AppID, envFileID)
			Expect(err).NotTo(HaveOccurred())

			// 环境实例已删除
			_, err = f.FileStore.GetByID(f.Ctx, envFileID)
			Expect(err).To(HaveOccurred())

			// 默认文件和 def 仍然存在
			_, err = f.FileStore.GetByID(f.Ctx, result.ID)
			Expect(err).NotTo(HaveOccurred())
			_, err = f.DefStore.GetByID(f.Ctx, defID)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
