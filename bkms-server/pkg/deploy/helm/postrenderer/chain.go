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

package postrenderer

import (
	"bytes"

	"helm.sh/helm/v3/pkg/postrender"
)

// ChainPostRenderer 链式组合多个 PostRenderer
// 按 renderers 切片顺序依次执行，前一个的输出作为后一个的输入
type ChainPostRenderer struct {
	renderers []postrender.PostRenderer
}

// 编译期接口实现检查
var _ postrender.PostRenderer = (*ChainPostRenderer)(nil)

// NewChainPostRenderer 创建链式 PostRenderer
// 自动过滤 nil 元素；如果过滤后为空，返回 nil
func NewChainPostRenderer(renderers ...postrender.PostRenderer) *ChainPostRenderer {
	var filtered []postrender.PostRenderer
	for _, r := range renderers {
		if r != nil {
			filtered = append(filtered, r)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return &ChainPostRenderer{renderers: filtered}
}

// Run 按顺序执行所有 PostRenderer，前一个的输出作为后一个的输入
func (c *ChainPostRenderer) Run(renderedManifests *bytes.Buffer) (*bytes.Buffer, error) {
	current := renderedManifests
	for _, r := range c.renderers {
		result, err := r.Run(current)
		if err != nil {
			return nil, err
		}
		current = result
	}
	return current, nil
}
