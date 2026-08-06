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

package lifecycle

import (
	"github.com/go-playground/validator/v10"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

var _ = Describe("validation", func() {
	var validate *validator.Validate

	BeforeEach(func() {
		validate = validator.New(validator.WithRequiredStructEnabled())
		RegisterValidation(validate)
	})

	It("should allow exec handler with shell command", func() {
		spec := Spec{
			PostStart: &Handler{
				Type:      appmodel.LifecycleTypeExec,
				ShCommand: "curl -sf localhost/ready",
			},
		}

		err := validate.Struct(spec)

		Expect(err).NotTo(HaveOccurred())
	})

	It("should reject exec handler without command, shell command, or sleep seconds", func() {
		spec := Spec{
			PostStart: &Handler{
				Type: appmodel.LifecycleTypeExec,
			},
		}

		err := validate.Struct(spec)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("required_command_or_sh_command_or_sleep_seconds"))
	})

	It("should reject exec handler with command and shell command", func() {
		spec := Spec{
			PreStop: &Handler{
				Type:      appmodel.LifecycleTypeExec,
				Command:   []string{"echo", "ok"},
				ShCommand: "echo ok",
			},
		}

		err := validate.Struct(spec)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("exec_command_or_sh_command_exclusive"))
	})
})
