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
})
