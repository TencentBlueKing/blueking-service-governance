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

package appruntime

// LegacyTypeQuery describes how a list type filter should match old and new records.
type LegacyTypeQuery struct {
	// ExactType is used when the query type is not a legacy AppModel type.
	ExactType string
	// LegacyType is the old persisted Application.Type (trpc or taf).
	LegacyType string
	// Framework is matched on type=bkmsApp records.
	Framework string
	// Expand is true when callers must OR legacy type with bkmsApp+framework.
	Expand bool
}

// ListTypeQuery converts a caller-supplied type filter into store match semantics.
// type=trpc matches old type=trpc records and type=bkmsApp + framework=trpc records.
func ListTypeQuery(appType string) LegacyTypeQuery {
	switch appType {
	case AppTypeTRPC:
		return LegacyTypeQuery{LegacyType: AppTypeTRPC, Framework: FrameworkTRPC, Expand: true}
	case AppTypeTAF:
		return LegacyTypeQuery{LegacyType: AppTypeTAF, Framework: FrameworkTAF, Expand: true}
	default:
		return LegacyTypeQuery{ExactType: appType}
	}
}
