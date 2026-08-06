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

package bkci

import (
	"time"
)

// 蓝盾 API 返回中的 Status 字段，表示某些特定的错误
const (
	// 蓝盾项目名或英文名重复
	projectNameExistStatus = 2119002
	// 蓝盾代码库别名已存在
	repoAliasExistStatus = 2115014
)

const (
	// PageForAllItems 用于拉取全量数据的页码
	PageForAllItems = 1
	// PageSizeForAllItems 用于拉取全量数据的分页大小
	PageSizeForAllItems = 1000
)

const (
	// RepoRefPropertyTypeGitRef BKCI 启动表单中的 Git 分支字段类型
	RepoRefPropertyTypeGitRef = "git_ref"
	// RepoRefPropertyTypeSvnTag BKCI 启动表单中的 Tag 字段类型
	RepoRefPropertyTypeSvnTag = "svn_tag"
	// RepoRefPropertyTypeRepoRef BKCI 启动表单中的仓库引用字段类型
	RepoRefPropertyTypeRepoRef = "repo_ref"
)

// Project 蓝盾项目
type Project struct {
	ID            string
	Code          string
	Name          string
	Creator       string
	HasManagePerm bool
}

// GitProject git 项目
type GitProject struct {
	ID    string
	Name  string
	Alias string
	Url   string
}

// RepoRefProperty 流水线启动表单中的代码分支/Tag相关字段
type RepoRefProperty struct {
	// ID 字段唯一标识（对应 BKCI property 的 id）
	ID string
	// Name 字段名（对应 BKCI property 的 name）
	Name string
	// Label 字段展示名称
	Label string
	// Type 字段类型，仅 RepoRefPropertyTypeGitRef、RepoRefPropertyTypeSvnTag、
	// RepoRefPropertyTypeRepoRef 为代码分支/Tag 相关类型
	Type string
	// Required 是否必填
	Required bool
	// ReadOnly 是否只读
	ReadOnly bool
	// Constant 是否为常量（不可修改）
	Constant bool
	// DefaultValue 默认值
	DefaultValue string
	// Value 当前值
	Value string
}

// PipelineVariableOption 流水线变量选项
type PipelineVariableOption struct {
	Key   string
	Value string
}

// PipelineVariable 流水线变量
type PipelineVariable struct {
	ID           string
	Name         string
	Description  string
	Required     bool
	ReadOnly     bool
	Constant     bool
	DefaultValue string
	Type         string
	Options      []PipelineVariableOption
}

// Pipeline 蓝盾流水线
type Pipeline struct {
	ID          string
	Name        string
	Description string
	Version     int64
	Creator     string
	Updater     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Variables   []PipelineVariable
}

// PipelineBuildReference 蓝盾流水线构建引用信息
type PipelineBuildReference struct {
	ID string
	// Num 构建编号（类型是数字，但是因为只作为展示，统一转成字符串）
	Num string
}

// PipelineBuildState 蓝盾流水线构建状态
type PipelineBuildState struct {
	PipelineID  string
	BuildID     string
	BuildNum    string
	UserID      string
	Status      string
	StartTime   int64
	EndTime     int64
	TotalTime   int64
	ExecuteTime int64
	Variables   map[string]string
}

// Repository 蓝盾代码库
type Repository struct {
	ID        string
	Alias     string
	Url       string
	Type      string
	UpdatedAt time.Time
}

// BuildLog 蓝盾构建日志查询结果
type BuildLog struct {
	// Status 日志查询状态
	Status int `mapstructure:"status"`
	// Message 日志查询结果消息
	Message string `mapstructure:"message"`
	// HasMore 是否还有更多日志可拉取
	HasMore bool `mapstructure:"hasMore"`
	// Finished 日志是否已完结（构建结束且日志打印完毕）
	Finished bool `mapstructure:"finished"`
	// Logs 日志行列表
	Logs []LogLine `mapstructure:"logs"`
}

// LogLine 蓝盾构建日志单行
type LogLine struct {
	// LineNo 行号
	LineNo int64 `mapstructure:"lineNo"`
	// Message 日志内容（可能包含 ANSI 颜色码）
	Message string `mapstructure:"message"`
	// Timestamp 时间戳（毫秒）
	Timestamp int64 `mapstructure:"timestamp"`
}

// MaxLineNo 返回本批日志中的最大行号，用于增量拉取
// BKCI 日志接口按行号升序返回，若无日志返回 -1。
func (bl *BuildLog) MaxLineNo() int64 {
	if len(bl.Logs) == 0 {
		return -1
	}
	return bl.Logs[len(bl.Logs)-1].LineNo
}

// IsComplete 返回日志是否已彻底结束，不会再有新日志产出。
func (bl *BuildLog) IsComplete() bool {
	if bl.Status == buildLogStatusEmpty {
		return true
	}
	return bl.Finished && !bl.HasMore
}

// ReachedCurrentTail 返回是否已拉到当前可获取日志的末尾。
func (bl *BuildLog) ReachedCurrentTail() bool {
	return !bl.HasMore
}
