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

// Package framework 提供 e2e 基础框架功能
package framework

import (
	"fmt"

	cenv "github.com/caarlos0/env/v11"
	"github.com/onsi/ginkgo/v2"
)

// EnvConfig 环境变量读取的 E2E 测试配置。
type EnvConfig struct {
	// ApiUrl BKMS 服务地址
	ApiUrl string `env:"BKMS_API_URL,required"`
	// Username 用户名
	Username string `env:"BKMS_USERNAME,required"`
	// Token 访问令牌
	Token string `env:"BKMS_TOKEN,required"`

	// WorkspaceID 工作区 ID
	WorkspaceID string `env:"BKMS_WORKSPACE_ID,required"`
	// AppID 应用 ID
	AppID string `env:"BKMS_APP_ID,required"`
	// EnvName 环境名称
	EnvName string `env:"BKMS_ENV_NAME,required"`

	// BCSToken BCS 令牌
	BCSToken string `env:"BCS_TOKEN,required"`
}

// LoadEnvConfig 从环境变量读取配置并返回 EnvConfig。
// required 字段缺失时直接终止测试。
func LoadEnvConfig() *EnvConfig {
	cfg := new(EnvConfig)

	if err := cenv.Parse(cfg); err != nil {
		ginkgo.Fail(fmt.Sprintf("failed to load env config: %v", err))
	}

	return cfg
}
