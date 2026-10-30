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

package client

import "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"

// IsAppModelType 判断应用是否由 AppModel 管理：trpc、taf 与 bkmsApp 都属于平台托管应用。
func IsAppModelType(appType string) bool {
	return appType == constant.AppTypeTrpc ||
		appType == constant.AppTypeTaf ||
		appType == constant.AppTypeBKMSApp
}

// FrameworkOf 返回应用的技术栈框架。
// 旧 trpc / taf 应用没有独立的 framework 字段（服务端会补成同名值），bkmsApp 应用以 framework 为准。
func FrameworkOf(appType, framework string) string {
	if !IsAppModelType(appType) {
		return ""
	}
	if framework != "" {
		return framework
	}
	// 兼容旧服务端未返回 framework 的情况
	if appType == constant.AppTypeBKMSApp {
		return ""
	}
	return appType
}

// IsBKMSApp 判断应用是否由新接口创建（type=bkmsApp）
func IsBKMSApp(appType string) bool {
	return appType == constant.AppTypeBKMSApp
}
