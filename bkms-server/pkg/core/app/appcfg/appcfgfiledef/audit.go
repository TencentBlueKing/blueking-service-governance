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
	"context"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
)

type appConfigFileDefAuditData struct {
	ID                  string         `json:"id"`
	Name                string         `json:"name"`
	ConfigKind          string         `json:"configKind"`
	MountDir            string         `json:"mountDir,omitempty"`
	IsUnifiedConfig     bool           `json:"isUnifiedConfig"`
	MountedEnvNames     []string       `json:"mountedEnvNames,omitempty"`
	EnableEnvVarRender  bool           `json:"enableEnvVarRender"`
	EnvContentStrategy  string         `json:"envContentStrategy,omitempty"`
	FileID              string         `json:"fileId,omitempty"`
	Type                string         `json:"type,omitempty"`
	EnvName             string         `json:"envName,omitempty"`
	ContentSourceType   string         `json:"contentSourceType,omitempty"`
	FileFormat          string         `json:"fileFormat,omitempty"`
	CurrentVersion      int64          `json:"currentVersion,omitempty"`
	BaseAppConfigFileID string         `json:"baseAppConfigFileID,omitempty"`
	BSCPConfig          *BSCPConfigObj `json:"bscpConfig,omitempty"`
}

func buildAppConfigFileDefAuditData(
	def *appcfg.AppConfigFileDef,
	file *appcfg.AppConfigFile,
) *appConfigFileDefAuditData {
	data := &appConfigFileDefAuditData{
		ID:                 def.ID.Hex(),
		Name:               def.Name,
		ConfigKind:         string(def.ConfigKind),
		MountDir:           def.MountDir,
		IsUnifiedConfig:    def.EnvConfigMode.IsUnifiedConfig,
		MountedEnvNames:    def.EnvConfigMode.MountedEnvNames,
		EnableEnvVarRender: def.EnableEnvVarRender,
		EnvContentStrategy: string(def.EnvConfigMode.EnvContentStrategy),
	}

	if file == nil {
		return data
	}

	data.FileID = file.ID.Hex()
	data.Type = string(file.Type)
	data.EnvName = file.EnvName
	data.ContentSourceType = string(file.ContentSourceType)
	data.FileFormat = string(file.GetConfigFormat())
	data.CurrentVersion = file.CurrentVersion
	if file.BaseAppConfigFileID != nil {
		data.BaseAppConfigFileID = file.BaseAppConfigFileID.Hex()
	}
	if file.BSCPConfig != nil {
		data.BSCPConfig = &BSCPConfigObj{
			BizID:     file.BSCPConfig.BizID,
			ServiceID: file.BSCPConfig.ServiceID,
			ID:        file.BSCPConfig.ConfigID,
		}
	}
	return data
}

func (h *Handler) addAppConfigFileDefAudit(
	ctx context.Context,
	app *bkmsapp.Application,
	envName string,
	opType audit.OperationType,
	before any,
	after any,
) {
	opts := []audit.Option{
		audit.WithAttribute(audit.AttributeAppConfigFile),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	}
	if envName != "" {
		opts = append(opts, audit.WithEnvName(envName))
	}
	if before != nil {
		opts = append(opts, audit.WithDataBefore(before))
	}
	if after != nil {
		opts = append(opts, audit.WithDataAfter(after))
	}
	go audit.AddOperationRecordAsync(ctx, opType, audit.ResourceTypeApp, app.ID, opts...)
}

func (h *Handler) auditContentUpsert(
	ctx context.Context,
	app *bkmsapp.Application,
	def *appcfg.AppConfigFileDef,
	result *appcfg.UpsertEnvContentResult,
) {
	if result == nil || result.File == nil {
		return
	}
	opType := audit.OperationTypeUpdate
	if result.IsNewFile {
		opType = audit.OperationTypeCreate
	}
	var before any
	if result.OldFile != nil {
		before = buildAppConfigFileDefAuditData(def, result.OldFile)
	}
	h.addAppConfigFileDefAudit(
		ctx,
		app,
		result.File.EnvName,
		opType,
		before,
		buildAppConfigFileDefAuditData(def, result.File),
	)
}
