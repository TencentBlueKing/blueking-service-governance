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

package json_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/json"
)

var _ = Describe("MarshalNestedFromDotPath", func() {
	Context("when path is a dot-separated field chain", func() {
		It("should marshal a nested object with the leaf at the path", func() {
			expr := "spec.replicas"
			result, err := json.MarshalNestedFromDotPath(expr, 3)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(MatchJSON(`{"spec": {"replicas": 3}}`))
		})
	})
})
