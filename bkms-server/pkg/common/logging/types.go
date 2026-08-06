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

package logging

const (
	defaultLevel       = "info"
	defaultHandlerName = "json"
	defaultWriterName  = "stdout"

	// HandlerText 表示文本格式日志处理器。
	HandlerText = "text"
	// HandlerJSON 表示 JSON 格式日志处理器。
	HandlerJSON = "json"

	// WriterStdout 表示日志输出到标准输出。
	WriterStdout = "stdout"
	// WriterStderr 表示日志输出到标准错误。
	WriterStderr = "stderr"
	// WriterFile 表示日志输出到本地文件。
	WriterFile = "file"

	// FieldTraceID 表示链路追踪 trace ID 字段名。
	FieldTraceID = "trace_id"
	// FieldSpanID 表示链路追踪 span ID 字段名。
	FieldSpanID = "span_id"
)
