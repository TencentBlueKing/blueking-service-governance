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
	"strings"

	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy/serializer"
)

func inputWithBatch(batch serializer.BatchInput) serializer.ReleaseStrategyInput {
	return serializer.ReleaseStrategyInput{Name: "abcde", Batches: []serializer.BatchInput{batch}}
}

func pauseInput(pauseMode string, pauseSeconds int) serializer.ReleaseStrategyInput {
	return inputWithBatch(serializer.BatchInput{Target: "1", PauseMode: pauseMode, PauseSeconds: pauseSeconds})
}

var _ = Describe("ReleaseStrategyInput", func() {
	DescribeTable(
		"validates strategy fields",
		func(input serializer.ReleaseStrategyInput, invalidField string) {
			err := binding.Validator.ValidateStruct(input)
			if invalidField == "" {
				Expect(err).NotTo(HaveOccurred())
				return
			}
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(invalidField))
		},
		// 名称长度按字符而非字节计算
		Entry("name with 32 multi-byte characters", serializer.ReleaseStrategyInput{
			Name:    strings.Repeat("名", 32),
			Batches: []serializer.BatchInput{{Target: "1", PauseMode: "none"}},
		}, ""),
		Entry("target rejected by batch_target",
			inputWithBatch(serializer.BatchInput{Target: "101%", PauseMode: "none"}), "Target"),

		// pauseSeconds 仅在 duration 时必填且只能在此时填写
		Entry("duration pause at lower bound", pauseInput("duration", 60), ""),
		Entry("duration pause at upper bound", pauseInput("duration", 86400), ""),
		Entry("duration pause without seconds", pauseInput("duration", 0), "PauseSeconds"),
		Entry("duration pause too short", pauseInput("duration", 59), "PauseSeconds"),
		Entry("duration pause too long", pauseInput("duration", 86401), "PauseSeconds"),
		Entry("seconds given for permanent pause", pauseInput("permanent", 120), "PauseSeconds"),
		Entry("seconds given for no pause", pauseInput("none", 120), "PauseSeconds"),
	)
})
