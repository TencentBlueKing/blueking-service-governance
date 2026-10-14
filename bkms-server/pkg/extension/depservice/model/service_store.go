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

package model

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

const serviceCollName = "depservice_services"

// ErrServiceNotFound 依赖服务不存在，AsNotFoundError 同样将其视为未找到
var ErrServiceNotFound = errors.New("service not found")

// ServiceStore 依赖服务存储，name 是服务的唯一标识
type ServiceStore = store.GenericStore[Service]

// ServiceMeta 依赖服务的存储元数据。
//
// MongoDB 中直接存储 map[string]any 会被解码为 bson.D，读回后不便使用，
// 因此 plan 的 config 在库中保存为 {"value": "<config 的 json 串>"}，由 Encode / Decode 钩子完成转换。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：name
var ServiceMeta = store.Meta[Service]{
	Collection: serviceCollName,
	NotFound:   ErrServiceNotFound,
	BeforeSave: []func(*Service) error{validateService},
	Encode:     []func(*Service) error{marshalPlanConfigs},
	Decode:     []func(*Service) error{unmarshalPlanConfigs},
}

func validateService(svc *Service) error {
	return errors.Wrap(validate.Struct(svc), "service validation failed")
}

func marshalPlanConfigs(svc *Service) error {
	for i := range svc.Plans {
		if err := svc.Plans[i].MarshalConfig(); err != nil {
			return errors.Wrapf(err, "marshal config of plan %s", svc.Plans[i].Name)
		}
	}
	return nil
}

func unmarshalPlanConfigs(svc *Service) error {
	for i := range svc.Plans {
		if err := svc.Plans[i].UnmarshalConfig(); err != nil {
			return errors.Wrapf(err, "unmarshal config of plan %s", svc.Plans[i].Name)
		}
	}
	return nil
}

// ByName 按服务名匹配依赖服务
func ByName(name string) store.Opt {
	return store.Where("name", name)
}
