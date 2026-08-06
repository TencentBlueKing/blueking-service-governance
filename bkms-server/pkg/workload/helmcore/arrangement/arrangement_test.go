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

package arrangement

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("Test AppArranger", func() {
	var appStore *bkmsapp.ApplicationStoreMongo
	var ctx context.Context
	var app bkmsapp.Application

	BeforeEach(func() {
		var err error

		appStore, err = bkmsapp.NewApplicationStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		ctx = context.Background()
		appName := "test-app-" + stringx.Random(6)
		app = bkmsapp.Application{
			ID:          appName + stringx.Random(6),
			Name:        appName,
			WorkspaceID: "test-workspace-" + stringx.Random(6),
		}

		// Create the application first to make the further operations such as saving override values file work
		err = appStore.CreateApp(ctx, &app)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Validate file content", func() {
		It("should return error if the YAML is malformed", func() {
			arranger := NewAppArranger(appStore)
			// Malformed YAML, a list instead of a map
			malformedYaml := []byte("[1, 2, 3]")

			// Validate the YAML
			_, err := arranger.ValidateFileContent(ctx, &app, malformedYaml, appcfg.FileFormatYAML)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("parsing values.yaml: "))
		})

		It("Should validate YAML with all placeholders correctly", func() {
			arranger := NewAppArranger(appStore)

			// YAML with all required placeholders
			yamlWithAllPlaceholders := []byte(`
global:
  image: ${{ bkms.ARTIFACT_IMAGE }}
  host: ${{ bkms.NETWORKING_INGRESS_DOMAIN }}
`)

			// Validate the YAML
			result, err := arranger.ValidateFileContent(ctx, &app, yamlWithAllPlaceholders, appcfg.FileFormatYAML)

			Expect(err).NotTo(HaveOccurred())
			// Both validations should pass
			Expect(result).To(Equal(ArrgResult{
				WorkloadImage: ArrgResultItem{
					Status:        ArrgStatusConfigured,
					SkippedReason: "",
				},
				IngressDomain: ArrgResultItem{
					Status:        ArrgStatusConfigured,
					SkippedReason: "",
				},
			}))
		})

		It("Should detect missing placeholders during validation", func() {
			arranger := NewAppArranger(appStore)

			// YAML without any of the required placeholders
			yamlWithoutPlaceholders := []byte(`
global:
  image: some-static-image:latest
  host: example.com
`)

			// Validate the YAML
			result, err := arranger.ValidateFileContent(ctx, &app, yamlWithoutPlaceholders, appcfg.FileFormatYAML)
			// Both validations should fail
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ArrgResult{
				WorkloadImage: ArrgResultItem{
					Status:        ArrgStatusSkipped,
					SkippedReason: "missing image placeholders in values.yaml",
				},
				IngressDomain: ArrgResultItem{
					Status:        ArrgStatusSkipped,
					SkippedReason: "missing ingress domain placeholder in values.yaml",
				},
			}))
		})
	})

	Context("arrangement type: workloadImage", func() {
		// Valid yaml with placeholders
		yamlWithImage := []byte(`global:
  image: ${{ bkms.ARTIFACT_IMAGE }}`)

		yamlWithImageNested := []byte(`global:
  workload:
    - first
    - name: second
      image: ${{ bkms.ARTIFACT_IMAGE }}`)

		yamlWithNameAndTag := []byte(`global:
  imageName: ${{ bkms.ARTIFACT_IMAGE_NAME }}
  tag: ${{ bkms.ARTIFACT_IMAGE_TAG }}`)

		yamlWithRegistryAndRepositoryAndTag := []byte(`global:
  registry: ${{ bkms.ARTIFACT_IMAGE_REGISTRY }}
  repository: ${{ bkms.ARTIFACT_IMAGE_REPOSITORY }}
  tag: ${{ bkms.ARTIFACT_IMAGE_TAG }}`)

		DescribeTable("Should succeed if contains placeholders", func(yamlContent []byte) {
			vars, err := render.ExtractVars(string(yamlContent))
			Expect(err).NotTo(HaveOccurred())

			err = validateValuesYamlWorkloadImage(vars)
			Expect(err).NotTo(HaveOccurred())
		},
			Entry("YAML with image", yamlWithImage),
			Entry("YAML with image name and tag", yamlWithNameAndTag),
			Entry("YAML with image registry and repository and tag", yamlWithRegistryAndRepositoryAndTag),
			Entry("deeply nested YAML with image", yamlWithImageNested),
		)

		// Invalid yaml without placeholders
		DescribeTable("Should fail if no placeholders can be found", func(yamlContent []byte) {
			vars, err := render.ExtractVars(string(yamlContent))
			Expect(err).NotTo(HaveOccurred())

			err = validateValuesYamlWorkloadImage(vars)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("missing image placeholders in values.yaml"))
		},
			Entry("YAML without placeholders", []byte(`foo: bar`)),
			Entry("YAML with only image name", []byte(`imageName: ${{ bkms.ARTIFACT_IMAGE_NAME }}`)),
		)
	})

	Context("arrangement type: ingressDomain", func() {
		// Valid yaml with placeholder
		validYaml := []byte(`global:
  ingress: ${{ bkms.NETWORKING_INGRESS_DOMAIN }}`)

		validYamlNested := []byte(`global:
  settings:
    host: ${{ bkms.NETWORKING_INGRESS_DOMAIN }}`)

		DescribeTable("Should succeed if contains ingress domain placeholder", func(yamlContent []byte) {
			vars, err := render.ExtractVars(string(yamlContent))
			Expect(err).NotTo(HaveOccurred())

			err = validateValuesYamlIngressDomain(vars)
			Expect(err).NotTo(HaveOccurred())
		},
			Entry("YAML with ingress", validYaml),
			Entry("deeply nested YAML with ingress", validYamlNested),
		)

		// Invalid yaml without ingress domain placeholder
		DescribeTable("Should fail if ingress domain placeholder is not found", func(yamlContent []byte) {
			vars, err := render.ExtractVars(string(yamlContent))
			Expect(err).NotTo(HaveOccurred())

			err = validateValuesYamlIngressDomain(vars)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("missing ingress domain placeholder in values.yaml"))
		},
			Entry("yaml without ingress", []byte(`foo: bar`)),
		)
	})
})
