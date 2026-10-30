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
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy/serializer"
)

func inputWithBatch(batch serializer.BatchInput) serializer.ReleaseStrategyInput {
	return serializer.ReleaseStrategyInput{Name: "abcde", Batches: []serializer.BatchInput{batch}}
}

func pauseInput(pauseMode string, pauseSeconds int) serializer.ReleaseStrategyInput {
	return inputWithBatch(serializer.BatchInput{Target: "1", PauseMode: pauseMode, PauseSeconds: pauseSeconds})
}

func repeatBatches(n int) []serializer.BatchInput {
	batches := make([]serializer.BatchInput, n)
	for i := range batches {
		batches[i] = serializer.BatchInput{Target: "1", PauseMode: "none"}
	}
	return batches
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
		// 名称长度按 trim 后的字符数计算，首尾空白不入库
		Entry("name with 64 multi-byte characters", serializer.ReleaseStrategyInput{
			Name:    strings.Repeat("名", 64),
			Batches: []serializer.BatchInput{{Target: "1", PauseMode: "none"}},
		}, ""),
		Entry("name with surrounding spaces within 64 characters", serializer.ReleaseStrategyInput{
			Name:    "  " + strings.Repeat("名", 64) + "  ",
			Batches: []serializer.BatchInput{{Target: "1", PauseMode: "none"}},
		}, ""),
		Entry("blank name", serializer.ReleaseStrategyInput{
			Name:    "   ",
			Batches: []serializer.BatchInput{{Target: "1", PauseMode: "none"}},
		}, "Name"),
		Entry("name longer than 64 characters after trim", serializer.ReleaseStrategyInput{
			Name:    " " + strings.Repeat("名", 65),
			Batches: []serializer.BatchInput{{Target: "1", PauseMode: "none"}},
		}, "Name"),
		Entry("eleven batches", serializer.ReleaseStrategyInput{
			Name:    "abcde",
			Batches: repeatBatches(11),
		}, ""),
		Entry("no batches", serializer.ReleaseStrategyInput{
			Name:    "abcde",
			Batches: nil,
		}, "Batches"),
		Entry("target rejected by batch_target",
			inputWithBatch(serializer.BatchInput{Target: "101%", PauseMode: "none"}), "Target"),

		// pauseSeconds 仅在 duration 时必填且只能在此时填写
		Entry("duration pause at lower bound", pauseInput("duration", 1), ""),
		Entry("duration pause longer than a day", pauseInput("duration", 7*86400), ""),
		Entry("duration pause without seconds", pauseInput("duration", 0), "PauseSeconds"),
		Entry("duration pause with negative seconds", pauseInput("duration", -1), "PauseSeconds"),
		Entry("seconds given for permanent pause", pauseInput("permanent", 120), "PauseSeconds"),
		Entry("seconds given for no pause", pauseInput("none", 120), "PauseSeconds"),
	)

	It("accepts every builtin strategy as user input", func() {
		for _, s := range releasestrategy.BuiltinStrategies() {
			input := serializer.ReleaseStrategyInput{
				Name:        s.Name,
				Description: s.Description,
				Batches: lo.Map(s.Batches, func(b releasestrategy.Batch, _ int) serializer.BatchInput {
					return serializer.BatchInput{
						Target:       b.Target,
						PauseMode:    string(b.PauseMode),
						PauseSeconds: b.PauseSeconds,
					}
				}),
			}
			Expect(binding.Validator.ValidateStruct(input)).To(Succeed(), s.Name)
		}
	})
})

var _ = Describe("ReleaseStrategyOutputObj", func() {
	It("marks builtin strategies and leaves their audit fields empty", func() {
		builtin := releasestrategy.BuiltinStrategies()[0]
		builtin.WorkspaceID = "demo-space"
		output := new(serializer.ReleaseStrategyOutputObj).FromModel(builtin)

		Expect(output.ID).To(Equal(builtin.ID.Hex()))
		Expect(output.IsBuiltin).To(BeTrue())
		Expect(output.WorkspaceID).To(Equal("demo-space"))
		Expect(output.Creator).To(BeEmpty())
		Expect(output.Updater).To(BeEmpty())
		Expect(output.CreatedAt).To(BeNil())
		Expect(output.UpdatedAt).To(BeNil())
		Expect(output.Batches).To(HaveLen(len(builtin.Batches)))

		raw, err := json.Marshal(output)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(raw)).To(ContainSubstring(`"createdAt":null`))
		Expect(string(raw)).To(ContainSubstring(`"isBuiltin":true`))
	})

	It("keeps ownership and audit fields of custom strategies", func() {
		createdAt := time.Date(2026, 10, 22, 2, 0, 0, 0, time.UTC)
		output := new(serializer.ReleaseStrategyOutputObj).FromModel(releasestrategy.ReleaseStrategy{
			ID:          bson.NewObjectID(),
			WorkspaceID: "demo-space",
			Name:        "careful",
			Batches:     []releasestrategy.Batch{{Target: "1", PauseMode: releasestrategy.PauseModeNone}},
			Creator:     "admin",
			CreatedAt:   createdAt,
			Updater:     "operator",
			UpdatedAt:   createdAt.Add(time.Hour),
		})

		Expect(output.IsBuiltin).To(BeFalse())
		Expect(output.WorkspaceID).To(Equal("demo-space"))
		Expect(output.Creator).To(Equal("admin"))
		Expect(output.Updater).To(Equal("operator"))
		Expect(output.CreatedAt).To(PointTo(Equal(createdAt)))
		Expect(output.UpdatedAt).To(PointTo(Equal(createdAt.Add(time.Hour))))
	})
})
