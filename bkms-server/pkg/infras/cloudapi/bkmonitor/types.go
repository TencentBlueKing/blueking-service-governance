// Package bkmonitor api client，如：蓝鲸监控的 apm、蓝鲸监控的 metadata
package bkmonitor

import (
	"fmt"

	"github.com/pkg/errors"
)

// SpaceUIDFormat 空间 UID 格式
const SpaceUIDFormat = "bkci__%s"

var (
	// ErrApmAppDuplicate APM 中 app 已经存在
	ErrApmAppDuplicate = errors.New("APM app duplicate name error")

	// ErrApmAppNotFound APM 的 app 不存在
	ErrApmAppNotFound = errors.New("APM app does not exist")

	// ErrSpaceNotFound 空间不存在
	ErrSpaceNotFound = errors.New("space not found")
)

// OTLPConfig OTLP 配置项
type OTLPConfig struct {
	// BkDataID 数据 ID
	BkDataID float64 `json:"bk_data_id" mapstructure:"bk_data_id"`
}

// ApmApp APM 应用
type ApmApp struct {
	// ID app id
	ID int64 `json:"id" mapstructure:"id"`
	// Token 口令
	Token string `json:"token" mapstructure:"token"`
	// BkBizID bkmonitor 划定的业务 id
	BkBizID int64 `json:"bk_biz_id" mapstructure:"bk_biz_id"`
	// AppName 名称(最大长度50)
	AppName string `json:"app_name" mapstructure:"app_name"`
	// Description 备注
	Description string `json:"description" mapstructure:"description"`

	// MetricConfig metric
	// 如果字段不为空，且对应的数据 ID 也不为空 0，则该功能生效，也意味着 APM 侧资源已经就绪；否则，不生效；下面三个字段也是一样；
	MetricConfig *OTLPConfig `json:"metric_config" mapstructure:"metric_config"`
	// TraceConfig trace
	TraceConfig *OTLPConfig `json:"trace_config" mapstructure:"trace_config"`
	// LogConfig log
	LogConfig *OTLPConfig `json:"log_config" mapstructure:"log_config"`
	// ProfilingConfig profiling
	ProfilingConfig *OTLPConfig `json:"profiling_config" mapstructure:"profiling_config"`

	// Creator 创建用户
	Creator string `json:"create_user" mapstructure:"create_user"`
	// CreatedAt 创建时间
	CreatedAt string `json:"create_time" mapstructure:"create_time"`
}

// Space 空间
type Space struct {
	// ID id
	ID int64 `json:"id" mapstructure:"id"`
	// SpaceTypeID 空间类型
	SpaceTypeID string `json:"space_type_id" mapstructure:"space_type_id"`
	// SpaceID 对应 bcs.projectCode
	SpaceID string `json:"space_id" mapstructure:"space_id"`
	// SpaceCode 对应 bcs.projectID
	SpaceCode string `json:"space_code" mapstructure:"space_code"`
	// SpaceName 对应 project.name
	SpaceName string `json:"space_name" mapstructure:"space_name"`
	// SpaceUid bkci__ + bcs.projectCode or bkci__ + SpaceTypeID
	SpaceUid string `json:"space_uid" mapstructure:"space_uid"`
	// IsBcsValid 是否有效
	IsBcsValid bool `json:"is_bcs_valid" mapstructure:"is_bcs_valid"`

	// Status 容器项目状态
	Status string `json:"status" mapstructure:"status"`
	// Creator 创建人
	Creator string `json:"creator" mapstructure:"creator"`
	// CreatedAt 创建时间
	CreatedAt string `json:"create_time" mapstructure:"create_time"`
}

// CreateApmAppReq 创建 APM 应用请求
type CreateApmAppReq struct {
	// SpaceUID 空间 uid
	SpaceUID string `json:"space_uid" validate:"required"`
	// AppName 名称
	AppName string `json:"app_name" validate:"required"`
	// BkBizID 蓝鲸监控下项目的业务 ID
	// 注意：
	//	容器项目 ID 是负数，这个是蓝鲸监控特殊规则
	BkBizID int64 `json:"bk_biz_id" validate:"lt=0"`
	// Operator 操作人
	Operator string `json:"-" validate:"required"`

	// AppAlias 别名
	AppAlias string `json:"app_alias"`
	// Description 备注
	Description string `json:"description"`
	// EnabledLog log
	EnabledLog bool `json:"enabled_log"`
	// EnabledMetric metric
	EnabledMetric bool `json:"enabled_metric"`
	// EnabledTrace trace
	EnabledTrace bool `json:"enabled_trace"`
	// EnabledProfiling profiling
	EnabledProfiling bool `json:"enabled_profiling"`
}

// NewDefaultCreateApmAppReq 创建默认的 APM 应用创建请求
func NewDefaultCreateApmAppReq(bkBizID int64, projectCode, name, description, operator string) *CreateApmAppReq {
	if bkBizID > 0 {
		bkBizID = -bkBizID
	}

	return &CreateApmAppReq{
		AppName:          name,
		AppAlias:         name,
		Description:      description,
		BkBizID:          bkBizID,
		EnabledLog:       true,
		EnabledMetric:    true,
		EnabledTrace:     true,
		EnabledProfiling: false,
		Operator:         operator,
		SpaceUID:         fmt.Sprintf(SpaceUIDFormat, projectCode),
	}
}

// GetApmAppReq 获取 APM 应用请求
type GetApmAppReq struct {
	// AppName 名称（与 ApmAppID 二选一）
	// envName
	AppName string `json:"app_name"`

	// ApmAppID（与 AppName 二选一）
	// apm app ID
	ApmAppID int64 `json:"application_id"`

	// BkBizID 蓝鲸监控下项目的业务 ID
	// 注意：
	//	容器项目 ID 是负数，这个是蓝鲸监控特殊规则
	BkBizID int64 `json:"bk_biz_id" validate:"lt=0"`
}

// NewGetApmAppReq 通过应用名称创建获取 APM 应用请求
func NewGetApmAppReq(bkBizID, apmAppID int64, appName string) *GetApmAppReq {
	if bkBizID > 0 {
		bkBizID = -bkBizID
	}

	return &GetApmAppReq{
		BkBizID:  bkBizID,
		ApmAppID: apmAppID,
		AppName:  appName,
	}
}

// ListApmAppReq 列出 APM 应用请求
type ListApmAppReq struct {
	// BkBizID 蓝鲸监控下项目的业务 ID
	// 注意：
	//	容器项目 ID 是负数，这个是蓝鲸监控特殊规则
	BkBizID int64 `json:"bk_biz_id" validate:"lt=0"`
}

// NewListApmAppReq 创建列出 APM 应用请求
// 注意：
//
//	蓝鲸监控的 容器项目 ID 必须是负数，这个是蓝鲸监控特殊规则
func NewListApmAppReq(bkBizID int64) *ListApmAppReq {
	if bkBizID > 0 {
		bkBizID = -bkBizID
	}

	return &ListApmAppReq{
		BkBizID: bkBizID,
	}
}

// UserGroupUser 告警组通知接收人员
type UserGroupUser struct {
	// ID 角色 key 或者用户 ID
	ID string `json:"id" mapstructure:"id"`
	// DisplayName 显示名
	DisplayName string `json:"display_name" mapstructure:"display_name"`
	// Type 类型，可选项 group、user
	Type string `json:"type" mapstructure:"type"`
	// Members 对应的人员信息（针对 group 类型）
	Members []map[string]any `json:"members" mapstructure:"members"`
}

// UserGroup 告警组（列表项）
type UserGroup struct {
	// ID 告警组 ID
	ID int64 `json:"id" mapstructure:"id"`
	// BkBizID 业务 ID
	BkBizID int64 `json:"bk_biz_id" mapstructure:"bk_biz_id"`
	// Name 名称
	Name string `json:"name" mapstructure:"name"`
	// NeedDuty 是否轮值
	NeedDuty bool `json:"need_duty" mapstructure:"need_duty"`
	// Channels 通知渠道，可选项 user(内部用户)、wxwork-bot(企业微信机器人)
	Channels []string `json:"channels" mapstructure:"channels"`
	// Desc 说明
	Desc string `json:"desc" mapstructure:"desc"`
	// Users 通知接收人员
	Users []UserGroupUser `json:"users" mapstructure:"users"`
	// StrategyCount 关联的告警策略数量
	StrategyCount int64 `json:"strategy_count" mapstructure:"strategy_count"`
	// DeleteAllowed 是否可删除
	DeleteAllowed bool `json:"delete_allowed" mapstructure:"delete_allowed"`
	// EditAllowed 是否可编辑
	EditAllowed bool `json:"edit_allowed" mapstructure:"edit_allowed"`
	// ConfigSource 配置来源
	ConfigSource string `json:"config_source" mapstructure:"config_source"`
	// Timezone 时区
	Timezone string `json:"timezone" mapstructure:"timezone"`
	// MentionList 提及人列表（企业微信机器人等渠道会 @ 这些人）
	MentionList []UserGroupUser `json:"mention_list" mapstructure:"mention_list"`
	// MentionType 提及类型：0-不提及，1-提及全部，2-按 MentionList 提及
	MentionType int64 `json:"mention_type" mapstructure:"mention_type"`
}

// NoticeWay 通知方式
type NoticeWay struct {
	// Name 通知方式名称，如 weixin、sms、voice、wxwork-bot
	Name string `json:"name" mapstructure:"name"`
	// Receivers 通知接收人员：企业微信机器人为 chatID，bkchat 为对应的选项 ID
	Receivers []string `json:"receivers" mapstructure:"receivers"`
}

// AlertNoticeConfig 告警通知配置项
type AlertNoticeConfig struct {
	// Level 告警级别：1(致命)，2(预警)，3(提醒)
	Level int64 `json:"level" mapstructure:"level"`
	// Type 通知场景类型列表（如 normal / ack / resolved / closed，空数组表示全部）
	Type []string `json:"type" mapstructure:"type"`
	// NoticeWays 通知方式
	NoticeWays []NoticeWay `json:"notice_ways" mapstructure:"notice_ways"`
}

// ActionNoticeConfig 告警处理通知配置项
type ActionNoticeConfig struct {
	// Phase 阶段：1(失败时)，2(成功时)，3(执行前)
	Phase int64 `json:"phase" mapstructure:"phase"`
	// Type 通知场景类型列表（如 normal / ack / resolved / closed，空数组表示全部）
	Type []string `json:"type" mapstructure:"type"`
	// NoticeWays 通知方式
	NoticeWays []NoticeWay `json:"notice_ways" mapstructure:"notice_ways"`
}

// AlertNotice 告警通知方式
type AlertNotice struct {
	// TimeRange 生效时间范围
	TimeRange string `json:"time_range" mapstructure:"time_range"`
	// NotifyConfig 通知配置
	NotifyConfig []AlertNoticeConfig `json:"notify_config" mapstructure:"notify_config"`
}

// ActionNotice 告警处理通知配置
type ActionNotice struct {
	// TimeRange 生效时间范围
	TimeRange string `json:"time_range" mapstructure:"time_range"`
	// NotifyConfig 通知配置
	NotifyConfig []ActionNoticeConfig `json:"notify_config" mapstructure:"notify_config"`
}

// DutyArrange 轮值安排
type DutyArrange struct {
	// ID 轮值 ID
	ID int64 `json:"id" mapstructure:"id"`
	// Hash 原始配置摘要
	Hash string `json:"hash" mapstructure:"hash"`
	// GroupType 分组类型，可选项 specified(指定), auto(自动)
	GroupType string `json:"group_type" mapstructure:"group_type"`
	// GroupNumber 	自动分组时每个班次对应的人数
	GroupNumber int64 `json:"group_number" mapstructure:"group_number"`
	// DutyRuleID 轮值规则ID。
	// 使用指针是为了：无值时序列化为 null（bkmonitor 读接口常回传 null，
	// 写接口也要求未设置时传 null 而非 0）。
	DutyRuleID *int64 `json:"duty_rule_id" mapstructure:"duty_rule_id"`

	// UserGroupID 告警组 ID
	UserGroupID int64 `json:"user_group_id" mapstructure:"user_group_id"`
	// NeedRotation 是否需要交接班
	NeedRotation bool `json:"need_rotation" mapstructure:"need_rotation"`
	// DutyTime 轮班时间安排
	DutyTime []map[string]any `json:"duty_time" mapstructure:"duty_time"`
	// EffectiveTime 生效时间。
	// 使用指针是为了：无值时序列化为 null（bkmonitor 读接口可能回传 null 或空串，
	// 写接口未设置时需要传 null）。
	EffectiveTime *string `json:"effective_time" mapstructure:"effective_time"`
	// HandoffTime 交接班时间配置
	HandoffTime map[string]any `json:"handoff_time" mapstructure:"handoff_time"`
	// DutyUsers 值班人员组
	DutyUsers [][]UserGroupUser `json:"duty_users" mapstructure:"duty_users"`
	// Users 值班人员（兼容老接口，不需要轮值的时候可以保留该字段）
	Users []UserGroupUser `json:"users" mapstructure:"users"`
	// Backups 备份人员
	Backups []map[string]any `json:"backups" mapstructure:"backups"`
	// Order 排序
	Order int64 `json:"order" mapstructure:"order"`
}

// DutyNotice 轮值通知设置
type DutyNotice struct {
	// HitFirstDuty 是否命中首班。
	// 使用指针 + omitempty 是为了：未配置时序列化时完全省略该字段，
	// 避免 bkmonitor 写接口在未设置时收到 false 产生歧义。
	HitFirstDuty *bool `json:"hit_first_duty,omitempty" mapstructure:"hit_first_duty"`
	// PlanNotice 轮值计划通知配置，未配置时不展示该字段
	PlanNotice map[string]any `json:"plan_notice,omitempty" mapstructure:"plan_notice"`
	// PersonalNotice 值班人员通知配置，未配置时不展示该字段
	PersonalNotice map[string]any `json:"personal_notice,omitempty" mapstructure:"personal_notice"`
}

// UserGroupDetail 告警组详情
type UserGroupDetail struct {
	UserGroup `json:",inline" mapstructure:",squash"`
	// DutyRules 轮值规则
	DutyRules []int64 `json:"duty_rules" mapstructure:"duty_rules"`
	// DutyArranges 轮值安排
	DutyArranges []DutyArrange `json:"duty_arranges" mapstructure:"duty_arranges"`
	// AlertNotice 告警通知方式
	AlertNotice []AlertNotice `json:"alert_notice" mapstructure:"alert_notice"`
	// ActionNotice 告警处理通知配置
	ActionNotice []ActionNotice `json:"action_notice" mapstructure:"action_notice"`
	// DutyNotice 轮值通知设置
	DutyNotice *DutyNotice `json:"duty_notice" mapstructure:"duty_notice"`
	// Path 业务路径
	Path string `json:"path" mapstructure:"path"`
}

// SearchUserGroupsReq 查询告警组列表请求
type SearchUserGroupsReq struct {
	// BkBizIDs 业务 ID 列表
	BkBizIDs []int64 `json:"bk_biz_ids,omitempty" validate:"required"`

	// IDs 通知组 ID 列表
	IDs []int64 `json:"ids,omitempty"`
	// Name 通知组名称
	Name string `json:"name,omitempty"`
}

// SearchUserGroupDetailReq 查询告警组详情请求
type SearchUserGroupDetailReq struct {
	// ID 通知组 ID
	ID int64 `json:"id" validate:"gt=0"`
}

// SaveUserGroupReq 保存告警组请求
type SaveUserGroupReq struct {
	// ID 告警组 ID（没有表示新建）
	ID *int64 `json:"id,omitempty"`
	// BkBizID 业务 ID
	BkBizID int64 `json:"bk_biz_id" validate:"required"`
	// Name 名称
	Name string `json:"name" validate:"required"`
	// Timezone 时区，默认 utc
	Timezone string `json:"timezone"`
	// NeedDuty 是否轮值
	NeedDuty bool `json:"need_duty"`
	// Channels 通知渠道，可选项 user(内部用户)、wxwork-bot(企业微信机器人)
	Channels []string `json:"channels" validate:"required,min=1"`
	// Desc 说明
	Desc string `json:"desc"`
	// AlertNotice 告警通知方式
	AlertNotice []AlertNotice `json:"alert_notice" validate:"required,min=1"`
	// ActionNotice 告警处理通知配置
	ActionNotice []ActionNotice `json:"action_notice" validate:"required,min=1"`
	// Operator 操作人
	Operator string `json:"-" validate:"required"`

	// DutyArranges 非轮值情况下通知接收人员
	DutyArranges []DutyArrange `json:"duty_arranges,omitempty"`
	// DutyRules 轮值对应的规则组（need_duty 情况下必填）
	DutyRules []int64 `json:"duty_rules,omitempty"`
	// DutyNotice 轮值相关的通知设置
	DutyNotice *DutyNotice `json:"duty_notice,omitempty"`
	// MentionList 提及人列表（企业微信机器人等渠道会 @ 这些人）
	MentionList []UserGroupUser `json:"mention_list,omitempty"`
	// MentionType 提及类型：0-不提及，1-提及全部，2-按 MentionList 提及
	MentionType int64 `json:"mention_type,omitempty"`
	// Path 业务路径
	Path string `json:"path" mapstructure:"path"`
}

// DeleteUserGroupReq 删除告警组请求
type DeleteUserGroupReq struct {
	// IDs 告警组 ID 列表
	IDs []int64 `json:"ids" validate:"required,min=1,dive,gt=0"`
	// BkBizIDs 业务 ID 列表
	BkBizIDs []int64 `json:"bk_biz_ids" validate:"required,min=1"`
	// Operator 操作人
	Operator string `json:"-" validate:"required"`
}

// ---- 告警策略（Alarm Strategy）相关类型 ----

// AlarmStrategyItem 告警策略检索返回的列表项（search_alarm_strategy_v3 返回）
type AlarmStrategyItem struct {
	// ID 策略 ID
	ID int64 `json:"id" mapstructure:"id"`
	// BkBizID 业务 ID
	BkBizID int64 `json:"bk_biz_id" mapstructure:"bk_biz_id"`
	// Name 策略名称
	Name string `json:"name" mapstructure:"name"`
	// Source 来源
	Source string `json:"source" mapstructure:"source"`
	// Scenario 监控场景
	Scenario string `json:"scenario" mapstructure:"scenario"`
	// IsEnabled 是否启用
	IsEnabled bool `json:"is_enabled" mapstructure:"is_enabled"`
	// Labels 标签列表
	Labels []string `json:"labels" mapstructure:"labels"`
}

// SearchAlarmStrategyReq search_alarm_strategy_v3 请求
type SearchAlarmStrategyReq struct {
	// BkBizID 业务 ID（必填）
	BkBizID int64 `json:"bk_biz_id" validate:"required"`
	// Conditions 筛选条件
	Conditions []map[string]any `json:"conditions,omitempty"`
	// Scenario 监控场景
	Scenario string `json:"scenario,omitempty"`
	// Page 页码，默认 1
	Page int `json:"page,omitempty"`
	// PageSize 每页数量，默认 10
	PageSize int `json:"page_size,omitempty"`
	// WithNoticeGroup 是否返回告警组信息
	WithNoticeGroup bool `json:"with_notice_group,omitempty"`
}

// SearchAlarmStrategyResp search_alarm_strategy_v3 响应
type SearchAlarmStrategyResp struct {
	// StrategyConfigList 策略列表
	StrategyConfigList []AlarmStrategyItem `json:"strategy_config_list" mapstructure:"strategy_config_list"`
	// Total 总数
	Total int64 `json:"total" mapstructure:"total"`
}

// SaveAlarmStrategyReq save_alarm_strategy_v3 请求体
// 完整的策略配置较为复杂，这里用 map[string]any 保持灵活性，由上层 service 构造完整 payload。
type SaveAlarmStrategyReq struct {
	// BkBizID 业务 ID（必填）
	BkBizID int64 `json:"bk_biz_id" validate:"required"`
	// ID 策略 ID（0 表示新建）
	ID int64 `json:"id,omitempty"`
	// Name 策略名称
	Name string `json:"name" validate:"required"`
	// Source 来源
	Source string `json:"source,omitempty"`
	// Scenario 监控场景（必填）
	Scenario string `json:"scenario" validate:"required"`
	// IsEnabled 是否启用
	IsEnabled bool `json:"is_enabled"`
	// Labels 标签列表
	Labels []string `json:"labels,omitempty"`
	// Items 监控项列表
	Items []map[string]any `json:"items" validate:"required"`
	// Detects 检测算法列表
	Detects []map[string]any `json:"detects" validate:"required"`
	// Actions 处理套餐列表（允许为空列表，不可省略）
	Actions []map[string]any `json:"actions"`
	// Notice 通知配置（必填）
	Notice map[string]any `json:"notice" validate:"required"`
	// Operator 操作人
	Operator string `json:"-" validate:"required"`
}

// SaveAlarmStrategyResp save_alarm_strategy_v3 响应
type SaveAlarmStrategyResp struct {
	// ID 策略 ID
	ID int64 `json:"id" mapstructure:"id"`
	// BkBizID 业务 ID
	BkBizID int64 `json:"bk_biz_id" mapstructure:"bk_biz_id"`
	// Name 策略名称
	Name string `json:"name" mapstructure:"name"`
}

// SwitchAlarmStrategyReq switch_alarm_strategy 请求（批量启停策略）
type SwitchAlarmStrategyReq struct {
	// BkBizID 业务 ID（必填）
	BkBizID int64 `json:"bk_biz_id" validate:"required"`
	// IDs 策略 ID 列表
	IDs []int64 `json:"ids" validate:"required,min=1"`
	// IsEnabled 是否启用
	IsEnabled bool `json:"is_enabled"`
}

// DeleteAlarmStrategyReq delete_alarm_strategy_v3 请求
type DeleteAlarmStrategyReq struct {
	// BkBizID 业务 ID（必填）
	BkBizID int64 `json:"bk_biz_id" validate:"required"`
	// IDs 策略 ID 列表
	IDs []int64 `json:"ids" validate:"required,min=1"`
}

// ---- 告警事件（Alert）相关类型 ----

// AlertEvent search_alert 返回的告警事件
type AlertEvent struct {
	// ID 告警 ID
	ID string `json:"id" mapstructure:"id"`
	// EventID 事件 ID
	EventID string `json:"event_id" mapstructure:"event_id"`
	// AlertName 告警名称
	AlertName string `json:"alert_name" mapstructure:"alert_name"`
	// Assignee 负责人列表
	Assignee []string `json:"assignee" mapstructure:"assignee"`
	// Status 告警状态：ABNORMAL / RECOVERED / CLOSED
	Status string `json:"status" mapstructure:"status"`
	// Severity 告警级别
	Severity int `json:"severity" mapstructure:"severity"`
	// Description 告警描述
	Description string `json:"description" mapstructure:"description"`
	// StrategyID 关联策略 ID
	StrategyID int64 `json:"strategy_id" mapstructure:"strategy_id"`
	// StrategyName 关联策略名称
	StrategyName string `json:"strategy_name" mapstructure:"strategy_name"`
	// TargetType 目标类型
	TargetType string `json:"target_type" mapstructure:"target_type"`
	// Target 目标
	Target string `json:"target" mapstructure:"target"`
	// Dimensions 维度信息
	Dimensions any `json:"dimensions" mapstructure:"dimensions"`
	// CurrentValue 当前值
	CurrentValue any `json:"current_value" mapstructure:"current_value"`
	// DataSource 数据源
	DataSource string `json:"data_source" mapstructure:"data_source"`
	// Content 告警内容
	Content string `json:"content" mapstructure:"content"`
	// Detail 详情内容
	Detail any `json:"detail" mapstructure:"detail"`
	// RelatedInfo 关联信息
	RelatedInfo any `json:"related_info" mapstructure:"related_info"`
	// BeginTime 开始时间（Unix 时间戳）
	BeginTime int64 `json:"begin_time" mapstructure:"begin_time"`
	// EndTime 结束时间（Unix 时间戳）
	EndTime int64 `json:"end_time" mapstructure:"end_time"`
	// LatestTime 最后异常时间
	LatestTime int64 `json:"latest_time" mapstructure:"latest_time"`
	// CreateTime 创建时间
	CreateTime int64 `json:"create_time" mapstructure:"create_time"`
}

// SearchAlertReq search_alert 请求
type SearchAlertReq struct {
	// BkBizIDs 业务 ID 列表（必填）
	BkBizIDs []int64 `json:"bk_biz_ids" validate:"required,min=1"`
	// Status 告警状态过滤：ABNORMAL / RECOVERED / CLOSED
	Status []string `json:"status,omitempty"`
	// Severity 告警级别过滤
	Severity []int `json:"severity,omitempty"`
	// Conditions 额外过滤条件
	Conditions []map[string]any `json:"conditions,omitempty"`
	// QueryString 查询字符串，支持按 description 等字段做文本检索。
	QueryString string `json:"query_string,omitempty"`
	// StartTime 开始时间（Unix 时间戳）
	StartTime int64 `json:"start_time" validate:"required"`
	// EndTime 结束时间（Unix 时间戳）
	EndTime int64 `json:"end_time" validate:"required"`
	// Page 页码
	Page int `json:"page,omitempty"`
	// PageSize 每页数量
	PageSize int `json:"page_size,omitempty"`
	// Ordering 排序字段列表
	Ordering []string `json:"ordering,omitempty"`
}

// AlertDetailReq alert_detail 请求
type AlertDetailReq struct {
	// BkBizID 业务 ID
	BkBizID int64 `json:"-" validate:"required"`
	// ID 告警 ID
	ID string `json:"-" validate:"required"`
}

// SearchAlertResp search_alert 响应
type SearchAlertResp struct {
	// Alerts 告警事件列表
	Alerts []AlertEvent `json:"alerts" mapstructure:"alerts"`
	// Total 总数
	Total int64 `json:"total" mapstructure:"total"`
}

// QueryFunctionParam 查询函数参数
type QueryFunctionParam struct {
	// ID 参数 ID
	ID string `json:"id" mapstructure:"id"`
	// Value 参数值
	Value string `json:"value" mapstructure:"value"`
}

// QueryFunction 查询计算函数
type QueryFunction struct {
	// ID 函数名
	ID string `json:"id" mapstructure:"id"`
	// Params 函数参数列表
	Params []QueryFunctionParam `json:"params" mapstructure:"params"`
}

// WhereCondition 查询过滤条件
type WhereCondition struct {
	// Key 过滤字段
	Key string `json:"key" mapstructure:"key"`
	// Method 过滤方法，如 eq、neq、contains 等
	Method string `json:"method" mapstructure:"method"`
	// Value 过滤值列表
	Value []string `json:"value" mapstructure:"value"`
}

// QueryMetric 查询指标
type QueryMetric struct {
	// Method 聚合方法，如 SUM、AVG、MAX 等
	Method string `json:"method,omitempty" mapstructure:"method"`
	// Field 指标字段名
	Field string `json:"field,omitempty" mapstructure:"field"`
	// Alias 别名
	Alias string `json:"alias,omitempty" mapstructure:"alias"`
	// Display 是否显示
	Display bool `json:"display,omitempty" mapstructure:"display"`
}

// QueryConfig 查询配置
type QueryConfig struct {
	// DataSourceLabel 数据来源（必填），如 bk_monitor、custom 等
	DataSourceLabel string `json:"data_source_label" mapstructure:"data_source_label" validate:"required"`
	// DataTypeLabel 数据类型，默认为 time_series
	DataTypeLabel string `json:"data_type_label,omitempty" mapstructure:"data_type_label"`
	// Table 结果表名
	Table string `json:"table,omitempty" mapstructure:"table"`
	// DataLabel 数据标签
	DataLabel string `json:"data_label,omitempty" mapstructure:"data_label"`
	// Metrics 查询指标列表
	Metrics []QueryMetric `json:"metrics,omitempty" mapstructure:"metrics"`
	// Where 过滤条件列表
	Where []WhereCondition `json:"where,omitempty" mapstructure:"where"`
	// GroupBy 聚合字段列表
	GroupBy []string `json:"group_by,omitempty" mapstructure:"group_by"`
	// Interval 时间间隔，可以是数字或字符串 "auto"
	Interval any `json:"interval,omitempty" mapstructure:"interval"`
	// IntervalUnit 聚合周期单位，可选值为 "s"(默认)、"m"
	IntervalUnit string `json:"interval_unit,omitempty" mapstructure:"interval_unit"`
	// FilterDict 过滤条件字典
	FilterDict map[string]any `json:"filter_dict,omitempty" mapstructure:"filter_dict"`
	// TimeField 时间字段
	TimeField string `json:"time_field,omitempty" mapstructure:"time_field"`
	// PromQL PromQL 查询语句
	PromQL string `json:"promql,omitempty" mapstructure:"promql"`
	// QueryString 日志查询语句
	QueryString string `json:"query_string,omitempty" mapstructure:"query_string"`
	// IndexSetID 索引集 ID
	IndexSetID *int64 `json:"index_set_id,omitempty" mapstructure:"index_set_id"`
	// Functions 计算函数列表
	Functions []QueryFunction `json:"functions,omitempty" mapstructure:"functions"`
	// Display 是否显示
	Display bool `json:"display,omitempty" mapstructure:"display"`
}

// TimeSeriesUnifyQueryReq 统一时序数据查询请求
type TimeSeriesUnifyQueryReq struct {
	// BkBizID 业务 ID（必填，原样传入，不做正负数转换）
	BkBizID int64 `json:"bk_biz_id" mapstructure:"bk_biz_id" validate:"required"`
	// QueryConfigs 查询配置列表（必填）
	QueryConfigs []QueryConfig `json:"query_configs" mapstructure:"query_configs" validate:"required,min=1,dive"`
	// Expression 查询表达式（必填）
	Expression string `json:"expression" mapstructure:"expression"`
	// StartTime 开始时间，Unix 时间戳（必填，大于 0）
	StartTime int64 `json:"start_time" mapstructure:"start_time" validate:"required,min=1"`
	// EndTime 结束时间，Unix 时间戳（必填，大于 0）
	EndTime int64 `json:"end_time" mapstructure:"end_time" validate:"required,min=1"`

	// Target 监控目标列表
	Target []any `json:"target,omitempty" mapstructure:"target"`
	// TargetFilterType 监控目标过滤方法，可选值为 auto(默认)、query、post-query
	TargetFilterType string `json:"target_filter_type,omitempty" mapstructure:"target_filter_type"`
	// PostQueryFilterDict 后置查询过滤条件
	PostQueryFilterDict map[string]any `json:"post_query_filter_dict,omitempty" mapstructure:"post_query_filter_dict"`
	// Stack 堆叠标识
	Stack string `json:"stack,omitempty" mapstructure:"stack"`
	// Function 功能函数
	Function map[string]any `json:"function,omitempty" mapstructure:"function"`
	// Functions 计算函数列表
	Functions []QueryFunction `json:"functions,omitempty" mapstructure:"functions"`
	// Limit 限制每个维度的点数
	Limit int `json:"limit,omitempty" mapstructure:"limit"`
	// Slimit 限制维度数量
	Slimit int `json:"slimit,omitempty" mapstructure:"slimit"`
	// DownSampleRange 降采样周期
	DownSampleRange string `json:"down_sample_range,omitempty" mapstructure:"down_sample_range"`
	// Format 输出格式，可选值为 time_series(默认)、heatmap、table
	Format string `json:"format,omitempty" mapstructure:"format"`
	// Type 类型，可选值为 instant、range(默认)
	Type string `json:"type,omitempty" mapstructure:"type"`
	// SeriesNum 查询数据条数
	SeriesNum int `json:"series_num,omitempty" mapstructure:"series_num"`
	// TimeAlignment 是否保留最后一个数据点，默认为 true
	TimeAlignment *bool `json:"time_alignment,omitempty" mapstructure:"time_alignment"`
	// NullAsZero 是否将空值转换为 0，默认为 false
	NullAsZero *bool `json:"null_as_zero,omitempty" mapstructure:"null_as_zero"`
	// QueryMethod 查询方法，默认为 query_data
	QueryMethod string `json:"query_method,omitempty" mapstructure:"query_method"`
	// Unit 单位
	Unit string `json:"unit,omitempty" mapstructure:"unit"`
	// WithMetric 是否返回 metric 信息，默认为 false
	WithMetric *bool `json:"with_metric,omitempty" mapstructure:"with_metric"`
	// NotTimeAlign 是否不对齐时间窗口，默认为 false
	NotTimeAlign *bool `json:"not_time_align,omitempty" mapstructure:"not_time_align"`
	// Step 步长
	Step string `json:"step,omitempty" mapstructure:"step"`
}

// MetricDimension 指标维度信息
type MetricDimension struct {
	// ID 维度唯一标识符
	ID string `json:"id" mapstructure:"id"`
	// Name 维度名称
	Name string `json:"name" mapstructure:"name"`
	// IsDimension 是否为维度
	IsDimension bool `json:"is_dimension" mapstructure:"is_dimension"`
	// Type 维度数据类型
	Type string `json:"type" mapstructure:"type"`
}

// TimeSeriesMetricInfo 时序查询返回的指标信息（完整映射）
type TimeSeriesMetricInfo struct {
	// ID 指标数据库 ID
	ID int64 `json:"id" mapstructure:"id"`
	// ResultTableID 结果表 ID
	ResultTableID string `json:"result_table_id" mapstructure:"result_table_id"`
	// ResultTableName 结果表名称
	ResultTableName string `json:"result_table_name" mapstructure:"result_table_name"`
	// MetricField 指标字段名称
	MetricField string `json:"metric_field" mapstructure:"metric_field"`
	// MetricFieldName 指标字段的描述名称
	MetricFieldName string `json:"metric_field_name" mapstructure:"metric_field_name"`
	// Unit 指标单位
	Unit string `json:"unit" mapstructure:"unit"`
	// UnitConversion 单位转换因子
	UnitConversion int `json:"unit_conversion" mapstructure:"unit_conversion"`
	// Dimensions 维度列表
	Dimensions []MetricDimension `json:"dimensions" mapstructure:"dimensions"`
	// PluginType 插件类型
	PluginType string `json:"plugin_type" mapstructure:"plugin_type"`
	// RelatedName 相关名称
	RelatedName string `json:"related_name" mapstructure:"related_name"`
	// RelatedID 相关 ID
	RelatedID string `json:"related_id" mapstructure:"related_id"`
	// CollectConfig 收集配置
	CollectConfig string `json:"collect_config" mapstructure:"collect_config"`
	// CollectConfigIDs 收集配置 ID 列表
	CollectConfigIDs string `json:"collect_config_ids" mapstructure:"collect_config_ids"`
	// ResultTableLabel 结果表标签
	ResultTableLabel string `json:"result_table_label" mapstructure:"result_table_label"`
	// DataSourceLabel 数据源标签
	DataSourceLabel string `json:"data_source_label" mapstructure:"data_source_label"`
	// DataTypeLabel 数据类型标签
	DataTypeLabel string `json:"data_type_label" mapstructure:"data_type_label"`
	// DataTarget 数据目标
	DataTarget string `json:"data_target" mapstructure:"data_target"`
	// DefaultDimensions 默认维度列表
	DefaultDimensions []string `json:"default_dimensions" mapstructure:"default_dimensions"`
	// DefaultCondition 默认条件列表
	DefaultCondition []any `json:"default_condition" mapstructure:"default_condition"`
	// Description 指标描述信息
	Description string `json:"description" mapstructure:"description"`
	// CollectInterval 数据收集时间间隔
	CollectInterval int `json:"collect_interval" mapstructure:"collect_interval"`
	// CategoryDisplay 类别显示名称
	CategoryDisplay string `json:"category_display" mapstructure:"category_display"`
	// ResultTableLabelName 结果表标签名称
	ResultTableLabelName string `json:"result_table_label_name" mapstructure:"result_table_label_name"`
	// ExtendFields 扩展字段
	ExtendFields map[string]any `json:"extend_fields" mapstructure:"extend_fields"`
	// UseFrequency 使用频率
	UseFrequency int `json:"use_frequency" mapstructure:"use_frequency"`
	// IsDuplicate 是否为重复记录
	IsDuplicate int `json:"is_duplicate" mapstructure:"is_duplicate"`
	// ReadableName 可读的指标名称
	ReadableName string `json:"readable_name" mapstructure:"readable_name"`
	// MetricMD5 指标 MD5 值，用于唯一标识
	MetricMD5 string `json:"metric_md5" mapstructure:"metric_md5"`
	// DataLabel 数据标签
	DataLabel string `json:"data_label" mapstructure:"data_label"`
	// MetricID 指标 ID
	MetricID string `json:"metric_id" mapstructure:"metric_id"`
}

// TimeSeriesDataStat 时序数据统计信息
type TimeSeriesDataStat struct {
	// Count 数据点计数，[0] 为时间戳，[1] 为值
	Count [2]float64 `json:"count" mapstructure:"count"`

	// Sum 数据点求和，[0] 为时间戳，[1] 为值
	Sum [2]float64 `json:"sum" mapstructure:"sum"`

	// Min 最小值，[0] 为时间戳，[1] 为值
	Min [2]float64 `json:"min" mapstructure:"min"`

	// Max 最大值，[0] 为时间戳，[1] 为值
	Max [2]float64 `json:"max" mapstructure:"max"`

	// Avg 平均值，[0] 为时间戳，[1] 为值
	Avg [2]float64 `json:"avg" mapstructure:"avg"`

	// Last 最后一个数据点，[0] 为时间戳，[1] 为值
	Last [2]float64 `json:"last" mapstructure:"last"`
}

// TimeSeriesData 时序数据样本信息
type TimeSeriesData struct {
	// Dimensions 时间序列的维度信息
	Dimensions map[string]string `json:"dimensions" mapstructure:"dimensions"`
	// Target 查询的目标表达式
	Target string `json:"target" mapstructure:"target"`
	// MetricField 指标字段名称
	MetricField string `json:"metric_field" mapstructure:"metric_field"`
	// Datapoints 时间序列数据点，[0] 为值，[1] 为毫秒级时间戳
	Datapoints [][2]float64 `json:"datapoints" mapstructure:"datapoints"`
	// Alias 指标别名
	Alias string `json:"alias" mapstructure:"alias"`
	// Type 数据展示类型，如折线图
	Type string `json:"type" mapstructure:"type"`
	// Stat 统计信息，包含 count、sum、min、max、avg、last
	Stat *TimeSeriesDataStat `json:"stat,omitempty" mapstructure:"stat"`
	// DimensionsTranslation 维度翻译信息
	DimensionsTranslation map[string]any `json:"dimensions_translation" mapstructure:"dimensions_translation"`
	// Unit 指标单位
	Unit string `json:"unit" mapstructure:"unit"`
}

// TimeSeriesUnifyQueryResp 统一时序数据查询响应
type TimeSeriesUnifyQueryResp struct {
	// Series 时序数据样本信息列表
	Series []TimeSeriesData `json:"series" mapstructure:"series"`
	// Metrics 指标信息列表
	Metrics []TimeSeriesMetricInfo `json:"metrics" mapstructure:"metrics"`
}
