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

// Package serializer defines Gin input and output serializers for AppSpec APIs.
package serializer

import _ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators

// AppURIInput is the path input for APIs scoped by application.
type AppURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
}

// AppEnvURIInput is the path input for APIs scoped by application and environment.
type AppEnvURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 环境名称
	EnvName string `uri:"envName" binding:"required,uri_slug"`
}

// AppEnvProbeTypeURIInput is the path input for deleting one probe type from an environment AppSpec.
type AppEnvProbeTypeURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 环境名称
	EnvName string `uri:"envName" binding:"required,uri_slug"`
	// 探针类型：liveness、readiness 或 startup
	ProbeType string `uri:"probeType" binding:"required,oneof=liveness readiness startup"`
}

// EmptyOutput is the JSON response for APIs that return no data.
type EmptyOutput struct{}
