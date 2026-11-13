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

package frameworkcfgtmpl

import (
	"embed"
	"strings"

	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
)

// BuiltinName 平台内置模板的名称，也是自定义模板不能使用的保留名称。
const BuiltinName = "平台默认"

const builtinIDPrefix = "builtin-"

//go:embed builtin/*
var builtinFS embed.FS

type builtinSource struct {
	group Group
	// fileName 对外的框架配置文件名，创建应用时以此名称挂载。
	fileName string
	// embedPath 模板内容在 builtinFS 中的路径，按「框架_语言」命名以免分组间重名。
	embedPath string
}

// builtins 平台内置模板，每个支持框架配置文件的分组恰好一个，顺序即列表中分组的展示顺序。
var builtins = lo.Map([]builtinSource{
	{Group{appruntime.FrameworkTRPC, appruntime.LanguageGo}, "trpc_go.yaml", "builtin/trpc_go.yaml"},
	{Group{appruntime.FrameworkTRPC, appruntime.LanguageCpp}, "trpc_cpp.yaml", "builtin/trpc_cpp.yaml"},
	{Group{appruntime.FrameworkTAF, appruntime.LanguageCpp}, "config.conf", "builtin/taf_cpp.conf"},
}, func(src builtinSource, _ int) View {
	return View{
		ID:       builtinID(src.group),
		Group:    src.group,
		Name:     BuiltinName,
		FileName: src.fileName,
		Content:  string(lo.Must(builtinFS.ReadFile(src.embedPath))),
		Builtin:  true,
	}
})

// builtinID 内置框架配置文件 id
func builtinID(group Group) string {
	return builtinIDPrefix + group.Framework + "-" + group.Language
}

func isBuiltinID(id string) bool {
	return strings.HasPrefix(id, builtinIDPrefix)
}

// lookupBuiltin 按 ID 查找内置模板，返回副本。
func lookupBuiltin(id string) (View, bool) {
	return lo.Find(builtins, func(v View) bool { return v.ID == id })
}
