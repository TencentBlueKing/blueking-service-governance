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

package workload

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

var _ = Describe("buildLifecycle", func() {
	It("should build exec shell lifecycle handler as exec with sh -c", func() {
		lifecycle := &appmodel.Lifecycle{
			PreStop: &appmodel.LifecycleHandler{
				TypeWrapper: appmodel.TypeWrapper{Type: appmodel.LifecycleTypeExec},
				ExecAction:  &appmodel.ExecAction{ShCommand: "echo stopping && exit 0"},
			},
		}

		result, err := buildLifecycle(lifecycle)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())
		Expect(result.PreStop).NotTo(BeNil())
		Expect(result.PreStop.Exec.Command).To(Equal([]string{"/bin/sh", "-c", "echo stopping && exit 0"}))
	})

	It("should reject exec lifecycle handler with command and shell command", func() {
		lifecycle := &appmodel.Lifecycle{
			PostStart: &appmodel.LifecycleHandler{
				TypeWrapper: appmodel.TypeWrapper{Type: appmodel.LifecycleTypeExec},
				ExecAction: &appmodel.ExecAction{
					Command:   []string{"echo", "ok"},
					ShCommand: "echo ok",
				},
			},
		}

		_, err := buildLifecycle(lifecycle)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("command and sh command are mutually exclusive"))
	})
})
