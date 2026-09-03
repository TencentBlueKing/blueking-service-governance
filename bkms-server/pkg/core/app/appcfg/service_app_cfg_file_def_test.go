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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

var _ = Describe("AppCfgFileDefService", func() {
	var diApp *fxtest.App
	var appStore bkmsapp.ApplicationStore
	var fileStore appcfg.AppConfigFileStore
	var defStore appcfg.AppConfigFileDefStore
	var versionStore appcfg.AppConfigFileVersionStore
	var svc *appcfg.AppCfgFileDefService
	var ctx context.Context
	var appID string

	BeforeEach(func() {
		ctx = context.Background()
		Expect(testutil.CleanupCollection("app_config_file_versions")).To(Succeed())
		Expect(testutil.CleanupCollection("app_config_files")).To(Succeed())
		Expect(testutil.CleanupCollection("app_config_file_defs")).To(Succeed())
		Expect(testutil.CleanupCollection("applications")).To(Succeed())

		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			appcfg.FxModule,
			fx.Populate(&appStore, &fileStore, &defStore, &versionStore),
		)
		diApp.RequireStart()

		app := dbfactory.Application(ctx, appStore)
		appID = app.ID

		base := appcfg.NewBaseAppCfgFileService(defStore, fileStore, versionStore)
		svc = appcfg.NewAppCfgFileDefService(base, nil)
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	createFile := func(name string) *appcfg.AppConfigFileWithDef {
		content := "key: value"
		result, err := svc.Create(ctx, appcfg.CreateCfgFileParams{
			AppID:             appID,
			EnvName:           appcfg.EnvNameDefault,
			Name:              name,
			Type:              appcfg.AppConfigFileTypeNormal,
			ContentSourceType: appcfg.ContentSourceTypeLocal,
			Format:            appcfg.FileFormatYAML,
			Content:           &content,
			Creator:           "tester",
			Description:       "init",
		})
		Expect(err).NotTo(HaveOccurred())
		return result
	}

	Context("Create", func() {
		It("should create def, file and initial version", func() {
			result := createFile("values.yaml")

			Expect(result.Def).NotTo(BeNil())
			Expect(result.Def.Name).To(Equal("values.yaml"))
			Expect(result.Def.ConfigKind).To(Equal(appcfg.ConfigKindFramework))
			Expect(result.Def.EnvConfigMode.IsUnifiedConfig).To(BeTrue())
			Expect(result.DefID).To(Equal(result.Def.ID))
			Expect(result.CurrentVersion).To(Equal(int64(1)))

			gotDef, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotDef.Name).To(Equal("values.yaml"))

			gotFile, err := fileStore.GetByID(ctx, result.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotFile.DefID).To(Equal(result.Def.ID))
		})

		It("should reject invalid YAML content", func() {
			content := "invalid: [yaml"
			_, err := svc.Create(ctx, appcfg.CreateCfgFileParams{
				AppID:             appID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              "bad.yaml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           "tester",
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("content validation"))
		})
	})

	Context("UpdateFileDef", func() {
		It("should update name", func() {
			result := createFile("old-name.yaml")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			newName := "new-name.yaml"
			err = svc.UpdateFileDef(ctx, def, appcfg.FileDefUpdate{
				Name:     &newName,
				Operator: "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			updated, err := defStore.GetByID(ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Name).To(Equal("new-name.yaml"))
		})

		It("should reject mountDir update for framework kind", func() {
			result := createFile("fw.yaml")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			newDir := "/new/path"
			err = svc.UpdateFileDef(ctx, def, appcfg.FileDefUpdate{
				MountDir: &newDir,
				Operator: "editor",
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mountDir"))
		})

		It("should clean up env instances when switching to unified config", func() {
			result := createFile("env-test.yaml")
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			// 先切换到独立配置
			isUnified := false
			err = svc.UpdateFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			// 创建一个环境实例
			envContent := "env: prod"
			envFile := appcfg.AppConfigFile{
				DefID:   def.ID,
				AppID:   appID,
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
			envFileID, err := fileStore.Add(ctx, envFile)
			Expect(err).NotTo(HaveOccurred())

			// 切回统一配置
			def, err = defStore.GetByID(ctx, def.ID)
			Expect(err).NotTo(HaveOccurred())
			isUnified = true
			err = svc.UpdateFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			// 环境实例应已被删除
			_, err = fileStore.GetByID(ctx, envFileID)
			Expect(err).To(HaveOccurred())

			// 默认实例仍然存在
			defaultFile, err := fileStore.GetByID(ctx, result.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(defaultFile.EnvName).To(Equal(appcfg.EnvNameDefault))
		})
	})

	Context("DeleteFile (cascade)", func() {
		It("should delete default file along with its def and sibling env instances", func() {
			result := createFile("cascade.yaml")

			// 添加一个环境实例
			envContent := "env: staging"
			envFile := appcfg.AppConfigFile{
				DefID:   result.Def.ID,
				AppID:   appID,
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
			envFileID, err := fileStore.Add(ctx, envFile)
			Expect(err).NotTo(HaveOccurred())

			// 删除默认文件
			_, err = svc.DeleteFile(ctx, appID, result.ID)
			Expect(err).NotTo(HaveOccurred())

			// 默认文件已删除
			_, err = fileStore.GetByID(ctx, result.ID)
			Expect(err).To(HaveOccurred())

			// 环境实例已被级联删除
			_, err = fileStore.GetByID(ctx, envFileID)
			Expect(err).To(HaveOccurred())

			// def 已被删除
			_, err = defStore.GetByID(ctx, result.Def.ID)
			Expect(err).To(HaveOccurred())
		})

		It("should only delete the env instance when deleting a non-default file", func() {
			result := createFile("partial.yaml")
			defID := result.Def.ID

			envContent := "env: prod"
			envFile := appcfg.AppConfigFile{
				DefID:   defID,
				AppID:   appID,
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
			envFileID, err := fileStore.Add(ctx, envFile)
			Expect(err).NotTo(HaveOccurred())

			// 仅删除环境实例
			_, err = svc.DeleteFile(ctx, appID, envFileID)
			Expect(err).NotTo(HaveOccurred())

			// 环境实例已删除
			_, err = fileStore.GetByID(ctx, envFileID)
			Expect(err).To(HaveOccurred())

			// 默认文件和 def 仍然存在
			_, err = fileStore.GetByID(ctx, result.ID)
			Expect(err).NotTo(HaveOccurred())
			_, err = defStore.GetByID(ctx, defID)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
