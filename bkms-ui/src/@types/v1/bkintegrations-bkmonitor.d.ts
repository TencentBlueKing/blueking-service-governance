/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：bkintegrations-bkmonitor

export interface GetApmServiceNameRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface GetInstanceTimeSeriesRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
  /**
   * 实例名称列表
   */
  instances: string[];
  /**
   * 指标标识
   */
  metricKey: string;
  /**
   * 开始时间（Unix 时间戳）
   */
  startTime: number;
  /**
   * 结束时间（Unix 时间戳）
   */
  endTime: number;
  /**
   * 汇聚周期（秒），默认 60
   */
  interval?: number;
}

export interface GetEnvApmRequest {
  /**
   * 环境 ID
   */
  envID: string;
}

export interface CreateEnvApmRequest {
  /**
   * 环境 ID
   */
  envID: string;
}

export interface BindApmToEnvRequest {
  /**
   * 环境 ID
   */
  envID: string;
  /**
   * APM ID
   */
  apmID: string;
}

export interface ListAlertStrategiesRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 应用 ID
   */
  appID: string;
}

export type CreateAlertStrategyRequest = CreateAlertStrategyBody & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 应用 ID
   */
  appID: string;
};

export interface GetAlertStrategyRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 本地策略 ID
   */
  strategyID: string;
}

export type UpdateAlertStrategyRequest = UpdateAlertStrategyBody & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 本地策略 ID
   */
  strategyID: string;
};

export interface DeleteAlertStrategyRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 本地策略 ID
   */
  strategyID: string;
}

export interface ListAlertEventsByStrategyRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 本地策略 ID
   */
  strategyID: string;
  /**
   * 告警状态
   */
  status?: string[];
  /**
   * 告警级别
   */
  severity?: number[];
  /**
   * 开始时间
   */
  startTime?: number;
  /**
   * 结束时间
   */
  endTime?: number;
  /**
   * 页码，从 1 开始
   */
  page: number;
  /**
   * 每页数量，仅支持 5/10/20/50/100
   */
  pageSize: number;
  /**
   * 按告警 ID 过滤
   */
  alertID?: string;
  /**
   * 按告警展示名称过滤
   */
  alertDisplayName?: string;
  /**
   * 按告警内容过滤（映射到 query_string）
   */
  description?: string;
  /**
   * 按策略名称过滤
   */
  strategyName?: string;
  /**
   * 按事件 ID 过滤
   */
  eventID?: string;
  /**
   * 按目标实例过滤
   */
  target?: string;
  /**
   * 排序字段列表，默认 -create_time
   */
  ordering?: string[];
}

export type SwitchAlertStrategyRequest = SwitchAlertStrategyBody & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 本地策略 ID
   */
  strategyID: string;
};

export interface SyncAlertStrategyRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 本地策略 ID
   */
  strategyID: string;
}

export interface ListAlertEventsRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName?: string;
  /**
   * 告警状态
   */
  status?: string[];
  /**
   * 告警级别
   */
  severity?: number[];
  /**
   * 开始时间
   */
  startTime?: number;
  /**
   * 结束时间
   */
  endTime?: number;
  /**
   * 页码，从 1 开始
   */
  page: number;
  /**
   * 每页数量，仅支持 5/10/20/50/100
   */
  pageSize: number;
  /**
   * 按告警 ID 过滤
   */
  alertID?: string;
  /**
   * 按告警展示名称过滤
   */
  alertDisplayName?: string;
  /**
   * 按告警内容过滤（映射到 query_string）
   */
  description?: string;
  /**
   * 按策略名称过滤
   */
  strategyName?: string;
  /**
   * 按事件 ID 过滤
   */
  eventID?: string;
  /**
   * 按目标实例过滤
   */
  target?: string;
  /**
   * 排序字段列表，默认 -create_time
   */
  ordering?: string[];
}

export interface GetAlertDetailRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 告警 ID
   */
  alertID: string;
}

export interface ListApmsRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export interface ListUserGroupsRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export type CreateUserGroupRequest = SaveUserGroupBody & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
};

export interface GetUserGroupRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 告警组 ID
   */
  groupID: number;
}

export type UpdateUserGroupRequest = SaveUserGroupBody & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 告警组 ID
   */
  groupID: number;
};

export interface DeleteUserGroupRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 告警组 ID
   */
  groupID: number;
}

export interface GetApmServiceNameResp {
  data?: GetApmServiceNameOutput;
}

export interface InstanceTimeSeriesResp {
  /**
   * Data 指标名称 -> 时序数据的映射
   */
  data?: Record<string, MetricTimeSeries>;
}

export interface GetEnvApmResp {
  data?: GetEnvApmOutput;
}

export interface CreateEnvApmResp {
  data?: ApmOutput;
}

export interface EmptyOutput {
}

export interface ListAlertStrategiesResp {
  data?: ListAlertStrategiesOutput;
}

export interface CreateAlertStrategyBody {
  displayName: string;
  effectiveScope: EffectiveScopeInput;
  effectiveTimeRange?: EffectiveTimeRangeInput;
  enabled?: boolean;
  noticeGroupIDs?: number[];
  recoverCondition?: RecoverConditionInput;
  severity: 1 | 2 | 3;
  strategyCode: string;
  threshold: ThresholdConfigInput;
  triggerCondition?: TriggerConditionInput;
}

export interface CreateAlertStrategyResp {
  data?: AlertStrategyOutput;
}

export interface GetAlertStrategyResp {
  data?: AlertStrategyOutput;
}

export interface UpdateAlertStrategyBody {
  displayName?: string;
  effectiveScope?: EffectiveScopeInput;
  effectiveTimeRange?: EffectiveTimeRangeInput;
  enabled?: boolean;
  noticeGroupIDs?: number[];
  recoverCondition?: RecoverConditionInput;
  severity?: 1 | 2 | 3;
  threshold?: ThresholdConfigInput;
  triggerCondition?: TriggerConditionInput;
}

export interface EmptyOutput {
}

export interface ListAlertEventsResp {
  data?: ListAlertEventsOutput;
}

export interface SwitchAlertStrategyBody {
  enabled?: boolean;
}

export interface GetAlertDetailResp {
  data?: Record<string, unknown>;
}

export interface ListApmsResp {
  data?: ListApmOutput;
}

export interface ListUserGroupsResp {
  data?: ListUserGroupsOutput;
}

export interface SaveUserGroupBody {
  actionNotice: ActionNotice[];
  alertNotice: AlertNotice[];
  channels: string[];
  desc?: string;
  name: string;
  users: UserGroupUser[];
}

export interface SaveUserGroupResp {
  data?: UserGroupDetail;
}

export interface GetUserGroupResp {
  data?: UserGroupDetail;
}

export interface EmptyOutput {
}

export interface UserGroupDetail {
  /**
   * ActionNotice 告警处理通知配置
   */
  action_notice?: ActionNotice[];
  /**
   * AlertNotice 告警通知方式
   */
  alert_notice?: AlertNotice[];
  /**
   * BkBizID 业务 ID
   */
  bk_biz_id?: number;
  /**
   * Channels 通知渠道，可选项 user(内部用户)、wxwork-bot(企业微信机器人)
   */
  channels?: string[];
  /**
   * ConfigSource 配置来源
   */
  config_source?: string;
  /**
   * DeleteAllowed 是否可删除
   */
  delete_allowed?: boolean;
  /**
   * Desc 说明
   */
  desc?: string;
  /**
   * DutyArranges 轮值安排
   */
  duty_arranges?: DutyArrange[];
  /**
   * DutyNotice 轮值通知设置
   */
  duty_notice?: DutyNotice;
  /**
   * DutyRules 轮值规则
   */
  duty_rules?: number[];
  /**
   * EditAllowed 是否可编辑
   */
  edit_allowed?: boolean;
  /**
   * ID 告警组 ID
   */
  id?: number;
  /**
   * MentionList 提及人列表（企业微信机器人等渠道会 @ 这些人）
   */
  mention_list?: UserGroupUser[];
  /**
   * MentionType 提及类型：0-不提及，1-提及全部，2-按 MentionList 提及
   */
  mention_type?: number;
  /**
   * Name 名称
   */
  name?: string;
  /**
   * NeedDuty 是否轮值
   */
  need_duty?: boolean;
  /**
   * Path 业务路径
   */
  path?: string;
  /**
   * StrategyCount 关联的告警策略数量
   */
  strategy_count?: number;
  /**
   * Timezone 时区
   */
  timezone?: string;
  /**
   * Users 通知接收人员
   */
  users?: UserGroupUser[];
}

export interface ActionNotice {
  /**
   * NotifyConfig 通知配置
   */
  notify_config?: ActionNoticeConfig[];
  /**
   * TimeRange 生效时间范围
   */
  time_range?: string;
}

export interface AlertNotice {
  /**
   * NotifyConfig 通知配置
   */
  notify_config?: AlertNoticeConfig[];
  /**
   * TimeRange 生效时间范围
   */
  time_range?: string;
}

export interface DutyArrange {
  /**
   * Backups 备份人员
   */
  backups?: Record<string, unknown>[];
  /**
   * DutyRuleID 轮值规则ID。
   * 使用指针是为了：无值时序列化为 null（bkmonitor 读接口常回传 null，
   * 写接口也要求未设置时传 null 而非 0）。
   */
  duty_rule_id?: number;
  /**
   * DutyTime 轮班时间安排
   */
  duty_time?: Record<string, unknown>[];
  /**
   * DutyUsers 值班人员组
   */
  duty_users?: UserGroupUser[][];
  /**
   * EffectiveTime 生效时间。
   * 使用指针是为了：无值时序列化为 null（bkmonitor 读接口可能回传 null 或空串，
   * 写接口未设置时需要传 null）。
   */
  effective_time?: string;
  /**
   * GroupNumber 	自动分组时每个班次对应的人数
   */
  group_number?: number;
  /**
   * GroupType 分组类型，可选项 specified(指定), auto(自动)
   */
  group_type?: string;
  /**
   * HandoffTime 交接班时间配置
   */
  handoff_time?: Record<string, unknown>;
  /**
   * Hash 原始配置摘要
   */
  hash?: string;
  /**
   * ID 轮值 ID
   */
  id?: number;
  /**
   * NeedRotation 是否需要交接班
   */
  need_rotation?: boolean;
  /**
   * Order 排序
   */
  order?: number;
  /**
   * UserGroupID 告警组 ID
   */
  user_group_id?: number;
  /**
   * Users 值班人员（兼容老接口，不需要轮值的时候可以保留该字段）
   */
  users?: UserGroupUser[];
}

export interface DutyNotice {
  /**
   * HitFirstDuty 是否命中首班。
   * 使用指针 + omitempty 是为了：未配置时序列化时完全省略该字段，
   * 避免 bkmonitor 写接口在未设置时收到 false 产生歧义。
   */
  hit_first_duty?: boolean;
  /**
   * PersonalNotice 值班人员通知配置，未配置时不展示该字段
   */
  personal_notice?: Record<string, unknown>;
  /**
   * PlanNotice 轮值计划通知配置，未配置时不展示该字段
   */
  plan_notice?: Record<string, unknown>;
}

export interface UserGroupUser {
  /**
   * DisplayName 显示名
   */
  display_name?: string;
  /**
   * ID 角色 key 或者用户 ID
   */
  id?: string;
  /**
   * Members 对应的人员信息（针对 group 类型）
   */
  members?: Record<string, unknown>[];
  /**
   * Type 类型，可选项 group、user
   */
  type?: string;
}

export interface AlertNoticeConfig {
  /**
   * Level 告警级别：1(致命)，2(预警)，3(提醒)
   */
  level?: number;
  /**
   * NoticeWays 通知方式
   */
  notice_ways?: NoticeWay[];
  /**
   * Type 通知场景类型列表（如 normal / ack / resolved / closed，空数组表示全部）
   */
  type?: string[];
}

export interface NoticeWay {
  /**
   * Name 通知方式名称，如 weixin、sms、voice、wxwork-bot
   */
  name?: string;
  /**
   * Receivers 通知接收人员：企业微信机器人为 chatID，bkchat 为对应的选项 ID
   */
  receivers?: string[];
}

export interface ActionNoticeConfig {
  /**
   * NoticeWays 通知方式
   */
  notice_ways?: NoticeWay[];
  /**
   * Phase 阶段：1(失败时)，2(成功时)，3(执行前)
   */
  phase?: number;
  /**
   * Type 通知场景类型列表（如 normal / ack / resolved / closed，空数组表示全部）
   */
  type?: string[];
}

export interface ListUserGroupsOutput {
  count?: string;
  results?: UserGroup[];
}

export interface UserGroup {
  /**
   * BkBizID 业务 ID
   */
  bk_biz_id?: number;
  /**
   * Channels 通知渠道，可选项 user(内部用户)、wxwork-bot(企业微信机器人)
   */
  channels?: string[];
  /**
   * ConfigSource 配置来源
   */
  config_source?: string;
  /**
   * DeleteAllowed 是否可删除
   */
  delete_allowed?: boolean;
  /**
   * Desc 说明
   */
  desc?: string;
  /**
   * EditAllowed 是否可编辑
   */
  edit_allowed?: boolean;
  /**
   * ID 告警组 ID
   */
  id?: number;
  /**
   * MentionList 提及人列表（企业微信机器人等渠道会 @ 这些人）
   */
  mention_list?: UserGroupUser[];
  /**
   * MentionType 提及类型：0-不提及，1-提及全部，2-按 MentionList 提及
   */
  mention_type?: number;
  /**
   * Name 名称
   */
  name?: string;
  /**
   * NeedDuty 是否轮值
   */
  need_duty?: boolean;
  /**
   * StrategyCount 关联的告警策略数量
   */
  strategy_count?: number;
  /**
   * Timezone 时区
   */
  timezone?: string;
  /**
   * Users 通知接收人员
   */
  users?: UserGroupUser[];
}

export interface ListApmOutput {
  count?: string;
  results?: ApmOutput[];
}

export interface ApmOutput {
  apmID?: string;
  associatedEnvs?: ApmEnvInfoOutput[];
  bkBizID?: string;
  createdAt?: string;
  creator?: string;
  description?: string;
  logReady?: boolean;
  metricReady?: boolean;
  name?: string;
  profilingReady?: boolean;
  token?: string;
  traceReady?: boolean;
  type?: string;
}

export interface ApmEnvInfoOutput {
  envID?: string;
  envName?: string;
}

export interface ListAlertEventsOutput {
  count?: string;
  results?: AlertEventOutput[];
}

export interface AlertEventOutput {
  /**
   * AlertDisplayName 是 BKMS 本地策略展示名，例如 `CPU 使用率过高`；
   * 对应的监控原始名称格式为 `CPU 使用率过高【demo-app】`。
   */
  alertDisplayName?: string;
  alertID?: string;
  assignee?: string[];
  beginTime?: number;
  bkMonitorStrategyID?: string;
  content?: string;
  createTime?: number;
  currentValue?: unknown;
  dataSource?: string;
  description?: string;
  detail?: unknown;
  dimensions?: unknown;
  duration?: string;
  endTime?: number;
  eventID?: string;
  latestTime?: number;
  relatedInfo?: unknown;
  severity?: number;
  status?: string;
  strategyID?: string;
  strategyName?: string;
  target?: string;
  targetType?: string;
}

export interface EffectiveScopeInput {
  envIDs?: string[];
  envTypes?: string[];
  type: "all" | "env_type" | "specific_envs";
}

export interface EffectiveTimeRangeInput {
  endTime?: string;
  startTime?: string;
}

export interface RecoverConditionInput {
  checkWindow?: number;
}

export interface ThresholdConfigInput {
  method: "gte" | "gt" | "lte" | "lt" | "eq" | "neq";
  value?: number;
}

export interface TriggerConditionInput {
  checkWindow?: number;
  count?: number;
}

export interface AlertStrategyOutput {
  appID?: string;
  appName?: string;
  createdAt?: string;
  creator?: string;
  displayName?: string;
  effectiveScope?: EffectiveScopeInput;
  effectiveTimeRange?: EffectiveTimeRangeInput;
  enabled?: boolean;
  id?: string;
  monitorMetric?: string;
  noticeGroupIDs?: number[];
  recoverCondition?: RecoverConditionInput;
  remoteRefs?: RemoteRefOutput[];
  severity?: number;
  strategyCode?: string;
  threshold?: ThresholdConfigInput;
  triggerCondition?: TriggerConditionInput;
  updatedAt?: string;
  updater?: string;
  workspaceID?: string;
}

export interface RemoteRefOutput {
  bkMonitorStrategyID?: string;
  envID?: string;
  envName?: string;
  remoteStrategyName?: string;
  trafficLaneName?: string;
}

export interface ListAlertStrategiesOutput {
  count?: string;
  results?: AlertStrategyOutput[];
}

export interface GetEnvApmOutput {
  apmID?: string;
  name?: string;
  token?: string;
}

export interface MetricTimeSeries {
  /**
   * DisplayName 指标展示名称
   */
  displayName?: string;
  /**
   * Series 各实例的时序数据列表
   */
  series?: TimeSeriesItem[];
  /**
   * Unit 指标单位
   */
  unit?: string;
}

export interface TimeSeriesItem {
  /**
   * DataPoints 时序数据点列表，每个元素为 [时间戳, 值]
   */
  dataPoints?: number[][];
  /**
   * Instance 实例（Pod）名称
   */
  instance?: string;
  /**
   * Stat 统计信息，包含 count、sum、min、max、avg、last
   */
  stat?: TimeSeriesItemStat;
}

export interface TimeSeriesItemStat {
  /**
   * Avg 平均值
   */
  avg?: number[];
  /**
   * Count 数据点计数
   * [0] 为时间戳，[1] 为值
   */
  count?: number[];
  /**
   * Last 最后一个数据点
   */
  last?: number[];
  /**
   * Max 最大值
   */
  max?: number[];
  /**
   * Min 最小值
   */
  min?: number[];
  /**
   * Sum 数据点求和
   */
  sum?: number[];
}

export interface GetApmServiceNameOutput {
  serviceName?: string;
}
