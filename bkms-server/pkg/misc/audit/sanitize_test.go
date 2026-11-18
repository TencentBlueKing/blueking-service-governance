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

package audit

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type secretHolder struct {
	Key    string
	Secret string
}

func init() {
	RegisterSanitizer(func(h secretHolder) any {
		h.Secret = MaskString(h.Secret)
		return h
	})
}

var _ = Describe("Sanitize", func() {
	apply := func(opt Option) *OperationRecord {
		r := &OperationRecord{}
		opt(r)
		return r
	}

	Describe("data options", func() {
		It("masks registered type and its pointer", func() {
			holder := &secretHolder{Key: "k", Secret: "plain"}

			Expect(string(apply(WithDataBefore(holder)).Data.Before)).To(MatchJSON(`{"Key":"k","Secret":"******"}`))
			Expect(string(apply(WithDataAfter(*holder)).Data.After)).To(MatchJSON(`{"Key":"k","Secret":"******"}`))
			Expect(string(apply(WithExtras(holder)).Data.Extras)).To(MatchJSON(`{"Key":"k","Secret":"******"}`))
			// 原对象不被修改
			Expect(holder.Secret).To(Equal("plain"))
		})

		It("keeps typed nil pointer as null", func() {
			var holder *secretHolder
			Expect(string(apply(WithDataBefore(holder)).Data.Before)).To(Equal("null"))
		})

		It("keeps unregistered data unchanged", func() {
			data := map[string]any{"password": "plain"}
			Expect(string(apply(WithDataAfter(data)).Data.After)).To(MatchJSON(`{"password":"plain"}`))
		})
	})

	Describe("MaskString", func() {
		It("masks non-empty string and keeps empty string", func() {
			Expect(MaskString("plain")).To(Equal(SensitiveValueMask))
			Expect(MaskString("")).To(BeEmpty())
		})
	})

	Describe("MaskSensitiveProperties", func() {
		It("returns nil for nil properties", func() {
			Expect(MaskSensitiveProperties(nil)).To(BeNil())
		})

		It("masks sensitive keys recursively without modifying the original", func() {
			props := map[string]any{
				"polarisToken": "tk",
				"Password":     "pw",
				"secretName":   "my-secret",
				"emptyToken":   "",
				"servicePort":  80,
				"nested": map[string]any{
					"apiSecret": "s",
					"name":      "n",
				},
				"items": []any{
					map[string]any{"passwd": "p"},
					"plain",
				},
			}

			masked := MaskSensitiveProperties(props)

			Expect(masked).To(Equal(map[string]any{
				"polarisToken": SensitiveValueMask,
				"Password":     SensitiveValueMask,
				"secretName":   "my-secret",
				"emptyToken":   "",
				"servicePort":  80,
				"nested": map[string]any{
					"apiSecret": SensitiveValueMask,
					"name":      "n",
				},
				"items": []any{
					map[string]any{"passwd": SensitiveValueMask},
					"plain",
				},
			}))
			Expect(props["polarisToken"]).To(Equal("tk"))
			Expect(props["nested"].(map[string]any)["apiSecret"]).To(Equal("s"))
			Expect(props["items"].([]any)[0].(map[string]any)["passwd"]).To(Equal("p"))
		})
	})
})
