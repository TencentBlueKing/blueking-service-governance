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
	"context"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	tafadmincmd "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/taf/admincmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/trpc/admincmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

// ListTrpcAdminCmds 查询 Trpc 管理命令。
//
//	@ID			ListTrpcAdminCmds
//	@Summary	查询 Trpc 管理命令
//	@Tags		instance
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		envName		path		string	true	"部署环境名称"
//	@Param		instanceIDs	query		[]string	true	"实例 ID 列表"
//	@Success	200			{object}	serializer.ListTrpcAdminCmdsOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/instances/admin-cmds [get]
func (h *Handler) ListTrpcAdminCmds(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var queryInput serializer.ListTrpcAdminCmdsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	// 校验 App 编辑权限
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	service, err := h.newTrpcAdminService(ctx, app, uriInput.EnvName, queryInput.InstanceIDs)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "new admin service failed"))
		return
	}
	// 预检查 admin 配置
	if err = service.Precheck(ctx); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.WrapTrpcAdminPrecheckFailed(err, app.ID, uriInput.EnvName))
		return
	}

	// 查询 admin cmds 列表
	results, err := service.ListTrpcAdminCmds(ctx)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list trpc admin cmds"))
		return
	}
	sort.Strings(results)
	ginutils.OK(c, serializer.ListTrpcAdminCmdsOutput{
		Data: &serializer.ListTrpcAdminCmdsOutputObjs{
			Count:   int64(len(results)),
			Results: results,
		},
	})
}

// ExecuteTrpcAdminCmd 执行 Trpc 管理命令。
//
//	@ID			ExecuteTrpcAdminCmd
//	@Summary	执行 Trpc 管理命令
//	@Tags		instance
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string								true	"应用 ID"
//	@Param		envName	path		string								true	"部署环境名称"
//	@Param		body	body		serializer.ExecuteTrpcAdminCmdInput	true	"执行 Trpc 管理命令请求"
//	@Success	200		{object}	serializer.ExecuteTrpcAdminCmdOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/instances/admin-cmds [post]
func (h *Handler) ExecuteTrpcAdminCmd(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var input serializer.ExecuteTrpcAdminCmdInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	// 校验 App 编辑权限
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	service, err := h.newTrpcAdminService(ctx, app, uriInput.EnvName, input.InstanceIDs)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "new admin service failed"))
		return
	}
	// 预检查 admin 配置
	if err = service.Precheck(ctx); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.WrapTrpcAdminPrecheckFailed(err, app.ID, uriInput.EnvName))
		return
	}

	// 执行 Admin 命令
	results, err := service.ExecuteTrpcAdminCmd(ctx, input.URL, input.Method, input.Body, input.Params)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "execute trpc admin cmd"))
		return
	}

	// 执行管理命令操作记录
	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx),
		audit.OperationTypeExecute,
		audit.ResourceTypeInstance,
		strings.Join(input.InstanceIDs, ","),
		audit.WithAttribute(audit.AttributeAdminCommand),
		audit.WithDataAfter(map[string]any{
			"url":    input.URL,
			"method": input.Method,
			"body":   input.Body,
			"params": input.Params,
		}),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
		audit.WithEnvName(uriInput.EnvName),
	)

	data := &serializer.ExecuteTrpcAdminCmdOutputObjs{
		Results: make([]*serializer.InstanceExecuteTrpcAdminCmdResultOutputObj, 0, len(results)),
	}
	for _, result := range results {
		data.Results = append(data.Results, &serializer.InstanceExecuteTrpcAdminCmdResultOutputObj{
			InstanceID: result.InstanceID,
			Success:    result.Success,
			Detail:     result.Detail,
		})
	}
	data.Count = int64(len(data.Results))
	ginutils.OK(c, serializer.ExecuteTrpcAdminCmdOutput{Data: data})
}

// ExecuteTafAdminCmd 执行 TAF 管理命令。
//
//	@ID			ExecuteTafAdminCmd
//	@Summary	执行 TAF 管理命令
//	@Tags		instance
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string							true	"应用 ID"
//	@Param		envName	path		string							true	"部署环境名称"
//	@Param		body	body		serializer.ExecuteTafAdminCmdInput	true	"执行 TAF 管理命令请求"
//	@Success	200		{object}	serializer.ExecuteTafAdminCmdOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/instances/taf-admin-cmds [post]
func (h *Handler) ExecuteTafAdminCmd(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var input serializer.ExecuteTafAdminCmdInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	// 校验 App 编辑权限
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	service, err := tafadmincmd.NewAdminService(
		app,
		uriInput.EnvName,
		input.InstanceIDs,
		&tafadmincmd.AdminServiceStores{
			TafDeployRecordStore: h.registry.AppModelDeployRecordStore,
			AppConfigFileStore:   h.registry.AppConfigFileStore,
			EnvStore:             h.registry.EnvStore,
			AppStore:             h.registry.AppStore,
			AppModelStore:        h.registry.AppModelStore,
			EnvVarsReader: envvars.NewUnifiedEnvVarsReader(
				h.registry.ScopedEnvVarStore,
				h.registry.AppDepsVarReader,
				h.registry.PolarisVarReader,
			),
		},
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "new taf admin service failed"))
		return
	}
	// 初始化服务
	if err = service.Init(ctx); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "init taf admin service failed"))
		return
	}

	// 执行 Admin 命令
	results, err := service.ExecuteTafAdminCmd(ctx, input.Command)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "execute taf admin cmd"))
		return
	}

	// 执行管理命令操作记录
	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx),
		audit.OperationTypeExecute,
		audit.ResourceTypeInstance,
		strings.Join(input.InstanceIDs, ","),
		audit.WithAttribute(audit.AttributeAdminCommand),
		audit.WithDataAfter(map[string]any{"command": input.Command}),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
		audit.WithEnvName(uriInput.EnvName),
	)

	data := &serializer.ExecuteTafAdminCmdOutputObjs{
		Results: make([]*serializer.InstanceExecuteTafAdminCmdResultOutputObj, 0, len(results)),
	}
	for _, result := range results {
		data.Results = append(data.Results, &serializer.InstanceExecuteTafAdminCmdResultOutputObj{
			InstanceID: result.InstanceID,
			Success:    result.Success,
			Detail:     result.Detail,
		})
	}
	data.Count = int64(len(data.Results))
	ginutils.OK(c, serializer.ExecuteTafAdminCmdOutput{Data: data})
}

// newTrpcAdminService 创建 Trpc 管理命令服务。
func (h *Handler) newTrpcAdminService(
	ctx context.Context,
	app *bkmsapp.Application,
	envName string,
	instanceIDs []string,
) (*admincmd.TrpcAdminService, error) {
	return admincmd.NewAdminService(
		ctx,
		app,
		envName,
		instanceIDs,
		h.registry.AppModelDeployRecordStore,
		h.registry.AppConfigFileStore,
		h.registry.EnvStore,
		h.registry.AppStore,
		h.registry.AppModelStore,
		h.registry.ScopedEnvVarStore,
		h.registry.AppDepsVarReader,
		h.registry.PolarisVarReader,
	)
}
