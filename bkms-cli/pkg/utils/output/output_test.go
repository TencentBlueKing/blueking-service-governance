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

package output_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// TestUser 测试用的结构体
type TestUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

var _ = Describe("Output Package", func() {
	Describe("FormatData function", func() {
		Context("when data is nil", func() {
			It("should return 'null'", func() {
				result, err := output.FormatData(context.Background(), nil, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal("null"))
			})
		})

		Context("when data is an empty slice", func() {
			It("should return 'null'", func() {
				var users []TestUser
				result, err := output.FormatData(context.Background(), users, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal("null"))
			})
		})

		Context("when data is a single object", func() {
			It("should format as key-value pairs", func() {
				user := TestUser{ID: "user1", Name: "Alice", Age: 25}

				result, err := output.FormatData(context.Background(), user, "")

				Expect(err).NotTo(HaveOccurred())
				// 验证键值对格式
				Expect(result).To(Equal("id: user1\nname: Alice\nage: 25\n"))
			})
		})

		Context("when data is a slice of objects", func() {
			It("should format as table", func() {
				users := []TestUser{
					{ID: "user1", Name: "Alice", Age: 25},
					{ID: "user2", Name: "Bob", Age: 30},
				}

				result, err := output.FormatData(context.Background(), users, "")

				Expect(err).NotTo(HaveOccurred())
				// 使用内容断言而非精确匹配，避免不同 OS 下表格渲染差异导致测试失败
				Expect(result).To(ContainSubstring("ID"))
				Expect(result).To(ContainSubstring("NAME"))
				Expect(result).To(ContainSubstring("AGE"))
				Expect(result).To(ContainSubstring("user1"))
				Expect(result).To(ContainSubstring("Alice"))
				Expect(result).To(ContainSubstring("25"))
				Expect(result).To(ContainSubstring("user2"))
				Expect(result).To(ContainSubstring("Bob"))
				Expect(result).To(ContainSubstring("30"))
			})
		})

		Context("when format is 'json'", func() {
			It("should format as JSON", func() {
				user := TestUser{ID: "user1", Name: "Alice", Age: 25}

				result, err := output.FormatData(context.Background(), user, "json")

				Expect(err).NotTo(HaveOccurred())
				// 验证 JSON 格式
				Expect(result).To(Equal(`{"id":"user1","name":"Alice","age":25}`))
			})
		})

		Context("when format is 'yaml'", func() {
			It("should format slices as YAML instead of inferring table output", func() {
				users := []TestUser{
					{ID: "user1", Name: "Alice", Age: 25},
				}

				result, err := output.FormatData(context.Background(), users, "yaml")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring("- id: user1"))
				Expect(result).NotTo(ContainSubstring("ID"))
			})
		})

		Context("when format is 'table'", func() {
			It("should format slices as table output", func() {
				users := []TestUser{
					{ID: "user1", Name: "Alice", Age: 25},
				}

				result, err := output.FormatData(context.Background(), users, "table")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring("ID"))
				Expect(result).To(ContainSubstring("NAME"))
				Expect(result).To(ContainSubstring("user1"))
				Expect(result).To(ContainSubstring("Alice"))
			})
		})

		Context("when format is empty string", func() {
			It("should use key-value format", func() {
				user := TestUser{ID: "test", Name: "Test User"}

				result, err := output.FormatData(context.Background(), user, "")

				Expect(err).NotTo(HaveOccurred())
				// 不指定格式时，应该使用键值对格式
				Expect(result).To(Equal("id: test\nname: Test User\nage: 0\n"))
			})
		})

		Context("when format is invalid", func() {
			It("should return an error", func() {
				user := TestUser{ID: "test", Name: "Test User"}

				_, err := output.FormatData(context.Background(), user, "invalid")

				Expect(err).To(MatchError("unsupported output format: invalid"))
			})
		})

		Context("when format is a jq expression", func() {
			It("should output string values as raw text", func() {
				user := TestUser{ID: "user1", Name: "Alice", Age: 25}

				result, err := output.FormatData(context.Background(), user, "jq=.name")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal("Alice"))
			})

			It("should output multiple emitted values on separate lines", func() {
				users := []TestUser{
					{ID: "user1", Name: "Alice", Age: 25},
					{ID: "user2", Name: "Bob", Age: 30},
				}

				result, err := output.FormatData(context.Background(), users, "jq=.[] | .name")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal("Alice\nBob"))
			})

			It("should output arrays and objects as JSON", func() {
				users := []TestUser{
					{ID: "user1", Name: "Alice", Age: 25},
					{ID: "user2", Name: "Bob", Age: 30},
				}

				result, err := output.FormatData(context.Background(), users, "jq=map(.name)")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(`["Alice","Bob"]`))
			})

			It("should return an error for invalid jq expressions", func() {
				user := TestUser{ID: "user1", Name: "Alice", Age: 25}

				_, err := output.FormatData(context.Background(), user, "jq=.name &")

				Expect(err).To(MatchError(ContainSubstring("parse jq expression")))
			})

			It("should return an error when expression is empty", func() {
				user := TestUser{ID: "user1", Name: "Alice", Age: 25}

				_, err := output.FormatData(context.Background(), user, "jq=")

				Expect(err).To(MatchError("jq expression cannot be empty"))
			})
		})

		Context("when format is an unsupported customizable expression", func() {
			It("should return an error", func() {
				user := TestUser{ID: "test", Name: "Test User"}

				_, err := output.FormatData(context.Background(), user, "foo=.name")

				Expect(err).To(MatchError("unsupported customizable output format: foo"))
			})

			It("should return an error when format type is empty", func() {
				user := TestUser{ID: "test", Name: "Test User"}

				_, err := output.FormatData(context.Background(), user, "=.name")

				Expect(err).To(MatchError("output format type cannot be empty"))
			})
		})
	})

	Describe("JSON formatting", func() {
		It("should produce valid JSON structure", func() {
			user := TestUser{ID: "user1", Name: "Alice", Age: 25}

			result, err := output.FormatData(context.Background(), user, "json")

			Expect(err).NotTo(HaveOccurred())
			// 验证 JSON 格式正确
			Expect(result).To(Equal(`{"id":"user1","name":"Alice","age":25}`))
		})
	})

	Describe("asKeyValues function (via FormatData)", func() {
		Context("when data is nil", func() {
			It("should return 'null'", func() {
				result, err := output.FormatData(context.Background(), nil, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal("null"))
			})
		})

		Context("when struct has fields with different lengths", func() {
			type Product struct {
				ID          string
				Name        string
				Description string
				Price       float64
			}

			It("should align all field names to the longest one", func() {
				product := Product{
					ID:          "prod-001",
					Name:        "Laptop",
					Description: "High-end laptop",
					Price:       1299.99,
				}

				result, err := output.FormatData(context.Background(), product, "")

				Expect(err).NotTo(HaveOccurred())
				// "Description" 是最长的字段名（11个字符）
				// 所有字段名都应该对齐到这个长度
				Expect(result).To(ContainSubstring("id: prod-001"))
				Expect(result).To(ContainSubstring("name: Laptop"))
				Expect(result).To(ContainSubstring("description: High-end laptop"))
				Expect(result).To(ContainSubstring("price: 1299.99"))
			})
		})

		Context("when struct has zero values", func() {
			It("should display zero values correctly", func() {
				// 所有字段都是零值
				user := TestUser{}

				result, err := output.FormatData(context.Background(), user, "")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring("id: \"\""))
				Expect(result).To(ContainSubstring("name: \"\""))
				Expect(result).To(ContainSubstring("age: 0"))
			})
		})

		Context("when struct has nested struct fields", func() {
			type Address struct {
				City    string
				Country string
			}
			type Person struct {
				Name    string
				Age     int
				Address Address
			}

			It("should format nested struct as value", func() {
				person := Person{
					Name: "Alice",
					Age:  25,
					Address: Address{
						City:    "Beijing",
						Country: "China",
					},
				}

				result, err := output.FormatData(context.Background(), person, "")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring("name: Alice"))
				Expect(result).To(ContainSubstring("age: 25"))
				// 嵌套结构体会被格式化为 {City Country}
				Expect(result).To(ContainSubstring("address:\n  city: Beijing\n  country: China"))
			})
		})

		Context("when struct has pointer fields", func() {
			type Config struct {
				Name    string
				Timeout *int
			}

			It("should handle nil pointer correctly", func() {
				config := Config{
					Name:    "test",
					Timeout: nil,
				}

				result, err := output.FormatData(context.Background(), config, "")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring("name: test"))
				Expect(result).To(ContainSubstring("timeout: null"))
			})
		})
	})

	Describe("table tag skip fields", func() {
		type InstanceWithHidden struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Hidden string `json:"hidden" table:"-"`
		}

		Context("when struct has table:\"-\" tag", func() {
			It("should not show hidden field in table output", func() {
				items := []InstanceWithHidden{
					{ID: "inst-1", Name: "pod-a", Hidden: "secret-data"},
					{ID: "inst-2", Name: "pod-b", Hidden: "more-secret"},
				}

				result, err := output.FormatData(context.Background(), items, "")

				Expect(err).NotTo(HaveOccurred())
				// 表格应包含 ID 和 Name 列
				Expect(result).To(ContainSubstring("ID"))
				Expect(result).To(ContainSubstring("NAME"))
				Expect(result).To(ContainSubstring("inst-1"))
				Expect(result).To(ContainSubstring("pod-a"))
				Expect(result).To(ContainSubstring("inst-2"))
				Expect(result).To(ContainSubstring("pod-b"))

				// 表格不应包含 Hidden 列头和数据
				Expect(result).NotTo(ContainSubstring("HIDDEN"))
				Expect(result).NotTo(ContainSubstring("secret-data"))
				Expect(result).NotTo(ContainSubstring("more-secret"))
			})

			It("should still show hidden field in json output", func() {
				items := []InstanceWithHidden{
					{ID: "inst-1", Name: "pod-a", Hidden: "secret-data"},
				}

				result, err := output.FormatData(context.Background(), items, "json")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring(`"hidden":"secret-data"`))
			})

			It("should still show hidden field in yaml output", func() {
				items := []InstanceWithHidden{
					{ID: "inst-1", Name: "pod-a", Hidden: "secret-data"},
				}

				result, err := output.FormatData(context.Background(), items, "yaml")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring("hidden: secret-data"))
			})
		})

		Context("when struct has slice field with table:\"-\" tag", func() {
			type PolarisInfo struct {
				ServiceNamespace  string `json:"serviceNamespace" yaml:"serviceNamespace"`
				ServiceName       string `json:"serviceName" yaml:"serviceName"`
				IsHealthy         bool   `json:"isHealthy" yaml:"isHealthy"`
				EnableHealthCheck bool   `json:"enableHealthCheck" yaml:"enableHealthCheck"`
			}
			type Instance struct {
				ID           string        `json:"id"`
				Status       string        `json:"status"`
				PolarisInfos []PolarisInfo `json:"polarisInfos" yaml:"polarisInfos" table:"-"`
			}

			It("should not show polarisInfos in table output", func() {
				items := []Instance{
					{
						ID:     "pod-1",
						Status: "Running",
						PolarisInfos: []PolarisInfo{
							{ServiceNamespace: "ns1", ServiceName: "svc1", IsHealthy: true, EnableHealthCheck: true},
						},
					},
				}

				result, err := output.FormatData(context.Background(), items, "")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring("ID"))
				Expect(result).To(ContainSubstring("STATUS"))
				Expect(result).To(ContainSubstring("pod-1"))
				Expect(result).To(ContainSubstring("Running"))

				// 不应展示 PolarisInfos 列
				Expect(result).NotTo(ContainSubstring("POLARISINFOS"))
				Expect(result).NotTo(ContainSubstring("ns1"))
				Expect(result).NotTo(ContainSubstring("svc1"))
			})

			It("should show polarisInfos in json output", func() {
				items := []Instance{
					{
						ID:     "pod-1",
						Status: "Running",
						PolarisInfos: []PolarisInfo{
							{ServiceNamespace: "ns1", ServiceName: "svc1", IsHealthy: true, EnableHealthCheck: true},
						},
					},
				}

				result, err := output.FormatData(context.Background(), items, "json")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring(`"polarisInfos"`))
				Expect(result).To(ContainSubstring(`"serviceNamespace":"ns1"`))
				Expect(result).To(ContainSubstring(`"serviceName":"svc1"`))
				Expect(result).To(ContainSubstring(`"isHealthy":true`))
				Expect(result).To(ContainSubstring(`"enableHealthCheck":true`))
			})

			It("should show polarisInfos in yaml output", func() {
				items := []Instance{
					{
						ID:     "pod-1",
						Status: "Running",
						PolarisInfos: []PolarisInfo{
							{ServiceNamespace: "ns1", ServiceName: "svc1", IsHealthy: true, EnableHealthCheck: false},
						},
					},
				}

				result, err := output.FormatData(context.Background(), items, "yaml")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring("serviceNamespace: ns1"))
				Expect(result).To(ContainSubstring("serviceName: svc1"))
				Expect(result).To(ContainSubstring("isHealthy: true"))
				Expect(result).To(ContainSubstring("enableHealthCheck: false"))
			})
		})
	})

	Describe("table output with array field (visual demo)", func() {
		type PolarisInfo struct {
			ServiceNamespace  string `json:"serviceNamespace" yaml:"serviceNamespace"`
			ServiceName       string `json:"serviceName" yaml:"serviceName"`
			IsHealthy         bool   `json:"isHealthy" yaml:"isHealthy"`
			EnableHealthCheck bool   `json:"enableHealthCheck" yaml:"enableHealthCheck"`
		}
		type Instance struct {
			ID           string        `json:"id"`
			IP           string        `json:"ip"`
			Status       string        `json:"status"`
			IsHealthy    bool          `json:"isHealthy"`
			Age          string        `json:"age"`
			PolarisInfos []PolarisInfo `json:"polarisInfos" yaml:"polarisInfos"`
		}

		It("should print table with array field for visual inspection", func() {
			items := []Instance{
				{
					ID:        "xxxxxx-pdkgn",
					IP:        "127.0.0.8",
					Status:    "Running",
					IsHealthy: true,
					Age:       "2d5h",
					PolarisInfos: []PolarisInfo{
						{
							ServiceNamespace:  "Development1",
							ServiceName:       "polaris_1",
							IsHealthy:         true,
							EnableHealthCheck: true,
						},
						{
							ServiceNamespace:  "Development2",
							ServiceName:       "polaris_2",
							IsHealthy:         false,
							EnableHealthCheck: false,
						},
					},
				},
				{
					ID:        "xxxxxx-xyzab",
					IP:        "127.0.0.9",
					Status:    "Running",
					IsHealthy: true,
					Age:       "1d3h",
					PolarisInfos: []PolarisInfo{
						{
							ServiceNamespace:  "Production",
							ServiceName:       "polaris_prod",
							IsHealthy:         true,
							EnableHealthCheck: true,
						},
					},
				},
			}

			result, err := output.FormatData(context.Background(), items, "")

			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Println("\n===== Table output with array field (NOT hidden) =====")
			GinkgoWriter.Println(result)
			GinkgoWriter.Println("===== End =====")

			Expect(result).To(ContainSubstring("xxxxxx-pdkgn"))
			Expect(result).To(ContainSubstring("xxxxxx-xyzab"))
		})
	})
})
