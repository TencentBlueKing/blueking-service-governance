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

import (
	"strings"

	"github.com/pkg/errors"
)

// 本文件的错误与 issue code 分为两组，用 BRIDGE-ONLY 标记区分，便于阶段 6 收缩时检索：
//
//   - 未标记的属于新模型自身的校验，迁移完成后仍然需要，调用方用 errors.Is 匹配。
//   - 标记 BRIDGE-ONLY 的仅服务于桥接期。它们描述的是"同一份技术栈散落在多个持久化
//     字段"带来的冲突与缺失，是迁移期的诊断设施而非长期契约。

// 新模型自身的校验结论，由 ValidateStack / DecodeStrict 产生，迁移完成后保留。
const (
	IssueUnknownAppType        = "unknown_app_type"
	IssueUnknownFramework      = "unknown_framework"
	IssueUnknownLanguage       = "unknown_language"
	IssueInvalidCombination    = "invalid_combination"
	IssueUnexpectedStackFields = "unexpected_stack_fields"
	IssueUnknownConfigVersion  = "unknown_config_version"
	IssueMissingConfigVersion  = "missing_config_version"
)

// BRIDGE-ONLY: 以下 issue code 全部源自新旧字段并存。
// 阶段 6 删除 TrpcSpec / Workload.Type / TrpcConfig / TafConfig 之后，这些冲突在数据
// 结构上不再可能出现，应连同 Inspect 中对应的判断分支一起删除。
const (
	IssueLanguageConflict        = "language_conflict"
	IssueTypeFrameworkConflict   = "type_framework_conflict"
	IssueIncompleteLanguage      = "incomplete_language"
	IssueDualFrameworkConfig     = "dual_framework_config"
	IssueFrameworkConfigConflict = "framework_config_conflict"
	IssueMissingFramework        = "missing_framework"
	// IssueOrphanApplication 描述的是真实的数据完整性故障而非字段冲突，
	// 但它只经扫描报告对外暴露，因此与扫描设施同期退场。
	IssueOrphanApplication = "orphan_application"
)

// 长期 sentinel，供调用方用 errors.Is 匹配。
var (
	ErrUnknownAppType        = errors.New("unknown app type")
	ErrUnknownFramework      = errors.New("unknown framework")
	ErrUnknownLanguage       = errors.New("unknown language")
	ErrInvalidCombination    = errors.New("invalid framework/language combination")
	ErrUnexpectedStackFields = errors.New("helm/agones must not carry language or framework")
	ErrUnknownConfigVersion  = errors.New("unknown frameworkConfigVersion")
	ErrMissingConfigVersion  = errors.New("frameworkConfigVersion is missing")
)

// BRIDGE-ONLY: Normalize 的唯一失败出口。Normalize 本身是为调和多个持久化来源而存在的，
// 新模型成为唯一事实来源后直接调用 ValidateStack 即可，该 sentinel 随之删除。
var ErrNormalizeFailed = errors.New("runtime snapshot normalize failed")

// BRIDGE-ONLY: Issue / Report 是桥接期的冲突上报载体，服务于扫描报告与迁移决策。
// Issue is a locatable normalize/scan finding. It never carries config body or secrets.
type Issue struct {
	Code     string
	Blocking bool
	Message  string
	Fields   map[string]string
}

// Report is the inspect result for one application.
type Report struct {
	AppID                  string
	PersistedType          string
	Stack                  Stack
	Sources                Sources
	FrameworkConfig        map[string]any
	FrameworkConfigVersion int
	Issues                 []Issue
}

// Sources records which persisted field supplied each normalized value.
type Sources struct {
	LanguageFrom  string
	FrameworkFrom string
}

// Blocking reports whether any issue forbids using the stack as a unified model.
func (r Report) Blocking() bool {
	for _, issue := range r.Issues {
		if issue.Blocking {
			return true
		}
	}
	return false
}

// Err returns a sentinel-wrapped error listing blocking issue codes.
func (r Report) Err() error {
	if !r.Blocking() {
		return nil
	}
	codes := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		if issue.Blocking {
			codes = append(codes, issue.Code)
		}
	}
	return errors.Wrapf(ErrNormalizeFailed, "app %s: %s", r.AppID, strings.Join(codes, ","))
}

func newIssue(code string, blocking bool, message string, fields map[string]string) Issue {
	return Issue{Code: code, Blocking: blocking, Message: message, Fields: fields}
}
