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

type tagSecret struct {
	Key         string
	Password    string `audit:"mask"`
	Value       string `audit:"mask,if=IsSensitive"`
	IsSensitive bool
}

type tagProps struct {
	Properties map[string]any `audit:"maskKeys"`
}

type tagEmbedded struct {
	tagProps
	Name string
}

type tagExportedEmbedded struct {
	TagExported
	Name string
}

// TagExported 用于验证导出的内嵌结构体
type TagExported struct {
	Token string `audit:"mask"`
}

type tagNested struct {
	Ptr      *tagSecret
	List     []tagSecret
	PtrList  []*tagSecret
	Array    [1]tagSecret
	Map      map[string]tagSecret
	Any      any
	AnyMap   map[string]any
	Plain    []string
	Untagged struct{ Password string }
	secret   tagSecret
}

type tagRecursive struct {
	Token string `audit:"mask"`
	Next  *tagRecursive
}

type tagBadKind struct {
	Port int `audit:"mask"`
}

type tagBadCond struct {
	Value string `audit:"mask,if=Missing"`
}

type tagBadName struct {
	Value string `audit:"hide"`
}

var _ = Describe("Sanitize", func() {
	apply := func(opt Option) *OperationRecord {
		r := &OperationRecord{}
		opt(r)
		return r
	}

	Describe("data options", func() {
		It("masks tagged fields for value and pointer without modifying the original", func() {
			s := &tagSecret{Key: "k", Password: "pw", Value: "v", IsSensitive: true}
			expected := `{"Key":"k","Password":"******","Value":"******","IsSensitive":true}`

			Expect(string(apply(WithDataBefore(s)).Data.Before)).To(MatchJSON(expected))
			Expect(string(apply(WithDataAfter(*s)).Data.After)).To(MatchJSON(expected))
			Expect(string(apply(WithExtras(s)).Data.Extras)).To(MatchJSON(expected))
			Expect(*s).To(Equal(tagSecret{Key: "k", Password: "pw", Value: "v", IsSensitive: true}))
		})

		It("keeps conditional field when condition is false and empty string unchanged", func() {
			s := tagSecret{Key: "k", Password: "", Value: "v"}
			Expect(string(apply(WithDataAfter(s)).Data.After)).To(
				MatchJSON(`{"Key":"k","Password":"","Value":"v","IsSensitive":false}`))
		})

		It("keeps nil and nil pointer as null", func() {
			var s *tagSecret
			Expect(string(apply(WithDataBefore(s)).Data.Before)).To(Equal("null"))
			Expect(string(apply(WithDataBefore(nil)).Data.Before)).To(Equal("null"))
		})

		It("keeps untagged data unchanged", func() {
			data := map[string]any{"password": "plain"}
			Expect(string(apply(WithDataAfter(data)).Data.After)).To(MatchJSON(`{"password":"plain"}`))
		})

		It("drops data with invalid audit tag", func() {
			Expect(string(apply(WithDataAfter(tagBadKind{Port: 1})).Data.After)).To(Equal("null"))
			Expect(string(apply(WithDataAfter(&tagBadCond{Value: "v"})).Data.After)).To(Equal("null"))
			Expect(string(apply(WithDataAfter([]tagBadName{{Value: "v"}})).Data.After)).To(Equal("null"))
		})
	})

	Describe("sanitize", func() {
		It("returns the original data when nothing is masked", func() {
			s := &tagSecret{Key: "k"}
			Expect(sanitize(s)).To(BeIdenticalTo(s))
		})

		It("walks pointers, slices, arrays, maps and interfaces", func() {
			secret := func() tagSecret { return tagSecret{Password: "pw", Value: "v", IsSensitive: true} }
			ptr := secret()
			listItem := secret()
			data := &tagNested{
				Ptr:      &ptr,
				List:     []tagSecret{secret()},
				PtrList:  []*tagSecret{&listItem, nil},
				Array:    [1]tagSecret{secret()},
				Map:      map[string]tagSecret{"a": secret()},
				Any:      secret(),
				AnyMap:   map[string]any{"s": &listItem, "n": 1},
				Plain:    []string{"x"},
				Untagged: struct{ Password string }{Password: "pw"},
				secret:   secret(),
			}

			masked := sanitize(data).(*tagNested)

			maskedSecret := tagSecret{Password: SensitiveValueMask, Value: SensitiveValueMask, IsSensitive: true}
			Expect(*masked.Ptr).To(Equal(maskedSecret))
			Expect(masked.List).To(Equal([]tagSecret{maskedSecret}))
			Expect(*masked.PtrList[0]).To(Equal(maskedSecret))
			Expect(masked.PtrList[1]).To(BeNil())
			Expect(masked.Array[0]).To(Equal(maskedSecret))
			Expect(masked.Map["a"]).To(Equal(maskedSecret))
			Expect(masked.Any).To(Equal(maskedSecret))
			Expect(*masked.AnyMap["s"].(*tagSecret)).To(Equal(maskedSecret))
			Expect(masked.AnyMap["n"]).To(Equal(1))
			Expect(masked.Plain).To(Equal([]string{"x"}))
			// 无 tag 的字段不处理
			Expect(masked.Untagged.Password).To(Equal("pw"))

			// 原对象不变
			Expect(ptr.Password).To(Equal("pw"))
			Expect(listItem.Password).To(Equal("pw"))
			Expect(data.List[0].Password).To(Equal("pw"))
			Expect(data.Array[0].Password).To(Equal("pw"))
			Expect(data.Map["a"].Password).To(Equal("pw"))
			Expect(data.Any.(tagSecret).Password).To(Equal("pw"))
		})

		It("masks properties and exported embedded struct fields", func() {
			props := map[string]any{"polarisToken": "tk", "polarisName": "n"}
			Expect(sanitize(tagProps{Properties: props}).(tagProps).Properties).To(Equal(map[string]any{
				"polarisToken": SensitiveValueMask, "polarisName": "n",
			}))
			Expect(props["polarisToken"]).To(Equal("tk"))

			e := tagExportedEmbedded{TagExported: TagExported{Token: "tk"}, Name: "n"}
			Expect(sanitize(e).(tagExportedEmbedded).Token).To(Equal(SensitiveValueMask))
			Expect(e.Token).To(Equal("tk"))

			// 未导出的内嵌结构体不处理
			u := tagEmbedded{tagProps: tagProps{Properties: props}}
			Expect(sanitize(u)).To(Equal(u))
		})

		It("handles recursive types", func() {
			data := &tagRecursive{Token: "a", Next: &tagRecursive{Token: "b"}}
			masked := sanitize(data).(*tagRecursive)
			Expect(masked.Token).To(Equal(SensitiveValueMask))
			Expect(masked.Next.Token).To(Equal(SensitiveValueMask))
			Expect(data.Next.Token).To(Equal("b"))
		})

		It("drops data with cyclic references", func() {
			data := &tagRecursive{Token: "a"}
			data.Next = data
			Expect(sanitize(data)).To(BeNil())
		})
	})

	Describe("maskString", func() {
		It("masks non-empty string and keeps empty string", func() {
			Expect(maskString("plain")).To(Equal(SensitiveValueMask))
			Expect(maskString("")).To(BeEmpty())
		})
	})

	Describe("maskSensitiveProperties", func() {
		It("returns nil for nil properties", func() {
			Expect(maskSensitiveProperties(nil)).To(BeNil())
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

			masked := maskSensitiveProperties(props)

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
