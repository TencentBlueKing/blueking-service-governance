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

package bkci

import (
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

// parsePipelineVariables 解析流水线变量
func parsePipelineVariables(data map[string]any) ([]PipelineVariable, error) {
	var variables []PipelineVariable

	// 获取流水线的第一个 Stage 配置
	stages := mapx.GetList(data, "stages")
	if len(stages) == 0 {
		return nil, errors.New("pipeline stages invalid (empty)")
	}

	stage, ok := stages[0].(map[string]any)
	if !ok {
		return nil, errors.New("first pipeline stage invalid (not map[string]any type)")
	}

	// 获取第一个 Stage 的第一个 Container 配置
	containers := mapx.GetList(stage, "containers")
	if len(containers) == 0 {
		return nil, errors.New("pipeline stage containers invalid (empty)")
	}

	container, ok := containers[0].(map[string]any)
	if !ok {
		return nil, errors.New("first pipeline stage container invalid (not map[string]any type)")
	}

	// 获取 Container 的 ClassType，并检查是否为 Trigger 类型
	classType := mapx.GetStr(container, "classType")
	if classType != "trigger" {
		return nil, errors.Errorf("first pipeline stage container classType is not trigger, got: %s", classType)
	}

	// 获取 Container 的 Variables
	for _, p := range mapx.GetList(container, "params") {
		v, ok := p.(map[string]any)
		if !ok {
			return nil, errors.Errorf("invalid pipeline variable (not map[string]any type): %v", p)
		}
		// 获取变量配置可选项
		var options []PipelineVariableOption
		for _, opt := range mapx.GetList(v, "options") {
			o, ok := opt.(map[string]any)
			if !ok {
				return nil, errors.Errorf("invalid pipeline variable option (not map[string]any type): %v", opt)
			}
			options = append(options, PipelineVariableOption{
				Key:   mapx.GetStr(o, "key"),
				Value: mapx.GetStr(o, "value"),
			})
		}
		// 组装变量数据
		variables = append(variables, PipelineVariable{
			ID:          mapx.GetStr(v, "id"),
			Name:        mapx.GetStr(v, "name"),
			Description: mapx.GetStr(v, "desc"),
			// 注：与蓝盾保持一致，使用 valueNotEmpty 控制参数是否页面上必填
			Required:     mapx.GetBool(v, "valueNotEmpty"),
			ReadOnly:     mapx.GetBool(v, "readOnly"),
			Constant:     mapx.GetBool(v, "constant"),
			DefaultValue: cast.ToString(mapx.Get(v, "defaultValue", "")),
			Type:         mapx.GetStr(v, "type"),
			Options:      options,
		})
	}

	return variables, nil
}

// parseRepositoryRefs 解析 BKCI 代码库分支/Tag响应
func parseRepositoryRefs(items []any) ([]RepositoryRef, error) {
	refs := make([]RepositoryRef, 0, len(items))
	for _, item := range items {
		ref, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Errorf("invalid repository ref (not map[string]any type): %v", item)
		}
		refs = append(refs, RepositoryRef{
			Name:    mapx.GetStr(ref, "name"),
			Path:    mapx.GetStr(ref, "path"),
			SHA:     mapx.GetStr(ref, "sha"),
			LinkURL: mapx.GetStr(ref, "linkUrl"),
		})
	}
	return refs, nil
}

// extractErrorMessage 从蓝盾/网关响应结果中提取错误消息
// 按优先级依次尝试 message、data.message、error.message、error 等常见字段，
// 如果都为空则返回明确的兜底提示
func extractErrorMessage(result map[string]any) string {
	// 按优先级尝试多个常见错误消息字段
	candidates := []string{
		"message",
		"data.message",
		"error.message",
		"error",
		"msg",
		"data.msg",
	}
	for _, key := range candidates {
		if msg := mapx.GetStr(result, key); msg != "" {
			return msg
		}
	}
	return ""
}
