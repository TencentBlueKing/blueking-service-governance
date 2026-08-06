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

package trafficmanager

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("New", func() {
	It("should return the local stub traffic manager", func() {
		manager := New()

		Expect(manager).To(BeAssignableToTypeOf(&StubTrafficManager{}))
	})
})

var _ = Describe("StubTrafficManager", func() {
	var (
		stub *StubTrafficManager
		ctx  context.Context
	)

	BeforeEach(func() {
		stub = &StubTrafficManager{}
		ctx = context.Background()
	})

	Describe("GetBaselineTrafficLane", func() {
		It("should return empty TrafficLane without error", func() {
			lane, err := stub.GetBaselineTrafficLane(ctx, "workspace-1", "staging")
			Expect(err).To(BeNil())
			Expect(lane).To(Equal(new(TrafficLane)))
		})
	})

	Describe("GetTrafficLane", func() {
		It("should return empty TrafficLane without error", func() {
			lane, err := stub.GetTrafficLane(ctx, "workspace-1", "staging", "feature-lane")
			Expect(err).To(BeNil())
			Expect(lane).To(Equal(new(TrafficLane)))
		})
	})

	Describe("ListTrafficLanes", func() {
		It("should return empty slice without error", func() {
			lanes, err := stub.ListTrafficLanes(ctx, "workspace-1", "staging")
			Expect(err).To(BeNil())
			Expect(lanes).To(BeEmpty())
			Expect(lanes).NotTo(BeNil())
		})
	})
})
