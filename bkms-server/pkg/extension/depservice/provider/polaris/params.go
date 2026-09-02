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

package polaris

import (
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/types"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// compile-time check
var (
	_ types.ProvisionParams = (*CreateParams)(nil)
	_ types.ProvisionParams = (*UpdateParams)(nil)
)

// CreateParams 创建北极星服务实例所需的参数
type CreateParams struct {
	PolarisName      string `mapstructure:"polarisName" validate:"required"`
	PolarisNamespace string `mapstructure:"polarisNamespace" validate:"required"`
	Owners           string `mapstructure:"Owners"`
	// Metadata 创建时写入北极星服务的 metadata；空则不下发该字段。
	Metadata map[string]string `mapstructure:"metadata"`
}

// Validate 使用 validator 校验 CreateParams 中所有必填字段是否已设置
func (p *CreateParams) Validate() error {
	return validate.Struct(p)
}

// UpdateParams 更新北极星服务实例所需的参数
type UpdateParams struct {
	// Owners 为空表示不改负责人，PUT 时不下发该字段。
	Owners string `mapstructure:"Owners"`
	// Metadata 要覆盖写入的键。与 MetadataKeysToDelete 一起作用在 GET 到的现有 metadata 上；
	Metadata map[string]string `mapstructure:"metadata"`
	// MetadataKeysToDelete 要从现有 metadata 中删除的键。
	MetadataKeysToDelete []string `mapstructure:"metadataKeysToDelete"`
}

// Validate 至少要改负责人或 metadata 之一。
func (p *UpdateParams) Validate() error {
	if p.Owners == "" && len(p.Metadata) == 0 && len(p.MetadataKeysToDelete) == 0 {
		return errors.New("owners or metadata is required")
	}
	return nil
}

// InstConfig 北极星服务实例配置
type InstConfig struct {
	PolarisName      string `mapstructure:"polarisName" validate:"required"`
	PolarisNamespace string `mapstructure:"polarisNamespace" validate:"required"`
	Token            string `mapstructure:"token" validate:"required"`
}

// Validate 使用 validator 校验 InstConfig 中所有必填字段是否已设置
func (c *InstConfig) Validate() error {
	return validate.Struct(c)
}

// ParseInstConfig 从 map 中解析北极星实例配置
func ParseInstConfig(raw map[string]any) (*InstConfig, error) {
	return types.ParseInstConfig[InstConfig](raw)
}

// Credentials 北极星服务实例的凭证信息
type Credentials struct {
	Token string `mapstructure:"token"`
}

// ParseCredentials 从 map 中解析北极星凭证
func ParseCredentials(raw map[string]any) (*Credentials, error) {
	return types.ParseInstConfig[Credentials](raw)
}
