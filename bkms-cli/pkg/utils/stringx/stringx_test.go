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

package stringx_test

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/stringx"
)

var _ = Describe("ToWords", func() {
	DescribeTable("should split identifiers into lowercase words",
		func(input, expected string) {
			Expect(stringx.ToWords(input)).To(Equal(expected))
		},
		Entry("camel case", "appName", "app name"),
		Entry("pascal case", "AppName", "app name"),
		Entry("single word", "id", "id"),
		Entry("multiple words", "keepNotReadyPod", "keep not ready pod"),
		Entry("trailing acronym", "appID", "app id"),
		Entry("acronym in the middle", "appConfigFileIDValue", "app config file id value"),
		Entry("leading acronym", "HTTPServer", "http server"),
		Entry("plural acronym", "refAppIDs", "ref app ids"),
		Entry("all upper case", "APP", "app"),
		Entry("snake case", "scope_env_names", "scope env names"),
		Entry("kebab case", "scope-env-names", "scope env names"),
		Entry("dotted", "app.name", "app name"),
		Entry("digits attached to the previous word", "v1Alpha", "v1 alpha"),
		Entry("already spaced", "app name", "app name"),
		Entry("redundant separators", "  app__name  ", "app name"),
		Entry("empty string", "", ""),
		Entry("separators only", "__", ""),
	)
})

var _ = Describe("TrimSpaceRecursive", func() {
	// 应去除简单结构体中字符串字段的前后空格
	It("should trim spaces from string fields in a simple struct", func() {
		type Simple struct {
			Name  string
			Value string
		}
		s := Simple{Name: "  hello  ", Value: "\tworld\n"}
		stringx.TrimSpaceRecursive(reflect.ValueOf(&s))
		Expect(s.Name).To(Equal("hello"))
		Expect(s.Value).To(Equal("world"))
	})

	// 应递归处理嵌套指针结构体
	It("should recursively trim nested pointer struct fields", func() {
		type Inner struct {
			Field string
		}
		type Outer struct {
			Inner *Inner
			Name  string
		}
		o := Outer{
			Inner: &Inner{Field: "  nested  "},
			Name:  " outer ",
		}
		stringx.TrimSpaceRecursive(reflect.ValueOf(&o))
		Expect(o.Inner.Field).To(Equal("nested"))
		Expect(o.Name).To(Equal("outer"))
	})

	// 应处理 nil 指针而不 panic
	It("should handle nil pointer without panic", func() {
		type Inner struct {
			Field string
		}
		type Outer struct {
			Inner *Inner
			Name  string
		}
		o := Outer{
			Inner: nil,
			Name:  " safe ",
		}
		Expect(func() {
			stringx.TrimSpaceRecursive(reflect.ValueOf(&o))
		}).NotTo(Panic())
		Expect(o.Name).To(Equal("safe"))
	})

	// 应递归处理切片中的元素
	It("should recursively trim elements in a slice", func() {
		type Item struct {
			Tag string
		}
		type Collection struct {
			Items []Item
		}
		c := Collection{
			Items: []Item{
				{Tag: " tag1 "},
				{Tag: "\ttag2\t"},
				{Tag: "  tag3  "},
			},
		}
		stringx.TrimSpaceRecursive(reflect.ValueOf(&c))
		Expect(c.Items[0].Tag).To(Equal("tag1"))
		Expect(c.Items[1].Tag).To(Equal("tag2"))
		Expect(c.Items[2].Tag).To(Equal("tag3"))
	})

	// 应处理字符串切片
	It("should trim string slices", func() {
		type WithSlice struct {
			Tags []string
		}
		w := WithSlice{
			Tags: []string{" a ", " b ", " c "},
		}
		stringx.TrimSpaceRecursive(reflect.ValueOf(&w))
		Expect(w.Tags).To(Equal([]string{"a", "b", "c"}))
	})

	// 应处理空结构体而不 panic
	It("should handle empty struct without panic", func() {
		type Empty struct{}
		e := Empty{}
		Expect(func() {
			stringx.TrimSpaceRecursive(reflect.ValueOf(&e))
		}).NotTo(Panic())
	})

	// 应处理多层嵌套结构体
	It("should handle deeply nested structs", func() {
		type Level3 struct {
			Deep string
		}
		type Level2 struct {
			L3 *Level3
		}
		type Level1 struct {
			L2 *Level2
		}
		l := Level1{
			L2: &Level2{
				L3: &Level3{Deep: "  deep value  "},
			},
		}
		stringx.TrimSpaceRecursive(reflect.ValueOf(&l))
		Expect(l.L2.L3.Deep).To(Equal("deep value"))
	})

	// 不应修改已经没有空格的字符串
	It("should not modify strings without spaces", func() {
		type Clean struct {
			Name string
		}
		c := Clean{Name: "already-clean"}
		stringx.TrimSpaceRecursive(reflect.ValueOf(&c))
		Expect(c.Name).To(Equal("already-clean"))
	})

	// 应处理包含 map 字段的结构体（map 不递归处理）
	It("should handle struct with map field and trim map keys and values", func() {
		type WithMap struct {
			Name   string
			Labels map[string]string
		}
		w := WithMap{
			Name:   " name ",
			Labels: map[string]string{" key ": " value "},
		}
		stringx.TrimSpaceRecursive(reflect.ValueOf(&w))
		Expect(w.Name).To(Equal("name"))
		// map 的 key 和 value 都会被 trim
		Expect(w.Labels).To(HaveKeyWithValue("key", "value"))
	})
})
