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

package deploy

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
)

var _ = Describe("BuildResourceKeys", func() {
	DescribeTable("builds resource keys in workload-first order",
		func(
			workloadKind, workloadName string,
			extras []unstructured.Unstructured,
			expected appmodeldeploy.ResourceKeys,
		) {
			Expect(BuildResourceKeys(workloadKind, workloadName, extras)).To(Equal(expected))
		},
		Entry("without extra objects",
			"GameDeployment",
			"my-app",
			nil,
			appmodeldeploy.ResourceKeys{
				{Kind: "GameDeployment", Name: "my-app"},
			},
		),
		Entry("with extra objects",
			"Deployment",
			"my-app",
			[]unstructured.Unstructured{
				makeUnstructured("Service", "my-app-svc"),
				makeUnstructured("ConfigMap", "my-app-cfg"),
			},
			appmodeldeploy.ResourceKeys{
				{Kind: "Deployment", Name: "my-app"},
				{Kind: "Service", Name: "my-app-svc"},
				{Kind: "ConfigMap", Name: "my-app-cfg"},
			},
		),
	)
})

func makeUnstructured(kind, name string) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       kind,
			"metadata": map[string]interface{}{
				"name": name,
			},
		},
	}
}
