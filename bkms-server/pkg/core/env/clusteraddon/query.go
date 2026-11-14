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

package clusteraddon

import (
	"cmp"
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod/mods"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
)

// GetSupportedActions 根据当前安装状态返回支持的操作列表
func GetSupportedActions(status AddonStatus) []string {
	switch status {
	case "", helm.StatusUninstalled, helm.StatusNotFound:
		return []string{"install"}
	case helm.StatusDeployed:
		return []string{"upgrade", "uninstall"}
	case helm.StatusFailed:
		return []string{"install", "uninstall"}
	default:
		// pending-xxx 等中间状态，不允许操作
		return nil
	}
}

// FillAvailableVersions 从仓库 Index 填充可用版本列表
func (c *HelmChartInfo) FillAvailableVersions(repoIndex *RepoIndex) {
	versions := repoIndex.ListChartVersions(c.ChartName)
	c.AvailableVersions = versions
	if len(versions) > 0 {
		// 使用仓库中最新版本作为默认版本
		c.DefaultChartVersion = versions[0]
	}
}

// FillClusterStatus 从已查询的 Release 列表中匹配组件并填充安装信息。
func (info *ClusterAddonInfo) FillClusterStatus(releases []*helm.Release) {
	defer func() {
		info.SupportedActions = GetSupportedActions(info.InstallInfo.Status)
	}()

	release := helm.FindReleaseByChart(releases, info.ChartInfo.ChartName,
		cmp.Or(info.ChartInfo.ReleaseName, info.ChartInfo.ChartName))
	if release == nil {
		info.InstallInfo.Status = helm.StatusNotFound
		return
	}

	// 填充状态信息
	info.InstallInfo.Status = release.DeployResult.Status
	info.InstallInfo.Message = release.DeployResult.Description
	info.InstallInfo.CurrentChartVersion = release.Chart.Version
	if len(release.Values) > 0 {
		yamlData, mErr := yaml.Marshal(release.Values)
		if mErr == nil {
			info.InstallInfo.CurrentValues = string(yamlData)
		}
	}
}

// BuildAddonInfoList 根据插件定义列表、仓库 Index 和集群状态，构建完整的 ClusterAddonInfo 列表
func BuildAddonInfoList(
	ctx context.Context,
	addonDefs []*ClusterAddonDef,
	target ClusterTarget,
	namespace string,
	repoIndex *RepoIndex,
) []*ClusterAddonInfo {
	var addons []*ClusterAddonInfo
	for _, addonDef := range ApplicableAddonDefs(addonDefs, target) {
		ns := addonDef.GetNamespace(namespace)
		info := NewAddonInfoFromDef(addonDef, ns)
		info.ChartInfo.FillAvailableVersions(repoIndex)
		addons = append(addons, info)
	}
	// 每个 namespace 只查询一次；填充原列表中的对象，保持组件定义的顺序。
	addonsByNamespace := lo.GroupBy(addons, func(info *ClusterAddonInfo) string { return info.InstallInfo.Namespace })
	for ns, infos := range addonsByNamespace {
		releases, err := queryAddonReleases(ctx, target.ClusterID, ns)
		if err != nil {
			log.Warnf(ctx, "query addon status: %v", err)
			for _, info := range infos {
				info.InstallInfo.Status = helm.StatusUnknown
				info.InstallInfo.Message = err.Error()
			}
			continue
		}
		for _, info := range infos {
			info.FillClusterStatus(releases)
		}
	}

	return addons
}

// ApplicableAddonDefs 返回适用于目标集群的组件定义，供列表展示和部署检查共用。
func ApplicableAddonDefs(defs []*ClusterAddonDef, target ClusterTarget) []*ClusterAddonDef {
	return lo.Filter(defs, func(def *ClusterAddonDef, _ int) bool {
		return def.IsApplicableTo(target)
	})
}

// AddonAppTypesOf 返回应用匹配 requiredForAppTypes / optionalForAppTypes 时使用的取值，命中任一即匹配。
//
// AppModel 类应用（含旧 trpc / taf）按 bkmsApp 匹配；同时带上框架，兼容仍按 trpc / taf 登记的组件定义。
// Helm 类应用按类型匹配（helm / agones）。
func AddonAppTypesOf(app *bkmsapp.Application) []string {
	if !appruntime.IsAppModelType(app.Type) {
		return []string{app.Type}
	}
	if framework := mods.Of(app).Stack.Framework; framework != "" {
		return []string{bkmsapp.AppTypeBKMSApp, framework}
	}
	return []string{bkmsapp.AppTypeBKMSApp}
}

// InspectRequiredAddons 查询必选组件状态。缺失作为检查结果，查询失败作为错误返回。
// addonAppTypes 为 AddonAppTypesOf 的结果，不是应用类型。
// namespace 的取值与组件列表、安装和卸载接口保持一致：优先使用传入值，为空时取各组件的 defaultNamespace，再回退到 bcs-system。
// 当前部署预检和创建部署均传空值，保留按 namespace 复用查询，不因前端安装表单固定使用 bcs-system 而在此写死。
func InspectRequiredAddons(
	ctx context.Context,
	defs []*ClusterAddonDef,
	addonAppTypes []string,
	env *envmodel.Environment,
	namespace string,
) ([]AddonReference, error) {
	var missing []AddonReference
	// 同一 namespace 的 Release 列表在本次检查中只读取一次，包括空列表。
	releasesByNamespace := make(map[string][]*helm.Release)
	// 只检查适用于目标环境、且当前应用必选的组件。
	for _, def := range ApplicableAddonDefs(defs, TargetOfEnv(env)) {
		if !lo.Some(def.RequiredForAppTypes, addonAppTypes) {
			continue
		}
		addon := AddonReference{Name: def.Name, DisplayName: def.DisplayName}
		ns := def.GetNamespace(namespace)
		releases, ok := releasesByNamespace[ns]
		if !ok {
			var err error
			releases, err = queryAddonReleases(ctx, env.Cluster.ClusterID, ns)
			if err != nil {
				return nil, errors.Wrapf(err, "inspect required addon %s", def.Name)
			}
			releasesByNamespace[ns] = releases
		}
		// 按 Chart 匹配实际安装实例，优先同名 Release，否则取名称字典序最小的候选。
		release := helm.FindReleaseByChart(releases, def.ChartInfo.ChartName, GenerateReleaseName(def))
		// 无法确定状态时中止预检，避免将查询异常当成组件缺失。
		if release != nil && release.DeployResult.Status == helm.StatusUnknown {
			return nil, errors.Errorf("addon %s release %s status is unknown in cluster %s namespace %s",
				def.Name, release.Name, env.Cluster.ClusterID, ns)
		}
		// 只有 deployed 满足部署要求，未安装或其他已知状态都计入缺失列表。
		if release == nil || release.DeployResult.Status != helm.StatusDeployed {
			missing = append(missing, addon)
		}
	}
	return missing, nil
}

func queryAddonReleases(ctx context.Context, clusterID, namespace string) ([]*helm.Release, error) {
	debugLog := helm.NewHelmDebugLogger(ctx, "", "query-addon-status")
	cfg, err := helm.NewActionConfiguration(clusterID, namespace, debugLog)
	if err != nil {
		return nil, errors.Wrapf(err, "init helm configuration in cluster %s namespace %s", clusterID, namespace)
	}
	releases, err := helm.ListReleases(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "list addon releases in cluster %s namespace %s", clusterID, namespace)
	}
	return releases, nil
}
