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

// Package json 提供 bkms-server 内部使用的 JSON 组装、点分路径转换等辅助工具
package json

import (
	"encoding/json"
	"strings"
)

// MarshalNestedFromDotPath 按点分路径 path 将叶节点 value 编码为逐层嵌套的 JSON 对象（两空格缩进）。
// 例如 path 为 "spec.replicas"、value 为 3 时得到 {"spec":{"replicas":3}}。
//
// path 的每一段是 map 的键，以 '.' 分隔。
//
// 点分键的约定来自历史 render-manager 实现。
func MarshalNestedFromDotPath(path string, value any) ([]byte, error) {
	// 把 path 分割成多个段，每个段作为 map 的键
	segments := strings.Split(path, ".")
	// 创建一个根节点，用于存储最终的 JSON 对象
	root := make(map[string]any)
	// 当前节点，初始化为根节点
	cur := root
	for i, seg := range segments {
		// 最后一层直接赋值
		if i == len(segments)-1 {
			cur[seg] = value
			continue
		}
		// 创建一个新的 map 节点，用于存储下一层
		next := make(map[string]any)
		cur[seg] = next
		cur = next
	}
	// 将最终的对象编码为 JSON 并返回
	return json.MarshalIndent(root, "", "  ")
}
