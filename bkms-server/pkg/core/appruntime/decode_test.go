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

package appruntime_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
)

type sampleConfig struct {
	Count  int            `json:"count"`
	Nested sampleNested   `json:"nested"`
	Tags   []string       `json:"tags"`
	Meta   map[string]any `json:"meta"`
}

type sampleNested struct {
	OK bool `json:"ok"`
}

var _ = Describe("DecodeStrict", func() {
	It("decodes nested objects, arrays and integers without unknown keys", func() {
		dest, err := appruntime.DecodeStrict[sampleConfig](map[string]any{
			"count":  int32(3),
			"nested": map[string]any{"ok": true},
			"tags":   []any{"a", "b"},
			"meta":   map[string]any{"retries": int64(2)},
		}, appruntime.FrameworkConfigVersionV1, appruntime.FrameworkConfigVersionV1)
		Expect(err).NotTo(HaveOccurred())
		Expect(dest.Count).To(Equal(3))
		Expect(dest.Nested.OK).To(BeTrue())
		Expect(dest.Tags).To(Equal([]string{"a", "b"}))
		// A JSON round trip would widen this into float64.
		Expect(dest.Meta["retries"]).To(Equal(int64(2)))
	})

	It("decodes BSON containers as they come back from MongoDB", func() {
		dest, err := appruntime.DecodeStrict[sampleConfig](map[string]any{
			"count":  int32(7),
			"nested": bson.D{{Key: "ok", Value: true}},
			"tags":   bson.A{"x"},
			"meta":   bson.D{{Key: "retries", Value: int32(9)}},
		}, appruntime.FrameworkConfigVersionV1, appruntime.FrameworkConfigVersionV1)
		Expect(err).NotTo(HaveOccurred())
		Expect(dest.Count).To(Equal(7))
		Expect(dest.Nested.OK).To(BeTrue())
		Expect(dest.Tags).To(Equal([]string{"x"}))
		Expect(dest.Meta["retries"]).To(Equal(int32(9)))
	})

	It("rejects unknown keys and wrong types", func() {
		_, err := appruntime.DecodeStrict[sampleConfig](map[string]any{
			"count": 1,
			"oops":  true,
		}, appruntime.FrameworkConfigVersionV1, appruntime.FrameworkConfigVersionV1)
		Expect(err).To(MatchError(ContainSubstring("invalid keys: oops")))

		_, err = appruntime.DecodeStrict[sampleConfig](map[string]any{
			"count": "three",
		}, appruntime.FrameworkConfigVersionV1, appruntime.FrameworkConfigVersionV1)
		Expect(err).To(MatchError(ContainSubstring("expected type 'int'")))
	})

	It("rejects unknown keys nested below the top level", func() {
		_, err := appruntime.DecodeStrict[sampleConfig](map[string]any{
			"nested": bson.D{{Key: "ok", Value: true}, {Key: "surprise", Value: 1}},
		}, appruntime.FrameworkConfigVersionV1, appruntime.FrameworkConfigVersionV1)
		Expect(err).To(MatchError(ContainSubstring("'nested' has invalid keys: surprise")))
	})

	It("rejects unknown and missing versions", func() {
		_, err := appruntime.DecodeStrict[sampleConfig](
			map[string]any{"count": 1}, 9, appruntime.FrameworkConfigVersionV1,
		)
		Expect(err).To(MatchError(appruntime.ErrUnknownConfigVersion))

		_, err = appruntime.DecodeStrict[sampleConfig](
			map[string]any{"count": 1}, 0, appruntime.FrameworkConfigVersionV1,
		)
		Expect(err).To(MatchError(appruntime.ErrMissingConfigVersion))
	})

	It("rejects every key when the destination declares none", func() {
		_, err := appruntime.DecodeStrict[struct{}](
			map[string]any{"fileName": "x"},
			appruntime.FrameworkConfigVersionV1,
			appruntime.FrameworkConfigVersionV1,
		)
		Expect(err).To(MatchError(ContainSubstring("invalid keys: fileName")))
	})
})
