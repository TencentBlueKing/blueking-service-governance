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

package workload

import (
	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
	gamedeploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/workload/gamedeployment"
)

// gameDeploymentDriver GameDeployment 的处理入口
type gameDeploymentDriver struct{}

// Kind ...
func (gameDeploymentDriver) Kind() string {
	return k8skind.GameDeploy
}

// GVR ...
func (gameDeploymentDriver) GVR() schema.GroupVersionResource {
	return gvr.GameDeploy
}

// ParseStatus 联邦集群只下发原生 Deployment，因此不区分 opts.Federation
func (gameDeploymentDriver) ParseStatus(manifest map[string]any, _ ParseOptions) (*k8sstatus.Result, error) {
	return gamedeploystatus.Parse(manifest)
}

// View ...
func (gameDeploymentDriver) View(manifest map[string]any) (*View, error) {
	var gd tkex.GameDeployment
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(manifest, &gd); err != nil {
		return nil, errors.Wrap(err, "convert unstructured to GameDeployment")
	}
	return &View{
		Replicas:   gd.Spec.Replicas,
		Containers: gd.Spec.Template.Spec.Containers,
	}, nil
}

// Capabilities ...
func (gameDeploymentDriver) Capabilities() Capabilities {
	return Capabilities{InplaceUpdate: true, SelectedPodDeletion: true}
}
