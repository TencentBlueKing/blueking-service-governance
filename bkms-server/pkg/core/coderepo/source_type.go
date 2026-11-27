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

// Package coderepo 代码库接入方式与代码库管理，代码库与凭证由 bkms 持有
package coderepo

import (
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

// LegacyTypeGitHub type 的历史取值，按默认接入方式处理
const LegacyTypeGitHub = "GitHub"

// legacyTypeNames type 的历史取值，未配置为接入方式时按默认接入方式处理
var legacyTypeNames = []string{config.DefaultRepoSourceTypeName, LegacyTypeGitHub}

// SourceType 代码库接入方式
type SourceType = config.RepoSourceTypeConfig

// SourceTypes 返回已启用的接入方式
func SourceTypes() []SourceType {
	return lo.Filter(config.G.CodeRepository.SourceTypes, func(t SourceType, _ int) bool { return t.Enabled })
}

// DefaultSourceType 默认接入方式：第一个蓝盾托管方式，没有时取第一个
func DefaultSourceType() SourceType {
	types := SourceTypes()
	if t, ok := lo.Find(types, func(t SourceType) bool { return t.Spec == config.RepoSourceSpecBkCIManaged }); ok {
		return t
	}
	// 配置校验保证至少启用一个接入方式
	return types[0]
}

// ResolveSourceType 按 type 取接入方式，为空或历史取值时取默认接入方式
func ResolveSourceType(name string) (SourceType, error) {
	types := SourceTypes()
	if t, ok := lo.Find(types, func(t SourceType) bool { return t.Name == name }); ok {
		return t, nil
	}
	if name == "" || lo.Contains(legacyTypeNames, name) {
		return DefaultSourceType(), nil
	}
	return SourceType{}, errors.Wrapf(ErrSourceTypeNotFound, "source type %q", name)
}
