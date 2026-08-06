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

// Package serializer 定义 Helm Chart Gin API 的输入和输出结构。
package serializer

import (
	"regexp"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"

	helmbuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart/semver"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
	helmrepo "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/source"
)

var appIDPattern = regexp.MustCompile("^[a-z][a-z0-9-]*$")

// init 注册 Helm Chart Gin serializer 使用的字段本地校验器。
func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("app_id", validateAppID); err != nil {
			panic("failed to register app_id validator: " + err.Error())
		}
	}
}

// validateAppID 检查路径参数 appID 是否符合应用 API 统一格式。
func validateAppID(fl validator.FieldLevel) bool {
	return appIDPattern.MatchString(fl.Field().String())
}

// AppURIInput 是按应用维度访问 API 的路径参数。
type AppURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
}

// ChartVersionURIInput 是按应用和 Chart 版本维度访问 API 的路径参数。
type ChartVersionURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// Chart 版本号
	ChartVersion string `uri:"chartVersion" binding:"required"`
}

// CreateHelmChartBuildInput 是触发 Helm Chart 构建的 JSON 输入。
type CreateHelmChartBuildInput struct {
	// semver 递增段类型（默认 patch）
	// 经典归零语义：递增 major 时 minor+patch 归零，递增 minor 时 patch 归零
	BumpType string `json:"bumpType" binding:"required,oneof=patch minor major"`
	// Git 分支名称
	Branch string `json:"branch" binding:"required,min=1"`
}

// GetHelmChartSemverQueryInput 是查询 Helm Chart semver 和预览下一个版本的 query 输入。
type GetHelmChartSemverQueryInput struct {
	// semver 递增段类型，非空时在响应中返回 next（预览值，不影响数据库）
	// 经典归零语义：递增 major 时 minor+patch 归零，递增 minor 时 patch 归零
	// 如果不提供 bumpType，则返回值中 next 为空
	BumpType string `form:"bumpType" binding:"omitempty,oneof=patch minor major"`
}

// ListQueryInput 是 Helm Chart 制品列表和构建记录列表共用的分页 query 输入。
type ListQueryInput struct {
	// 搜索关键字（制品列表按版本号模糊匹配；构建记录列表按版本号 / 构建号 / 操作人模糊匹配）
	Keyword string `form:"keyword"`
	// 分页页码（从 1 开始）
	Page int64 `form:"page" binding:"required,gte=1"`
	// 分页大小
	PageSize int64 `form:"pageSize" binding:"required,oneof=5 10 20 50 100"`
}

// CreateHelmChartBuildOutputObj 是触发 Helm Chart 构建后的输出对象。
type CreateHelmChartBuildOutputObj struct {
	// 本次构建的 Chart 版本号（semver 格式：major.minor.patch）
	ChartVersion string `json:"chartVersion"`
	// 蓝盾构建 ID（预留查询能力）
	BuildID string `json:"buildID"`
}

// CreateHelmChartBuildOutput 是触发 Helm Chart 构建的 JSON 响应。
type CreateHelmChartBuildOutput struct {
	// 触发 Helm Chart 构建 - 输出对象
	Data *CreateHelmChartBuildOutputObj `json:"data"`
}

// SemverOutputObj 是语义化版本计数器的 JSON 表示。
type SemverOutputObj struct {
	// 主版本号
	Major int64 `json:"major,string"`
	// 次版本号
	Minor int64 `json:"minor,string"`
	// 修订版本号
	Patch int64 `json:"patch,string"`
	// 格式化版本字符串（格式：major.minor.patch）
	Version string `json:"version"`
}

// FromCounter 将 semver counter 转换为输出对象。
func (o *SemverOutputObj) FromCounter(c *semver.Counter) *SemverOutputObj {
	*o = SemverOutputObj{
		Major:   c.Major,
		Minor:   c.Minor,
		Patch:   c.Patch,
		Version: c.FormatSemver(),
	}
	return o
}

// GetHelmChartSemverOutputObj 是查询 Helm Chart semver counter 当前值的输出对象。
type GetHelmChartSemverOutputObj struct {
	// 当前最新的 semver 值
	Latest *SemverOutputObj `json:"latest"`
	// 按 bumpType 递增后的下一个 semver 值（仅当请求中 bumpType 非空时返回）
	Next *SemverOutputObj `json:"next"`
}

// GetHelmChartSemverOutput 是查询 Helm Chart semver counter 当前值的 JSON 响应。
type GetHelmChartSemverOutput struct {
	// 查询 Helm Chart semver counter 当前值 - 输出对象
	Data *GetHelmChartSemverOutputObj `json:"data"`
}

// DeployedEnvInfo 是 Chart 版本已部署到的一个环境。
type DeployedEnvInfo struct {
	// envName 环境名称
	EnvName string `json:"envName"`
	// envType 环境类型
	EnvType string `json:"envType"`
}

// AppHelmChartOutputObj 是 Helm Chart 制品列表中的一条记录。
type AppHelmChartOutputObj struct {
	// 版本号（semver）
	ChartVersion string `json:"chartVersion"`
	// 制品产生时间（来自 Helm Repo index entry 的 created 字段）
	CreatedAt time.Time `json:"createdAt"`
	// Chart 产物摘要（来自 Helm Repo index entry 的 digest 字段）
	Digest string `json:"digest"`
	// 已部署到的环境列表
	DeployedEnvs []*DeployedEnvInfo `json:"deployedEnvs"`
}

// FromChartEntry 将 Helm Repo index 中的 Chart entry 转换为制品输出对象。
func (o *AppHelmChartOutputObj) FromChartEntry(
	entry helmrepo.ChartEntry,
	deployedEnvs []*DeployedEnvInfo,
) *AppHelmChartOutputObj {
	if deployedEnvs == nil {
		deployedEnvs = make([]*DeployedEnvInfo, 0)
	}
	*o = AppHelmChartOutputObj{
		ChartVersion: entry.Version,
		CreatedAt:    entry.Created,
		Digest:       entry.Digest,
		DeployedEnvs: deployedEnvs,
	}
	return o
}

// PaginatedAppHelmChartsOutputObjs 是分页 Helm Chart 制品列表。
type PaginatedAppHelmChartsOutputObjs struct {
	// 总记录数（去重后的版本数）
	Count int64 `json:"count,string"`
	// 当前页 Chart 制品列表
	Results []*AppHelmChartOutputObj `json:"results"`
}

// ListAppHelmChartsOutput 是获取 Helm Chart 制品列表的 JSON 响应。
type ListAppHelmChartsOutput struct {
	// 分页 Helm Chart 制品列表
	Data *PaginatedAppHelmChartsOutputObjs `json:"data"`
}

// HelmChartBuildRecordOutputObj 是 Helm Chart 构建记录列表中的一条记录。
type HelmChartBuildRecordOutputObj struct {
	// 构建序号（每个 AppID 独立自增）
	Num int64 `json:"num,string"`
	// 蓝盾流水线 ID
	PipelineID string `json:"pipelineID"`
	// 蓝盾构建 ID
	BuildID string `json:"buildID"`
	// 本次构建产出的 Chart 版本号
	ChartVersion string `json:"chartVersion"`
	// 构建状态
	Status string `json:"status"`
	// 触发人
	Operator string `json:"operator"`
	// 构建参数（包含代码库、分支等信息）
	Params map[string]string `json:"params"`
	// 构建额外信息（包含 commit ID 等，由轮询任务回写）
	Extras map[string]string `json:"extras"`
	// 构建开始时间
	StartedAt time.Time `json:"startedAt"`
	// 构建结束时间
	EndedAt *time.Time `json:"endedAt,omitempty"`
}

// FromBuildRecord 将 Helm Chart 构建记录模型转换为输出对象。
func (o *HelmChartBuildRecordOutputObj) FromBuildRecord(record helmbuild.Record) *HelmChartBuildRecordOutputObj {
	*o = HelmChartBuildRecordOutputObj{
		Num:          record.Num,
		PipelineID:   record.PipelineID,
		BuildID:      record.BuildID,
		ChartVersion: record.ChartVersion,
		Status:       string(record.Status),
		Operator:     record.Operator,
		Params:       record.Params,
		Extras:       record.Extras,
		StartedAt:    record.StartedAt,
		EndedAt:      record.EndedAt,
	}
	return o
}

// PaginatedHelmChartBuildRecordOutputObjs 是分页 Helm Chart 构建记录列表。
type PaginatedHelmChartBuildRecordOutputObjs struct {
	// 总记录数
	Count int64 `json:"count,string"`
	// 当前页构建记录列表
	Results []*HelmChartBuildRecordOutputObj `json:"results"`
}

// ListHelmChartBuildRecordsOutput 是获取 Helm Chart 构建记录列表的 JSON 响应。
type ListHelmChartBuildRecordsOutput struct {
	// 分页 Helm Chart 构建记录列表
	Data *PaginatedHelmChartBuildRecordOutputObjs `json:"data"`
}

// HelmChartFileNode 是 Helm Chart 文件树中的一个节点。
type HelmChartFileNode struct {
	// 节点名称（不含父级路径）
	Name string `json:"name"`
	// 节点相对路径（相对于 chart 根目录）
	Path string `json:"path"`
	// 是否为目录
	IsDir bool `json:"isDir"`
	// 文件大小（字节，仅文件有效）
	Size int64 `json:"size,string"`
	// 是否为二进制文件（true 时不返回 content）
	IsBinary bool `json:"isBinary"`
	// 文件内容（仅文本文件且大小未超限时返回 UTF-8 文本）
	Content string `json:"content"`
	// 子节点（仅目录有效）
	Children []*HelmChartFileNode `json:"children"`
}

// FromRepoFileNode 将 Helm Repo 文件树节点递归转换为输出对象。
func (o *HelmChartFileNode) FromRepoFileNode(node *helmrepo.FileNode) *HelmChartFileNode {
	if node == nil {
		return nil
	}
	*o = HelmChartFileNode{
		Name:     node.Name,
		Path:     node.Path,
		IsDir:    node.IsDir,
		Size:     node.Size,
		IsBinary: node.IsBinary,
	}
	if !node.IsBinary && len(node.Content) > 0 {
		o.Content = string(node.Content)
	}
	o.Children = lo.Map(node.Children, func(child *helmrepo.FileNode, _ int) *HelmChartFileNode {
		return new(HelmChartFileNode).FromRepoFileNode(child)
	})
	return o
}

// GetHelmChartFilesOutputObj 是获取 Helm Chart 文件树的输出对象。
type GetHelmChartFilesOutputObj struct {
	// Chart 名称
	ChartName string `json:"chartName"`
	// Chart 版本号
	ChartVersion string `json:"chartVersion"`
	// Chart 根目录节点
	Root *HelmChartFileNode `json:"root"`
}

// GetHelmChartFilesOutput 是获取 Helm Chart 文件树的 JSON 响应。
type GetHelmChartFilesOutput struct {
	// Chart 文件输出对象
	Data *GetHelmChartFilesOutputObj `json:"data"`
}

// ChartVersionOutputObj 是 Helm Chart 版本列表中的一条记录。
type ChartVersionOutputObj struct {
	// Helm Chart 版本名
	Name string `json:"name"`
	// 版本创建时间
	CreatedAt string `json:"createdAt"`
	// 版本更新时间
	UpdatedAt string `json:"updatedAt"`
}

// FromRepoVersion 将 Helm Repo 版本信息转换为输出对象。
func (o *ChartVersionOutputObj) FromRepoVersion(version helmrepo.Version) *ChartVersionOutputObj {
	*o = ChartVersionOutputObj{
		Name:      version.Name,
		CreatedAt: version.CreatedAt,
		UpdatedAt: version.UpdatedAt,
	}
	return o
}

// ListChartVersionsOutput 是获取 Helm Chart 版本列表的 JSON 响应。
type ListChartVersionsOutput struct {
	// Helm Chart 版本列表
	Data []*ChartVersionOutputObj `json:"data"`
}

// GetValuesFileOutput 是获取 Helm Chart Values 文件的 JSON 响应。
type GetValuesFileOutput struct {
	// 应用配置文件内容
	Data string `json:"data"`
}
