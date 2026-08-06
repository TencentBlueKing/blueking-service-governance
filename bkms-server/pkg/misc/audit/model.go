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

package audit

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// AccessType 访问类型
type AccessType string

const (
	// AccessTypeWeb 网页访问
	AccessTypeWeb AccessType = "web"
	// AccessTypeAPI API 访问
	AccessTypeAPI AccessType = "api"
	// AccessTypeClient 客户端访问
	AccessTypeClient AccessType = "client"
)

// OperationType 操作类型
type OperationType string

const (
	// OperationTypeCreate 创建操作
	OperationTypeCreate OperationType = "create"
	// OperationTypeUpdate 更新操作
	OperationTypeUpdate OperationType = "update"
	// OperationTypeRollback 回滚操作
	OperationTypeRollback OperationType = "rollback"
	// OperationTypeDelete 删除操作
	OperationTypeDelete OperationType = "delete"
	// OperationTypeBuild 构建操作
	OperationTypeBuild OperationType = "build"
	// OperationTypeDeploy 部署操作
	OperationTypeDeploy OperationType = "deploy"
	// OperationTypeUninstall 卸载操作
	OperationTypeUninstall OperationType = "uninstall"
	// OperationTypeScale 扩缩容操作
	OperationTypeScale OperationType = "scale"
	// OperationTypeGray 灰度操作
	OperationTypeGray OperationType = "gray"
	// OperationTypeExecute 执行操作
	OperationTypeExecute OperationType = "execute"
)

// AllOperationTypes 所有操作类型
var AllOperationTypes = []OperationType{
	OperationTypeCreate,
	OperationTypeUpdate,
	OperationTypeRollback,
	OperationTypeDelete,
	OperationTypeBuild,
	OperationTypeDeploy,
	OperationTypeScale,
	OperationTypeGray,
	OperationTypeExecute,
}

// DisplayName 操作类型展示用名称 TODO 国际化
func (t OperationType) DisplayName() string {
	switch t {
	case OperationTypeCreate:
		return "创建"
	case OperationTypeUpdate:
		return "更新"
	case OperationTypeRollback:
		return "回滚"
	case OperationTypeDelete:
		return "删除"
	case OperationTypeBuild:
		return "构建"
	case OperationTypeDeploy:
		return "部署"
	case OperationTypeUninstall:
		return "卸载"
	case OperationTypeScale:
		return "扩缩容"
	case OperationTypeGray:
		return "灰度"
	case OperationTypeExecute:
		return "执行"
	default:
		// 默认返回原始值
		return string(t)
	}
}

// ResourceType 资源类型
type ResourceType string

const (
	// ResourceTypeWorkspace 工作空间
	ResourceTypeWorkspace ResourceType = "workspace"
	// ResourceTypeEnv 环境
	ResourceTypeEnv ResourceType = "env"
	// ResourceTypeApp 应用
	ResourceTypeApp ResourceType = "app"
	// ResourceTypeInstance 实例
	ResourceTypeInstance ResourceType = "instance"
	// ResourceTypeComponentDef 组件定义
	ResourceTypeComponentDef ResourceType = "componentDef"
	// ResourceTypeClusterAddon 集群插件
	ResourceTypeClusterAddon ResourceType = "clusterAddon"
	// ResourceTypePortPool 端口池
	ResourceTypePortPool ResourceType = "portPool"
	// ResourceTypeAlertStrategy 告警策略
	ResourceTypeAlertStrategy ResourceType = "alertStrategy"
)

// AllResourceTypes 所有资源类型
var AllResourceTypes = []ResourceType{
	ResourceTypeWorkspace,
	ResourceTypeEnv,
	ResourceTypeApp,
	ResourceTypeInstance,
	ResourceTypeComponentDef,
	ResourceTypeClusterAddon,
	ResourceTypePortPool,
	ResourceTypeAlertStrategy,
}

// DisplayName 资源类型展示用名称 TODO 国际化
func (t ResourceType) DisplayName() string {
	switch t {
	case ResourceTypeWorkspace:
		return "工作空间"
	case ResourceTypeEnv:
		return "环境"
	case ResourceTypeApp:
		return "应用"
	case ResourceTypeInstance:
		return "应用实例"
	case ResourceTypeComponentDef:
		return "组件定义"
	case ResourceTypeClusterAddon:
		return "集群插件"
	case ResourceTypePortPool:
		return "端口池"
	case ResourceTypeAlertStrategy:
		return "告警策略"
	default:
		// 默认返回原始值
		return string(t)
	}
}

// Attribute 资源属性
type Attribute string

const (
	// AttributeDisplayName 展示用名称
	AttributeDisplayName Attribute = "displayName"
	// AttributeHelmSpec Helm 配置
	AttributeHelmSpec Attribute = "helmSpec"
	// AttributeAppModel 应用模型
	AttributeAppModel Attribute = "appModel"
	// AttributeAppComponents 应用组件
	AttributeAppComponents Attribute = "appComponents"
	// AttributeBuildConfig 构建配置
	AttributeBuildConfig Attribute = "buildConfig"
	// AttributeUserRole 空间下的角色
	AttributeUserRole Attribute = "userRole"
	// AttributeWorkspaceComponent 空间组件
	AttributeWorkspaceComponent Attribute = "workspaceComponent"
	// AttributeReplicas 副本数
	AttributeReplicas Attribute = "replicas"
	// AttributeInstance 应用实例
	AttributeInstance Attribute = "instance"
	// AttributeWebConsole Web 控制台
	AttributeWebConsole Attribute = "webConsole"
	// AttributePortForward 端口转发
	AttributePortForward Attribute = "portForward"
	// AttributeAdminCommand 管理命令
	AttributeAdminCommand Attribute = "adminCommand"
	// AttributeEnvVars 环境变量
	AttributeEnvVars Attribute = "envVars"
	// AttributeHelmChart Helm Chart
	AttributeHelmChart Attribute = "helmChart"
	// AttributeScopedEnvVars 作用域环境变量
	AttributeScopedEnvVars Attribute = "scopedEnvVars"
	// AttributePolaris 北极星
	AttributePolaris Attribute = "polaris"
	// AttributeGPA 自动扩缩容（GPA）配置
	AttributeGPA Attribute = "gpa"
)

// DisplayName 属性展示用名称 TODO 国际化
func (t Attribute) DisplayName() string {
	switch t {
	case AttributeDisplayName:
		return "展示名称"
	case AttributeHelmSpec:
		return "Helm 配置"
	case AttributeAppModel:
		return "应用模型"
	case AttributeBuildConfig:
		return "构建配置"
	case AttributeUserRole:
		return "空间角色"
	case AttributeAppComponents:
		return "应用组件"
	case AttributeWorkspaceComponent:
		return "空间组件"
	case AttributeReplicas:
		return "副本数"
	case AttributeInstance:
		return "应用实例"
	case AttributeWebConsole:
		return "WebConsole"
	case AttributePortForward:
		return "端口转发"
	case AttributeAdminCommand:
		return "管理命令"
	case AttributeEnvVars:
		return "环境变量"
	case AttributeHelmChart:
		return "Helm Chart"
	case AttributeScopedEnvVars:
		return "作用域环境变量"
	case AttributePolaris:
		return "北极星"
	case AttributeGPA:
		return "自动扩缩容配置"
	default:
		// 默认返回原始值
		return string(t)
	}
}

// Result 操作结果
type Result string

const (
	// ResultSuccess 操作成功
	ResultSuccess Result = "success"
	// ResultFailed 操作失败
	ResultFailed Result = "failed"
)

// AllOperationResults 所有操作结果
var AllOperationResults = []Result{ResultSuccess, ResultFailed}

// DisplayName 操作结果展示用名称 TODO 国际化
func (r Result) DisplayName() string {
	switch r {
	case ResultSuccess:
		return "成功"
	case ResultFailed:
		return "失败"
	default:
		// 默认返回原始值
		return string(r)
	}
}

// OperationData 操作数据
type OperationData struct {
	// Before 操作前数据（如果是 create 操作，该字段为空）
	Before []byte `bson:"before,omitempty"`
	// After 操作后数据（如果是 delete 操作，该字段为空）
	After []byte `bson:"after,omitempty"`
	// Extras 额外信息（不推荐使用，除非真的有必要）
	Extras []byte `bson:"extras,omitempty"`
}

// OperationGroup 操作分组，用于全局过滤查询
type OperationGroup struct {
	// WorkspaceID 工作空间 ID
	WorkspaceID string `bson:"workspaceID,omitempty"`
	// AppID 应用 ID
	AppID string `bson:"appID,omitempty"`
	// EnvName 环境名称，需结合 workspaceID 一起使用
	EnvName string `bson:"envName,omitempty"`
}

// OperationRecord 操作记录
type OperationRecord struct {
	// ID 操作记录 ID
	ID bson.ObjectID `bson:"_id"`
	// Username 操作人
	Username string `bson:"username"`
	// AccessType 访问类型（如：web, api, client），默认为 web
	AccessType AccessType `bson:"accessType"`
	// OperationType 操作类型（如：create, update, delete）
	OperationType OperationType `bson:"operationType"`
	// ResourceType 操作资源（如：workspace, env, app）
	ResourceType ResourceType `bson:"resourceType"`
	// ResourceID 操作资源 ID（如：workspaceID, envID, appID）
	ResourceID string `bson:"resourceID"`
	// Attribute 资源属性，用于更具体描述操作资源的某一部分的信息
	Attribute Attribute `bson:"attribute"`
	// Result 操作结果（如：success, failed, running, canceled），默认为 success
	Result Result `bson:"result"`
	// Data 操作数据，包含操作前后的数据和额外信息
	Data OperationData `bson:"data"`
	// Group 分组字段，用于全局过滤查询
	Group OperationGroup `bson:"group"`
	// CreatedAt 创建时间（即操作时间）
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间（可能为操作结束时间）
	UpdatedAt time.Time `bson:"updatedAt"`
}

// newOperationRecord 新建操作记录
func newOperationRecord(
	username string,
	opType OperationType,
	resType ResourceType,
	resID string,
	opts ...Option,
) *OperationRecord {
	now := time.Now()
	r := OperationRecord{
		ID:            bson.NewObjectID(),
		Username:      username,
		AccessType:    AccessTypeWeb,
		OperationType: opType,
		ResourceType:  resType,
		ResourceID:    resID,
		Attribute:     "",
		Result:        ResultSuccess,
		Data: OperationData{
			Before: nil,
			After:  nil,
			Extras: nil,
		},
		Group:     OperationGroup{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	// 设置选项
	for _, opt := range opts {
		opt(&r)
	}
	return &r
}

// Option 操作记录选项
type Option func(*OperationRecord)

// WithAccessType 指定访问类型（默认为：web）
func WithAccessType(accessType AccessType) Option {
	return func(r *OperationRecord) {
		r.AccessType = accessType
	}
}

// WithAttribute 指定资源属性
func WithAttribute(attribute Attribute) Option {
	return func(r *OperationRecord) {
		r.Attribute = attribute
	}
}

// WithResult 指定操作结果（默认为：success）
func WithResult(result Result) Option {
	return func(r *OperationRecord) {
		r.Result = result
	}
}

// WithDataBefore 添加操作前数据
func WithDataBefore(data any) Option {
	return func(r *OperationRecord) {
		bytes, err := json.Marshal(data)
		if err != nil {
			log.ErrorNoContextf("marshal dataBefore %+v failed: %v", data, err)
			return
		}
		r.Data.Before = bytes
	}
}

// WithDataAfter 添加操作后数据
func WithDataAfter(data any) Option {
	return func(r *OperationRecord) {
		bytes, err := json.Marshal(data)
		if err != nil {
			log.ErrorNoContextf("marshal dataAfter %+v failed: %v", data, err)
			return
		}
		r.Data.After = bytes
	}
}

// WithExtras 添加额外信息
func WithExtras(extras any) Option {
	return func(r *OperationRecord) {
		bytes, err := json.Marshal(extras)
		if err != nil {
			log.ErrorNoContextf("marshal extras %+v failed: %v", extras, err)
			return
		}
		r.Data.Extras = bytes
	}
}

// WithGroup 设置分组信息
func WithGroup(group OperationGroup) Option {
	return func(r *OperationRecord) {
		r.Group = group
	}
}

// WithWorkspaceID 设置工作空间 ID
func WithWorkspaceID(workspaceID string) Option {
	return func(r *OperationRecord) {
		r.Group.WorkspaceID = workspaceID
	}
}

// WithAppID 设置应用 ID
func WithAppID(appID string) Option {
	return func(r *OperationRecord) {
		r.Group.AppID = appID
	}
}

// WithEnvName 设置环境名称
func WithEnvName(envName string) Option {
	return func(r *OperationRecord) {
		r.Group.EnvName = envName
	}
}
