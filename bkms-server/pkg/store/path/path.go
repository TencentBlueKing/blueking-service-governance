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

// Package path 处理模型结构体中参与 bson 编码的字段：扫描字段，并在 bson key 与 Go 字段名之间转换字段路径
package path

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
)

// StepKind 字段路径中一段的类型
type StepKind uint8

const (
	// StepField 结构体字段（含 inline 展开的字段）
	StepField StepKind = iota
	// StepMapKey key 为字符串的 map 中的条目
	StepMapKey
	// StepOpaque 无法静态解析的段，如数组下标、interface 中的字段、模型中不存在的字段，原样保留
	StepOpaque
)

// Step 字段路径中的一段
type Step struct {
	// Key 该段的 bson key
	Key  string
	Kind StepKind
	// Index 仅 StepField 使用，为字段在所在结构体中的 index 路径
	Index []int
}

// Path 解析后的字段路径，见 SchemaFields.Resolve
type Path struct {
	Steps []Step
	// Known 每一段都能在模型中找到；为 false 时，第一个找不到的段及之后的段原样保留
	Known bool
	// Assignable 路径只经过结构体字段与 map 条目，可以直接在对象上定位、赋值
	Assignable bool
}

// String 返回 bson 路径
func (p Path) String() string {
	return strings.Join(lo.Map(p.Steps, func(st Step, _ int) string { return st.Key }), ".")
}

// IsAncestorOf 判断 p 是否是 other 的上级字段，如 "spec" 是 "spec.color" 的上级，但不是 "specs" 的
func (p Path) IsAncestorOf(other Path) bool {
	return strings.HasPrefix(other.String(), p.String()+".")
}

// Positional 判断路径中是否包含位置操作符（$、$[]、$[id]）
func (p Path) Positional() bool {
	return lo.ContainsBy(p.Steps, func(st Step) bool { return isPositional(st.Key) })
}

// Dedupe 去掉重复的字段（如同一字段分别以 bson key 与 Go 字段名指定），并拒绝互为父子的字段，
// 后者会被 MongoDB 判定为更新冲突
func Dedupe(paths []Path) ([]Path, error) {
	paths = lo.UniqBy(paths, Path.String)
	for _, a := range paths {
		for _, b := range paths {
			if a.IsAncestorOf(b) {
				return nil, errors.Errorf("conflicting fields %q and %q, keep only one of them", a, b)
			}
		}
	}
	return paths, nil
}

// isArrayIndex 判断 seg 是否指向数组中的元素：数字下标，或更新操作中的位置操作符（$、$[]、$[id]）
func isArrayIndex(seg string) bool {
	_, err := strconv.ParseUint(seg, 10, 64)
	return err == nil || isPositional(seg)
}

// isPositional 判断 seg 是否是更新操作中的位置操作符：$、$[] 或 $[id]
func isPositional(seg string) bool {
	return seg == "$" || (strings.HasPrefix(seg, "$[") && strings.HasSuffix(seg, "]"))
}
