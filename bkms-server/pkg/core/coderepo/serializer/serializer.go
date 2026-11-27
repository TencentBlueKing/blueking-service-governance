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

// Package serializer 代码库接口的 Gin Input/Output 结构体
package serializer

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/coderepo"
)

// workspaceIDPattern 工作空间 ID 校验规则
var workspaceIDPattern = regexp.MustCompile("^[a-z]([a-z0-9-]*[a-z0-9])?$")

// WorkspaceURIInput 路径参数
type WorkspaceURIInput struct {
	WorkspaceID string `uri:"workspaceID" binding:"required,min=1,max=27,workspace_id"`
}

// RepositoryInput 代码库配置
type RepositoryInput struct {
	// 接入方式名，取值见前端开关 repoSourceTypes
	Type string `json:"type" binding:"required,max=32"`
	// 仓库 HTTP(S) 地址
	RepoURL string `json:"repoURL" binding:"required,max=512,url"`
	// 代码库别名，仅蓝盾托管方式使用
	RepoAlias string `json:"repoAlias" binding:"omitempty,max=256"`
	// 仓库账号，仅 basic 方式使用
	Username string `json:"username" binding:"omitempty,max=128"`
	// 仓库密码，仅 basic 方式使用，不传时沿用已保存的
	Password string `json:"password" binding:"omitempty,max=256"`
}

// ConnectivityTestInput 连通性测试请求
type ConnectivityTestInput struct {
	RepositoryInput
}

// ListRefsInput 查询分支或 Tag 请求
type ListRefsInput struct {
	RepositoryInput
	// 引用类型：branch 分支 / tag Tag
	Kind string `json:"kind" binding:"required,oneof=branch tag"`
	// 名称关键字，忽略大小写
	Search string `json:"search" binding:"omitempty,max=128"`
	// 页码，最小为 1
	Page int64 `json:"page" binding:"required,min=1"`
	// 每页数量
	PageSize int64 `json:"pageSize" binding:"required,oneof=5 10 20 50 100"`
}

// RefOutput 分支或 Tag
type RefOutput struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	SHA     string `json:"sha"`
	LinkURL string `json:"linkUrl"`
}

// ListRefsOutput 查询分支或 Tag 响应
type ListRefsOutput struct {
	Data []*RefOutput `json:"data"`
}

// EmptyOutput 空响应
type EmptyOutput struct{}

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("workspace_id", func(fl validator.FieldLevel) bool {
			return workspaceIDPattern.MatchString(fl.Field().String())
		}); err != nil {
			panic("failed to register workspace_id validator: " + err.Error())
		}
	}
}

// ToSpec 转为保存参数
func (in RepositoryInput) ToSpec(workspaceID string) coderepo.SaveSpec {
	return coderepo.SaveSpec{
		WorkspaceID: workspaceID,
		Type:        in.Type,
		URL:         in.RepoURL,
		Alias:       in.RepoAlias,
		Username:    in.Username,
		Password:    in.Password,
	}
}

// ToQuery 转为查询参数
func (in ListRefsInput) ToQuery() coderepo.RefsQuery {
	return coderepo.RefsQuery{
		Kind: coderepo.RefKind(in.Kind), Search: in.Search, Page: in.Page, PageSize: in.PageSize,
	}
}

// NewListRefsOutput 生成响应
func NewListRefsOutput(refs []coderepo.Ref) *ListRefsOutput {
	return &ListRefsOutput{Data: lo.Map(refs, func(ref coderepo.Ref, _ int) *RefOutput {
		return &RefOutput{Name: ref.Name, Path: ref.Path, SHA: ref.SHA, LinkURL: ref.LinkURL}
	})}
}
