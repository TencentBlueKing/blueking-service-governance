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

package serializer

import (
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/lifecycle"
)

// ToBKMSAppCompat 将旧创建接口的 trpc / taf 请求转换为 bkmsApp 的框架、语言与创建参数。
// TAF 请求没有语言，固定为 cpp。
//
// Deprecated: 仅供旧创建接口兼容，旧接口下线后删除。
func (input *CreateAppInput) ToBKMSAppCompat() (framework, language string, params *lifecycle.CreateParams) {
	spec := input.AppModelSpec
	params = &lifecycle.CreateParams{
		Command:   spec.Command,
		Args:      spec.Args,
		ShCommand: spec.ShCommand,
		EnvVars:   variableInputsToModel(spec.EnvVars),
	}
	switch input.Type {
	case bkmsapp.AppTypeTRPC:
		framework, language = appruntime.FrameworkTRPC, spec.TrpcSpec.Language
		params.FrameworkFile = &lifecycle.FrameworkFileParams{
			Name:     spec.TrpcSpec.FileName,
			MountDir: spec.TrpcSpec.FilePath,
			Content:  spec.TrpcSpec.FileContent,
		}
	case bkmsapp.AppTypeTAF:
		framework, language = appruntime.FrameworkTAF, appruntime.LanguageCpp
		params.FrameworkFile = &lifecycle.FrameworkFileParams{
			Name:     spec.TafSpec.FileName,
			MountDir: spec.TafSpec.FilePath,
			Content:  spec.TafSpec.FileContent,
		}
	}
	return framework, language, params
}

// ToSpecCompatParams 将旧 trpc-spec / taf-spec 请求转换为兼容修改参数。
// 环境变量为兼容旧客户端接收但忽略，应通过独立接口修改。
//
// Deprecated: 仅供旧 spec 接口兼容，旧接口下线后删除。
func (input *AppModelSpecInput) ToSpecCompatParams(framework string) *lifecycle.SpecCompatParams {
	params := &lifecycle.SpecCompatParams{
		Framework: framework,
		Command:   input.Command,
		Args:      input.Args,
		ShCommand: input.ShCommand,
	}
	switch framework {
	case appruntime.FrameworkTRPC:
		if input.TrpcSpec != nil {
			params.Language = input.TrpcSpec.Language
			params.FileName = input.TrpcSpec.FileName
			params.FilePath = input.TrpcSpec.FilePath
		}
	case appruntime.FrameworkTAF:
		if input.TafSpec != nil {
			params.FileName = input.TafSpec.FileName
			params.FilePath = input.TafSpec.FilePath
		}
	}
	return params
}
