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

package component

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// InstanceCount 某个租户下由一个组件定义生成的组件实例数量。
// 内置定义被所有租户共用，计数不能放在定义上，因此单独成表、按租户划分
type InstanceCount struct {
	TenantID string `json:"-" bson:"tenantID"`
	// DefName 组件定义名称
	DefName string `bson:"defName"`
	// Version 组件定义版本
	Version string `bson:"version"`
	// AppCompInstanceCount 应用组件实例数量，引用空间组件的应用组件不计入
	AppCompInstanceCount int32 `bson:"appCompInstanceCount"`
	// WorkspaceCompInstanceCount 空间组件实例数量
	WorkspaceCompInstanceCount int32 `bson:"workspaceCompInstanceCount"`

	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

// InstanceCountStore is the storage of component instance counts.
type InstanceCountStore = store.GenericStore[InstanceCount]

// InstanceCountMeta 组件实例计数的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：tenantID + defName + version
var InstanceCountMeta = store.Meta[InstanceCount]{
	Collection:   "component_def_instance_counts",
	TenantScoped: true,
}

// ByDef 按组件定义的名称与版本匹配计数
func ByDef(name, version string) store.Opt {
	return store.Filter(bson.M{"defName": name, "version": version})
}

// instanceCountField 返回计数类型对应的 bson 字段名，以及 count 上对应字段的指针
func instanceCountField(count *InstanceCount, instanceType InstanceType) (string, *int32, error) {
	switch instanceType {
	case FieldAppCompInstance:
		return "appCompInstanceCount", &count.AppCompInstanceCount, nil
	case FieldWorkspaceCompInstance:
		return "workspaceCompInstanceCount", &count.WorkspaceCompInstanceCount, nil
	default:
		return "", nil, errors.Errorf("invalid instance type: %s", instanceType)
	}
}

// UpdateInstanceCount 原子调整当前租户下组件定义的实例计数，计数记录在首次增加时创建。
//
// - name: 组件定义名称（使用默认版本 DefaultComponentDefVersion）
// - instanceType: 计数字段，FieldAppCompInstance 或 FieldWorkspaceCompInstance
// - delta: 增量，正数表示增加，负数表示减少
func UpdateInstanceCount(
	ctx context.Context, counts *InstanceCountStore, name string, instanceType InstanceType, delta int,
) error {
	count := &InstanceCount{DefName: name, Version: DefaultComponentDefVersion}
	field, value, err := instanceCountField(count, instanceType)
	if err != nil {
		return err
	}
	if delta == 0 {
		return nil
	}
	wrap := func(err error) error {
		return errors.Wrapf(err, "adjust instance count of component def(%s) field(%s) by %d", name, field, delta)
	}
	incr := func() error {
		_, incrErr := counts.Modify(ctx, bson.M{"$inc": bson.M{field: delta}}, ByDef(name, DefaultComponentDefVersion))
		return incrErr
	}

	if err = incr(); !errors.Is(err, store.ErrNotFound) {
		return wrap(err)
	}
	// 记录不存在时的减少只可能来自计数表建立之前的数据，没有可减的值，忽略
	if delta < 0 {
		return nil
	}
	*value = int32(delta) //nolint:gosec // G115: counts fit in int32
	err = counts.Create(ctx, count)
	// 并发下记录可能已被其他请求创建，此时改为递增
	if errors.Is(err, store.ErrDuplicate) {
		err = incr()
	}
	return wrap(err)
}

// ListInstanceCounts 返回当前租户下指定组件定义的实例计数，key 为 ComponentDef.Key()，没有记录的定义不在结果中
func ListInstanceCounts(
	ctx context.Context, counts *InstanceCountStore, defs []ComponentDef,
) (map[string]InstanceCount, error) {
	if len(defs) == 0 {
		return map[string]InstanceCount{}, nil
	}
	names := lo.Uniq(lo.Map(defs, func(d ComponentDef, _ int) string { return d.Name }))
	items, err := counts.List(ctx, store.Where("defName", bson.M{"$in": names}))
	if err != nil {
		return nil, errors.Wrap(err, "list component instance counts")
	}
	return lo.SliceToMap(items, func(c InstanceCount) (string, InstanceCount) {
		return (&ComponentDef{Name: c.DefName, Version: c.Version}).Key(), c
	}), nil
}
