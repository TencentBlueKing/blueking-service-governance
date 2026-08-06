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
	"context"
	"strings"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"

	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
)

// GenerateReleaseName 生成 Helm Release 名称
func GenerateReleaseName(addonDef *ClusterAddonDef) string {
	if addonDef.ChartInfo.ReleaseName != "" {
		return addonDef.ChartInfo.ReleaseName
	}
	// 对于没有指定 ReleaseName 的，使用 ChartName 作为托底
	return addonDef.ChartInfo.ChartName
}

// InstallOrUpgradeClusterAddon 部署或更新集群 Addon
func InstallOrUpgradeClusterAddon(
	ctx context.Context,
	addonDef *ClusterAddonDef,
	clusterID, namespace, chartVersion string,
	valuesMap map[string]any,
) error {
	releaseName := GenerateReleaseName(addonDef)

	// 获取全局 Helm 仓库配置
	repoURL, repoUsername, repoPassword := GetConfigHelmRepoInfo()

	// 1. 从源仓库拉取 Chart 并进行 Lint 校验
	chart, lintResult, err := helmdeploy.PullChart(
		repoURL,
		addonDef.ChartInfo.ChartName,
		chartVersion,
		repoUsername,
		repoPassword,
	)
	if err != nil {
		return errors.Wrapf(err, "pull chart %s version %s", addonDef.ChartInfo.ChartName, chartVersion)
	}

	// 2. 检查 Lint 校验结果
	if lintResult.HasErrors() {
		return errors.Errorf("chart lint failed: %s", strings.Join(lintResult.Errors, "; "))
	}

	// 3. 初始化 Helm SDK
	debugLog := helm.NewHelmDebugLogger(ctx, releaseName, "cluster-addon-deploy")
	cfg, err := helm.NewActionConfiguration(clusterID, namespace, debugLog)
	if err != nil {
		return errors.Wrapf(err, "init action configuration for deploy %s", releaseName)
	}

	// 4. 执行 Upgrade 或 Install
	if _, err = helmdeploy.RunHelmRelease(cfg, releaseName, namespace, chart, valuesMap, false, nil); err != nil {
		return errors.Wrapf(err, "upgrade or install release %s", releaseName)
	}

	return nil
}

// UninstallClusterAddon 卸载集群 Addon
func UninstallClusterAddon(
	ctx context.Context,
	addonDef *ClusterAddonDef,
	clusterID, namespace string,
) error {
	releaseName := GenerateReleaseName(addonDef)

	// 初始化 Helm SDK
	debugLog := helm.NewHelmDebugLogger(ctx, releaseName, "cluster-addon-uninstall")
	cfg, err := helm.NewActionConfiguration(clusterID, namespace, debugLog)
	if err != nil {
		return errors.Wrapf(err, "init action configuration for uninstall %s", releaseName)
	}

	// 执行卸载操作
	uninstall := action.NewUninstall(cfg)
	if _, err = uninstall.Run(releaseName); err != nil {
		return errors.Wrapf(err, "uninstall helm release %s", releaseName)
	}

	return nil
}
