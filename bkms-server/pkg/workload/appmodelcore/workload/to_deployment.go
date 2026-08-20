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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

// gameDeploymentToDeployment 将已构建的 GameDeployment 转成原生 Deployment。
// 原因是目前联邦集群对 GameDeployment 支持有限；
func gameDeploymentToDeployment(gd tkex.GameDeployment) *appsv1.Deployment {
	template := gd.Spec.Template.DeepCopy()

	var selector *metav1.LabelSelector
	if gd.Spec.Selector != nil {
		selector = gd.Spec.Selector.DeepCopy()
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       kind.Deploy,
			APIVersion: gvr.Deploy.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        gd.Name,
			Namespace:   gd.Namespace,
			Labels:      gd.Labels,
			Annotations: gd.Annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: gd.Spec.Replicas,
			Selector: selector,
			Template: *template,
			Strategy: appsv1.DeploymentStrategy{
				// 目前 Deployment 只支持滚动更新
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: gd.Spec.UpdateStrategy.MaxUnavailable,
					MaxSurge:       gd.Spec.UpdateStrategy.MaxSurge,
				},
			},
		},
	}
}
