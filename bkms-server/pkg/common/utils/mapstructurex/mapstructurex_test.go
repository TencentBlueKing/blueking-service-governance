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

package mapstructurex

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ = Describe("Test Decode", func() {
	It("test ForceToTimestamppbHook", func() {
		now := time.Now()

		input := struct {
			T time.Time
		}{T: now}

		output := struct {
			T *timestamppb.Timestamp
		}{}

		err := DecodeWithHooks(input, &output, TimeToTimestamppbHook())
		Expect(err).NotTo(HaveOccurred())
		Expect(output.T).To(Equal(timestamppb.New(now)))
	})

	It("test BsonIDToStringHook", func() {
		bonsID := bson.NewObjectID()

		input := struct {
			ID bson.ObjectID
		}{ID: bonsID}

		output := struct {
			ID string
		}{}

		err := DecodeWithHooks(input, &output, BsonIDToStringHook())
		Expect(err).NotTo(HaveOccurred())
		Expect(output.ID).To(Equal(bonsID.Hex()))
	})

	It("test BsonDocToMapHook with nested documents", func() {
		type contact struct {
			Email string `mapstructure:"email"`
			Phone string `mapstructure:"phone"`
		}
		type owner struct {
			Name    string  `mapstructure:"name"`
			Contact contact `mapstructure:"contact"`
		}
		type service struct {
			Name string   `mapstructure:"name"`
			Port int      `mapstructure:"port"`
			Tags []string `mapstructure:"tags"`
		}
		type app struct {
			Name     string            `mapstructure:"name"`
			Owner    *owner            `mapstructure:"owner"`
			Services []service         `mapstructure:"services"`
			Labels   map[string]string `mapstructure:"labels"`
			Extra    map[string]any    `mapstructure:"extra"`
		}

		// 模拟从 MongoDB 读回的 map[string]any：嵌套文档为 bson.D，数组为 bson.A
		input := map[string]any{
			"name": "demo",
			"owner": bson.D{
				{Key: "name", Value: "alice"},
				{Key: "contact", Value: bson.D{
					{Key: "email", Value: "alice@example.com"},
					{Key: "phone", Value: "123456"},
				}},
			},
			"services": bson.A{
				bson.D{
					{Key: "name", Value: "api"},
					{Key: "port", Value: int32(8080)},
					{Key: "tags", Value: bson.A{"web", "http"}},
				},
				bson.D{
					{Key: "name", Value: "worker"},
					{Key: "port", Value: int32(9090)},
				},
			},
			"labels": bson.D{
				{Key: "env", Value: "prod"},
				{Key: "team", Value: "infra"},
			},
			"extra": bson.D{
				{Key: "replicas", Value: int32(3)},
				{Key: "debug", Value: true},
			},
		}

		var output app
		err := DecodeWithHooks(input, &output, BsonDocToMapHook())
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(Equal(app{
			Name: "demo",
			Owner: &owner{
				Name:    "alice",
				Contact: contact{Email: "alice@example.com", Phone: "123456"},
			},
			Services: []service{
				{Name: "api", Port: 8080, Tags: []string{"web", "http"}},
				{Name: "worker", Port: 9090},
			},
			Labels: map[string]string{"env": "prod", "team": "infra"},
			Extra:  map[string]any{"replicas": int32(3), "debug": true},
		}))

		// 不使用 hook 时，bson.D 会被当作 slice，无法写入 struct 字段
		err = DecodeWithHooks(input, &app{})
		Expect(err).To(HaveOccurred())
	})
})
