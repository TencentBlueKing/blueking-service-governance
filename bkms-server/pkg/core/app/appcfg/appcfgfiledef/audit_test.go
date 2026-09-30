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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

var _ = Describe("buildAppConfigFileDefAuditData", func() {
	It("should include def and file fields for auditing", func() {
		defID := bson.NewObjectID()
		fileID := bson.NewObjectID()
		baseFileID := bson.NewObjectID()
		content := "plain content"
		def := &appcfg.AppConfigFileDef{
			ID:         defID,
			AppID:      "app1",
			Name:       "app.yaml",
			ConfigKind: appcfg.ConfigKindPlain,
			MountDir:   "/etc/app",
			EnvConfigMode: appcfg.EnvConfigMode{
				IsUnifiedConfig: false,
				MountedEnvNames: []string{"prod", "stag"},
			},
			EnableEnvVarRender: true,
		}
		file := &appcfg.AppConfigFile{
			ID:      fileID,
			DefID:   defID,
			AppID:   "app1",
			EnvName: appcfg.EnvNameDefault,
			Type:    appcfg.AppConfigFileTypeNormal,
			VersionedContent: appcfg.VersionedContent{
				ContentSourceType:   appcfg.ContentSourceTypeBSCP,
				BaseAppConfigFileID: &baseFileID,
				BSCPConfig: &appcfg.BSCPConfig{
					BizID:     "2",
					ServiceID: "3",
					ConfigID:  "4",
				},
				Content: &content,
			},
			CurrentVersion: 7,
		}

		data := buildAppConfigFileDefAuditData(def, file)

		Expect(data.ID).To(Equal(defID.Hex()))
		Expect(data.FileID).To(Equal(fileID.Hex()))
		Expect(data.ConfigKind).To(Equal("plain"))
		Expect(data.MountDir).To(Equal("/etc/app"))
		Expect(data.IsUnifiedConfig).To(BeFalse())
		Expect(data.MountedEnvNames).To(Equal([]string{"prod", "stag"}))
		Expect(data.EnableEnvVarRender).To(BeTrue())
		Expect(data.BaseAppConfigFileID).To(Equal(baseFileID.Hex()))
		Expect(*data.Content).To(Equal("plain content"))
		Expect(data.BSCPConfig).NotTo(BeNil())
		Expect(*data.BSCPConfig).To(Equal(BSCPConfigObj{
			BizID:     "2",
			ServiceID: "3",
			ID:        "4",
		}))
	})
})
