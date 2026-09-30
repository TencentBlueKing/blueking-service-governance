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

package batchrelease_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
)

var _ = Describe("BuildManifest", func() {
	It("maps a release strategy to a canary rollout of the GameDeployment", func() {
		manifest, err := batchrelease.BuildManifest("GameDeployment", "demo", []releasestrategy.Batch{
			{Target: "1", PauseMode: releasestrategy.PauseModePermanent},
			{Target: "50%", PauseMode: releasestrategy.PauseModeNone},
			{Target: "80%", PauseMode: releasestrategy.PauseModeDuration, PauseSeconds: 1800},
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(manifest).To(Equal(map[string]any{
			"apiVersion": "tkex.tencent.com/v1alpha1",
			"kind":       "Rollout",
			"metadata":   map[string]any{"name": "demo"},
			"spec": map[string]any{
				"type":          "Canary",
				"failurePolicy": "Present",
				"objectRef": map[string]any{
					"workloadRef": map[string]any{
						"apiVersion": "tkex.tencent.com/v1alpha1",
						"kind":       "GameDeployment",
						"name":       "demo",
					},
				},
				"steps": []any{
					map[string]any{"replicas": int64(1), "pause": map[string]any{}},
					map[string]any{"replicas": "50%"},
					map[string]any{"replicas": "80%", "pause": map[string]any{"duration": "1800s"}},
				},
			},
		}))

		// 生成结果必须是 JSON 兼容的 unstructured 对象，且不包含 hook 与 routeControl
		obj := unstructured.Unstructured{Object: manifest}
		Expect(func() { obj.DeepCopy() }).NotTo(Panic())
		steps, _, err := unstructured.NestedSlice(manifest, "spec", "steps")
		Expect(err).NotTo(HaveOccurred())
		for _, step := range steps {
			Expect(step).NotTo(HaveKey("hook"))
			Expect(step).NotTo(HaveKey("routeControl"))
		}
	})

	It("rejects workloads other than GameDeployment", func() {
		_, err := batchrelease.BuildManifest("Deployment", "demo", []releasestrategy.Batch{
			{Target: "1", PauseMode: releasestrategy.PauseModeNone},
		})
		Expect(errors.Is(err, batchrelease.ErrWorkloadNotSupported)).To(BeTrue())
	})

	DescribeTable("rejects invalid batches",
		func(batches []releasestrategy.Batch) {
			_, err := batchrelease.BuildManifest("GameDeployment", "demo", batches)
			Expect(err).To(HaveOccurred())
		},
		Entry("no batches", []releasestrategy.Batch{}),
		Entry("invalid target", []releasestrategy.Batch{
			{Target: "0", PauseMode: releasestrategy.PauseModeNone},
		}),
		Entry("duration pause without seconds", []releasestrategy.Batch{
			{Target: "1", PauseMode: releasestrategy.PauseModeDuration},
		}),
		Entry("unknown pause mode", []releasestrategy.Batch{
			{Target: "1", PauseMode: "forever"},
		}),
	)
})
