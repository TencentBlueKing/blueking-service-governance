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

// Package revision 定义一次构建所使用的代码修订（分支 / Tag / CommitID），供镜像构建与 Helm Chart 构建共用
package revision

import (
	"regexp"

	"github.com/pkg/errors"
)

// Type 代码来源类型，取值与接口字段 revisionType 一致
type Type string

const (
	// TypeBranch 按分支检出
	TypeBranch Type = "branch"
	// TypeTag 按 Tag 检出
	TypeTag Type = "tag"
	// TypeCommit 按 CommitID 检出
	TypeCommit Type = "commit"
)

// 蓝盾拉代码插件 pullType 的取值，与流水线模板中 BKMS_REPO_CHECKOUT_BY 的枚举保持一致
const (
	checkoutByBranch   = "BRANCH"
	checkoutByTag      = "TAG"
	checkoutByCommitID = "COMMIT_ID"
)

// imageTagCommitIDLen CommitID 拼进镜像 Tag 时保留的位数，与 git 默认短 SHA 一致
const imageTagCommitIDLen = 7

// commitIDRegexp CommitID 允许 7–40 位十六进制，不区分大小写
var commitIDRegexp = regexp.MustCompile(`^[0-9a-fA-F]{7,40}$`)

// ErrInvalidCommitID CommitID 格式不合法
var ErrInvalidCommitID = errors.New("commit id must be 7-40 hexadecimal characters")

// ErrRequired 构建未指定代码修订
// 镜像构建只在代码仓库模式下返回；Helm Chart 构建始终要求有修订
var ErrRequired = errors.New("code revision is required")

// CodeRevision 一次构建使用的代码修订
type CodeRevision struct {
	// Type 代码来源类型
	Type Type
	// Value 分支名、Tag 名或 CommitID，按用户填写原样保存
	Value string
}

// Resolve 合并接口的新旧字段得到代码修订
//
// typ 为空时按分支处理；只校验非空 CommitID 的格式，是否允许为空由调用方按构建模式决定
func Resolve(typ, value, legacyBranch string) (CodeRevision, error) {
	rev := CodeRevision{
		Type:  Type(typ),
		Value: value,
	}
	if rev.Type == "" {
		rev.Type = TypeBranch
	}

	// 兼容字段 branch 的含义是「按分支检出」，只能补给分支类型；
	// 显式选了 Tag / CommitID 却没填修订时不能拿分支名顶替，否则会按错误的方式检出
	if rev.Value == "" && rev.Type == TypeBranch {
		rev.Value = legacyBranch
	}

	// 空值放行：推荐版本号要在 CommitID 未填时返回不含修订的版本号，自定义流水线模式也允许不传
	if rev.Type == TypeCommit && rev.Value != "" && !commitIDRegexp.MatchString(rev.Value) {
		return CodeRevision{}, errors.Wrapf(ErrInvalidCommitID, "resolve commit id %q", rev.Value)
	}
	return rev, nil
}

// ResolveForPreview 解析推荐镜像 Tag 用的修订
// CommitID 还没填完时按空修订返回，格式校验留给真正提交构建
func ResolveForPreview(typ, value, legacyBranch string) CodeRevision {
	rev, err := Resolve(typ, value, legacyBranch)
	if err == nil {
		return rev
	}

	// Resolve 目前只会对未填完的 CommitID 报错，预览不能因此整段失败
	return CodeRevision{Type: Type(typ)}
}

// Presented 把已落库的类型和值还原成对外展示的修订
// 类型为空的历史记录按分支处理；分支值为空时回退到旧的 branch 字段
func Presented(storedType, storedValue, legacyBranch string) CodeRevision {
	rev := CodeRevision{
		Type:  Type(storedType),
		Value: storedValue,
	}
	if rev.Type == "" {
		rev.Type = TypeBranch
	}

	if rev.Value == "" && rev.Type == TypeBranch {
		rev.Value = legacyBranch
	}
	return rev
}

// CheckoutBy 返回传给流水线变量 BKMS_REPO_CHECKOUT_BY 的检出方式
// 修订为空时回落 BRANCH：自定义流水线允许不填修订，不能传 COMMIT_ID / TAG 配空值
func (r CodeRevision) CheckoutBy() string {
	if r.Value == "" {
		return checkoutByBranch
	}

	switch r.Type {
	case TypeTag:
		return checkoutByTag
	case TypeCommit:
		return checkoutByCommitID
	default:
		return checkoutByBranch
	}
}

// LegacyBranch 仅在按分支检出时返回修订值，供仍读取 branch 的记录和接口兼容
func (r CodeRevision) LegacyBranch() string {
	if r.Type != TypeBranch {
		return ""
	}
	return r.Value
}

// ImageTagSegment 返回拼进推荐镜像 Tag 的修订片段，CommitID 只保留前 7 位
func (r CodeRevision) ImageTagSegment() string {
	if r.Type == TypeCommit && len(r.Value) > imageTagCommitIDLen {
		return r.Value[:imageTagCommitIDLen]
	}
	return r.Value
}

// TypeFromCheckoutBy 由构建参数中的检出方式反推代码来源类型
//
// 本功能上线前的构建记录只会是 BRANCH，空值与未知值同样视为分支
func TypeFromCheckoutBy(checkoutBy string) Type {
	switch checkoutBy {
	case checkoutByTag:
		return TypeTag
	case checkoutByCommitID:
		return TypeCommit
	default:
		return TypeBranch
	}
}
