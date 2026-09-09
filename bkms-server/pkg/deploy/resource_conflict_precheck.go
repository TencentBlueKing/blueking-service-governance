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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
)

// resourceConflictPreCheckImage 预检构建时使用的占位镜像，不会真正拉取。
const resourceConflictPreCheckImage = "precheck.invalid/bkms:latest"

// ResourceConflictPreCheckResult 资源冲突预检结果，包含目标 namespace 中已存在的与本次部署资源同名的 K8s 资源列表。
type ResourceConflictPreCheckResult struct {
	ConflictResources []appmodeldeploy.ResourceKey
}

// ResourceConflictPreChecker 部署前 K8s 资源名称冲突检测器。
type ResourceConflictPreChecker struct {
	appModelStore  appmodel.AppModelStore
	builderService *workload.BuilderService
	recordStore    appmodeldeploy.RecordStore
}

// NewResourceConflictPreChecker 创建 ResourceConflictPreChecker。
func NewResourceConflictPreChecker(
	appModelStore appmodel.AppModelStore,
	builderService *workload.BuilderService,
	recordStore appmodeldeploy.RecordStore,
) *ResourceConflictPreChecker {
	return &ResourceConflictPreChecker{
		appModelStore:  appModelStore,
		builderService: builderService,
		recordStore:    recordStore,
	}
}

// Check 执行资源冲突预检：dry-run 构建 → 与最新部署记录对比提取新增资源 → 逐个探测 K8s 是否已存在。
func (c *ResourceConflictPreChecker) Check(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
) (*ResourceConflictPreCheckResult, error) {
	if app == nil || env == nil {
		return nil, errors.New("app and environment are required")
	}

	// dry-run 构建，提取本次部署会产生的所有资源 Key
	appModel, err := c.appModelStore.GetAppModel(ctx, app.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "get app %s model", app.ID)
	}
	modelForBuild := *appModel
	modelForBuild.Workload.Image = resourceConflictPreCheckImage

	buildResult, err := workload.NewBuilder(c.builderService, app, &modelForBuild).Build(ctx, env)
	if err != nil {
		return nil, errors.Wrap(err, "building workload for resource conflict pre-check")
	}
	currentKeys := BuildResourceKeys(buildResult.WorkloadKind, buildResult.Name(), buildResult.ExtraObjects)

	// 与最新部署记录对比，只保留新增的资源 Key；首次部署时检查全部
	keysToCheck := currentKeys
	record, err := c.recordStore.GetLatest(ctx, app.ID, env.Name, "")
	if err != nil && !errors.Is(err, appmodeldeploy.ErrDeployRecordNotFound) {
		return nil, errors.Wrap(err, "get latest deploy record for resource conflict pre-check")
	}
	if record != nil {
		keysToCheck = currentKeys.Diff(record.ResourceKeys)
	}

	// 逐个探测 K8s，返回已存在的资源
	conflicts, err := c.probeConflicts(ctx, env.Cluster.ClusterID, env.Cluster.Namespace, keysToCheck)
	if err != nil {
		return nil, err
	}
	return &ResourceConflictPreCheckResult{ConflictResources: conflicts}, nil
}

// probeConflicts 逐个探测 K8s，返回目标 namespace 中已存在的资源列表。
func (c *ResourceConflictPreChecker) probeConflicts(
	ctx context.Context,
	clusterID, namespace string,
	keys appmodeldeploy.ResourceKeys,
) ([]appmodeldeploy.ResourceKey, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	clusterCfg := cluster.NewConfig(clusterID)
	var conflicts []appmodeldeploy.ResourceKey

	for _, key := range keys {
		resGVR, err := discovery.GetGroupVersionResource(clusterCfg, key.Kind, "")
		if err != nil {
			return nil, errors.Wrapf(err, "get GVR for kind %s", key.Kind)
		}

		client := k8sclient.NewWithGVR(clusterCfg, *resGVR)
		if _, err = client.Get(ctx, namespace, key.Name, metav1.GetOptions{}); err != nil {
			if errors.Is(err, k8sclient.ErrResourceNotFound) {
				continue
			}
			return nil, errors.Wrapf(err, "check resource %s/%s", key.Kind, key.Name)
		}
		conflicts = append(conflicts, key)
	}
	return conflicts, nil
}

// BuildResourceKeys 从构建结果中提取资源 Key 列表。
// 导出供 Deployer 和预检器共用同一逻辑。
func BuildResourceKeys(
	workloadKind, workloadName string,
	extraObjs []unstructured.Unstructured,
) appmodeldeploy.ResourceKeys {
	resources := appmodeldeploy.ResourceKeys{{Kind: workloadKind, Name: workloadName}}
	for _, obj := range extraObjs {
		resources = append(resources, appmodeldeploy.ResourceKey{
			Kind: obj.GetKind(), Name: obj.GetName(),
		})
	}
	return resources
}
