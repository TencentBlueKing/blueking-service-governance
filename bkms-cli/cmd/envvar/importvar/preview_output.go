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

// Package importvar provides the 'envvar import' sub-command group.
package importvar

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// formatPreviewOutput 格式化预览结果输出。
// 默认以表格展示 items，表格下方输出 summary 汇总。
// 支持 json/yaml 格式输出完整预览数据。
func formatPreviewOutput(ctx context.Context, preview *client.EnvVarImportPreview, outputFormat string) error {
	// 如果指定了 json/yaml 格式，直接输出完整数据
	if outputFormat != "" {
		formatted, err := output.FormatData(ctx, preview, outputFormat)
		if err != nil {
			return errors.Wrap(err, "format preview output")
		}
		fmt.Println(formatted)
		return nil
	}

	// 默认表格输出
	if len(preview.Items) > 0 {
		formatted, err := output.FormatData(ctx, toPreviewTableItems(preview.Items), "table")
		if err != nil {
			return errors.Wrap(err, "format preview table")
		}
		fmt.Println(formatted)
	}

	// 输出汇总信息
	console.Info("\nSummary: total=%d, new=%d, overwrite=%d\n",
		preview.Summary.Total, preview.Summary.New, preview.Summary.Overwrite)
	return nil
}

// previewTableItem 用于表格展示的预览条目。
type previewTableItem struct {
	Key           string `json:"key" yaml:"key"`
	Value         string `json:"value" yaml:"value"`
	Action        string `json:"action" yaml:"action"`
	OriginalValue string `json:"originalValue" yaml:"originalValue"`
	Messages      string `json:"messages" yaml:"messages"`
}

// toPreviewTableItems 将预览结果转换为表格展示格式。
func toPreviewTableItems(items []client.EnvVarImportPreviewItem) []previewTableItem {
	result := make([]previewTableItem, 0, len(items))
	for _, item := range items {
		tableItem := previewTableItem{
			Key:           item.Key,
			Value:         item.Value,
			Action:        item.Action,
			OriginalValue: item.OriginalValue,
		}
		if len(item.Messages) > 0 {
			tableItem.Messages = strings.Join(item.Messages, "; ")
		}
		result = append(result, tableItem)
	}
	return result
}
