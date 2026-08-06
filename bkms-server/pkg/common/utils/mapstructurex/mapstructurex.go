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

// Package mapstructurex 提供 mapstructure 扩展功能，用于特殊类型转换
package mapstructurex

import (
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DecodeWithHooks 在原 mapstructure.Decode 基础上，支持设置转换 hooks
func DecodeWithHooks(input, output any, hook ...mapstructure.DecodeHookFunc) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(hook...),
		Result:     output,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

// TimeToTimestamppbHook 将 time.Time 转换为 *timestamppb.Timestamp
func TimeToTimestamppbHook() mapstructure.DecodeHookFunc {
	return func(f, t reflect.Type, data any) (any, error) {
		if f != reflect.TypeOf(&time.Time{}) {
			return data, nil
		}

		inputTime, ok := data.(*time.Time)
		if !ok {
			return nil, errors.Errorf("unable to convert %v to timestamp", data)
		}

		if inputTime == nil {
			return nil, errors.New("nil time pointer")
		}
		return timestamppb.New(*inputTime), nil
	}
}

// BsonIDToStringHook 将 bson.ObjectId 转换为 string
func BsonIDToStringHook() mapstructure.DecodeHookFunc {
	return func(f, t reflect.Type, data any) (any, error) {
		if f != reflect.TypeOf(bson.ObjectID{}) {
			return data, nil
		}

		objID, ok := data.(bson.ObjectID)
		if !ok {
			return nil, errors.Errorf("unable to convert %v to string", data)
		}

		if objID.IsZero() {
			return "", nil
		}

		return objID.Hex(), nil
	}
}
