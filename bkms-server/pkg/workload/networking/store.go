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

package networking

import (
	"context"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	// ErrServiceNotFound is returned when a service is not found in the store.
	ErrServiceNotFound = errors.New("service not found")
	// ErrServiceAlreadyExists is returned when a service with the same name already exists in the app.
	ErrServiceAlreadyExists = errors.New("service already exists")
)

// use a single instance of Validate, it caches struct info
var validate = validator.New(validator.WithRequiredStructEnabled())

// ServiceStore is the storage of networking services.
type ServiceStore = store.GenericStore[Service]

// ServiceMeta is the storage metadata of networking services, documents are validated before being saved.
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID + name
var ServiceMeta = store.Meta[Service]{
	Collection: "networking_services",
	NotFound:   ErrServiceNotFound,
	Duplicate:  ErrServiceAlreadyExists,
	BeforeSave: []func(*Service) error{validateService},
}

func validateService(svc *Service) error {
	return errors.Wrap(validate.Struct(svc), "service validation failed")
}

// ByApp matches the services of the given app.
func ByApp(appID string) store.Opt {
	return store.Where("appID", appID)
}

// ByAppAndName matches the service with the given name in the app.
func ByAppAndName(appID, name string) store.Opt {
	return store.Combine(ByApp(appID), store.Where("name", name))
}

// GroupByAppID fetches all services for the given appIDs from storage, newest first,
// and groups them by appID. Only appIDs with at least one service are returned.
func GroupByAppID(ctx context.Context, services *ServiceStore, appIDs []string) (map[string][]Service, error) {
	if len(appIDs) == 0 {
		return make(map[string][]Service), nil
	}
	list, err := services.List(ctx, store.Where("appID", bson.M{"$in": appIDs}), store.Desc("createdAt"))
	if err != nil {
		return nil, errors.Wrap(err, "list services by app ids")
	}
	return lo.GroupBy(list, func(svc Service) string { return svc.AppID }), nil
}

// SvcUpdateData is the data for updating the networking service.
// Note: 不支持置空 selector 和 ports, 空值会被忽略更新
type SvcUpdateData struct {
	Selector           map[string]string
	Ports              []ServicePort
	TrafficLaneEnabled *bool
}

// ToBSON converts the update data to the $set document used by Update, empty fields are skipped.
func (d *SvcUpdateData) ToBSON() bson.M {
	set := bson.M{}
	if len(d.Selector) > 0 {
		set["selector"] = d.Selector
	}
	if len(d.Ports) > 0 {
		set["ports"] = d.Ports
	}
	if d.TrafficLaneEnabled != nil {
		set["trafficLaneEnabled"] = *d.TrafficLaneEnabled
	}
	return set
}
