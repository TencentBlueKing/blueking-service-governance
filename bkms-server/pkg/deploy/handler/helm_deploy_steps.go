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

// Package handler 包含部署相关 Gin API 的 handler。
package handler

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	deploypkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	networkingdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/networking"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/secret"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

// attachHelmDeployRecordsValues 给部署记录添加 Values 相关信息。
//
// 通过 Helm SDK History action 获取 Release 历史，受 Helm 存储限制，历史久远的数据不支持。
func (h *Handler) attachHelmDeployRecordsValues(
	ctx context.Context,
	app *bkmsapp.Application,
	envName string,
	trafficLaneName string,
	records []*serializer.HelmDeployRecordOutputObj,
) error {
	// 1. 获取部署环境信息
	env, err := h.registry.EnvStore.GetByName(ctx, app.WorkspaceID, app.ID, envName)
	if err != nil {
		return errors.Wrapf(err, "get workspace %s env %s", app.WorkspaceID, envName)
	}
	// 2. 获取 Release Record 列表
	releases, err := helmdeploy.ListHelmReleases(ctx, app, env, trafficLaneName)
	if err != nil {
		deployInfo := genDeployInfo(app.WorkspaceID, app.ID, envName, trafficLaneName)
		return errors.Wrapf(err, "list helm releases for deploy %s", deployInfo)
	}
	// 3. 构建 Revision -> Values 的映射以便后续查询
	revisionValuesMap := map[string]string{}
	for _, rel := range releases {
		key := strings.Join([]string{rel.Chart, rel.ChartVersion, rel.Revision}, ":")
		revisionValuesMap[key] = rel.Values
	}
	// 4. 将 Values 信息添加到部署记录中
	for _, record := range records {
		key := strings.Join([]string{record.ChartName, record.ChartVersion, record.Revision}, ":")
		if values, ok := revisionValuesMap[key]; ok {
			record.Values = values
			// 按时间倒序遍历，如果版本 Values 已经关联，不能再次被关联
			// 目的是防止 Release 卸载重装后，旧部署拿的新的 Values 作展示
			delete(revisionValuesMap, key)
		}
	}
	return nil
}

// execHelmAppDeploySteps 执行 Helm 应用部署相关步骤，返回部署记录 ID。
func (h *Handler) execHelmAppDeploySteps(
	ctx context.Context,
	app *bkmsapp.Application,
	env *bkmsenv.Environment,
	trafficLaneName string,
	chartVersion string,
	valuesFileID string,
	imageTag string,
) (string, error) {
	deployInfo := genDeployInfo(app.WorkspaceID, app.ID, env.Name, trafficLaneName)
	// 1. 部署前检查
	if err := deploypkg.NewPreDeployChecker(
		h.registry.EnvStore, h.registry.PromotionStore, h.newSnapshotService(),
	).Do(ctx, &deploypkg.PreDeployCheckParams{
		WorkspaceID:     app.WorkspaceID,
		EnvName:         env.Name,
		TrafficLaneName: trafficLaneName,
		AppType:         app.Type,
		AppID:           app.ID,
		ImageTag:        imageTag,
	}); err != nil {
		return "", errors.Wrapf(err, "pre deploy check %s", deployInfo)
	}

	// 2. 确保部署环境存在相应的 ImagePullSecret
	buildCfg, err := h.registry.BuildConfigStore.Get(ctx, app.ID)
	if err != nil {
		return "", errors.Wrapf(err, "get build config for %s", deployInfo)
	}
	if err = secret.NewImagePullSecretSyncer(env, app.ID, buildCfg).Sync(ctx); err != nil {
		return "", errors.Wrapf(err, "ensure image pull secret for %s", deployInfo)
	}

	// 3. 同步 app services 到集群中
	services, err := h.registry.AppServiceStore.ListByApp(ctx, app.ID)
	if err != nil {
		return "", errors.Wrapf(err, "list app services for %s", deployInfo)
	}
	if err = networkingdeploy.NewServiceSyncer(env).Sync(ctx, app.ID, services); err != nil {
		return "", errors.Wrapf(err, "sync app services for %s", deployInfo)
	}

	// 4. 使用指定版本的 Chart 进行部署（通过 Helm SDK 直接部署，无需 Chart 转存）
	envVarsReader := envvars.NewUnifiedEnvVarsReader(
		h.registry.ScopedEnvVarStore,
		h.registry.AppDepsVarReader,
		h.registry.PolarisVarReader,
	)
	release, err := helmdeploy.UpgradeOrInstallHelmChart(
		ctx,
		h.registry.BkCIProjectStore,
		h.registry.BkRepoProjectStore,
		h.registry.HelmRepoCredentialStore,
		envVarsReader,
		app,
		env,
		trafficLaneName,
		chartVersion,
		valuesFileID,
		imageTag,
	)
	if err != nil {
		return "", errors.Wrapf(err, "deploy chart for %s", deployInfo)
	}

	// 5. 创建部署记录
	timeNow := time.Now()
	record := helmdeploy.Record{
		WorkspaceID:     app.WorkspaceID,
		AppID:           app.ID,
		EnvName:         env.Name,
		TrafficLaneName: trafficLaneName,
		Revision:        release.Revision,
		ProjectCode:     release.ProjectCode,
		ClusterID:       release.ClusterID,
		Namespace:       release.Namespace,
		ReleaseName:     release.Name,
		ChartName:       release.Chart,
		ChartVersion:    chartVersion,
		ValuesFileID:    valuesFileID,
		ImageTag:        imageTag,
		Status:          helm.StatusUnknown,
		Operator:        auth.MustGetUser(ctx).ID,
		StartedAt:       timeNow,
		CreatedAt:       timeNow,
		UpdatedAt:       timeNow,
	}
	recordID, err := h.registry.HelmDeployRecordStore.Create(ctx, &record)
	if err != nil {
		return "", errors.Wrapf(err, "create deploy record for %s", deployInfo)
	}

	return recordID, nil
}

// getDeployRecordForRollback 获取回滚用的部署记录。
func (h *Handler) getDeployRecordForRollback(ctx context.Context, appID, deployID string) (*helmdeploy.Record, error) {
	// 查询数据库获取部署记录
	record, err := h.registry.HelmDeployRecordStore.Get(ctx, appID, deployID)
	if err != nil {
		return nil, errors.Wrapf(err, "get app %s deploy record %s", appID, deployID)
	}

	// 检查指定回滚到的版本必须是成功部署的
	if record.Status != helm.StatusDeployed {
		return nil, errors.New("target revision for rollback not in deployed status")
	}
	return record, nil
}

// execHelmAppRollbackSteps 执行 Helm 应用回滚相关步骤，返回新的部署记录 ID。
func (h *Handler) execHelmAppRollbackSteps(ctx context.Context, record *helmdeploy.Record) (string, error) {
	deployInfo := genDeployInfo(record.WorkspaceID, record.AppID, record.EnvName, record.TrafficLaneName)
	// 1. 部署前检查
	if err := deploypkg.NewPreDeployChecker(
		h.registry.EnvStore, h.registry.PromotionStore, h.newSnapshotService(),
	).Do(ctx, &deploypkg.PreDeployCheckParams{
		WorkspaceID:     record.WorkspaceID,
		EnvName:         record.EnvName,
		TrafficLaneName: record.TrafficLaneName,
		AppType:         bkmsapp.AppTypeHelm,
		AppID:           record.AppID,
		ImageTag:        record.ImageTag,
	}); err != nil {
		return "", errors.Wrapf(err, "pre deploy check %s", deployInfo)
	}

	// 2. 根据部署记录回滚（用旧配置创建新的一次部署）
	release, err := helmdeploy.RollbackHelmRelease(ctx, record)
	if err != nil {
		return "", errors.Wrapf(err, "rollback deploy for %s", deployInfo)
	}

	// 3. 记录新的部署记录
	timeNow := time.Now()
	newRecord := helmdeploy.Record{
		WorkspaceID:     record.WorkspaceID,
		AppID:           record.AppID,
		EnvName:         record.EnvName,
		TrafficLaneName: record.TrafficLaneName,
		Revision:        release.Revision,
		ProjectCode:     release.ProjectCode,
		ClusterID:       release.ClusterID,
		Namespace:       release.Namespace,
		ReleaseName:     release.Name,
		ChartName:       record.ChartName,
		ChartVersion:    record.ChartVersion,
		ImageTag:        record.ImageTag,
		Status:          helm.StatusUnknown,
		Operator:        auth.MustGetUser(ctx).ID,
		StartedAt:       timeNow,
		CreatedAt:       timeNow,
		UpdatedAt:       timeNow,
	}
	recordID, err := h.registry.HelmDeployRecordStore.Create(ctx, &newRecord)
	if err != nil {
		return "", errors.Wrapf(err, "create deploy record for %s", deployInfo)
	}

	return recordID, nil
}
