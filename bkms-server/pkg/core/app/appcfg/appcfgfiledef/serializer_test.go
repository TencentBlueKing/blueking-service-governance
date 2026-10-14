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

package appcfgfiledef

import (
	"strings"

	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg/serializer" // register validators
)

var _ = Describe("DefDetailObj", func() {
	It("should include bscp config when filling from def and file", func() {
		defID := bson.NewObjectID()
		fileID := bson.NewObjectID()
		def := appcfg.AppConfigFileDef{
			ID:         defID,
			Name:       "values.yaml",
			ConfigKind: appcfg.ConfigKindFramework,
		}
		file := &appcfg.AppConfigFile{
			ID:    fileID,
			DefID: defID,
			Type:  appcfg.AppConfigFileTypeNormal,
			VersionedContent: appcfg.VersionedContent{
				ContentSourceType: appcfg.ContentSourceTypeBSCP,
				BSCPConfig: &appcfg.BSCPConfig{
					BizID:     "2",
					ServiceID: "3",
					ConfigID:  "4",
				},
			},
		}

		obj := new(DefDetailObj).FromDefAndFile(def, file)

		Expect(obj.BSCPConfig).NotTo(BeNil())
		Expect(*obj.BSCPConfig).To(Equal(BSCPConfigObj{
			BizID:     "2",
			ServiceID: "3",
			ID:        "4",
		}))
	})

	It("should keep default file bscp config when display file is nil", func() {
		defID := bson.NewObjectID()
		fileID := bson.NewObjectID()
		def := &appcfg.AppConfigFileDef{
			ID:         defID,
			Name:       "values.yaml",
			ConfigKind: appcfg.ConfigKindFramework,
		}
		defaultFile := &appcfg.AppConfigFile{
			ID:    fileID,
			DefID: defID,
			Type:  appcfg.AppConfigFileTypeNormal,
			VersionedContent: appcfg.VersionedContent{
				ContentSourceType: appcfg.ContentSourceTypeBSCP,
				BSCPConfig: &appcfg.BSCPConfig{
					BizID:     "20",
					ServiceID: "30",
					ConfigID:  "40",
				},
			},
		}

		obj := new(DefDetailObj).FromEnvFileDetail(&appcfg.EnvFileDetailResult{
			Def:         def,
			DefaultFile: defaultFile,
			DisplayFile: nil,
		})

		Expect(obj.FileID).To(Equal(fileID.Hex()))
		Expect(obj.BSCPConfig).NotTo(BeNil())
		Expect(*obj.BSCPConfig).To(Equal(BSCPConfigObj{
			BizID:     "20",
			ServiceID: "30",
			ID:        "40",
		}))
	})

	It("should not leak default file bscp config onto a local env instance", func() {
		defID := bson.NewObjectID()
		defaultFileID := bson.NewObjectID()
		envFileID := bson.NewObjectID()
		baseFileID := bson.NewObjectID()
		def := &appcfg.AppConfigFileDef{
			ID:         defID,
			Name:       "trpc_go.yaml",
			ConfigKind: appcfg.ConfigKindFramework,
		}
		defaultFile := &appcfg.AppConfigFile{
			ID:    defaultFileID,
			DefID: defID,
			Type:  appcfg.AppConfigFileTypeNormal,
			VersionedContent: appcfg.VersionedContent{
				ContentSourceType:   appcfg.ContentSourceTypeBSCP,
				BaseAppConfigFileID: &baseFileID,
				BSCPConfig: &appcfg.BSCPConfig{
					BizID:     "2",
					ServiceID: "3",
					ConfigID:  "4",
				},
			},
		}
		envFile := &appcfg.AppConfigFile{
			ID:      envFileID,
			DefID:   defID,
			EnvName: "prod",
			Type:    appcfg.AppConfigFileTypeOverlay,
			VersionedContent: appcfg.VersionedContent{
				ContentSourceType: appcfg.ContentSourceTypeLocal,
			},
		}

		obj := new(DefDetailObj).FromEnvFileDetail(&appcfg.EnvFileDetailResult{
			Def:            def,
			DefaultFile:    defaultFile,
			DisplayFile:    envFile,
			HasEnvInstance: true,
		})

		Expect(obj.FileID).To(Equal(envFileID.Hex()))
		Expect(obj.ContentSourceType).To(Equal("local"))
		Expect(obj.BSCPConfig).To(BeNil())
		Expect(obj.BaseAppConfigFileID).To(BeEmpty())
	})
})

var _ = Describe("mountDir binding", func() {
	It("should accept an absolute directory including root", func() {
		root := "/"
		err := binding.Validator.ValidateStruct(UpdateDefInput{MountDir: &root})
		Expect(err).NotTo(HaveOccurred())

		longDir := "/" + strings.Repeat("a", 255)
		err = binding.Validator.ValidateStruct(CreateDefInput{
			Name:              "app.conf",
			ConfigKind:        "plain",
			MountDir:          "/usr/local/trpc/bin/",
			FileFormat:        "yaml",
			FileType:          "normal",
			ContentSourceType: "local",
		})
		Expect(err).NotTo(HaveOccurred())

		err = binding.Validator.ValidateStruct(UpdateDefInput{MountDir: &longDir})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should reject a directory that does not start with slash", func() {
		relative := "etc/app"
		err := binding.Validator.ValidateStruct(UpdateDefInput{MountDir: &relative})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("mount_dir"))
	})
})

var _ = Describe("mountedEnvNames binding", func() {
	It("should accept env names and an empty list", func() {
		mounted := []string{"prod", "stag-1"}
		err := binding.Validator.ValidateStruct(CreateDefInput{
			Name:              "app.conf",
			ConfigKind:        "plain",
			FileFormat:        "yaml",
			FileType:          "normal",
			ContentSourceType: "local",
			MountedEnvNames:   &mounted,
		})
		Expect(err).NotTo(HaveOccurred())

		empty := []string{}
		err = binding.Validator.ValidateStruct(UpdateDefInput{MountedEnvNames: &empty})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should reject mounted env names that are not uri slugs", func() {
		mounted := []string{"bad env"}
		err := binding.Validator.ValidateStruct(UpdateDefInput{MountedEnvNames: &mounted})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("uri_slug"))
	})
})
