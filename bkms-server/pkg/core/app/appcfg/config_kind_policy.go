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
)

// EnvInstanceStrategy 环境实例的产生方式。
type EnvInstanceStrategy string

const (
	// EnvInstanceStrategyOverlay 以 overlay（补丁）方式叠加基础配置。
	EnvInstanceStrategyOverlay EnvInstanceStrategy = "overlay"
	// EnvInstanceStrategyOverwrite 以 overwrite（全量覆盖）方式替换基础配置。
	EnvInstanceStrategyOverwrite EnvInstanceStrategy = "overwrite"
)

// ConfigKindPolicy 定义某个 ConfigKind 的校验与行为规则。
type ConfigKindPolicy interface {
	// ValidateCreateParams 校验创建参数是否满足该 kind 的语义约束。
	ValidateCreateParams(params CreateCfgFileParams) error
	// ValidateContent 对原始内容做 kind 级别的约束校验。
	ValidateContent(content string, format FileFormat) error
	// GetEnvInstanceStrategy 返回环境实例产生策略。
	GetEnvInstanceStrategy() EnvInstanceStrategy
	// IsEffectiveForEnv 判断该 def 是否在指定环境下生效。
	IsEffectiveForEnv(def *AppConfigFileDef, envName string) bool
	// AllowMountDirUpdate 是否允许通过 def 接口修改挂载目录。
	AllowMountDirUpdate() bool
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

// ValidateCreateParams framework 创建无额外约束。
func (FrameworkPolicy) ValidateCreateParams(_ CreateCfgFileParams) error {
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

// GetEnvInstanceStrategy framework 使用 overlay 策略。
func (FrameworkPolicy) GetEnvInstanceStrategy() EnvInstanceStrategy {
	return EnvInstanceStrategyOverlay
}

// IsEffectiveForEnv framework 始终挂载到全部环境。
func (FrameworkPolicy) IsEffectiveForEnv(_ *AppConfigFileDef, _ string) bool {
	return true
}

// AllowMountDirUpdate framework 挂载路径由 app_models 管理，不允许通过 def 修改。
// TODO: 待挂载路径迁移至 def 后放开此限制。
func (FrameworkPolicy) AllowMountDirUpdate() bool {
	return false
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

// GetEnvInstanceStrategy plain 使用 overwrite 策略（完整复制默认内容到环境实例）。
func (PlainPolicy) GetEnvInstanceStrategy() EnvInstanceStrategy {
	return EnvInstanceStrategyOverwrite
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

// AllowMountDirUpdate plain 挂载路径由用户指定，允许通过 def 修改。
func (PlainPolicy) AllowMountDirUpdate() bool {
	return true
}
