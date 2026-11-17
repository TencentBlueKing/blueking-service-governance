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
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	devmode "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/taskqtask/devmoderestart"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
)

var (
	errRestartTargetRequired = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"EXTENSION_DEVMODE_RESTART_TARGET_REQUIRED",
		"no restart target specified",
	)
	errMountPathEmpty = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"EXTENSION_DEVMODE_MOUNT_PATH_EMPTY",
		"devmode mount path is empty",
	)
)

// restartEnvTarget 单个环境解析后的重启目标
type restartEnvTarget struct {
	env         *envmodel.Environment
	devModeSpec *appspec.DevModeSpec
	instanceIDs []string
}

// RestartDevModeProcesses 开发模式进程重启接口（异步）。
// 同步创建 pending 记录并投递任务后立即返回，由 worker 逐个实例执行 stop → 等待 monitor 拉起。
//
//	@ID			RestartDevModeProcesses
//	@Summary	开发模式进程重启
//	@Tags		devmode
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string				true	"应用 ID"
//	@Param		body	body		slz.RestartInput	true	"进程重启请求体"
//	@Success	202		{object}	slz.RestartOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/devmode/{appID}/restart [post]
func (h *Handler) RestartDevModeProcesses(c *gin.Context) {
	var uriInput slz.RestartURIInput
	var input slz.RestartInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if err := ginutils.BindJSON(c, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if err := input.Validate(); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()

	if err := checkClusterAPIConfigured(); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 解析重启目标
	envTargets, skipped, err := h.resolveRestartTargets(ctx, app, input)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if len(envTargets) == 0 {
		ginutils.JSON(c, http.StatusAccepted, buildRestartOutput(time.Now(), nil, skipped))
		return
	}

	// 获取 BCS 项目 Code，worker 执行时按用户身份重新获取 Token
	bcsProjectCode, err := h.getBCSProjectCode(ctx, app.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	requestedAt := time.Now()
	restarter := devmode.NewRestarter(store.Of(h.registry.DB, devmode.PublishRecordMeta))
	prepared, err := restarter.Prepare(
		ctx, app.ID, auth.MustGetUser(ctx).ID, buildRestartTargets(envTargets),
	)
	if err != nil {
		bkerrs.AbortWithErr(c, errors.Wrap(err, "prepare devmode restart records"))
		return
	}

	// 一次请求只投递一条任务消息，本次所有实例都在该消息中，由 worker 按最大 10 并发执行
	instances := lo.Map(prepared, func(item devmode.PreparedInstance, _ int) devmoderestart.Instance {
		return devmoderestart.Instance{
			RecordID:  item.RecordID.Hex(),
			EnvName:   item.Target.EnvName,
			Instance:  item.Instance,
			ClusterID: item.Target.ClusterID,
			Address:   item.Target.Address,
			Namespace: item.Target.Namespace,
			Container: item.Target.Container,
			MountPath: item.Target.MountPath,
		}
	})
	if err = devmoderestart.Enqueue(ctx, devmoderestart.Args{
		AppID:          app.ID,
		WorkspaceID:    app.WorkspaceID,
		BCSProjectCode: bcsProjectCode,
		Instances:      instances,
	}); err != nil {
		for _, item := range prepared {
			restarter.MarkFailed(ctx, item.RecordID, fmt.Sprintf("failed to enqueue restart task: %v", err))
		}
		bkerrs.AbortWithErr(c, errors.Wrap(err, "enqueue devmode restart task"))
		return
	}

	recordIDs := lo.Map(prepared, func(item devmode.PreparedInstance, _ int) string {
		return item.RecordID.Hex()
	})
	ginutils.JSON(c, http.StatusAccepted, buildRestartOutput(requestedAt, recordIDs, skipped))
}

// resolveRestartTargets 按目标类型解析重启目标。
// envs / instances 为严格模式，任一环境不满足条件即返回错误；
// envTypes 模式下不满足条件的环境被跳过，并返回跳过原因。
func (h *Handler) resolveRestartTargets(
	ctx context.Context, app *bkmsapp.Application, input slz.RestartInput,
) ([]*restartEnvTarget, []*slz.RestartSkippedEnv, error) {
	switch {
	case input.Envs != nil:
		targets, err := lo.MapErr(input.Envs.Names, func(envName string, _ int) (*restartEnvTarget, error) {
			return h.resolveRestartTargetByEnvName(ctx, app, envName, nil)
		})
		return targets, nil, err

	case input.Instances != nil:
		target, err := h.resolveRestartTargetByEnvName(
			ctx, app, input.Instances.EnvName, input.Instances.InstanceIDs,
		)
		if err != nil {
			return nil, nil, err
		}
		return []*restartEnvTarget{target}, nil, nil

	case input.EnvTypes != nil:
		return h.resolveRestartTargetsByEnvTypes(ctx, app, input.EnvTypes.Types)

	default:
		return nil, nil, errRestartTargetRequired
	}
}

// resolveRestartTargetsByEnvTypes 解析匹配环境类型的所有环境，不满足条件的环境被跳过并返回跳过原因
func (h *Handler) resolveRestartTargetsByEnvTypes(
	ctx context.Context, app *bkmsapp.Application, envTypes []string,
) ([]*restartEnvTarget, []*slz.RestartSkippedEnv, error) {
	envs, err := store.Of(h.registry.DB, envmodel.EnvMeta).
		List(ctx, envmodel.InWorkspace(app.WorkspaceID), envmodel.AvailableToApp(app.ID), envmodel.NewestFirst())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "list envs of app %s", app.ID)
	}
	matched := lo.Filter(envs, func(env envmodel.Environment, _ int) bool {
		return lo.Contains(envTypes, env.Type)
	})

	var targets []*restartEnvTarget
	var skipped []*slz.RestartSkippedEnv
	for i := range matched {
		env := &matched[i]
		target, err := h.resolveRestartTarget(ctx, app, env, nil)
		if err != nil {
			if !isSkippableRestartErr(err) {
				return nil, nil, err
			}
			skipped = append(
				skipped,
				&slz.RestartSkippedEnv{EnvName: env.Name, EnvType: env.Type, Reason: localizeSkipReason(ctx, err)},
			)
			continue
		}
		targets = append(targets, target)
	}
	return targets, skipped, nil
}

// localizeSkipReason 将被跳过环境的错误渲染为可读的本地化原因，无 detail code 或翻译失败时回退原文
func localizeSkipReason(ctx context.Context, err error) string {
	if bkErr := bkerrs.FromError(err); bkErr != nil && bkErr.DetailCode() != "" {
		if msg, ok := i18n.Localize(ctx, string(bkErr.DetailCode()), bkErr.Params()); ok {
			return msg
		}
	}
	return err.Error()
}

// isSkippableRestartErr 判断错误是否只是环境不适用；系统错误需直接返回
func isSkippableRestartErr(err error) bool {
	var bkErr *bkerrs.Error
	if !errors.As(err, &bkErr) {
		return false
	}
	switch bkErr.Code() {
	case bkerrs.ErrCodeInvalidArgument, bkerrs.ErrCodeInvalidRequest, bkerrs.ErrCodeNotFound:
		return true
	default:
		return false
	}
}

// resolveRestartTargetByEnvName 按环境名称解析重启目标
func (h *Handler) resolveRestartTargetByEnvName(
	ctx context.Context, app *bkmsapp.Application, envName string, instanceIDs []string,
) (*restartEnvTarget, error) {
	env, err := store.Of(h.registry.DB, envmodel.EnvMeta).
		Get(ctx, envmodel.AppEnvByName(app.WorkspaceID, app.ID, envName))
	if err != nil {
		if errors.Is(err, envmodel.ErrEnvNotFound) {
			return nil, envmodel.ErrEnvNotFound.With("envName", envName).WithCause(err)
		}
		return nil, errors.Wrapf(err, "get env %s by name", envName)
	}
	return h.resolveRestartTarget(ctx, app, env, instanceIDs)
}

// resolveRestartTarget 校验环境并解析目标实例；instanceIDs 为空时取所有 Running 实例
func (h *Handler) resolveRestartTarget(
	ctx context.Context, app *bkmsapp.Application, env *envmodel.Environment, instanceIDs []string,
) (*restartEnvTarget, error) {
	if err := checkEnvSupported(env); err != nil {
		return nil, err
	}
	devModeSpec, err := h.getEffectiveDevMode(ctx, app, env.Name)
	if err != nil {
		return nil, err
	}
	if lo.FromPtr(devModeSpec.MountPath) == "" {
		return nil, errMountPathEmpty.With("appID", app.ID).With("envName", env.Name)
	}
	ids, err := h.resolveInstanceIDs(ctx, app.ID, env.Name, instanceIDs, len(instanceIDs) == 0)
	if err != nil {
		return nil, err
	}
	return &restartEnvTarget{env: env, devModeSpec: devModeSpec, instanceIDs: ids}, nil
}

// buildRestartTargets 构造 Restarter 所需的重启目标
func buildRestartTargets(envTargets []*restartEnvTarget) []devmode.RestartTarget {
	return lo.Map(envTargets, func(target *restartEnvTarget, _ int) devmode.RestartTarget {
		clusterID := target.env.Cluster.ClusterID
		return devmode.RestartTarget{
			EnvName:     target.env.Name,
			ClusterID:   clusterID,
			Address:     buildClusterAddress(clusterID),
			Namespace:   target.env.Cluster.Namespace,
			Container:   defaults.WorkloadMainContainerName,
			MountPath:   lo.FromPtr(target.devModeSpec.MountPath),
			InstanceIDs: target.instanceIDs,
		}
	})
}

// buildRestartOutput 组装异步重启的提交结果
func buildRestartOutput(
	requestedAt time.Time, recordIDs []string, skipped []*slz.RestartSkippedEnv,
) *slz.RestartOutput {
	return &slz.RestartOutput{
		Data: &slz.RestartData{
			RequestedAt:   requestedAt,
			InstanceCount: len(recordIDs),
			RecordIDs:     lo.Ternary(recordIDs == nil, []string{}, recordIDs),
			Skipped:       lo.Ternary(skipped == nil, []*slz.RestartSkippedEnv{}, skipped),
		},
	}
}
