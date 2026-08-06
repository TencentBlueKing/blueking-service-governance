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

package component

import "github.com/hashicorp/go-set/v3"

// PropType 表示组件不同的参数类型，不同类型将影响数据校验、前端渲染等行为。
type PropType string

const (
	// PropTypeInt 整数
	// - UI 侧用整数输入框
	PropTypeInt PropType = "INT"
	// PropTypeString 字符串
	// - UI 侧使用单行文本输入框
	PropTypeString PropType = "STRING"
	// PropTypeText 文本
	// - UI 侧使用多行文本输入框
	PropTypeText PropType = "TEXT"
	// PropTypeSelect 单选
	// - UI 侧使用下拉单选框
	PropTypeSelect PropType = "SELECT"
	// PropTypeBool 布尔值
	// - UI 侧使用滑动开关
	PropTypeBool PropType = "BOOL"
	// PropTypeMap 映射
	// - UI 侧要求传入有效的 JSON object 字符串
	PropTypeMap PropType = "MAP"
)

var propTypeSet = set.From([]PropType{
	PropTypeInt, PropTypeString, PropTypeText, PropTypeSelect, PropTypeBool, PropTypeMap,
})

// IsValid 检查 PropType 是否为有效值
func (p PropType) IsValid() bool {
	return propTypeSet.Contains(p)
}

// Builtin property names for template context.
// Most are prefixed with "bkms"; "name" is the legacy resource-name helper.
const (
	PropNameAppName       = "bkmsAppName"
	PropNameContainerName = "bkmsContainerName"
	PropNameEnvName       = "bkmsEnvName"
	PropNameEnvNS         = "bkmsEnvNamespace"
	PropNameEnvCluster    = "bkmsEnvCluster"
	PropNameName          = "name"
)

// BuiltinVar 是组件 patcher/spec 模板中由系统注入的变量。
type BuiltinVar struct {
	// Key 是 Go template 中可引用的变量名，如 {{ .bkmsAppName }}。
	Key string
	// Description 是变量含义说明。
	Description string
}

var BuiltinVars = []BuiltinVar{
	{Key: PropNameAppName, Description: "组件实例所在的应用名称"},
	{Key: PropNameContainerName, Description: "主容器名称（目前固定为 main）"},
	{Key: PropNameEnvName, Description: "组件实例所在的环境名称"},
	{Key: PropNameEnvNS, Description: "环境绑定集群中的命名空间; 未绑定集群时为空"},
	{Key: PropNameEnvCluster, Description: "环境绑定的集群 ID; 未绑定集群时为空"},
	{Key: PropNameName, Description: "组件实例生成资源时使用的默认名称，格式为 `应用名称-组件实例名称的小写形式`"},
}
