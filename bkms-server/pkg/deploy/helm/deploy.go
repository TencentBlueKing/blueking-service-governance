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

// Package helm deploy provides deploy related functions.
package helm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/postrender"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkrepo"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm/postrenderer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/secret"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/arrangement"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/credential"
)

// helmReleaseNameMaxLen Helm Release 名称最大长度
const helmReleaseNameMaxLen = 53

// releaseNameHashSuffixLen 截断时追加的哈希后缀长度（含分隔符 '-'，格式为 -xxxxx）
const releaseNameHashSuffixLen = 6

// MaxHelmHistoryCount Helm Release 历史版本最大数量（受 Helm 存储限制）
const MaxHelmHistoryCount = 10

// PreviewHelmRelease 预览 Helm Release
func PreviewHelmRelease(
	ctx context.Context,
	bkciProjectStore bkci.ProjectStore,
	bkrepoProjectStore bkrepo.ProjectStore,
	credentialStore credential.HelmRepoCredentialStore,
	envVarsReader *envvars.UnifiedEnvVarsReader,
	app *bkmsapp.Application,
	env *bkmsenv.Environment,
	trafficLaneName, chartVersion, valuesFileID, imageTag string,
) (*PreviewResult, error) {
	// 解析 Helm 仓库配置
	sourceRepo, err := resolveRepoConfig(ctx, bkciProjectStore, bkrepoProjectStore, credentialStore, app)
	if err != nil {
		return nil, errors.Wrap(err, "resolve helm repo config")
	}

	// 生成部署环境信息
	spec, err := genHelmReleaseSpec(app, env, trafficLaneName, sourceRepo.ChartName)
	if err != nil {
		return nil, errors.Wrapf(err, "generate app %s helm release spec", app.ID)
	}

	// 获取 Values 配置（预览场景下敏感 env 变量渲染为掩码）
	values, missingVars, missingEnvVars, err := genHelmDeployValues(
		ctx, envVarsReader, app, env, valuesFileID, imageTag, true,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "generate release %s helm deploy values", spec.ReleaseName)
	}

	// 从源仓库拉取 Chart 并进行 Lint 校验
	chart, lintResult, err := PullChart(
		sourceRepo.RepoURL, spec.ChartName, chartVersion, sourceRepo.Username, sourceRepo.Password,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "pull chart %s version %s", spec.ChartName, chartVersion)
	}

	// 检查 Lint 校验结果
	if lintResult.HasErrors() {
		return nil, errors.Errorf("chart lint failed: %s", strings.Join(lintResult.Errors, "; "))
	}

	// 初始化 Helm SDK
	debugLog := helm.NewHelmDebugLogger(ctx, spec.ReleaseName, "preview")
	cfg, err := helm.NewActionConfiguration(spec.ClusterID, spec.Namespace, debugLog)
	if err != nil {
		return nil, errors.Wrapf(err, "init action configuration for preview %s", spec.ReleaseName)
	}

	// 获取当前已部署的 Manifest（如果存在）
	result := &PreviewResult{
		MissingVars:    missingVars,
		MissingEnvVars: missingEnvVars,
	}
	statusAction := action.NewStatus(cfg)
	currentRelease, err := statusAction.Run(spec.ReleaseName)
	if err == nil && currentRelease != nil {
		result.CurrentManifests = currentRelease.Manifest
	}

	// 构建 PostRenderer（组件 + 泳道）
	postRenderer, err := postrenderer.Build(ctx, app, env, trafficLaneName)
	if err != nil {
		return nil, errors.Wrapf(err, "build post renderer for preview %s", spec.ReleaseName)
	}

	// 解析 Values 字符串为 map 用于 Helm SDK
	valuesMap, err := ParseValuesString(values)
	if err != nil {
		return nil, errors.Wrapf(err, "parse values for preview %s", spec.ReleaseName)
	}

	// 通过 DryRun 获取目标 Manifest
	// 注意：Helm SDK 的 Upgrade.Install 标志只是信息性的，并不会在 release 不存在时自动执行 install，
	// 因此需要先判断 release 是否存在，不存在时使用 Install DryRun，存在时使用 Upgrade DryRun
	targetRelease, err := RunHelmRelease(cfg, spec.ReleaseName, spec.Namespace, chart, valuesMap, true, postRenderer)
	if err != nil {
		return nil, errors.Wrapf(err, "dry-run release %s", spec.ReleaseName)
	}
	result.TargetManifests = targetRelease.Manifest

	return result, nil
}

// UpgradeOrInstallHelmChart 更新/安装 Helm Chart
func UpgradeOrInstallHelmChart(
	ctx context.Context,
	bkciProjectStore bkci.ProjectStore,
	bkrepoProjectStore bkrepo.ProjectStore,
	credentialStore credential.HelmRepoCredentialStore,
	envVarsReader *envvars.UnifiedEnvVarsReader,
	app *bkmsapp.Application,
	env *bkmsenv.Environment,
	trafficLaneName, chartVersion, valuesFileID, imageTag string,
) (*DeployResult, error) {
	// 解析 Helm 仓库配置
	sourceRepo, err := resolveRepoConfig(ctx, bkciProjectStore, bkrepoProjectStore, credentialStore, app)
	if err != nil {
		return nil, errors.Wrap(err, "resolve helm repo config")
	}

	// 生成部署环境信息
	spec, err := genHelmReleaseSpec(app, env, trafficLaneName, sourceRepo.ChartName)
	if err != nil {
		return nil, errors.Wrapf(err, "generate app %s helm release spec", app.ID)
	}

	// 从源仓库拉取 Chart 并进行 Lint 校验
	chart, lintResult, err := PullChart(
		sourceRepo.RepoURL, spec.ChartName, chartVersion, sourceRepo.Username, sourceRepo.Password,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "pull chart %s version %s", spec.ChartName, chartVersion)
	}

	// 检查 Lint 校验结果
	if lintResult.HasErrors() {
		return nil, errors.Errorf("chart lint failed: %s", strings.Join(lintResult.Errors, "; "))
	}

	// 获取 Values 配置（正式部署下发真实 env 值，不掩码；忽略未定义变量提示）
	values, _, _, err := genHelmDeployValues(ctx, envVarsReader, app, env, valuesFileID, imageTag, false)
	if err != nil {
		return nil, errors.Wrapf(err, "generate release %s helm deploy values", spec.ReleaseName)
	}

	// 初始化 Helm SDK
	debugLog := helm.NewHelmDebugLogger(ctx, spec.ReleaseName, "deploy")
	cfg, err := helm.NewActionConfiguration(spec.ClusterID, spec.Namespace, debugLog)
	if err != nil {
		return nil, errors.Wrapf(err, "init action configuration for deploy %s", spec.ReleaseName)
	}

	// 构建 PostRenderer（组件 + 泳道）
	postRenderer, err := postrenderer.Build(ctx, app, env, trafficLaneName)
	if err != nil {
		return nil, errors.Wrapf(err, "build post renderer for %s", spec.ReleaseName)
	}

	// 解析 Values 字符串为 map
	valuesMap, err := ParseValuesString(values)
	if err != nil {
		return nil, errors.Wrapf(err, "parse values for deploy %s", spec.ReleaseName)
	}

	// 执行 Upgrade 或 Install
	// 注意：Helm SDK 的 Upgrade.Install 标志只是信息性的，并不会在 release 不存在时自动执行 install，
	// 因此需要先判断 release 是否存在，不存在时使用 Install，存在时使用 Upgrade
	rel, err := RunHelmRelease(cfg, spec.ReleaseName, spec.Namespace, chart, valuesMap, false, postRenderer)
	if err != nil {
		return nil, errors.Wrapf(err, "upgrade or install release %s", spec.ReleaseName)
	}

	return &DeployResult{
		ProjectCode:  spec.ProjectCode,
		ClusterID:    spec.ClusterID,
		Name:         spec.ReleaseName,
		Namespace:    spec.Namespace,
		Revision:     strconv.Itoa(rel.Version),
		Status:       rel.Info.Status.String(),
		Chart:        spec.ChartName,
		ChartVersion: chartVersion,
	}, nil
}

// ListHelmReleases 获取 Helm Release 历史
func ListHelmReleases(
	ctx context.Context,
	app *bkmsapp.Application,
	env *bkmsenv.Environment,
	trafficLaneName string,
) ([]*ReleaseHistory, error) {
	// 查询 Helm Release 历史只依赖 releaseName、集群和命名空间信息，不需要 ChartName 参与
	spec, err := genHelmReleaseSpec(app, env, trafficLaneName, "")
	if err != nil {
		return nil, errors.Wrapf(err, "generate app %s helm release spec", app.ID)
	}

	// 初始化 Helm SDK
	debugLog := helm.NewHelmDebugLogger(ctx, spec.ReleaseName, "history")
	cfg, err := helm.NewActionConfiguration(spec.ClusterID, spec.Namespace, debugLog)
	if err != nil {
		return nil, errors.Wrapf(err, "init action configuration for list releases %s", spec.ReleaseName)
	}

	// 通过 Helm SDK History action 获取 Release 历史
	historyAction := action.NewHistory(cfg)
	historyAction.Max = MaxHelmHistoryCount
	releases, err := historyAction.Run(spec.ReleaseName)
	if err != nil {
		// 如果 Release 不存在（尚未部署过），返回空列表而非报错
		if errors.Is(err, driver.ErrReleaseNotFound) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "get release %s history", spec.ReleaseName)
	}

	var result []*ReleaseHistory
	for _, rel := range releases {
		rh := &ReleaseHistory{
			Name:         rel.Name,
			Namespace:    rel.Namespace,
			Revision:     strconv.Itoa(rel.Version),
			Status:       rel.Info.Status.String(),
			Message:      rel.Info.Description,
			Chart:        rel.Chart.Metadata.Name,
			ChartVersion: rel.Chart.Metadata.Version,
			UpdatedAt:    rel.Info.LastDeployed.Time,
		}
		// 获取 Values（合并后的用户 Values）
		if rel.Config != nil {
			valuesStr, _ := marshalValues(rel.Config)
			rh.Values = valuesStr
		}
		result = append(result, rh)
	}

	return result, nil
}

// PreviewRollbackHelmRelease 预览 Helm Release 回滚的 Manifest
func PreviewRollbackHelmRelease(ctx context.Context, record *Record) (*PreviewResult, error) {
	// 初始化 Helm SDK
	debugLog := helm.NewHelmDebugLogger(ctx, record.ReleaseName, "preview-rollback")
	cfg, err := helm.NewActionConfiguration(record.ClusterID, record.Namespace, debugLog)
	if err != nil {
		return nil, errors.Wrapf(err, "init action configuration for preview rollback %s", record.ReleaseName)
	}

	result := &PreviewResult{}

	// 1. 获取当前已部署的 Manifest
	statusAction := action.NewStatus(cfg)
	currentRelease, err := statusAction.Run(record.ReleaseName)
	if err == nil && currentRelease != nil {
		result.CurrentManifests = currentRelease.Manifest
	}

	// 2. 获取目标版本的渲染后 Manifest
	revision, pErr := strconv.Atoi(record.Revision)
	if pErr != nil {
		return nil, errors.Wrapf(pErr, "parse revision %s", record.Revision)
	}
	getAction := action.NewGet(cfg)
	getAction.Version = revision
	targetRelease, gErr := getAction.Run(record.ReleaseName)
	if gErr != nil {
		return nil, errors.Wrapf(
			gErr, "get release %s revision %d manifest for preview", record.ReleaseName, revision,
		)
	}
	result.TargetManifests = targetRelease.Manifest

	return result, nil
}

// RollbackHelmRelease 回滚 Helm Release
func RollbackHelmRelease(ctx context.Context, record *Record) (*DeployResult, error) {
	// 初始化 Helm SDK
	debugLog := helm.NewHelmDebugLogger(ctx, record.ReleaseName, "rollback")
	cfg, err := helm.NewActionConfiguration(record.ClusterID, record.Namespace, debugLog)
	if err != nil {
		return nil, errors.Wrapf(err, "init action configuration for rollback %s", record.ReleaseName)
	}

	// 执行 Helm Release 回滚操作
	rollback := action.NewRollback(cfg)
	rollback.Version, err = strconv.Atoi(record.Revision)
	if err != nil {
		return nil, errors.Wrapf(err, "parse revision %s for rollback %s", record.Revision, record.ReleaseName)
	}
	if err = rollback.Run(record.ReleaseName); err != nil {
		return nil, errors.Wrapf(err, "rollback helm release %s to revision %s", record.ReleaseName, record.Revision)
	}

	// 获取最新的 Release 版本信息
	statusAction := action.NewStatus(cfg)
	rel, err := statusAction.Run(record.ReleaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "get release %s status after rollback", record.ReleaseName)
	}

	return &DeployResult{
		ProjectCode:  record.ProjectCode,
		ClusterID:    record.ClusterID,
		Name:         rel.Name,
		Namespace:    rel.Namespace,
		Revision:     strconv.Itoa(rel.Version),
		Status:       rel.Info.Status.String(),
		Chart:        rel.Chart.Metadata.Name,
		ChartVersion: rel.Chart.Metadata.Version,
	}, nil
}

// UninstallHelmRelease 卸载 Helm Release
func UninstallHelmRelease(ctx context.Context, record *Record) error {
	// 初始化 Helm SDK
	debugLog := helm.NewHelmDebugLogger(ctx, record.ReleaseName, "uninstall")
	cfg, err := helm.NewActionConfiguration(record.ClusterID, record.Namespace, debugLog)
	if err != nil {
		return errors.Wrapf(err, "init action configuration for uninstall %s", record.ReleaseName)
	}

	// 执行 Helm Release 卸载操作
	uninstall := action.NewUninstall(cfg)
	if _, err = uninstall.Run(record.ReleaseName); err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			return nil
		}
		return errors.Wrapf(err, "uninstall helm release %s", record.ReleaseName)
	}

	return nil
}

// RunHelmRelease 根据 Release 是否已存在选择 Install 或 Upgrade 模式执行操作
// dryRun 为 true 时仅渲染 Manifest 不实际部署，postRenderer 为可选的 PostRenderer
func RunHelmRelease(
	cfg *action.Configuration,
	releaseName, namespace string,
	chart *helmchart.Chart,
	values map[string]any,
	dryRun bool,
	postRenderer postrender.PostRenderer,
) (*release.Release, error) {
	// 检查指定的 Helm Release 是否存在（至少有一个版本记录）
	if _, err := action.NewHistory(cfg).Run(releaseName); err != nil {
		// Release 不存在，使用 Install
		instClient := action.NewInstall(cfg)
		instClient.CreateNamespace = true
		instClient.ReleaseName = releaseName
		instClient.Namespace = namespace
		instClient.DryRun = dryRun
		if postRenderer != nil {
			instClient.PostRenderer = postRenderer
		}
		return instClient.Run(chart, values)
	}

	// Release 已存在，使用 Upgrade
	upgradeClient := action.NewUpgrade(cfg)
	upgradeClient.Namespace = namespace
	upgradeClient.DryRun = dryRun
	if postRenderer != nil {
		upgradeClient.PostRenderer = postRenderer
	}
	return upgradeClient.Run(releaseName, chart, values)
}

// ParseValuesString 将 Values YAML 字符串解析为 map
func ParseValuesString(valuesStr string) (map[string]any, error) {
	if valuesStr == "" {
		return map[string]any{}, nil
	}
	var values map[string]any
	if err := yaml.Unmarshal([]byte(valuesStr), &values); err != nil {
		return nil, errors.Wrap(err, "unmarshal values yaml string")
	}
	if values == nil {
		return map[string]any{}, nil
	}
	return values, nil
}

// marshalValues 将 Values map 序列化为 YAML 字符串
func marshalValues(values map[string]any) (string, error) {
	if len(values) == 0 {
		return "", nil
	}
	data, err := yaml.Marshal(values)
	if err != nil {
		return "", errors.Wrap(err, "marshal values to yaml")
	}
	return string(data), nil
}

// genHelmReleaseSpec 生成部署的环境信息，当 trafficLaneName 为空时，表示不使用泳道功能
func genHelmReleaseSpec(
	app *bkmsapp.Application,
	env *bkmsenv.Environment,
	trafficLaneName string,
	chartName string,
) (*ReleaseSpec, error) {
	// 对 App 进行参数校验
	if err := validateApp(app); err != nil {
		return nil, errors.Wrap(err, "invalid app")
	}
	// 对 Env 进行参数校验
	if err := validateEnv(env); err != nil {
		return nil, errors.Wrap(err, "invalid env")
	}

	// 默认情况使用 EnvName & AppName 组合作为 Release 名称
	releaseName := fmt.Sprintf("%s-%s", env.Name, app.Name)
	// 如果有使用泳道功能，则 Release 名称也需要包含泳道信息
	if trafficLaneName != "" {
		releaseName = fmt.Sprintf("%s-%s-%s", env.Name, trafficLaneName, app.Name)
	}
	// 限制 Helm Release 名称长度
	// 当超长时，截断并追加原名称的哈希后缀以保证唯一性
	// https://github.com/helm/helm/blob/fc22b6df/pkg/chart/v2/util/validate_name.go#L60
	// https://helm.sh/docs/chart_template_guide/getting_started/#adding-a-simple-template-call
	if len(releaseName) > helmReleaseNameMaxLen {
		hash := sha256.Sum256([]byte(releaseName))
		hashSuffix := hex.EncodeToString(hash[:])[:releaseNameHashSuffixLen-1]
		releaseName = releaseName[:helmReleaseNameMaxLen-releaseNameHashSuffixLen] + "-" + hashSuffix
	}

	// 生成部署需要的配置
	return &ReleaseSpec{
		// 项目，集群，命名空间信息
		ProjectCode: env.Cluster.ProjectCode,
		ClusterID:   env.Cluster.ClusterID,
		Namespace:   env.Cluster.Namespace,
		ReleaseName: releaseName,
		// 默认使用项目的 Repo 仓库，即与 projectCode 同名
		ChartRepoName:   env.Cluster.ProjectCode,
		ChartName:       chartName,
		TrafficLaneName: trafficLaneName,
	}, nil
}

// genHelmDeployValues 生成应用部署使用的 Values（通过 Gonja 引擎渲染）。
//
// 渲染上下文同时包含 bkms（构建/镜像）与 env（应用环境变量）两个命名空间。
//
// Args:
//   - ctx 上下文
//   - envVarsReader 统一环境变量读取器
//   - app 应用信息
//   - env 环境信息
//   - appConfigFileID AppConfigFile ID，用于获取 Helm Values 模板内容
//   - imageTag 镜像标签，用于构建 bkms 命名空间变量
//   - maskSensitive 为 true 时（如部署预览），敏感 env 变量值渲染为掩码，避免明文暴露。
//
// Returns:
//   - helm values 渲染结果
//   - values 中引用但未定义的非 env 命名空间变量列表（以 "ns.var" 形式，如 bkms.BAR）
//   - values 中引用但未定义的 env 命名空间变量列表
//   - error 遇到的错误
func genHelmDeployValues(
	ctx context.Context,
	envVarsReader *envvars.UnifiedEnvVarsReader,
	app *bkmsapp.Application,
	env *bkmsenv.Environment,
	appConfigFileID, imageTag string,
	maskSensitive bool,
) (string, []string, []string, error) {
	// 1. 从数据库中获取 AppConfigFile
	store, err := appcfg.NewAppConfigFileStoreMongo(database.Client(), database.Name())
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "initial app config file store")
	}
	acfID, err := bson.ObjectIDFromHex(appConfigFileID)
	if err != nil {
		return "", nil, nil, errors.Wrapf(err, "invalid app config file id %s", appConfigFileID)
	}
	acf, err := store.GetByID(ctx, acfID)
	if err != nil {
		return "", nil, nil, errors.Wrapf(err, "get app config file %s", appConfigFileID)
	}

	// 2. 获取 AppConfigFile 内容
	editor, err := appcfg.NewAppConfigFileEditor(store, acf)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "creating app config file editor")
	}
	content, err := editor.GetCompiledContent(ctx)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "compiling app config file")
	}

	// 3. 生成 bkms 命名空间变量（构建/镜像相关）
	buildVars, err := genValuesVariables(ctx, app.WorkspaceID, app.ID, app.Name, imageTag)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "generate values variables")
	}

	// 4. 生成 env 命名空间变量（应用在该环境下生效的环境变量）
	envVars, err := buildEnvContext(ctx, envVarsReader, app, env, maskSensitive)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "build env context")
	}

	// 5. 收集 values 中引用但未定义的变量（覆盖 bkms 与 env 命名空间，不报错，仅用于预览提示）
	missingVars, missingEnvVars, err := collectMissingVars(content, map[string]map[string]string{
		string(render.ContextBkms): buildVars,
		string(render.ContextEnv):  envVars,
	})
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "collect missing vars")
	}

	// 6. 使用 Gonja 引擎渲染 ${{ bkms.* }} 与 ${{ env.* }} 模板
	renderer := render.New(render.SetBkmsContext(buildVars), render.SetEnvContext(envVars))
	rendered, err := renderer.Render(content)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "render helm values with gonja")
	}
	return rendered, missingVars, missingEnvVars, nil
}

// buildEnvContext 读取应用在指定环境下生效的环境变量，构建 env 命名空间渲染 map
// maskSensitive 为 true 时敏感变量值替换为掩码（用于预览，避免明文暴露）
func buildEnvContext(
	ctx context.Context,
	envVarsReader *envvars.UnifiedEnvVarsReader,
	app *bkmsapp.Application,
	env *bkmsenv.Environment,
	maskSensitive bool,
) (map[string]string, error) {
	envList, err := envVarsReader.ListVars(ctx, *env, app, nil)
	if err != nil {
		return nil, errors.Wrap(err, "list env vars")
	}
	envVars := make(map[string]string, len(envList))
	for _, v := range envList {
		if maskSensitive && v.IsSensitive {
			envVars[v.Key] = envvartypes.SensitiveValueMask
			continue
		}
		envVars[v.Key] = v.Value
	}
	return envVars, nil
}

// collectMissingVars 提取 values 中引用的所有 ${{ ns.var }} 变量，返回未在对应命名空间
// 上下文中定义的变量。env 命名空间仅返回变量名，其他命名空间以 "ns.var" 形式返回。
func collectMissingVars(content string, contexts map[string]map[string]string) ([]string, []string, error) {
	varsSet, err := render.ExtractVars(content)
	if err != nil {
		return nil, nil, errors.Wrap(err, "extract vars from helm values")
	}
	var missingVars []string
	var missingEnvVars []string
	for ns, names := range varsSet {
		ctxVars := contexts[ns]
		for name := range names {
			if _, ok := ctxVars[name]; ok {
				continue
			}
			if ns == string(render.ContextEnv) {
				missingEnvVars = append(missingEnvVars, name)
				continue
			}
			if ns == "" {
				missingVars = append(missingVars, name)
				continue
			}
			missingVars = append(missingVars, ns+"."+name)
		}
	}
	sort.Strings(missingVars)
	sort.Strings(missingEnvVars)
	return missingVars, missingEnvVars, nil
}

// genValuesVariables 生成 Helm 部署的 bkms 命名空间变量（用于 Gonja 引擎渲染 ${{ bkms.* }} 模板）
func genValuesVariables(ctx context.Context, workspaceID, appID, appName, imageTag string) (map[string]string, error) {
	buildCfgStore, err := build.NewConfigStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "initial build config store")
	}
	buildCfg, err := buildCfgStore.Get(ctx, appID)
	if err != nil {
		return nil, errors.Wrapf(err, "get build config for %s", appID)
	}

	// 解析镜像相关配置信息
	var registry, repository string
	if buildCfg.SourceType == build.SourceTypeCodeRepository {
		// 镜像来源是代码仓库时，使用内建镜像源（bkrepo），镜像名称即为 AppName
		imageRegistry, gErr := workspace.GetWorkspaceImageRegistry(ctx, workspaceID)
		if gErr != nil {
			return nil, errors.Wrapf(gErr, "get image registry for workspace %s", workspaceID)
		}
		registry, repository = imageRegistry.Registry, appName
	} else {
		// 镜像来源是镜像仓库时，需要解析镜像仓库的地址，可能格式如下：
		// 1. registry.example.com/group/repository
		// 2. group/repository
		// 3. repository
		registry, repository = "", buildCfg.Image.Name
		if strings.Contains(repository, "/") {
			// 按找到的第一个 `/` 分割为两部分
			before, after, _ := strings.Cut(repository, "/")
			// 再次检查 registry 是否包含 `.` 或 `:`（是否为合法域名/IP）
			if strings.Contains(before, ".") || strings.Contains(before, ":") {
				registry, repository = before, after
			}
		}
	}

	// 构建模板上下文数据：bkms 命名空间变量 map
	imageName := lo.Ternary(registry == "", repository, fmt.Sprintf("%s/%s", registry, repository))
	return map[string]string{
		arrangement.BkmsArtifactImage:           fmt.Sprintf("%s:%s", imageName, imageTag),
		arrangement.BkmsArtifactImageName:       imageName,
		arrangement.BkmsArtifactImageTag:        imageTag,
		arrangement.BkmsArtifactImageRegistry:   registry,
		arrangement.BkmsArtifactImageRepository: repository,
		arrangement.BkmsAppImagePullSecret:      secret.ResolveImagePullSecretName(workspaceID, appID, buildCfg),

		// TODO: 目前还没有域名相关配置，先使用固定值
		arrangement.BkmsNetworkingIngressDomain: "not-implemented.example.com",
	}, nil
}
