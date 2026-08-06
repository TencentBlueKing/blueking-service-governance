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

package labels

import (
	"maps"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// FromAppModel builds the section from an AppModel.
func FromAppModel(appModel *appmodel.AppModel) *Spec {
	if appModel == nil {
		return nil
	}
	return Clone(&Spec{Labels: appModel.Labels})
}

// ApplyToAppModel applies the section into AppModel, fully replacing appModel.Labels.
// A nil spec clears the labels.
func ApplyToAppModel(spec *Spec, appModel *appmodel.AppModel) *appmodel.AppModel {
	if spec == nil {
		appModel.Labels = nil
		return appModel
	}
	appModel.Labels = maps.Clone(spec.Labels)
	return appModel
}
