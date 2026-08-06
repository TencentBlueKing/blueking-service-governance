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

package appspec

import (
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
)

// FromAppModel builds an app spec from an AppModel.
func FromAppModel(appID, envName string, appModel *appmodel.AppModel) *AppSpec {
	spec := &AppSpec{
		AppID:   appID,
		EnvName: envName,
	}
	for _, section := range registeredSections {
		section.fillFromAppModel(spec, appModel)
	}
	return spec
}

// ApplyToAppModel applies fields managed by appspec into the AppModel.
func ApplyToAppModel(spec *AppSpec, appModel *appmodel.AppModel) *appmodel.AppModel {
	if spec == nil {
		return appModel
	}
	for _, section := range registeredSections {
		section.applyToAppModel(spec, appModel)
	}
	return appModel
}

// ResetAppModelToDefaultValues reset the values of the fields related with DeploySpec of the
// given AppModel to the default, it's mainly used when initializing a newly created AppModel.
func ResetAppModelToDefaultValues(appModel *appmodel.AppModel) {
	appModel.Replicas = lo.ToPtr(int32(1))
	appModel.UpdateStrategy = &appmodel.UpdateStrategy{
		MaxUnavailable: lo.ToPtr(defaults.MaxUnavailable),
		MaxSurge:       lo.ToPtr(defaults.MaxSurge),
	}
	appModel.Workload.Resources = map[string]string{
		"cpu":    "1-2",
		"memory": "2Gi-4Gi",
	}
}
