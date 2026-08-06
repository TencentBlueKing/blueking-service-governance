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

package component_test

import (
	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
)

func mustUnmarshalYAML[T any](s string) T {
	GinkgoHelper()
	var v T
	Expect(yaml.Unmarshal([]byte(s), &v)).To(Succeed())
	return v
}

var _ = Describe("PreviewBuilder", func() {
	It("renders spec and patch preview with partial property overrides and typed expectations", func() {
		// 多属性：WithPropertyValues 只覆盖 mountPath；theme、replicas 走定义默认值
		builder := component.NewPreviewBuilder("mixed-preview", []component.Property{
			{Name: "mountPath", Type: component.PropTypeString, DefaultValue: "/data/default"},
			{Name: "theme", Type: component.PropTypeString, DefaultValue: "dark"},
			{Name: "replicas", Type: component.PropTypeInt, DefaultValue: 2},
		}, []string{`spec:
  replicas: {{ .replicas }}
`, `spec:
  template:
    spec:
      containers:
        - name: "{{ .bkmsContainerName }}"
          volumeMounts:
            - name: data-vol
              mountPath: "{{ .mountPath }}"
`}, []string{`apiVersion: v1
kind: ConfigMap
metadata:
  name: "{{ .name }}-cm"
  namespace: "{{ .bkmsEnvNamespace }}"
data:
  mountPath: "{{ .mountPath }}"
  theme: "{{ .theme }}"
`},
		).WithPropertyValues(map[string]any{
			"mountPath": "/mnt/override",
		})

		result, err := builder.Build()
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Resources).To(HaveLen(1))
		res := result.Resources[0]
		Expect(res.Kind).To(Equal("ConfigMap"))
		Expect(res.Name).To(Equal("preview-app-name-mixed-preview-cm"))
		Expect(res.APIVersion).To(Equal("v1"))

		cm := mustUnmarshalYAML[corev1.ConfigMap](res.Manifest)
		wantCM := corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "preview-app-name-mixed-preview-cm",
				Namespace: "preview-env-namespace",
			},
			Data: map[string]string{
				"mountPath": "/mnt/override",
				"theme":     "dark",
			},
		}
		Expect(cm.TypeMeta).To(Equal(wantCM.TypeMeta))
		Expect(cm.Name).To(Equal(wantCM.Name))
		Expect(cm.Namespace).To(Equal(wantCM.Namespace))
		Expect(cm.Data).To(Equal(wantCM.Data))

		Expect(result.Patches).To(HaveLen(1))
		pp := result.Patches[0]
		Expect(pp.TargetKind).To(Equal("GameDeployment"))

		gd := mustUnmarshalYAML[tkex.GameDeployment](pp.PatchedManifest)
		Expect(gd.Spec.Replicas).To(Equal(lo.ToPtr(int32(2))))
		Expect(gd.Spec.Template.Spec.Containers).To(HaveLen(1))
		main := gd.Spec.Template.Spec.Containers[0]
		// bkmsContainerName is hardcoded in buildBasicBuiltin @component.buildBasicBuiltin
		Expect(main.Name).To(Equal(defaults.WorkloadMainContainerName))
		Expect(main.VolumeMounts).To(Equal([]corev1.VolumeMount{
			{Name: "data-vol", MountPath: "/mnt/override"},
		}))

		gdBase := mustUnmarshalYAML[tkex.GameDeployment](pp.BaseManifest)
		Expect(gdBase.Spec.Replicas).To(BeNil(), "sample base should not set replicas until patch")
	})

	It("leaves GameDeployment manifest unchanged when patcher is empty", func() {
		builder := component.NewPreviewBuilder("no-patch", nil, []string{}, []string{
			"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: no-patch\n",
		})
		result, err := builder.Build()
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Resources).To(HaveLen(1))
		Expect(result.Patches).To(HaveLen(1))
		pp := result.Patches[0]
		Expect(pp.PatchedManifest).To(Equal(pp.BaseManifest))
	})

	It("applies multiple patchers in array order", func() {
		builder := component.NewPreviewBuilder("ordered-patches", nil, []string{
			"spec:\n  replicas: 1\n",
			"spec:\n  replicas: 3\n",
		}, []string{})
		result, err := builder.Build()
		Expect(err).NotTo(HaveOccurred())

		gameDeployment := mustUnmarshalYAML[tkex.GameDeployment](result.Patches[0].PatchedManifest)
		Expect(gameDeployment.Spec.Replicas).To(Equal(lo.ToPtr(int32(3))))
	})

	It("rejects a preview without patchers and specs", func() {
		builder := component.NewPreviewBuilder("empty", nil, []string{}, []string{})
		_, err := builder.Build()
		Expect(err).To(MatchError(ContainSubstring("at least one patcher or spec")))
	})
})
