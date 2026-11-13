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

package deviceid

import (
	"regexp"

	"github.com/keygen-sh/machineid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("compute", func() {
	hexPattern := regexp.MustCompile(`^[0-9a-f]{64}$`)

	fixed := func(id string) func() (string, error) {
		return func() (string, error) { return id, nil }
	}

	DescribeTable("accepts trusted machine ids",
		func(id string) {
			deviceID := compute(fixed(id))
			Expect(deviceID).To(MatchRegexp(hexPattern.String()))
			Expect(deviceID).NotTo(ContainSubstring(id))
		},
		Entry("linux machine-id", "4c4c4544004d3610804bb4c04f4e4a32"),
		Entry("macOS IOPlatformUUID", "8C2F3A4B-1D2E-4F5A-9B8C-7D6E5F4A3B2C"),
		Entry("windows MachineGuid", "8c2f3a4b-1d2e-4f5a-9b8c-7d6e5f4a3b2c"),
		Entry("container id", "3f4e5d6c7b8a99887766554433221100ffeeddccbbaa99887766554433221100"),
	)

	DescribeTable("rejects untrusted machine ids",
		func(id string) {
			Expect(compute(fixed(id))).To(BeEmpty())
		},
		Entry("cgroup slice", "user-1000.slice"),
		Entry("empty", ""),
		Entry("short hex", "abcdef"),
	)

	It("returns empty when machine id cannot be read", func() {
		Expect(compute(func() (string, error) { return "", errors.New("no machine-id found") })).To(BeEmpty())
	})

	It("is stable for the same machine id and differs across machines", func() {
		a := compute(fixed("4c4c4544004d3610804bb4c04f4e4a32"))
		Expect(compute(fixed("4c4c4544004d3610804bb4c04f4e4a32"))).To(Equal(a))
		Expect(compute(fixed("5c4c4544004d3610804bb4c04f4e4a32"))).NotTo(Equal(a))
	})

	It("matches machineid.ProtectedID on the current machine", func() {
		id, err := machineid.ID()
		if err != nil || compute(fixed(id)) == "" {
			Skip("no trusted machine id on the current machine")
		}
		protected, err := machineid.ProtectedID(appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(Get()).To(Equal(protected))
	})
})
