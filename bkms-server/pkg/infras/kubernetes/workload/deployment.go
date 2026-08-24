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
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
	deploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/workload/deployment"
)

// deploymentDriver 原生 Deployment 的处理入口
type deploymentDriver struct{}

// Kind ...
func (deploymentDriver) Kind() string {
	return k8skind.Deploy
}

// GVR ...
func (deploymentDriver) GVR() schema.GroupVersionResource {
	return gvr.Deploy
}

// ParseStatus ...
func (deploymentDriver) ParseStatus(manifest map[string]any, opts ParseOptions) (*k8sstatus.Result, error) {
	if opts.Federation {
		return deploystatus.ParseForFederation(manifest), nil
	}
	return deploystatus.Parse(manifest), nil
}

// View ...
func (deploymentDriver) View(manifest map[string]any) (*View, error) {
	var deploy appsv1.Deployment
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(manifest, &deploy); err != nil {
		return nil, errors.Wrap(err, "convert unstructured to Deployment")
	}
	return &View{
		Replicas:   deploy.Spec.Replicas,
		Containers: deploy.Spec.Template.Spec.Containers,
	}, nil
}

// Capabilities 原生 Deployment 只能整体滚动更新，既不支持原地升级，也无法指定要删除的 Pod
func (deploymentDriver) Capabilities() Capabilities {
	return Capabilities{}
}
