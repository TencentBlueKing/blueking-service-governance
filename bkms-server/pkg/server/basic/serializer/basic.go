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

// Package serializer 定义 basic 模块的 Gin Input/Output 结构体。
package serializer

// VersionOutput 是 Version 接口的 JSON 响应。
type VersionOutput struct {
	// 版本信息
	Data *VersionData `json:"data"`
}

// VersionData 包含服务版本详细信息。
type VersionData struct {
	// 版本号
	Version string `json:"version"`
	// Git Hash
	GitHash string `json:"gitHash"`
	// 构建时间
	BuildTime string `json:"buildTime"`
	// Go 版本号
	GoVersion string `json:"goVersion"`
}

// PingOutput 是 Ping 接口的 JSON 响应。
type PingOutput struct {
	// 联通性测试结果
	Data string `json:"data"`
}
