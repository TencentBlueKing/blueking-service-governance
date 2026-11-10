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
package serializer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/lifecycle"
)

var _ = Describe("Legacy API compatibility", func() {
	Describe("CreateAppInput.ToBKMSAppCompat", func() {
		It("converts a trpc request into a bkmsApp create request", func() {
			input := &serializer.CreateAppInput{
				Type: "trpc",
				AppModelSpec: &serializer.AppModelSpecInput{
					Command: []string{"/app/server"},
					EnvVars: []serializer.VariableInput{{Key: "FOO", Value: "bar"}},
					TrpcSpec: &serializer.TrpcSpecInput{
						Language: "cpp", FileName: "trpc_cpp.yaml", FilePath: "/usr/local/trpc/bin", FileContent: "a: b",
					},
				},
			}

			framework, language, params := input.ToBKMSAppCompat()

			Expect(framework).To(Equal("trpc"))
			Expect(language).To(Equal("cpp"))
			Expect(params.Command).To(Equal([]string{"/app/server"}))
			Expect(params.EnvVars).To(Equal([]appmodel.Variable{{Key: "FOO", Value: "bar"}}))
			Expect(params.FrameworkFile).To(Equal(&lifecycle.FrameworkFileParams{
				Name: "trpc_cpp.yaml", MountDir: "/usr/local/trpc/bin", Content: "a: b",
			}))
		})

		It("converts a taf request with cpp as the language", func() {
			input := &serializer.CreateAppInput{
				Type: "taf",
				AppModelSpec: &serializer.AppModelSpecInput{
					ShCommand: "./start.sh",
					TafSpec:   &serializer.TafSpecInput{FileName: "app.conf", FilePath: "/usr/local/taf/conf"},
				},
			}

			framework, language, params := input.ToBKMSAppCompat()

			Expect(framework).To(Equal("taf"))
			Expect(language).To(Equal("cpp"))
			Expect(params.ShCommand).To(Equal("./start.sh"))
			Expect(params.FrameworkFile.Name).To(Equal("app.conf"))
			Expect(params.FrameworkFile.MountDir).To(Equal("/usr/local/taf/conf"))
		})
	})

	Describe("AppModelSpecInput.ToSpecCompatParams", func() {
		It("takes language and file location from trpcSpec", func() {
			input := &serializer.AppModelSpecInput{
				Args:     []string{"--debug"},
				TrpcSpec: &serializer.TrpcSpecInput{Language: "go", FileName: "trpc_go.yaml", FilePath: "/conf"},
				TafSpec:  &serializer.TafSpecInput{FileName: "ignored.conf"},
			}

			Expect(input.ToSpecCompatParams("trpc")).To(Equal(&lifecycle.SpecCompatParams{
				Framework: "trpc",
				Args:      []string{"--debug"},
				Language:  "go",
				FileName:  "trpc_go.yaml",
				FilePath:  "/conf",
			}))
		})

		It("takes the file location from tafSpec and leaves language empty", func() {
			input := &serializer.AppModelSpecInput{
				TafSpec: &serializer.TafSpecInput{FileName: "app.conf", FilePath: "/conf"},
			}

			Expect(input.ToSpecCompatParams("taf")).To(Equal(&lifecycle.SpecCompatParams{
				Framework: "taf", FileName: "app.conf", FilePath: "/conf",
			}))
		})
	})
})
