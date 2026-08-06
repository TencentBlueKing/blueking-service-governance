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

// Package log 提供统一的构建日志读取能力，同时覆盖应用镜像构建与 Helm Chart 构建两条链路
package log

import "fmt"

// BuildLogQuery 从构建记录和 BKCI 项目解析出的日志查询对象
type BuildLogQuery struct {
	// ProjectCode BKCI 项目 Code
	ProjectCode string
	// PipelineID BKCI 流水线 ID
	PipelineID string
	// BuildID BKCI 构建 ID
	BuildID string
	// AppID 应用 ID（用于生成下载文件名等场景）
	AppID string
}

// DownloadFilename 返回默认的构建日志下载文件名
func (q *BuildLogQuery) DownloadFilename() string {
	return fmt.Sprintf("build-log_%s_%s.log", q.AppID, q.BuildID)
}
