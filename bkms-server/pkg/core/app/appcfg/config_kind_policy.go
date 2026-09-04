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
	// ValidateContent 对原始内容做 kind 级别的约束校验。
	ValidateContent(content string, format FileFormat) error
	// GetEnvInstanceStrategy 返回环境实例产生策略。
	GetEnvInstanceStrategy() EnvInstanceStrategy
	// IsAlwaysMount 是否挂载环境，framework 始终挂载到环境
	IsAlwaysMount(def *AppConfigFileDef, envName string) bool
	// AllowMountDirUpdate 是否允许通过 def 接口修改挂载目录。
	AllowMountDirUpdate() bool
}

// DefaultPolicies 已注册 ConfigKind 到 policy 的映射。
var DefaultPolicies = map[ConfigKind]ConfigKindPolicy{
	ConfigKindFramework: FrameworkPolicy{},
}

// --- framework policy ---

// FrameworkPolicy 是 ConfigKindFramework 的策略实现。
type FrameworkPolicy struct{}

var _ ConfigKindPolicy = FrameworkPolicy{}

// ValidateContent 校验内容为合法 YAML。
func (FrameworkPolicy) ValidateContent(content string, _ FileFormat) error {
	if content == "" {
		return nil
	}
	var out any
	if err := yaml.Unmarshal([]byte(content), &out); err != nil {
		return errors.Wrap(err, "content is not valid YAML")
	}
	return nil
}

// GetEnvInstanceStrategy framework 使用 overlay 策略。
func (FrameworkPolicy) GetEnvInstanceStrategy() EnvInstanceStrategy {
	return EnvInstanceStrategyOverlay
}

// IsAlwaysMount framework 始终挂载到全部环境。
func (FrameworkPolicy) IsAlwaysMount(_ *AppConfigFileDef, _ string) bool {
	return true
}

// AllowMountDirUpdate framework 挂载路径由 app_models 管理，不允许通过 def 修改。
// TODO: 待挂载路径迁移至 def 后放开此限制。
func (FrameworkPolicy) AllowMountDirUpdate() bool {
	return false
}
