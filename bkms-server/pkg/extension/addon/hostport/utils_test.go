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

package hostport_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/hostport"
)

var _ = Describe("NormalizePorts", func() {
	It("returns empty slice for nil or empty input", func() {
		Expect(hostport.NormalizePorts(nil)).To(Equal([]int32{}))
		Expect(hostport.NormalizePorts([]int32{})).To(Equal([]int32{}))
	})

	It("deduplicates and sorts ports ascending", func() {
		Expect(hostport.NormalizePorts([]int32{8080, 80, 80, 443})).To(Equal([]int32{80, 443, 8080}))
	})
})

var _ = Describe("DiffPorts", func() {
	It("returns empty diffs when sets are equal", func() {
		add, remove := hostport.DiffPorts([]int32{80, 8080}, []int32{8080, 80})
		Expect(add).To(BeEmpty())
		Expect(remove).To(BeEmpty())
	})

	It("detects pending add and remove", func() {
		add, remove := hostport.DiffPorts([]int32{80, 443}, []int32{80, 8080})
		Expect(add).To(Equal([]int32{443}))
		Expect(remove).To(Equal([]int32{8080}))
	})
})

var _ = Describe("ComputeEnvState", func() {
	It("treats missing snapshot with desired ports as pending create", func() {
		view := hostport.ComputeEnvState([]int32{80, 8080}, nil)
		Expect(view.AppliedPorts).To(BeEmpty())
		Expect(view.PendingAddPorts).To(Equal([]int32{80, 8080}))
		Expect(view.PendingRemovePorts).To(BeEmpty())
	})

	It("treats missing snapshot with empty desired as not pending", func() {
		view := hostport.ComputeEnvState(nil, nil)
		Expect(view.PendingAddPorts).To(BeEmpty())
		Expect(view.PendingRemovePorts).To(BeEmpty())
	})

	It("computes add and remove against applied snapshot", func() {
		state := &hostport.HostPortEnvState{AppliedPorts: []int32{80, 8080}}
		view := hostport.ComputeEnvState([]int32{80, 443}, state)
		Expect(view.AppliedPorts).To(Equal([]int32{80, 8080}))
		Expect(view.PendingAddPorts).To(Equal([]int32{443}))
		Expect(view.PendingRemovePorts).To(Equal([]int32{8080}))
	})
})

var _ = Describe("BuildPodAnnotations", func() {
	It("returns nil for empty ports", func() {
		Expect(hostport.BuildPodAnnotations(nil)).To(BeNil())
	})

	It("builds webhook annotations with sorted ports", func() {
		anns := hostport.BuildPodAnnotations([]int32{8080, 80})
		Expect(anns).To(Equal(map[string]string{
			hostport.AnnotationEnabledKey: hostport.AnnotationEnabledVal,
			hostport.AnnotationPortsKey:   "80,8080",
		}))
	})
})

var _ = Describe("FormatPortsAnnotationValue", func() {
	It("joins sorted unique ports as comma-separated string", func() {
		Expect(hostport.FormatPortsAnnotationValue([]int32{8080, 80, 80})).To(Equal("80,8080"))
	})

	It("returns empty string for empty ports", func() {
		Expect(hostport.FormatPortsAnnotationValue(nil)).To(Equal(""))
	})
})
