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

package appcfg

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
)

// EnvContentStrategy 环境内容相对默认配置的处理策略。
type EnvContentStrategy string

const (
	// EnvContentStrategyOverlay 以 overlay（补丁）方式叠加基础配置。
	EnvContentStrategyOverlay EnvContentStrategy = "overlay"
	// EnvContentStrategyOverwrite 以 overwrite（全量覆盖）方式替换基础配置。
	EnvContentStrategyOverwrite EnvContentStrategy = "overwrite"
)

// ConfigKindPolicy 定义某个 ConfigKind 的校验与行为规则。
type ConfigKindPolicy interface {
	// ValidateCreateParams 校验创建参数是否满足该 kind 的语义约束。
	ValidateCreateParams(params CreateCfgFileParams) error
	// ValidateContent 对原始内容做 kind 级别的约束校验。
	ValidateContent(content string, format FileFormat) error
	// InitializeEnvConfigMode 在创建 def 时填充该 kind 需要持久化的环境配置模式字段。
	InitializeEnvConfigMode(mode *EnvConfigMode, params CreateCfgFileParams)
	// GetEnvContentStrategy 返回后续创建环境实例、展示生效内容时要使用的策略。
	GetEnvContentStrategy(def *AppConfigFileDef) (EnvContentStrategy, error)
	// IsEffectiveForEnv 判断该 def 是否在指定环境下生效。
	IsEffectiveForEnv(def *AppConfigFileDef, envName string) bool
	// AllowNameUpdate 是否允许通过 def 接口修改文件名。
	// appType 用来区分同一 ConfigKind 下的应用类型，例如 framework 里的 helm 与 trpc。
	AllowNameUpdate(appType string) bool
	// AllowMountDirUpdate 是否允许通过 def 接口修改挂载目录。
	// appType 用来区分同一 ConfigKind 下的应用类型，例如 framework 里的 helm 与 trpc。
	AllowMountDirUpdate(appType string) bool
	// AllowMountedEnvNamesUpdate 是否允许通过 def 接口修改挂载环境范围。
	AllowMountedEnvNamesUpdate() bool
	// AllowEnableEnvVarRenderUpdate 是否允许通过 def 接口修改环境变量渲染开关。
	AllowEnableEnvVarRenderUpdate() bool
	// AllowEnvContentStrategyUpdate 是否允许通过 def 接口修改环境内容策略。
	AllowEnvContentStrategyUpdate() bool
	// DefaultEnableEnvVarRender 返回该 kind 创建时 EnableEnvVarRender 的默认值。
	DefaultEnableEnvVarRender() bool
}

// DefaultPolicies 已注册 ConfigKind 到 policy 的映射。
var DefaultPolicies = map[ConfigKind]ConfigKindPolicy{
	ConfigKindFramework: FrameworkPolicy{},
	ConfigKindPlain:     PlainPolicy{},
}

// --- framework policy ---

// FrameworkPolicy 是 ConfigKindFramework 的策略实现。
type FrameworkPolicy struct{}

var _ ConfigKindPolicy = FrameworkPolicy{}

// ValidateCreateParams framework 创建时不允许关闭环境变量渲染。
func (p FrameworkPolicy) ValidateCreateParams(params CreateCfgFileParams) error {
	if params.EnableEnvVarRender != nil && !*params.EnableEnvVarRender {
		return errors.Wrap(ErrInvalidConfigSpec, "framework config file does not allow disable enableEnvVarRender")
	}
	if params.MountedEnvNames != nil {
		return errors.Wrap(ErrInvalidConfigSpec, "framework config file does not support mountedEnvNames")
	}
	// 挂载目录是否允许与更新使用同一规则：只有 trpc/taf 可以带 mountDir。
	if params.MountDir != "" && !p.AllowMountDirUpdate(params.AppType) {
		return errors.Wrap(ErrInvalidConfigSpec, "framework config file does not allow mountDir")
	}
	return nil
}

// ValidateContent 校验内容为合法 YAML。
func (FrameworkPolicy) ValidateContent(content string, _ FileFormat) error {
	if content == "" {
		return nil
	}
	var out any
	if err := yaml.Unmarshal([]byte(content), &out); err != nil {
		return errors.Wrapf(ErrInvalidConfigSpec, "content is not valid YAML: %v", err)
	}
	return nil
}

// InitializeEnvConfigMode 在创建 framework def 时写入初始环境实例策略。
// AppModel 应用写入完整配置，其余类型写入与默认配置合并。
func (FrameworkPolicy) InitializeEnvConfigMode(mode *EnvConfigMode, params CreateCfgFileParams) {
	if mode == nil {
		return
	}
	if appruntime.IsAppModelType(params.AppType) {
		mode.EnvContentStrategy = EnvContentStrategyOverwrite
		return
	}
	mode.EnvContentStrategy = EnvContentStrategyOverlay
}

// GetEnvContentStrategy 返回框架定义上已持久化的环境内容策略。
func (FrameworkPolicy) GetEnvContentStrategy(def *AppConfigFileDef) (EnvContentStrategy, error) {
	if def == nil {
		return "", errors.Wrap(ErrInvalidConfigSpec, "framework def is required")
	}
	strategy := def.EnvConfigMode.EnvContentStrategy
	if err := validateFrameworkEnvContentStrategy(strategy); err != nil {
		return "", err
	}
	return strategy, nil
}

// IsEffectiveForEnv framework 始终挂载到全部环境。
func (FrameworkPolicy) IsEffectiveForEnv(_ *AppConfigFileDef, _ string) bool {
	return true
}

// AllowNameUpdate trpc 的框架文件名创建后不能改。taf 和 Helm 可以改。
func (FrameworkPolicy) AllowNameUpdate(appType string) bool {
	return appType != "trpc"
}

// AllowMountDirUpdate framework 默认不允许改挂载目录，避免 Helm values 被写上目录。
// trpc/taf 的框架文件目录存在 def 上，允许修改。
func (FrameworkPolicy) AllowMountDirUpdate(appType string) bool {
	return appType == "trpc" || appType == "taf"
}

// AllowMountedEnvNamesUpdate framework 始终对全部环境生效，不允许设置挂载范围。
func (FrameworkPolicy) AllowMountedEnvNamesUpdate() bool {
	return false
}

// AllowEnableEnvVarRenderUpdate framework 始终启用环境变量渲染，不允许关闭。
func (FrameworkPolicy) AllowEnableEnvVarRenderUpdate() bool {
	return false
}

// AllowEnvContentStrategyUpdate framework 允许修改环境内容策略。
func (FrameworkPolicy) AllowEnvContentStrategyUpdate() bool {
	return true
}

// DefaultEnableEnvVarRender framework 创建时始终启用环境变量渲染。
func (FrameworkPolicy) DefaultEnableEnvVarRender() bool {
	return true
}

// --- plain policy ---

// PlainPolicy 是 ConfigKindPlain 的策略实现。
type PlainPolicy struct{}

var _ ConfigKindPolicy = PlainPolicy{}

// ValidateCreateParams 校验 plain 文件创建参数：mountDir 必填，不允许 overlay/base 引用，仅支持 local 来源。
func (PlainPolicy) ValidateCreateParams(params CreateCfgFileParams) error {
	if params.MountDir == "" {
		return errors.Wrap(ErrInvalidConfigSpec, "plain config file requires mountDir")
	}
	if params.BaseAppConfigFileID != nil {
		return errors.Wrap(ErrInvalidConfigSpec, "plain config file does not support base reference")
	}
	if params.OverlayContent != nil {
		return errors.Wrap(ErrInvalidConfigSpec, "plain config file does not support overlay content")
	}
	if params.ContentSourceType != "" && params.ContentSourceType != ContentSourceTypeLocal {
		return errors.Wrap(ErrInvalidConfigSpec, "plain config file only supports local content source")
	}
	return nil
}

// ValidateContent plain 文件接受任意文本内容，不做格式校验。
func (PlainPolicy) ValidateContent(_ string, _ FileFormat) error {
	return nil
}

// InitializeEnvConfigMode plain 不需要写入额外的环境实例策略字段。
func (PlainPolicy) InitializeEnvConfigMode(_ *EnvConfigMode, _ CreateCfgFileParams) {
}

// GetEnvContentStrategy plain 固定使用 overwrite 策略。
func (PlainPolicy) GetEnvContentStrategy(_ *AppConfigFileDef) (EnvContentStrategy, error) {
	return EnvContentStrategyOverwrite, nil
}

// IsEffectiveForEnv plain 文件是否在指定环境生效仅由挂载范围决定；
// IsUnifiedConfig 只决定内容是否按环境独立，不改变挂载范围语义。
//
// MountedEnvNames 语义：nil = 全环境生效；空切片 = 不挂载任何环境；非空 = 仅列出的环境。
func (PlainPolicy) IsEffectiveForEnv(def *AppConfigFileDef, envName string) bool {
	if def == nil {
		return false
	}
	if def.EnvConfigMode.MountedEnvNames == nil {
		return true
	}
	return def.EnvConfigMode.ContainsEnv(envName)
}

// AllowNameUpdate plain 文件名由用户指定，允许通过 def 修改。
func (PlainPolicy) AllowNameUpdate(string) bool {
	return true
}

// AllowMountDirUpdate plain 挂载路径由用户指定，允许通过 def 修改。
func (PlainPolicy) AllowMountDirUpdate(string) bool {
	return true
}

// AllowMountedEnvNamesUpdate plain 允许用户指定挂载环境范围。
func (PlainPolicy) AllowMountedEnvNamesUpdate() bool {
	return true
}

// AllowEnableEnvVarRenderUpdate plain 允许用户自行开关环境变量渲染。
func (PlainPolicy) AllowEnableEnvVarRenderUpdate() bool {
	return true
}

// AllowEnvContentStrategyUpdate plain 不支持修改环境内容策略。
func (PlainPolicy) AllowEnvContentStrategyUpdate() bool {
	return false
}

// DefaultEnableEnvVarRender plain 创建时默认不启用环境变量渲染。
func (PlainPolicy) DefaultEnableEnvVarRender() bool {
	return false
}

// validateFrameworkEnvContentStrategy 校验框架定义上持久化的环境内容策略字段。
func validateFrameworkEnvContentStrategy(strategy EnvContentStrategy) error {
	switch strategy {
	case EnvContentStrategyOverlay, EnvContentStrategyOverwrite:
		return nil
	default:
		return errors.Wrapf(
			ErrInvalidConfigSpec, "framework envContentStrategy %q is invalid", strategy,
		)
	}
}
