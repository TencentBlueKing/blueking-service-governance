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

// Package serializer 自动触发镜像构建相关 API 的请求与响应结构。
// 契约详见 design_notes/build_trigger_contract.md
package serializer

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
)

var (
	// policyNamePattern 策略名称字符集：汉字、大小写字母、数字、- 与 _
	policyNamePattern = regexp.MustCompile(`^[\p{Han}a-zA-Z0-9_-]+$`)
	// versionPrefixPattern 自定义版本前缀字符集：字母、数字与 -，需满足容器镜像 tag 规范
	versionPrefixPattern = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
)

func init() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	if err := v.RegisterValidation("trigger_policy_name", validatePolicyName); err != nil {
		panic("failed to register trigger_policy_name validator: " + err.Error())
	}
	if err := v.RegisterValidation("trigger_version_prefix", validateVersionPrefix); err != nil {
		panic("failed to register trigger_version_prefix validator: " + err.Error())
	}
}

// validatePolicyName 校验策略名称字符集，长度由 min / max tag 单独约束
func validatePolicyName(fl validator.FieldLevel) bool {
	return policyNamePattern.MatchString(fl.Field().String())
}

// validateVersionPrefix 校验自定义版本前缀字符集，长度由 max tag 单独约束
func validateVersionPrefix(fl validator.FieldLevel) bool {
	return versionPrefixPattern.MatchString(fl.Field().String())
}

// AppURIInput is the path input for APIs scoped by application.
type AppURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,max=63,uri_slug"`
}

// PolicyURIInput is the path input for APIs scoped by one trigger policy.
type PolicyURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,max=63,uri_slug"`
	// 触发策略 ID
	PolicyID string `uri:"policyID" binding:"required,min=1,max=63"`
}
