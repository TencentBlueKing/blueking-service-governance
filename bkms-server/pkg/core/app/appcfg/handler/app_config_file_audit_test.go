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

package handler

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

func TestBuildAppConfigFileAuditDataIncludesBSCPConfig(t *testing.T) {
	t.Parallel()

	fileID := bson.NewObjectID()
	content := "content"
	data := buildAppConfigFileAuditData(&appcfg.AppConfigFile{
		ID:      fileID,
		EnvName: appcfg.EnvNameDefault,
		Type:    appcfg.AppConfigFileTypeNormal,
		VersionedContent: appcfg.VersionedContent{
			ContentSourceType: appcfg.ContentSourceTypeBSCP,
			Format:            appcfg.FileFormatYAML,
			Content:           &content,
			BSCPConfig: &appcfg.BSCPConfig{
				BizID:     "2",
				ServiceID: "3",
				ConfigID:  "4",
				VersionID: "5",
			},
		},
		CurrentVersion: 7,
	}, "values.yaml")

	bscpCfg, ok := data["bscpConfig"].(map[string]any)
	if !ok {
		t.Fatalf("expected bscpConfig in audit data, got %#v", data["bscpConfig"])
	}
	if bscpCfg["bizID"] != "2" ||
		bscpCfg["serviceID"] != "3" ||
		bscpCfg["configID"] != "4" ||
		bscpCfg["versionID"] != "5" {
		t.Fatalf("unexpected bscpConfig: %#v", bscpCfg)
	}
}
