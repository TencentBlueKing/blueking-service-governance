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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	lifecyclesection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/lifecycle"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/serializer"
)

var _ = Describe("Lifecycle serializer", func() {
	It("should convert nil model to nil output", func() {
		Expect(new(serializer.AppSpecLifecycleOutput).FromModel(nil)).To(BeNil())
	})

	It("should convert handlers from model to output", func() {
		sleepSeconds := int64(5)
		spec := &appspec.LifecycleSpec{
			PostStart: &lifecyclesection.Handler{
				Type:         appmodel.LifecycleTypeExec,
				ShCommand:    "curl -sf localhost/ready",
				SleepSeconds: &sleepSeconds,
			},
			PreStop: &lifecyclesection.Handler{
				Type: appmodel.LifecycleTypeHTTP,
				URL:  "http://localhost:8080/stop",
			},
		}

		output := new(serializer.AppSpecLifecycleOutput).FromModel(spec)

		Expect(output.PostStart.Type).To(Equal(appmodel.LifecycleTypeExec))
		Expect(output.PostStart.Exec.ShCommand).To(Equal("curl -sf localhost/ready"))
		Expect(output.PostStart.Exec.SleepSeconds).NotTo(BeNil())
		Expect(*output.PostStart.Exec.SleepSeconds).To(Equal("5"))
		Expect(output.PostStart.HTTP).To(BeNil())
		Expect(output.PreStop.Type).To(Equal(appmodel.LifecycleTypeHTTP))
		Expect(output.PreStop.HTTP.URL).To(Equal("http://localhost:8080/stop"))
		Expect(output.PreStop.Exec).To(BeNil())
	})

	It("should convert shell command input to model", func() {
		input := &serializer.AppSpecLifecycleInput{
			PostStart: &serializer.LifecycleHandlerInput{
				Type: appmodel.LifecycleTypeExec,
				Exec: &serializer.LifecycleExecActionInput{
					ShCommand: "curl -sf localhost/ready",
				},
			},
		}

		model := input.ToModel()

		Expect(model.PostStart).NotTo(BeNil())
		Expect(model.PostStart.Type).To(Equal(appmodel.LifecycleTypeExec))
		Expect(model.PostStart.ShCommand).To(Equal("curl -sf localhost/ready"))
		Expect(model.PostStart.Command).To(BeEmpty())
	})
})
