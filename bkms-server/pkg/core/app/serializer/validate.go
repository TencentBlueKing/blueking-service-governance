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
	"regexp"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register mount_dir
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var validIDPattern = regexp.MustCompile("^[a-z](?:[a-z0-9-]*[a-z0-9])?$")

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("app_id", validateID); err != nil {
			panic("failed to register app_id validator: " + err.Error())
		}
		if err := v.RegisterValidation("workspace_id", validateID); err != nil {
			panic("failed to register workspace_id validator: " + err.Error())
		}
		if err := v.RegisterValidation("component_name", validateComponentName); err != nil {
			panic("failed to register component_name validator: " + err.Error())
		}
		if err := v.RegisterValidation("envvar_key", validateEnvVarKey); err != nil {
			panic("failed to register envvar_key validator: " + err.Error())
		}
		// Register struct-level validator for CreateAppInput to enforce
		// conditional required fields based on app type.
		v.RegisterStructValidation(validateCreateAppInputStruct, CreateAppInput{})
		v.RegisterStructValidation(validateAppModelSpecInputStruct, AppModelSpecInput{})
		v.RegisterStructValidation(validateBKMSAppModelSpecInputStruct, BKMSAppModelSpecInput{})
		v.RegisterStructValidation(validateUpdateAppStartupConfigInputStruct, UpdateAppStartupConfigInput{})
		v.RegisterStructValidation(validateCreateAppComponentInputStruct, CreateAppComponentInput{})
		// Register struct-level validator for HelmSourceInput to enforce
		// the matching repo config based on RepoType. This applies to both
		// CreateApp (via CreateAppInput.HelmSpec.HelmSource) and
		// UpdateHelmSpec (via UpdateHelmSpecInput.HelmSpec.HelmSource).
		v.RegisterStructValidation(validateHelmSourceInputStruct, HelmSourceInput{})
	}
}

func validateID(fl validator.FieldLevel) bool {
	input := fl.Field().String()
	if len(input) < 1 || len(input) > 63 {
		return false
	}
	return validIDPattern.MatchString(input)
}

func validateComponentName(fl validator.FieldLevel) bool {
	input := fl.Field().String()
	if len(input) < 1 || len(input) > 20 {
		return false
	}
	return validIDPattern.MatchString(input)
}

func validateEnvVarKey(fl validator.FieldLevel) bool {
	return envvartypes.ValidateEnvVarKey(fl.Field().String()) == nil
}

// validateCreateAppInputStruct is a struct-level validator for CreateAppInput.
// It enforces conditional required fields based on the app Type
func validateCreateAppInputStruct(sl validator.StructLevel) {
	input := sl.Current().Interface().(CreateAppInput)

	switch input.Type {
	case appruntime.AppTypeTRPC: //nolint:staticcheck // bkmsApp 迁移兼容：旧创建接口仍接受 trpc / taf
		if input.AppModelSpec == nil {
			sl.ReportError(nil, "AppModelSpec", "AppModelSpec", "required", "")
			return
		}
		if input.AppModelSpec.TrpcSpec == nil {
			sl.ReportError(nil, "AppModelSpec.TrpcSpec", "TrpcSpec", "required", "")
			return
		}
		validateFrameworkFileLocationRequired(
			sl, "AppModelSpec.TrpcSpec", input.AppModelSpec.TrpcSpec.FileName, input.AppModelSpec.TrpcSpec.FilePath,
		)
		validateAppModelSpecEnvVarsUnique(sl, input.AppModelSpec)
	case appruntime.AppTypeTAF: //nolint:staticcheck // bkmsApp 迁移兼容：旧创建接口仍接受 trpc / taf
		if input.AppModelSpec == nil {
			sl.ReportError(nil, "AppModelSpec", "AppModelSpec", "required", "")
			return
		}
		if input.AppModelSpec.TafSpec == nil {
			sl.ReportError(nil, "AppModelSpec.TafSpec", "TafSpec", "required", "")
			return
		}
		validateFrameworkFileLocationRequired(
			sl, "AppModelSpec.TafSpec", input.AppModelSpec.TafSpec.FileName, input.AppModelSpec.TafSpec.FilePath,
		)
		validateAppModelSpecEnvVarsUnique(sl, input.AppModelSpec)
	case appruntime.AppTypeHelm, appruntime.AppTypeAgones:
		if input.HelmSpec == nil {
			sl.ReportError(nil, "HelmSpec", "HelmSpec", "required", "")
			return
		}
		// HelmSpec.HelmSource 的 required 校验由字段 binding tag 保证；
		// HelmSource 内部 RepoType 与各 Config 的关联校验由
		// HelmSourceInput 的 struct-level validator 统一处理。
	}
}

// validateFrameworkFileLocationRequired 要求创建 trpc/taf 应用时提供框架配置文件名和路径。
// 更新 spec 时这两个字段可省略，因此不放在字段 binding tag 上。
func validateFrameworkFileLocationRequired(sl validator.StructLevel, specField, fileName, filePath string) {
	if fileName == "" {
		sl.ReportError(fileName, specField+".FileName", "FileName", "required", "")
	}
	if filePath == "" {
		sl.ReportError(filePath, specField+".FilePath", "FilePath", "required", "")
	}
}

// validateAppModelSpecEnvVarsUnique validate app unique env vars key
func validateAppModelSpecEnvVarsUnique(sl validator.StructLevel, spec *AppModelSpecInput) {
	if spec == nil {
		return
	}
	validateEnvVarsUnique(sl, spec.EnvVars)
}

// validateEnvVarsUnique 校验应用环境变量的 key 不重复。
func validateEnvVarsUnique(sl validator.StructLevel, envVars []VariableInput) {
	seen := make(map[string]struct{}, len(envVars))
	for _, envVar := range envVars {
		if _, ok := seen[envVar.Key]; ok {
			sl.ReportError(
				envVars,
				"AppModelSpec.EnvVars",
				"EnvVars",
				"env_var_key_unique",
				envVar.Key)
			return
		}
		seen[envVar.Key] = struct{}{}
	}
}

// validateAppModelSpecInputStruct ensures ShCommand is not blank and is mutually exclusive with Command and Args.
func validateAppModelSpecInputStruct(sl validator.StructLevel) {
	input := sl.Current().Interface().(AppModelSpecInput)
	validateStartCommand(sl, input.Command, input.Args, input.ShCommand)
}

// validateStartCommand 校验启动脚本不能为纯空白，且与启动命令、参数互斥。
func validateStartCommand(sl validator.StructLevel, command, args []string, shCommand string) {
	if shCommand == "" {
		return
	}
	// 仅拒绝纯空白脚本，非空脚本原样保留，不做 TrimSpace 归一化
	if strings.TrimSpace(shCommand) == "" {
		sl.ReportError(shCommand, "ShCommand", "ShCommand", "sh_command_not_blank", "")
	}
	if len(command) > 0 || len(args) > 0 {
		sl.ReportError(shCommand, "ShCommand", "ShCommand", "exec_command_or_sh_command_exclusive", "")
	}
}

func validateCreateAppComponentInputStruct(sl validator.StructLevel) {
	input := sl.Current().Interface().(CreateAppComponentInput)
	if input.RefWorkspaceCompName != nil && *input.RefWorkspaceCompName != "" {
		return
	}
	if input.Type == "" {
		sl.ReportError(input.Type, "Type", "Type", "required", "")
	}
}

// validateHelmSourceInputStruct ensures that the matching repo config is
// provided based on RepoType. Applies to any place HelmSourceInput is used
// (e.g. CreateApp and UpdateHelmSpec).
func validateHelmSourceInputStruct(sl validator.StructLevel) {
	src := sl.Current().Interface().(HelmSourceInput)

	switch src.RepoType {
	case string(bkmsapp.HelmSourceRepoTypeHelm):
		if src.HelmRepoConfig == nil {
			sl.ReportError(nil, "HelmRepoConfig", "HelmRepoConfig", "required", "")
		}
	case string(bkmsapp.HelmSourceRepoTypeBCS):
		if src.BCSRepoConfig == nil {
			sl.ReportError(nil, "BCSRepoConfig", "BCSRepoConfig", "required", "")
		}
	case string(bkmsapp.HelmSourceRepoTypeGit):
		if src.GitRepoConfig == nil {
			sl.ReportError(nil, "GitRepoConfig", "GitRepoConfig", "required", "")
		}
	}
}

// validateBKMSAppModelSpecInputStruct 校验 bkmsApp 创建请求中的启动配置与环境变量。
func validateBKMSAppModelSpecInputStruct(sl validator.StructLevel) {
	input := sl.Current().Interface().(BKMSAppModelSpecInput)
	validateStartCommand(sl, input.Command, input.Args, input.ShCommand)
	validateEnvVarsUnique(sl, input.EnvVars)
}

// validateUpdateAppStartupConfigInputStruct 校验 bkmsApp 修改请求中的启动配置。
func validateUpdateAppStartupConfigInputStruct(sl validator.StructLevel) {
	input := sl.Current().Interface().(UpdateAppStartupConfigInput)
	validateStartCommand(sl, input.Command, input.Args, input.ShCommand)
}
