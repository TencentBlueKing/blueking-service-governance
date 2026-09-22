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

// Package appruntime 定义应用技术栈值类型、框架目录以及旧数据规范化合同。
//
// 本包不 import appmodel、component、具体框架适配器或 handler。
package appruntime

const (
	// AppTypeHelm helm 应用。
	AppTypeHelm = "helm"
	// AppTypeAgones agones 应用。
	AppTypeAgones = "agones"
	// AppTypeBKMSApp 使用 AppModel 管理的默认应用类型。
	AppTypeBKMSApp = "bkmsApp"
	// AppTypeTRPC 旧持久化值，桥接期仍有效。
	AppTypeTRPC = "trpc"
	// AppTypeTAF 旧持久化值，桥接期仍有效。
	AppTypeTAF = "taf"
)

const (
	// LanguageGo Go 技术栈。
	LanguageGo = "go"
	// LanguageCpp C++ 技术栈。
	LanguageCpp = "cpp"
)

const (
	// FrameworkTRPC tRPC 框架。
	FrameworkTRPC = "trpc"
	// FrameworkTAF TAF 框架。
	FrameworkTAF = "taf"
	// FrameworkBlank 显式空框架，不是空字符串。
	FrameworkBlank = "blank"
)

const (
	// FrameworkConfigVersionV1 首期 frameworkConfig schema 版本。
	FrameworkConfigVersionV1 = 1
)

// Stack is the normalized management type plus tech stack.
type Stack struct {
	// Type is helm, agones or bkmsApp. Legacy trpc/taf persistences normalize to bkmsApp.
	Type string
	// Framework is trpc, taf or blank. Empty for Helm/Agones.
	Framework string
	// Language is go or cpp. Empty when incomplete and not guessed.
	Language string
}

// Snapshot is the compatibility DTO for persisted tech-stack fields.
// Callers assemble it from Application and AppModel; this package does not load storage.
type Snapshot struct {
	AppID string

	AppType   string
	Language  string
	Framework string

	TrpcSpecLanguage   string
	WorkloadType       string
	TrpcConfigLanguage string

	HasTrpcConfig bool
	HasTafConfig  bool

	TrpcFileName string
	TrpcFilePath string
	TafFileName  string
	TafFilePath  string

	HasTrpcFileContent bool
	HasTafFileContent  bool

	FrameworkConfig        map[string]any
	FrameworkConfigVersion int

	// AppModelMissing is set by scanners when an AppModel-managed app has no AppModel row.
	AppModelMissing bool
}

// IsAppModelType reports whether appType is managed by AppModel.
// During the bridge period this includes legacy trpc/taf and bkmsApp.
func IsAppModelType(appType string) bool {
	return appType == AppTypeTRPC || appType == AppTypeTAF || appType == AppTypeBKMSApp
}

// IsHelmBasedType reports whether appType is a Helm chart based management style.
func IsHelmBasedType(appType string) bool {
	return appType == AppTypeHelm || appType == AppTypeAgones
}

// FrameworkFromLegacyType projects an old Application.Type onto a framework name.
// Non trpc/taf types return empty.
func FrameworkFromLegacyType(appType string) string {
	switch appType {
	case AppTypeTRPC:
		return FrameworkTRPC
	case AppTypeTAF:
		return FrameworkTAF
	default:
		return ""
	}
}
