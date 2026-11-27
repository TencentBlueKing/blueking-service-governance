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

package testutil

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

// BasicRepoSourceTypeName 单测使用的账号密码接入方式名
const BasicRepoSourceTypeName = "Git"

// BkCIManagedRepoSourceType 蓝盾托管接入方式，名称沿用历史取值 TGit
func BkCIManagedRepoSourceType() config.RepoSourceTypeConfig {
	return config.RepoSourceTypeConfig{
		Name:    config.DefaultRepoSourceTypeName,
		Spec:    config.RepoSourceSpecBkCIManaged,
		Label:   config.DefaultRepoSourceTypeName,
		Enabled: true,
	}
}

// BasicRepoSourceType 单测使用的账号密码接入方式
func BasicRepoSourceType() config.RepoSourceTypeConfig {
	return config.RepoSourceTypeConfig{
		Name:    BasicRepoSourceTypeName,
		Spec:    config.RepoSourceSpecBasic,
		Label:   BasicRepoSourceTypeName,
		Enabled: true,
	}
}

// UseRepoSourceTypes 在当前测试内替换接入方式配置
func UseRepoSourceTypes(t CleanupTB, types ...config.RepoSourceTypeConfig) {
	t.Helper()
	prev := config.G
	var cfg config.Config
	if prev != nil {
		cfg = *prev
	}
	cfg.CodeRepository = config.CodeRepositoryConfig{SourceTypes: types}
	config.G = &cfg
	t.Cleanup(func() { config.G = prev })
}
