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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode/serializer"
)

var _ = Describe("RestartInput", func() {
	// validate 依次执行 binding 校验与 Validate，返回第一个错误
	validate := func(input slz.RestartInput) error {
		if err := binding.Validator.ValidateStruct(input); err != nil {
			return err
		}
		return input.Validate()
	}

	DescribeTable("accepts exactly one valid target",
		func(input slz.RestartInput) {
			Expect(validate(input)).To(Succeed())
		},
		Entry("envs", slz.RestartInput{Envs: &slz.RestartEnvsTarget{Names: []string{"stag", "test"}}}),
		Entry("envTypes", slz.RestartInput{
			EnvTypes: &slz.RestartEnvTypesTarget{Types: []string{"development", "staging"}},
		}),
		Entry("instances", slz.RestartInput{
			Instances: &slz.RestartInstancesTarget{EnvName: "stag", InstanceIDs: []string{"pod-1", "pod-2"}},
		}),
	)

	DescribeTable("rejects invalid input",
		func(input slz.RestartInput, expectedDetail, msg string) {
			err := validate(input)
			Expect(err).To(HaveOccurred())
			if expectedDetail != "" {
				bkErr := bkerrs.FromError(err)
				Expect(bkErr).NotTo(BeNil())
				Expect(bkErr.Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))
				Expect(bkErr.DetailCode()).To(Equal(bkerrs.ErrDetailCode(expectedDetail)))
				return
			}
			Expect(err.Error()).To(ContainSubstring(msg))
		},
		Entry("no target", slz.RestartInput{}, "EXTENSION_DEVMODE_RESTART_SCOPE_CONFLICT", ""),
		Entry("multiple targets", slz.RestartInput{
			Envs:      &slz.RestartEnvsTarget{Names: []string{"stag"}},
			Instances: &slz.RestartInstancesTarget{EnvName: "stag", InstanceIDs: []string{"pod-1"}},
		}, "EXTENSION_DEVMODE_RESTART_SCOPE_CONFLICT", ""),
		Entry("empty env names", slz.RestartInput{Envs: &slz.RestartEnvsTarget{}}, "", "'required' tag"),
		Entry("duplicated env names", slz.RestartInput{
			Envs: &slz.RestartEnvsTarget{Names: []string{"stag", "stag"}},
		}, "", "'unique' tag"),
		Entry("invalid env name", slz.RestartInput{
			Envs: &slz.RestartEnvsTarget{Names: []string{"stag/1"}},
		}, "", "'uri_slug' tag"),
		Entry("production env type", slz.RestartInput{
			EnvTypes: &slz.RestartEnvTypesTarget{Types: []string{"production"}},
		}, "", "'oneof' tag"),
		Entry("unknown env type", slz.RestartInput{
			EnvTypes: &slz.RestartEnvTypesTarget{Types: []string{"dev"}},
		}, "", "'oneof' tag"),
		Entry("instances without env name", slz.RestartInput{
			Instances: &slz.RestartInstancesTarget{InstanceIDs: []string{"pod-1"}},
		}, "", "'required' tag"),
		Entry("instances with empty instance id", slz.RestartInput{
			Instances: &slz.RestartInstancesTarget{EnvName: "stag", InstanceIDs: []string{""}},
		}, "", "'required' tag"),
	)
})
