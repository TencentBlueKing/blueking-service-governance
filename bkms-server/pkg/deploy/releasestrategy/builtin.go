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

package releasestrategy

import (
	"slices"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ErrBuiltinStrategyReadOnly 平台内置发布策略不可修改或删除
var ErrBuiltinStrategyReadOnly = errors.New("builtin release strategy cannot be modified or deleted")

// builtinStrategies 平台内置发布策略，所有空间可见且只读，按批次数升序。
//
// ID 只有最后一个字节非 0（取值为批次数）：生成的 ObjectID 前 4 字节是创建时间，不会与之冲突。
// ID 是部署请求和部署快照引用的契约，发布后不能修改或复用；调整批次内容时保持 ID 不变。
var builtinStrategies = []ReleaseStrategy{
	{
		ID:   bson.ObjectID{11: 3},
		Name: "分三批灰度(1个,50%,全量)",
		Batches: []Batch{
			{Target: "1", PauseMode: PauseModePermanent},
			{Target: "50%", PauseMode: PauseModeDuration, PauseSeconds: 30},
			{Target: "100%", PauseMode: PauseModeNone},
		},
	},
	{
		ID:   bson.ObjectID{11: 4},
		Name: "分四批灰度(1个,33%,66%,全量)",
		Batches: []Batch{
			{Target: "1", PauseMode: PauseModePermanent},
			{Target: "33%", PauseMode: PauseModeDuration, PauseSeconds: 30},
			{Target: "66%", PauseMode: PauseModeDuration, PauseSeconds: 30},
			{Target: "100%", PauseMode: PauseModeNone},
		},
	},
	{
		ID:   bson.ObjectID{11: 5},
		Name: "分五批灰度(1个,25%,50%,75%,全量)",
		Batches: []Batch{
			{Target: "1", PauseMode: PauseModePermanent},
			{Target: "25%", PauseMode: PauseModeDuration, PauseSeconds: 30},
			{Target: "50%", PauseMode: PauseModeDuration, PauseSeconds: 30},
			{Target: "75%", PauseMode: PauseModeDuration, PauseSeconds: 30},
			{Target: "100%", PauseMode: PauseModeNone},
		},
	},
}

// BuiltinStrategies 返回平台内置发布策略的副本，按批次数升序
func BuiltinStrategies() []ReleaseStrategy {
	return lo.Map(builtinStrategies, func(s ReleaseStrategy, _ int) ReleaseStrategy {
		s.Batches = slices.Clone(s.Batches)
		return s
	})
}

// IsBuiltin 判断策略是否为平台内置策略
func (s ReleaseStrategy) IsBuiltin() bool {
	_, ok := getBuiltinStrategy(s.ID.Hex())
	return ok
}

// IsBuiltinName 判断名称是否被平台内置策略占用
func IsBuiltinName(name string) bool {
	return lo.ContainsBy(builtinStrategies, func(s ReleaseStrategy) bool { return s.Name == name })
}

// getBuiltinStrategy 按 ID 查找平台内置策略，返回副本
func getBuiltinStrategy(id string) (*ReleaseStrategy, bool) {
	for _, s := range BuiltinStrategies() {
		if s.ID.Hex() == id {
			return &s, true
		}
	}
	return nil, false
}
