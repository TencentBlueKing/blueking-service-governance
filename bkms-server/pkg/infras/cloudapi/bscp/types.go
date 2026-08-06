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

package bscp

import (
	"context"
	"encoding/base64"
	"path"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

// Biz BSCP 业务（接口中被称为空间）
type Biz struct {
	ID   string
	Name string
}

// ConfigType BSCP 配置类型
type ConfigType string

const (
	// ConfigTypeKV 为键值型配置
	ConfigTypeKV ConfigType = "kv"
	// ConfigTypeFile 为文件型配置
	ConfigTypeFile ConfigType = "file"
)

// DataType BSCP 数据类型，用于描述 kv 型配置的数据类型
type DataType string

const (
	// DataTypeAny 为任意类型
	DataTypeAny DataType = "any"
	// DataTypeString 为字符串类型
	DataTypeString DataType = "string"
	// DataTypeNumber 为数字类型
	DataTypeNumber DataType = "number"
	// DataTypeText 为文本类型
	DataTypeText DataType = "text"
	// DataTypeJSON 为 json 类型
	DataTypeJSON DataType = "json"
	// DataTypeXML 为 xml 类型
	DataTypeXML DataType = "xml"
	// DataTypeYAML 为 yaml 类型
	DataTypeYAML DataType = "yaml"
	// DataTypeSecret 为敏感信息类型
	DataTypeSecret DataType = "secret"
)

// ApproveType BSCP 审批类型
type ApproveType string

const (
	// ApproveTypeCountSign 为会签（需所有审批人同意）
	ApproveTypeCountSign ApproveType = "count_sign"
	// ApproveTypeOrSign 为或签（任一审批人同意即可）
	ApproveTypeOrSign ApproveType = "or_sign"
)

// Service BSCP 服务（接口中称为应用）
type Service struct {
	ID         string
	Name       string
	Alias      string
	Desc       string
	ConfigType ConfigType
	DataType   DataType
}

// CreateServiceReq 创建 BSCP 服务
type CreateServiceReq struct {
	// BizID 业务ID，作为 URL path 参数传递
	BizID string `json:"-" validate:"required"`

	// Name 服务名称
	Name string `json:"name" validate:"required"`

	// ConfigType 服务类型
	ConfigType ConfigType `json:"config_type" validate:"required"`

	// Alias 服务别名
	Alias string `json:"alias" validate:"required"`

	// DataType 数据类型
	DataType DataType `json:"data_type" validate:"required"`

	// IsApprove 是否需要审批
	IsApprove bool `json:"is_approve,omitempty"`

	// ApproveType 审批类型，仅 IsApprove 为 true 时生效
	ApproveType ApproveType `json:"approve_type,omitempty"`

	// Approver 审批人列表（字符串，多个审批人以","分隔符拼接），仅 IsApprove 为 true 时生效
	Approver string `json:"approver,omitempty"`

	// Memo 服务描述
	Memo string `json:"memo,omitempty"`
}

// NewCreateServiceReq 以最少必填参数创建 CreateServiceReq
// 默认不需要审批
func NewCreateServiceReq(
	bizID, name, alias string,
	configType ConfigType,
	dataType DataType,
) *CreateServiceReq {
	return &CreateServiceReq{
		BizID:      bizID,
		Name:       name,
		Alias:      alias,
		DataType:   dataType,
		ConfigType: configType,
		IsApprove:  false,
	}
}

// Version BSCP 版本
type Version struct {
	ID   string
	Name string
	Desc string
	// 是否为全量发布
	IsFullyReleased bool
	Creator         string
	CreatedAt       string
}

// Versions BSCP 版本列表
type Versions []Version

// LatestFullyReleased 获取最新的全量发布版本
func (versions Versions) LatestFullyReleased() *Version {
	// 默认逆序，因此按顺序找第一个即可
	for _, v := range versions {
		if v.IsFullyReleased {
			return &v
		}
	}
	return nil
}

// Config 配置项接口，统一处理文件型和键值型配置
type Config interface {
	// ID 获取配置项 id
	ID() string
	// Name 获取配置项名称
	Name() string
	// Desc 获取配置项描述
	Desc() string
	// Type 获取配置类型
	Type() ConfigType
	// Content 获取配置内容
	Content(ctx context.Context) (string, error)
}

// FileType BSCP 配置文件类型
type FileType string

const (
	// FileTypeText 文本类型
	FileTypeText FileType = "text"
	// FileTypeBinary 二进制类型
	FileTypeBinary FileType = "binary"
)

// File BSCP 配置（文件型）
type File struct {
	name      string
	path      string
	desc      string
	bizID     string
	svcID     string
	signature string
}

// ID ...
func (f *File) ID() string {
	// 注：与 bscp 客户端逻辑保持一致，使用 path + name 拼接作为文件唯一标识
	// 如：/path/to/file.txt，/root.txt，这样即使服务端删除并重建，也可以拿到配置
	// base64 编码避免 ID 中包含特殊字符
	return base64.URLEncoding.EncodeToString([]byte(f.Name()))
}

// Name ...
func (f *File) Name() string {
	// Path: /etc/nginx, Name: nginx.conf
	return path.Join(f.path, f.name)
}

// Desc ...
func (f *File) Desc() string {
	return f.desc
}

// Type ...
func (f *File) Type() ConfigType {
	return ConfigTypeFile
}

// Content 通过 API 下载文件内容
func (f *File) Content(ctx context.Context) (string, error) {
	// stub 模式下直接返回模拟内容
	if config.G.Development.UseStubBSCP {
		return "stub-file-content", nil
	}

	apiClient, err := newApiClient(auth.MustGetUser(ctx))
	if err != nil {
		return "", errors.Wrap(err, "initial bscp client")
	}

	return apiClient.getFileContent(ctx, f.bizID, f.svcID, f.signature)
}

var _ Config = &File{}

// KeyValue BSCP 配置（键值型）
type KeyValue struct {
	key   string
	value string
	desc  string
}

// NewKeyValue 创建键值型配置
func NewKeyValue(key, value, desc string) *KeyValue {
	return &KeyValue{key: key, value: value, desc: desc}
}

// ID ...
func (kv *KeyValue) ID() string {
	// 注：键值对类型的，固定索引是 Key，ID 会随版本发布而变化
	// base64 编码避免 ID 中包含特殊字符
	return base64.URLEncoding.EncodeToString([]byte(kv.key))
}

// Name ...
func (kv *KeyValue) Name() string {
	return kv.key
}

// Desc ...
func (kv *KeyValue) Desc() string {
	return kv.desc
}

// Type ...
func (kv *KeyValue) Type() ConfigType {
	return ConfigTypeKV
}

// Content 直接返回 Value 字段
func (kv *KeyValue) Content(_ context.Context) (string, error) {
	return kv.value, nil
}

var _ Config = &KeyValue{}

// Credential BSCP 客户端密钥
type Credential struct {
	// ID 密钥ID
	ID int64
	// Name 密钥名称
	Name string
	// Memo 密钥描述
	Memo string
	// Enable 是否启用
	Enable bool
	// CredentialType 凭证类型（如 bearToken）
	CredentialType string
	// EncCredential 加密后的凭证
	EncCredential string
	// CredentialScopes 关联规则列表
	CredentialScopes []string
}

// CredentialScope 密钥关联服务规则
type CredentialScope struct {
	// ID 关联规则ID
	ID int64
	// App 服务名称
	App string
	// Scope 关联规则
	Scope string
}

// CreateCredentialReq 创建客户端密钥请求
type CreateCredentialReq struct {
	// BizID 业务ID，作为 URL path 参数传递
	BizID string `json:"-" validate:"required"`
	// Name 密钥名称
	Name string `json:"name" validate:"required"`
	// Memo 密钥描述
	Memo string `json:"memo,omitempty"`
}

// Validate 校验创建凭证请求
func (r *CreateCredentialReq) Validate() error {
	return Validate(r)
}

// UpdateCredentialReq 更新客户端密钥请求
type UpdateCredentialReq struct {
	// BizID 业务ID，作为 URL path 参数传递
	BizID string `json:"-" validate:"required"`
	// ID 密钥ID
	ID int64 `json:"id" validate:"required"`
	// Enable 是否启用
	Enable *bool `json:"enable,omitempty"`
	// Memo 密钥描述
	Memo string `json:"memo,omitempty"`
	// Name 密钥名称
	Name string `json:"name,omitempty"`
}

// Validate 校验更新凭证请求
func (r *UpdateCredentialReq) Validate() error {
	return Validate(r)
}

// CredentialScopeItem 新增关联规则项
type CredentialScopeItem struct {
	// App 服务名称
	App string `json:"app"`
	// Scope 关联规则
	Scope string `json:"scope"`
}

// AlterScopeItem 更新关联规则项
type AlterScopeItem struct {
	// ID 关联规则ID
	ID int64 `json:"id"`
	// App 服务名称
	App string `json:"app"`
	// Scope 关联规则
	Scope string `json:"scope"`
}

// UpdateCredentialScopeReq 更新密钥关联服务规则请求
type UpdateCredentialScopeReq struct {
	// BizID 业务ID，作为 URL path 参数传递
	BizID string `json:"-" validate:"required"`
	// CredentialID 密钥ID，作为 URL path 参数传递
	CredentialID string `json:"-" validate:"required"`
	// AddScope 新增规则
	AddScope []CredentialScopeItem `json:"addScope,omitempty"`
	// AlterScope 更新规则
	AlterScope []AlterScopeItem `json:"alterScope,omitempty"`
	// DelID 删除规则ID列表
	DelID []int64 `json:"delId,omitempty"`
}

// Validate 校验更新密钥关联规则请求
func (r *UpdateCredentialScopeReq) Validate() error {
	return Validate(r)
}

// --- Hook 相关类型 ---

// Hook BSCP 脚本
type Hook struct {
	// ID 脚本ID
	ID int64
	// Name 脚本名称
	Name string
	// Type 脚本类型：shell、python、bat、powershell
	Type string
	// Tags 脚本标签
	Tags []string
	// Memo 脚本描述
	Memo string
	// NotReleaseID 未发布版本ID
	NotReleaseID int64
	// Creator 创建人
	Creator string
	// Reviser 更新人
	Reviser string
	// CreateAt 创建时间
	CreateAt string
	// UpdateAt 更新时间
	UpdateAt string
}

// ReleaseHookDetail 版本绑定的脚本详情
type ReleaseHookDetail struct {
	// HookID 脚本ID
	HookID int64
	// HookName 脚本名称
	HookName string
	// HookRevisionID 脚本版本ID
	HookRevisionID int64
	// HookRevisionName 脚本版本名称
	HookRevisionName string
	// Type 脚本类型
	Type string
	// Content 脚本内容
	Content string
}

// ReleaseHook 版本绑定的前后置脚本
type ReleaseHook struct {
	// PreHook 前置脚本
	PreHook *ReleaseHookDetail
	// PostHook 后置脚本
	PostHook *ReleaseHookDetail
}

// HookListItem 脚本列表项
type HookListItem struct {
	// Hook 脚本信息
	Hook Hook
	// BoundNum 绑定数量
	BoundNum int64
	// ConfirmDelete 是否确认删除
	ConfirmDelete bool
	// PublishedRevisionID 已发布版本ID
	PublishedRevisionID int64
}

// ListHooksResp 获取脚本列表响应
type ListHooksResp struct {
	// Count 总数
	Count int64
	// Details 脚本列表
	Details []HookListItem
}

// CreateHookReq 创建脚本请求
type CreateHookReq struct {
	// BizID 业务ID，作为 URL path 参数传递
	BizID string `json:"-" validate:"required"`
	// Name 脚本名称
	Name string `json:"name" validate:"required"`
	// Type 脚本类型：shell、python、bat、powershell
	Type string `json:"type" validate:"required"`
	// Content 脚本内容
	Content string `json:"content" validate:"required"`
	// RevisionName 版本号
	RevisionName string `json:"revision_name" validate:"required"`
	// Tags 分类标签
	Tags []string `json:"tags,omitempty"`
	// Memo 脚本描述
	Memo string `json:"memo,omitempty"`
}

// Validate 校验创建脚本请求
func (r *CreateHookReq) Validate() error {
	return Validate(r)
}

// DeleteHookReq 删除脚本请求
type DeleteHookReq struct {
	// BizID 业务ID，作为 URL path 参数传递
	BizID string `validate:"required"`
	// HookID 脚本ID
	HookID int64 `validate:"required"`
	// Force 是否强制删除
	Force bool
}

// Validate 校验删除脚本请求
func (r *DeleteHookReq) Validate() error {
	return Validate(r)
}

// ListHooksReq 获取脚本列表请求
type ListHooksReq struct {
	// BizID 业务ID，作为 URL path 参数传递
	BizID string `validate:"required"`
	// Name 脚本名称过滤
	Name string
	// Tag 标签过滤
	Tag string
	// All 是否获取所有
	All bool
	// SearchKey 搜索关键字
	SearchKey string
	// Start 当前页码
	Start int64
	// Limit 每页条数
	Limit int64
}

// UpdateConfigHookReq 更新服务绑定的前后置脚本请求
type UpdateConfigHookReq struct {
	// BizID 业务ID，作为 URL path 参数传递
	BizID string `json:"-" validate:"required"`
	// AppID 服务ID，作为 URL path 参数传递
	AppID int64 `json:"-" validate:"required"`
	// PreHookID 前置脚本ID，0 表示不绑定
	PreHookID int64 `json:"pre_hook_id"`
	// PostHookID 后置脚本ID，0 表示不绑定
	PostHookID int64 `json:"post_hook_id"`
}

// Validate 校验更新服务绑定脚本请求
func (r *UpdateConfigHookReq) Validate() error {
	return Validate(r)
}

// UpdateHookReq 更新脚本信息请求
type UpdateHookReq struct {
	// BizID 业务ID，作为 URL path 参数传递
	BizID string `json:"-" validate:"required"`
	// HookID 脚本ID，作为 URL path 参数传递
	HookID int64 `json:"-" validate:"required"`
	// Tags 脚本标签
	Tags []string `json:"tags,omitempty"`
	// Memo 脚本描述
	Memo string `json:"memo,omitempty"`
}

// Validate 校验更新脚本请求
func (r *UpdateHookReq) Validate() error {
	return Validate(r)
}
