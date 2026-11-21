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
	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace/serializer"
)

var _ = Describe("Linked project serializer", func() {
	DescribeTable("DisableWorkspaceLinkedProjectsInput validation",
		func(systems []string, valid bool) {
			err := binding.Validator.ValidateStruct(serializer.DisableWorkspaceLinkedProjectsInput{
				LinkedProjects: systems,
			})
			Expect(err == nil).To(Equal(valid))
		},
		Entry("bkci and bkmonitor", []string{"bkci", "bkmonitor"}, true),
		Entry("empty", []string{}, false),
		Entry("bcs is not supported", []string{"bcs"}, false),
		Entry("duplicated", []string{"bkci", "bkci"}, false),
	)

	DescribeTable("SetWorkspaceStateInput validation",
		func(systems []string, valid bool) {
			err := binding.Validator.ValidateStruct(serializer.SetWorkspaceStateInput{
				State: "Disabled", LinkedProjects: systems,
			})
			Expect(err == nil).To(Equal(valid))
		},
		Entry("without linked projects", nil, true),
		Entry("with bkmonitor", []string{"bkmonitor"}, true),
		Entry("with unknown system", []string{"bcs"}, false),
	)
})
