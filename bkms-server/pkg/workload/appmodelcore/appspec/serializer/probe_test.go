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
	probesection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/probe"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/serializer"
)

var _ = Describe("Probe serializer", func() {
	It("should convert nil model to nil output", func() {
		Expect(new(serializer.AppSpecProbeOutput).FromModel(nil)).To(BeNil())
	})

	It("should preserve nil probe types in output", func() {
		initialDelaySeconds := int32(3)
		spec := &appspec.ProbeSpec{
			Liveness: &probesection.Probe{
				Handler: &probesection.Handler{
					Type:      appmodel.ProbeTypeExec,
					ShCommand: "curl -f http://127.0.0.1/health || exit 1",
				},
				InitialDelaySeconds: &initialDelaySeconds,
			},
		}

		output := new(serializer.AppSpecProbeOutput).FromModel(spec)

		Expect(output.Liveness).NotTo(BeNil())
		Expect(output.Liveness.ProbeHandler.Type).To(Equal(appmodel.ProbeTypeExec))
		Expect(output.Liveness.ProbeHandler.ShCommand).To(Equal("curl -f http://127.0.0.1/health || exit 1"))
		Expect(output.Liveness.InitialDelaySeconds).To(Equal(int32(3)))
		Expect(output.Readiness).To(BeNil())
		Expect(output.Startup).To(BeNil())
	})

	It("should convert input to model", func() {
		timeoutSeconds := int32(2)
		input := &serializer.ProbeInput{
			ProbeHandler: &serializer.ProbeHandlerInput{
				Type: appmodel.ProbeTypeTCP,
				Port: 8080,
			},
			TimeoutSeconds: &timeoutSeconds,
		}

		probe := input.ToModel()

		Expect(probe.Handler.Type).To(Equal(appmodel.ProbeTypeTCP))
		Expect(probe.Handler.Port).To(Equal(int32(8080)))
		Expect(probe.TimeoutSeconds).NotTo(BeNil())
		Expect(*probe.TimeoutSeconds).To(Equal(int32(2)))
	})
})
