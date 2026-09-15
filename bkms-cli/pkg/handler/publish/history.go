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

package publish

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// PublishRecordRow 发布记录展示行
type PublishRecordRow struct {
	Instance   string `json:"instance"`
	BinaryName string `json:"binaryName"`
	FileSize   string `json:"fileSize"`
	MD5        string `json:"md5"`
	Status     string `json:"status"`
	// Message 失败原因等附加信息，仅在 JSON/YAML 输出中保留，表格不展示
	Message   string `json:"message" table:"-"`
	Operator  string `json:"operator"`
	UpdatedAt string `json:"updatedAt"`
}

// ListHistory 获取开发模式发布历史记录（展示格式）。
func ListHistory(
	ctx context.Context,
	cli client.Client,
	appID, envName, keyword string,
) ([]PublishRecordRow, error) {
	records, err := cli.ListDevModePublishRecords(ctx, appID, envName, keyword)
	if err != nil {
		return nil, errors.Wrap(err, "list dev mode publish records")
	}

	rows := lo.Map(records, func(record client.DevModePublishRecord, _ int) PublishRecordRow {
		return PublishRecordRow{
			Instance:   record.Instance,
			BinaryName: record.BinaryName,
			FileSize:   formatFileSize(record.FileSize),
			MD5:        record.MD5,
			Status:     record.Status,
			Message:    record.Message,
			Operator:   record.Operator,
			UpdatedAt:  record.UpdatedAt,
		}
	})
	return rows, nil
}

// formatFileSize 将字节数格式化为人类可读字符串（B / KB / MB / GB ...）。
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
