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

// Package cfgfile 基于配置文件定义（def）接口管理应用配置文件，同时支持 framework 和 plain 两类文件。
package cfgfile

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
)

const (
	// defaultEnvLabel 用于 CLI 展示默认文件，避免直接展示空环境名。
	defaultEnvLabel = "<default>"
	// allEnvsLabel 用于 CLI 展示「挂载到所有环境」。
	allEnvsLabel = "*"

	envModeUnified = "unified"
	envModePerEnv  = "per-env"

	editableContentFieldOverlayContent = "overlayContent"
)

// FileRef 定位一个配置文件定义：按 ID，或按文件名（同名文件位于不同目录时，再用挂载目录区分）。
type FileRef struct {
	// ID 配置文件定义 ID，与 Name 二选一。
	ID string
	// Name 文件名。
	Name string
	// MountDir 挂载目录，可选，用于区分同名文件。
	MountDir string
}

// ResolveDef 按 FileRef 在应用的配置文件定义中唯一定位一个文件。
func ResolveDef(ctx context.Context, cli client.Client, appID string, ref FileRef) (*client.AppConfigFileDef, error) {
	if ref.ID == "" && ref.Name == "" {
		return nil, errors.New("one of config file ID or name is required")
	}

	defs, err := cli.ListAppConfigFileDefs(ctx, appID)
	if err != nil {
		return nil, errors.Wrap(err, "list app config files")
	}

	matches := lo.Filter(defs, func(def client.AppConfigFileDef, _ int) bool {
		if ref.ID != "" {
			return def.ID == ref.ID
		}
		if def.Name != ref.Name {
			return false
		}
		return ref.MountDir == "" || path.Clean(def.MountDir) == path.Clean(ref.MountDir)
	})
	switch len(matches) {
	case 0:
		return nil, errors.Errorf(
			"config file %s not found in app %s, available: %s", ref, appID, formatDefCandidates(defs),
		)
	case 1:
		return &matches[0], nil
	default:
		return nil, errors.Errorf(
			"multiple config files match %s, use --mount-dir or --id to select one: %s",
			ref, formatDefCandidates(matches),
		)
	}
}

// String 返回用于错误提示的文件定位描述。
func (r FileRef) String() string {
	if r.ID != "" {
		return fmt.Sprintf("id=%s", r.ID)
	}
	if r.MountDir != "" {
		return fmt.Sprintf("name=%s mountDir=%s", r.Name, r.MountDir)
	}
	return fmt.Sprintf("name=%s", r.Name)
}

// fileTarget 是某个配置文件在指定环境下实际生效的文件。
type fileTarget struct {
	// Def 配置文件定义（来自列表接口，带默认文件信息）。
	Def *client.AppConfigFileDef
	// Detail 指定环境下的详情，FileID / CurrentVersion 指向实际生效的文件。
	Detail *client.AppConfigFileDef
	// EnvName 环境名，空字符串表示默认文件。
	EnvName string
}

// SharesDefault 表示指定了环境，但该环境没有独立实例，生效的是默认文件。
func (t *fileTarget) SharesDefault() bool {
	return t.EnvName != "" && !lo.FromPtr(t.Detail.HasEnvInstance)
}

func formatDefCandidates(defs []client.AppConfigFileDef) string {
	if len(defs) == 0 {
		return "<none>"
	}
	candidates := lo.Map(defs, func(def client.AppConfigFileDef, _ int) string {
		return fmt.Sprintf("%s(mountDir: %s, id: %s)", def.Name, lo.CoalesceOrEmpty(def.MountDir, "-"), def.ID)
	})
	return strings.Join(candidates, ", ")
}

// listAppEnvNames 返回应用可见的环境名列表。
func listAppEnvNames(ctx context.Context, cli client.Client, appID string) ([]string, error) {
	envs, err := cli.ListAppEnvs(ctx, appID)
	if err != nil {
		return nil, errors.Wrap(err, "list app envs")
	}
	return lo.Map(envs, func(env client.Env, _ int) string { return env.Name }), nil
}

// ensureAppEnv 校验环境属于应用可见环境。
// 后端在文件挂载到所有环境时不会校验环境是否存在，写错环境名会静默生效，因此由 CLI 先行校验。
func ensureAppEnv(ctx context.Context, cli client.Client, appID, envName string) error {
	envNames, err := listAppEnvNames(ctx, cli, appID)
	if err != nil {
		return err
	}
	if !slices.Contains(envNames, envName) {
		return errors.Errorf(
			"env %s not found in app %s, available: %s", envName, appID, strings.Join(envNames, ", "),
		)
	}
	return nil
}

// ensureEnvConfigurable 校验应用支持按环境配置（Helm 类应用没有环境级配置文件），且环境属于该应用。
func ensureEnvConfigurable(ctx context.Context, cli client.Client, appID, envName string) error {
	app, err := cli.GetApp(ctx, appID)
	if err != nil {
		return errors.Wrap(err, "get app")
	}
	if app.Type == constant.AppTypeHelm || app.Type == constant.AppTypeAgones {
		return errors.Errorf("%s app %s has no per-environment config files, omit --env", app.Type, appID)
	}
	return ensureAppEnv(ctx, cli, appID, envName)
}

// ensureDefMountedOnEnv 校验 plain 文件挂载到了指定环境；framework 文件始终挂载到所有环境。
func ensureDefMountedOnEnv(def *client.AppConfigFileDef, envName string) error {
	if def.ConfigKind != client.ConfigKindPlain || len(def.MountedEnvNames) == 0 {
		return nil
	}
	if !slices.Contains(def.MountedEnvNames, envName) {
		return errors.Errorf(
			"config file %s is not mounted on env %s (mounted envs: %s), use 'update --mount-envs' to change",
			def.Name, envName, strings.Join(def.MountedEnvNames, ","),
		)
	}
	return nil
}

// resolveMountEnvNames 校验并去重用户指定的挂载环境。
func resolveMountEnvNames(ctx context.Context, cli client.Client, appID string, names []string) ([]string, error) {
	envNames, err := listAppEnvNames(ctx, cli, appID)
	if err != nil {
		return nil, err
	}
	names = lo.Uniq(lo.Compact(lo.Map(names, func(name string, _ int) string { return strings.TrimSpace(name) })))
	if len(names) == 0 {
		return nil, errors.New("mount envs cannot be empty")
	}
	if missing, _ := lo.Difference(names, envNames); len(missing) > 0 {
		return nil, errors.Errorf(
			"envs %s not found in app %s, available: %s",
			strings.Join(missing, ","), appID, strings.Join(envNames, ", "),
		)
	}
	return names, nil
}

func formatEnvName(envName string) string {
	if envName == "" {
		return defaultEnvLabel
	}
	return envName
}

func formatMountEnvs(def *client.AppConfigFileDef) string {
	if len(def.MountedEnvNames) == 0 {
		return allEnvsLabel
	}
	return strings.Join(def.MountedEnvNames, ",")
}

func formatEnvMode(def *client.AppConfigFileDef) string {
	if def.IsUnifiedConfig {
		return envModeUnified
	}
	return envModePerEnv
}

// loadTarget 定位配置文件，并加载其在指定环境下实际生效的文件详情。
func loadTarget(
	ctx context.Context,
	cli client.Client,
	appID string,
	ref FileRef,
	envName string,
) (*fileTarget, error) {
	def, err := ResolveDef(ctx, cli, appID, ref)
	if err != nil {
		return nil, err
	}
	if envName != "" {
		if err = ensureEnvConfigurable(ctx, cli, appID, envName); err != nil {
			return nil, err
		}
		if err = ensureDefMountedOnEnv(def, envName); err != nil {
			return nil, err
		}
	}

	detail, err := cli.GetAppConfigFileDef(ctx, appID, def.ID, envName)
	if err != nil {
		return nil, errors.Wrap(err, "get app config file details")
	}
	return &fileTarget{Def: def, Detail: detail, EnvName: envName}, nil
}
