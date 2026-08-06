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

package topology

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
	gpastatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/autoscaler/gpa"
	hpastatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/autoscaler/hpa"
	ingstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/network/ingress"
	polarisstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/network/polaris"
	dsstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/workload/daemonset"
	deploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/workload/deployment"
	gamedeploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/workload/gamedeployment"
	gamestsstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/workload/gamestatefulset"
	podstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/workload/pod"
	rsstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/workload/replicaset"
	stsstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/workload/statefulset"
)

// getResourceStatus 计算资源的综合状态评估结果
// 根据 kind 调度到对应的 status parser 进行解析，返回包含 Code 和 Message 的 k8sstatus.Result
func getResourceStatus(kind string, obj *unstructured.Unstructured) *k8sstatus.Result {
	if obj == nil {
		return &k8sstatus.Result{Code: k8sstatus.Unknown}
	}

	manifest := obj.Object

	switch kind {
	// 工作负载类资源：调用专属 parser
	case k8skind.Deploy:
		return deploystatus.Parse(manifest)
	case k8skind.STS:
		return stsstatus.Parse(manifest)
	case k8skind.DS:
		return dsstatus.Parse(manifest)
	case k8skind.GameDeploy:
		result, err := gamedeploystatus.Parse(manifest)
		if err != nil {
			return &k8sstatus.Result{Code: k8sstatus.Unknown}
		}
		return result
	case k8skind.GameSTS:
		result, err := gamestsstatus.Parse(manifest)
		if err != nil {
			return &k8sstatus.Result{Code: k8sstatus.Unknown}
		}
		return result
	case k8skind.Po:
		return podstatus.NewParser(manifest).Parse()
	case k8skind.RS:
		return rsstatus.Parse(manifest)

	// 自动扩缩容类资源
	case k8skind.HPA:
		return hpastatus.Parse(manifest)
	case k8skind.GPA:
		return gpastatus.Parse(manifest)

	// 网络类资源
	case k8skind.Ing:
		return ingstatus.Parse(manifest)
	case k8skind.PolarisCfg:
		return polarisstatus.Parse(manifest)

	// 稳态资源：始终健康
	case k8skind.SVC, k8skind.CM, k8skind.Secret, k8skind.SA, k8skind.NS:
		return &k8sstatus.Result{Code: k8sstatus.Healthy}

	default:
		return &k8sstatus.Result{Code: k8sstatus.Unknown}
	}
}
