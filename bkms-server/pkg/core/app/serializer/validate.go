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

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
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
	case bkmsapp.AppTypeTRPC:
		if input.AppModelSpec == nil {
			sl.ReportError(nil, "AppModelSpec", "AppModelSpec", "required", "")
			return
		}
		if input.AppModelSpec.TrpcSpec == nil {
			sl.ReportError(nil, "AppModelSpec.TrpcSpec", "TrpcSpec", "required", "")
			return
		}
		validateAppModelSpecEnvVarsUnique(sl, input.AppModelSpec)
	case bkmsapp.AppTypeTAF:
		if input.AppModelSpec == nil {
			sl.ReportError(nil, "AppModelSpec", "AppModelSpec", "required", "")
			return
		}
		if input.AppModelSpec.TafSpec == nil {
			sl.ReportError(nil, "AppModelSpec.TafSpec", "TafSpec", "required", "")
			return
		}
		validateAppModelSpecEnvVarsUnique(sl, input.AppModelSpec)
	case bkmsapp.AppTypeHelm, bkmsapp.AppTypeAgones:
		if input.HelmSpec == nil {
			sl.ReportError(nil, "HelmSpec", "HelmSpec", "required", "")
			return
		}
		// HelmSpec.HelmSource 的 required 校验由字段 binding tag 保证；
		// HelmSource 内部 RepoType 与各 Config 的关联校验由
		// HelmSourceInput 的 struct-level validator 统一处理。
	}
}

// validateAppModelSpecEnvVarsUnique validate app unique env vars key
func validateAppModelSpecEnvVarsUnique(sl validator.StructLevel, spec *AppModelSpecInput) {
	if spec == nil || len(spec.EnvVars) == 0 {
		return
	}

	seen := make(map[string]struct{}, len(spec.EnvVars))
	for _, envVar := range spec.EnvVars {
		if _, ok := seen[envVar.Key]; ok {
			sl.ReportError(
				spec.EnvVars,
				"AppModelSpec.EnvVars",
				"EnvVars",
				"env_var_key_unique",
				envVar.Key)
			return
		}
		seen[envVar.Key] = struct{}{}
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
