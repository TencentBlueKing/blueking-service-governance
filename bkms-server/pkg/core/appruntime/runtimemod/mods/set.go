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

package mods

import (
	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/capability/admincommand"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/capability/apmservicename"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/capability/bscpworkloadmeta"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/capability/configmerge"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/capability/devmode"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/capability/frameworkfile"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/capability/platformbuild"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/capability/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod"
)

// Set 应用的全部能力。
//
// 新增能力时，需同步修改本结构体、resolve、invalid、all，测试会检查四者是否一致。
type Set struct {
	// Stack 解析所用的技术栈。
	Stack appruntime.Stack

	// FrameworkFile 框架配置文件。
	FrameworkFile runtimemod.Mod[frameworkfile.Props]
	// DevMode 开发模式。
	DevMode runtimemod.Mod[devmode.Props]
	// PlatformBuild 平台生成 Dockerfile 构建。
	PlatformBuild runtimemod.Mod[platformbuild.Props]
	// AdminCommand admin 命令。
	AdminCommand runtimemod.Mod[admincommand.Props]
	// APMServiceName APM 服务名提取。
	APMServiceName runtimemod.Mod[apmservicename.Props]
	// PolarisServiceCheck 北极星服务名校验。
	PolarisServiceCheck runtimemod.Mod[polaris.ServiceCheckProps]
	// PolarisFrameworkPatch 北极星注册信息注入。
	PolarisFrameworkPatch runtimemod.Mod[polaris.FrameworkPatchProps]
	// BSCPWorkloadMeta BSCP 元信息自动填充。
	BSCPWorkloadMeta runtimemod.Mod[bscpworkloadmeta.Props]
	// ConfigMerge 配置文件合并。
	ConfigMerge runtimemod.Mod[configmerge.Props]
}

// Of 解析应用的技术栈，得出全部能力。
//
// 只读 Application 自身的字段，不加载 AppModel。技术栈非法时记录告警日志，
// 返回全部能力不支持（原因为 invalid_stack）的结果；执行器调用 Require 时会得到能力错误。
func Of(app *bkmsapp.Application) Set {
	stack, err := appruntime.CapabilityStack(app.RuntimeSnapshot())
	if err != nil {
		log.WarnNoContextf("resolve tech stack of app %s failed, all runtime mods are unsupported: %v", app.ID, err)
		return invalid(appruntime.Stack{Type: app.Type})
	}
	return resolve(stack)
}

// Resolve 按已规范化的技术栈得出全部能力，用于调用方已自行取得技术栈的场景。
func Resolve(stack appruntime.Stack) (Set, error) {
	if err := appruntime.ValidateCapabilityStack(stack); err != nil {
		return Set{}, errors.Wrap(err, "resolve runtime mods")
	}
	return resolve(stack), nil
}

func resolve(stack appruntime.Stack) Set {
	return Set{
		Stack:                 stack,
		FrameworkFile:         frameworkfile.Spec.ResolveStack(stack),
		DevMode:               devmode.Spec.ResolveStack(stack),
		PlatformBuild:         platformbuild.Spec.ResolveStack(stack),
		AdminCommand:          admincommand.Spec.ResolveStack(stack),
		APMServiceName:        apmservicename.Spec.ResolveStack(stack),
		PolarisServiceCheck:   polaris.ServiceCheck.ResolveStack(stack),
		PolarisFrameworkPatch: polaris.FrameworkPatch.ResolveStack(stack),
		BSCPWorkloadMeta:      bscpworkloadmeta.Spec.ResolveStack(stack),
		ConfigMerge:           configmerge.Spec.ResolveStack(stack),
	}
}

// invalid 技术栈非法时的能力集合，所有能力都不支持。
func invalid(stack appruntime.Stack) Set {
	return Set{
		Stack:                 stack,
		FrameworkFile:         frameworkfile.Spec.Invalid(),
		DevMode:               devmode.Spec.Invalid(),
		PlatformBuild:         platformbuild.Spec.Invalid(),
		AdminCommand:          admincommand.Spec.Invalid(),
		APMServiceName:        apmservicename.Spec.Invalid(),
		PolarisServiceCheck:   polaris.ServiceCheck.Invalid(),
		PolarisFrameworkPatch: polaris.FrameworkPatch.Invalid(),
		BSCPWorkloadMeta:      bscpworkloadmeta.Spec.Invalid(),
		ConfigMerge:           configmerge.Spec.Invalid(),
	}
}

// all 以能力 ID 为 key 返回全部能力，供 Public 使用。
func (s Set) all() map[runtimemod.ModID]runtimemod.View {
	return map[runtimemod.ModID]runtimemod.View{
		s.FrameworkFile.ID:         s.FrameworkFile,
		s.DevMode.ID:               s.DevMode,
		s.PlatformBuild.ID:         s.PlatformBuild,
		s.AdminCommand.ID:          s.AdminCommand,
		s.APMServiceName.ID:        s.APMServiceName,
		s.PolarisServiceCheck.ID:   s.PolarisServiceCheck,
		s.PolarisFrameworkPatch.ID: s.PolarisFrameworkPatch,
		s.BSCPWorkloadMeta.ID:      s.BSCPWorkloadMeta,
		s.ConfigMerge.ID:           s.ConfigMerge,
	}
}

// Public 转为对外结构。
func (s Set) Public() runtimemod.PublicView {
	return runtimemod.BuildPublicView(s.Stack, s.all())
}

// Matrix 返回框架目录中每个合法组合的能力，供前端创建应用时展示。
func Matrix() ([]runtimemod.PublicView, error) {
	var out []runtimemod.PublicView
	for _, def := range appruntime.Catalog() {
		for _, language := range def.SupportedLanguages {
			out = append(out, resolve(appruntime.Stack{
				Type:      appruntime.AppTypeBKMSApp,
				Framework: def.Name,
				Language:  language,
			}).Public())
		}
	}
	return out, nil
}
