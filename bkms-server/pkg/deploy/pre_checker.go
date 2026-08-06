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

package deploy

import (
	"context"

	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/trafficmanager"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/promotion"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// PreDeployCheckParams 部署前检查的请求参数
type PreDeployCheckParams struct {
	// 环境信息
	WorkspaceID     string
	EnvName         string
	TrafficLaneName string
	// 应用信息
	AppType string
	AppID   string
	// 镜像晋级检查所需参数
	ImageTag string
}

// PreDeployChecker 部署前检查器
type PreDeployChecker struct {
	envStore       envmodel.EnvironmentStore
	promotionStore promotion.PromotionStore
	snapshotSvc    *snapshot.Service
}

// NewPreDeployChecker 创建检查器，适用于 app model 应用 / Helm 应用等的部署前检查
// 目前支持的检查器有：
//   - 检查基线泳道是否已部署
//   - 检查部署到生产环境的镜像是否已晋级
func NewPreDeployChecker(
	envStore envmodel.EnvironmentStore,
	promotionStore promotion.PromotionStore,
	snapshotSvc *snapshot.Service,
) *PreDeployChecker {
	return &PreDeployChecker{
		envStore:       envStore,
		promotionStore: promotionStore,
		snapshotSvc:    snapshotSvc,
	}
}

// Do 逐个执行检查
func (c *PreDeployChecker) Do(ctx context.Context, params *PreDeployCheckParams) error {
	// 检查基线泳道是否已部署
	if err := c.checkIfBaselineLaneDeployed(ctx, params); err != nil {
		return errors.Wrapf(err, "check if baseline lane deployed")
	}
	// 检查镜像是否已晋级
	if err := c.checkIfImagePromoted(ctx, params); err != nil {
		return errors.Wrapf(err, "check if image promoted")
	}
	// 通过所有检查
	return nil
}

// checkIfImagePromoted 检查部署到生产环境的镜像是否已晋级
func (c *PreDeployChecker) checkIfImagePromoted(ctx context.Context, params *PreDeployCheckParams) error {
	if params == nil || params.ImageTag == "" {
		return errors.New("image tag is required")
	}

	// 查询目标环境信息，获取环境类型
	env, err := c.envStore.GetByName(ctx, params.WorkspaceID, params.AppID, params.EnvName)
	if err != nil {
		return errors.Wrapf(err, "get env %s in workspace %s", params.EnvName, params.WorkspaceID)
	}

	// 非生产环境跳过检查
	if !bkmsenv.IsProductionType(bkmsenv.Type(env.Type)) {
		return nil
	}

	// 获取应用对应的 repoKey
	repoInfo, err := c.snapshotSvc.ResolveRepoKeyForApp(ctx, params.AppID)
	if err != nil {
		return errors.Wrapf(err, "resolve repo key for app %s", params.AppID)
	}

	// 查询指定 tag 是否已晋级
	exists, err := c.promotionStore.IsTagPromoted(ctx, params.AppID, repoInfo.RepoKey, params.ImageTag)
	if err != nil {
		return errors.Wrapf(err, "check promotion exists for app %s tag %s", params.AppID, params.ImageTag)
	}

	if !exists {
		return errors.Errorf(
			"image tag %q has not been promoted to production, please promote it first", params.ImageTag,
		)
	}

	return nil
}

// checkIfBaselineLaneDeployed 检查基线是否已部署
func (c *PreDeployChecker) checkIfBaselineLaneDeployed(ctx context.Context, params *PreDeployCheckParams) error {
	// 如果没有指定泳道，说明没有启用泳道功能，无需检查
	if params == nil || params.TrafficLaneName == "" {
		return nil
	}

	// 如果指定了泳道，检查基线是否已部署
	baselineLane, err := trafficmanager.New().GetBaselineTrafficLane(ctx, params.WorkspaceID, params.EnvName)
	if err != nil {
		return errors.Wrapf(err, "get baseline lane (workspace: %s, env: %s)", params.WorkspaceID, params.EnvName)
	}
	// 如果没有指定泳道，说明没有启用泳道功能，无需检查
	if baselineLane == nil || baselineLane.LaneName == "" {
		return nil
	}
	// 待部署的泳道就是基线泳道，无需检查
	if params.TrafficLaneName == baselineLane.LaneName {
		return nil
	}

	switch params.AppType {
	case bkmsapp.AppTypeHelm, bkmsapp.AppTypeAgones:
		return helmdeploy.CheckIfTrafficLaneDeployed(ctx, params.AppID, params.EnvName, baselineLane.LaneName)
	case bkmsapp.AppTypeTRPC, bkmsapp.AppTypeTAF:
		return appmodeldeploy.CheckIfTrafficLaneDeployed(ctx, params.AppID, params.EnvName, baselineLane.LaneName)
	default:
		return errors.Errorf("pre deploy checker not support app type %s", params.AppType)
	}
}
