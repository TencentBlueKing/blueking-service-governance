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

	"github.com/pkg/errors"

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
	FileID              string         `json:"fileId,omitempty"`
	Type                string         `json:"type,omitempty"`
	EnvName             string         `json:"envName,omitempty"`
	ContentSourceType   string         `json:"contentSourceType,omitempty"`
	FileFormat          string         `json:"fileFormat,omitempty"`
	CurrentVersion      int64          `json:"currentVersion,omitempty"`
	BaseAppConfigFileID string         `json:"baseAppConfigFileID,omitempty"`
	BSCPConfig          *BSCPConfigObj `json:"bscpConfig,omitempty"`
	Content             *string        `json:"content,omitempty"`
	OverlayContent      *string        `json:"overlayContent,omitempty"`
}

type updateDefAuditContext struct {
	oldDef          appcfg.AppConfigFileDef
	defaultFile     *appcfg.AppConfigFile
	deletedEnvFiles []appcfg.AppConfigFile
}

type updateContentAuditContext struct {
	oldFile *appcfg.AppConfigFile
	opType  audit.OperationType
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
	data.Content = file.Content
	data.OverlayContent = file.OverlayContent
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
		audit.WithAttribute(audit.AttributeAppModel),
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

func (h *Handler) prepareUpdateDefAuditContext(
	ctx context.Context,
	def *appcfg.AppConfigFileDef,
	update appcfg.FileDefUpdate,
) (*updateDefAuditContext, error) {
	preview, err := h.newAppCfgFileDefService().PreviewDefUpdateImpact(ctx, def, update)
	if err != nil {
		return nil, errors.Wrap(err, "previewing def update impact")
	}
	oldDef := *def
	return &updateDefAuditContext{
		oldDef:          oldDef,
		defaultFile:     preview.DefaultFile,
		deletedEnvFiles: preview.DeletedEnvFiles,
	}, nil
}

func (h *Handler) auditDefUpdate(
	ctx context.Context,
	app *bkmsapp.Application,
	auditCtx *updateDefAuditContext,
	newDef *appcfg.AppConfigFileDef,
) {
	h.addAppConfigFileDefAudit(
		ctx,
		app,
		appcfg.EnvNameDefault,
		audit.OperationTypeUpdate,
		buildAppConfigFileDefAuditData(&auditCtx.oldDef, auditCtx.defaultFile),
		buildAppConfigFileDefAuditData(newDef, auditCtx.defaultFile),
	)
	for _, file := range auditCtx.deletedEnvFiles {
		fileCopy := file
		h.addAppConfigFileDefAudit(
			ctx,
			app,
			fileCopy.EnvName,
			audit.OperationTypeDelete,
			buildAppConfigFileDefAuditData(&auditCtx.oldDef, &fileCopy),
			nil,
		)
	}
}

func (h *Handler) prepareUpdateContentAuditContext(
	ctx context.Context,
	def *appcfg.AppConfigFileDef,
	envName string,
) (*updateContentAuditContext, error) {
	targetFile, isNewFile, err := h.newAppCfgFileDefService().FindContentUpdateTarget(ctx, def, envName)
	if err != nil {
		return nil, errors.Wrap(err, "finding content update target")
	}
	if isNewFile {
		return &updateContentAuditContext{opType: audit.OperationTypeCreate}, nil
	}
	oldCopy := *targetFile
	return &updateContentAuditContext{
		oldFile: &oldCopy,
		opType:  audit.OperationTypeUpdate,
	}, nil
}

func (h *Handler) auditContentUpsert(
	ctx context.Context,
	app *bkmsapp.Application,
	def *appcfg.AppConfigFileDef,
	auditCtx *updateContentAuditContext,
	file *appcfg.AppConfigFile,
) {
	var before any
	if auditCtx.oldFile != nil {
		before = buildAppConfigFileDefAuditData(def, auditCtx.oldFile)
	}
	h.addAppConfigFileDefAudit(
		ctx,
		app,
		file.EnvName,
		auditCtx.opType,
		before,
		buildAppConfigFileDefAuditData(def, file),
	)
}

func (h *Handler) auditEnvResetDelete(
	ctx context.Context,
	app *bkmsapp.Application,
	def *appcfg.AppConfigFileDef,
	file *appcfg.AppConfigFile,
) {
	if file == nil {
		return
	}
	h.addAppConfigFileDefAudit(
		ctx,
		app,
		file.EnvName,
		audit.OperationTypeDelete,
		buildAppConfigFileDefAuditData(def, file),
		nil,
	)
}
