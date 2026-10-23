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

// CapabilityStack 从存量快照得出用于能力查表的技术栈。
//
// 与 Normalize 的区别是缺少语言不算错误，其他冲突照常报错。存量 TAF 应用都未记录语言，
// 部分 tRPC 应用的语言只记录在 workload 上，若报错，这些应用会失去现有能力。
// BRIDGE-ONLY：存量语言回填后删除，改用 ValidateStack。
func CapabilityStack(s Snapshot) (Stack, error) {
	report := Inspect(s)
	for _, issue := range report.Issues {
		if issue.Blocking && issue.Code != IssueIncompleteLanguage {
			return Stack{}, report.Err()
		}
	}
	return report.Stack, nil
}
