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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/serializer"
)

var _ = Describe("Labels serializer", func() {
	It("should convert nil model to nil output", func() {
		Expect(new(serializer.AppSpecLabelsOutput).FromModel(nil)).To(BeNil())
	})

	It("should preserve labels when converting model to output", func() {
		spec := &appspec.LabelsSpec{Labels: map[string]string{"team": "sre"}}
		output := new(serializer.AppSpecLabelsOutput).FromModel(spec)
		Expect(output).NotTo(BeNil())
		Expect(output.Labels).To(Equal(map[string]string{"team": "sre"}))
	})

	It("should convert nil input to nil model", func() {
		var input *serializer.AppSpecLabelsInput
		Expect(input.ToModel()).To(BeNil())
	})

	It("should trim spaces from keys and values", func() {
		input := &serializer.AppSpecLabelsInput{Labels: map[string]string{"  team  ": " sre "}}
		spec := input.ToModel()
		Expect(spec).NotTo(BeNil())
		Expect(spec.Labels).To(Equal(map[string]string{"team": "sre"}))
	})
})
