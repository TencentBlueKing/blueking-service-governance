/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { GetApmServiceNameRequest, GetApmServiceNameOutput, GetInstanceTimeSeriesRequest, MetricTimeSeries, GetEnvApmRequest, GetEnvApmOutput, CreateEnvApmRequest, ApmOutput, BindApmToEnvRequest, EmptyOutput, ListAlertStrategiesRequest, ListAlertStrategiesOutput, CreateAlertStrategyRequest, AlertStrategyOutput, GetAlertStrategyRequest, UpdateAlertStrategyRequest, DeleteAlertStrategyRequest, ListAlertEventsByStrategyRequest, ListAlertEventsOutput, SwitchAlertStrategyRequest, SyncAlertStrategyRequest, ListAlertEventsRequest, GetAlertDetailRequest, ListApmsRequest, ListApmOutput, ListUserGroupsRequest, ListUserGroupsOutput, CreateUserGroupRequest, UserGroupDetail, GetUserGroupRequest, UpdateUserGroupRequest, DeleteUserGroupRequest } from '~/@types/v1/bkintegrations-bkmonitor';

export const BkintegrationsBkmonitorService = {
  /**
   * 获取应用环境的 APM 服务名称
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/bkmonitor/apm-service-name
   * @tag bkintegrations-bkmonitor
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 GetApmServiceNameResp OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getApmServiceName: async <Request extends GetApmServiceNameRequest = GetApmServiceNameRequest, ResponseData = GetApmServiceNameOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/bkmonitor/apm-service-name')(params, config),
  /**
   * 查询实例监控指标时序数据
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/bkmonitor/instance-time-series
   * @tag bkintegrations-bkmonitor
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param instances query string[] required 实例名称列表
   * @param metricKey query string required 指标标识
   * @param startTime query number required 开始时间（Unix 时间戳）
   * @param endTime query number required 结束时间（Unix 时间戳）
   * @param interval query number 汇聚周期（秒），默认 60
   * @response 200 InstanceTimeSeriesResp OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getInstanceTimeSeries: async <Request extends GetInstanceTimeSeriesRequest = GetInstanceTimeSeriesRequest, ResponseData = Record<string, MetricTimeSeries>>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/bkmonitor/instance-time-series')(params, config),
  /**
   * 查询环境绑定的 APM
   *
   * @method GET
   * @path /envs/{envID}/bkmonitor/apms
   * @tag bkintegrations-bkmonitor
   * @param envID path string required 环境 ID
   * @response 200 GetEnvApmResp OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvApm: async <Request extends GetEnvApmRequest = GetEnvApmRequest, ResponseData = GetEnvApmOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/envs/{envID}/bkmonitor/apms')(params, config),
  /**
   * 为环境创建 APM 并绑定
   *
   * @method POST
   * @path /envs/{envID}/bkmonitor/apms
   * @tag bkintegrations-bkmonitor
   * @param envID path string required 环境 ID
   * @response 200 CreateEnvApmResp OK
   * @response 400 GinErrorOutput Bad Request
   */
  createEnvApm: async <Request extends CreateEnvApmRequest = CreateEnvApmRequest, ResponseData = ApmOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/envs/{envID}/bkmonitor/apms')(params, config),
  /**
   * 将环境绑定到指定 APM
   *
   * @method PUT
   * @path /envs/{envID}/bkmonitor/apms/{apmID}
   * @tag bkintegrations-bkmonitor
   * @param envID path string required 环境 ID
   * @param apmID path string required APM ID
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  bindApmToEnv: async <Request extends BindApmToEnvRequest = BindApmToEnvRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/envs/{envID}/bkmonitor/apms/{apmID}')(params, config),
  /**
   * 查询告警策略列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies
   * @tag bkintegrations-bkmonitor
   * @param workspaceID path string required 工作空间 ID
   * @param appID path string required 应用 ID
   * @response 200 ListAlertStrategiesResp OK
   * @response 400 GinErrorOutput Bad Request
   */
  listAlertStrategies: async <Request extends ListAlertStrategiesRequest = ListAlertStrategiesRequest, ResponseData = ListAlertStrategiesOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies')(params, config),
  /**
   * 创建告警策略
   *
   * @method POST
   * @path /workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies
   * @tag bkintegrations-bkmonitor
   * @param workspaceID path string required 工作空间 ID
   * @param appID path string required 应用 ID
   * @param body body CreateAlertStrategyBody required 请求体
   * @response 200 CreateAlertStrategyResp OK
   * @response 400 GinErrorOutput Bad Request
   */
  createAlertStrategy: async <Request extends CreateAlertStrategyRequest = CreateAlertStrategyRequest, ResponseData = AlertStrategyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies')(params, config),
  /**
   * 获取告警策略详情
   *
   * @method GET
   * @path /workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID}
   * @tag bkintegrations-bkmonitor
   * @param workspaceID path string required 工作空间 ID
   * @param appID path string required 应用 ID
   * @param strategyID path string required 本地策略 ID
   * @response 200 GetAlertStrategyResp OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getAlertStrategy: async <Request extends GetAlertStrategyRequest = GetAlertStrategyRequest, ResponseData = AlertStrategyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID}')(params, config),
  /**
   * 更新告警策略
   *
   * @method PUT
   * @path /workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID}
   * @tag bkintegrations-bkmonitor
   * @param workspaceID path string required 工作空间 ID
   * @param appID path string required 应用 ID
   * @param strategyID path string required 本地策略 ID
   * @param body body UpdateAlertStrategyBody required 请求体
   * @response 200 GetAlertStrategyResp OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateAlertStrategy: async <Request extends UpdateAlertStrategyRequest = UpdateAlertStrategyRequest, ResponseData = AlertStrategyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID}')(params, config),
  /**
   * 删除告警策略
   *
   * @method DELETE
   * @path /workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID}
   * @tag bkintegrations-bkmonitor
   * @param workspaceID path string required 工作空间 ID
   * @param appID path string required 应用 ID
   * @param strategyID path string required 本地策略 ID
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteAlertStrategy: async <Request extends DeleteAlertStrategyRequest = DeleteAlertStrategyRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID}')(params, config),
  /**
   * 查询规则关联的告警事件
   *
   * @method GET
   * @path /workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID}/alerts
   * @tag bkintegrations-bkmonitor
   * @param workspaceID path string required 工作空间 ID
   * @param appID path string required 应用 ID
   * @param strategyID path string required 本地策略 ID
   * @param status query string[] 告警状态
   * @param severity query number[] 告警级别
   * @param startTime query number 开始时间
   * @param endTime query number 结束时间
   * @param page query number required 页码，从 1 开始
   * @param pageSize query number required 每页数量，仅支持 5/10/20/50/100
   * @param alertID query string 按告警 ID 过滤
   * @param alertName query string 按告警名称过滤
   * @param description query string 按告警内容过滤（映射到 query_string）
   * @param strategyName query string 按策略名称过滤
   * @param eventID query string 按事件 ID 过滤
   * @param target query string 按目标实例过滤
   * @param ordering query string[] 排序字段列表，默认 -create_time
   * @response 200 ListAlertEventsResp OK
   * @response 400 GinErrorOutput Bad Request
   */
  listAlertEventsByStrategy: async <Request extends ListAlertEventsByStrategyRequest = ListAlertEventsByStrategyRequest, ResponseData = ListAlertEventsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID}/alerts')(params, config),
  /**
   * 切换告警策略启停状态
   *
   * @method POST
   * @path /workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID}/switch
   * @tag bkintegrations-bkmonitor
   * @param workspaceID path string required 工作空间 ID
   * @param appID path string required 应用 ID
   * @param strategyID path string required 本地策略 ID
   * @param body body SwitchAlertStrategyBody required 请求体
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  switchAlertStrategy: async <Request extends SwitchAlertStrategyRequest = SwitchAlertStrategyRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID}/switch')(params, config),
  /**
   * 同步告警策略到远端
   *
   * @method POST
   * @path /workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID}/sync
   * @tag bkintegrations-bkmonitor
   * @param workspaceID path string required 工作空间 ID
   * @param appID path string required 应用 ID
   * @param strategyID path string required 本地策略 ID
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  syncAlertStrategy: async <Request extends SyncAlertStrategyRequest = SyncAlertStrategyRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID}/sync')(params, config),
  /**
   * 查询应用下的告警事件列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/apps/{appID}/bkmonitor/alerts
   * @tag bkintegrations-bkmonitor
   * @param workspaceID path string required 工作空间 ID
   * @param appID path string required 应用 ID
   * @param envName query string 环境名称
   * @param status query string[] 告警状态
   * @param severity query number[] 告警级别
   * @param startTime query number 开始时间
   * @param endTime query number 结束时间
   * @param page query number required 页码，从 1 开始
   * @param pageSize query number required 每页数量，仅支持 5/10/20/50/100
   * @param alertID query string 按告警 ID 过滤
   * @param alertName query string 按告警名称过滤
   * @param description query string 按告警内容过滤（映射到 query_string）
   * @param strategyName query string 按策略名称过滤
   * @param eventID query string 按事件 ID 过滤
   * @param target query string 按目标实例过滤
   * @param ordering query string[] 排序字段列表，默认 -create_time
   * @response 200 ListAlertEventsResp OK
   * @response 400 GinErrorOutput Bad Request
   */
  listAlertEvents: async <Request extends ListAlertEventsRequest = ListAlertEventsRequest, ResponseData = ListAlertEventsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alerts')(params, config),
  /**
   * 查询单条告警详情
   *
   * @method GET
   * @path /workspaces/{workspaceID}/bkmonitor/alerts/{alertID}
   * @tag bkintegrations-bkmonitor
   * @param workspaceID path string required 工作空间 ID
   * @param alertID path string required 告警 ID
   * @response 200 GetAlertDetailResp OK
   * @response 400 GinErrorOutput Bad Request
   */
  getAlertDetail: async <Request extends GetAlertDetailRequest = GetAlertDetailRequest, ResponseData = Record<string, unknown>>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/bkmonitor/alerts/{alertID}')(params, config),
  /**
   * 获取工作空间下的 APM 列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/bkmonitor/apms
   * @tag bkintegrations-bkmonitor
   * @param workspaceID path string required 工作空间 ID
   * @response 200 ListApmsResp OK
   * @response 400 GinErrorOutput Bad Request
   */
  listApms: async <Request extends ListApmsRequest = ListApmsRequest, ResponseData = ListApmOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/bkmonitor/apms')(params, config),
  /**
   * 查询告警组列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/bkmonitor/user-groups
   * @tag bkintegrations-bkmonitor
   * @param workspaceID path string required 工作空间 ID
   * @response 200 ListUserGroupsResp OK
   * @response 400 GinErrorOutput Bad Request
   */
  listUserGroups: async <Request extends ListUserGroupsRequest = ListUserGroupsRequest, ResponseData = ListUserGroupsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/bkmonitor/user-groups')(params, config),
  /**
   * 创建告警组
   *
   * @method POST
   * @path /workspaces/{workspaceID}/bkmonitor/user-groups
   * @tag bkintegrations-bkmonitor
   * @param workspaceID path string required 工作空间 ID
   * @param body body SaveUserGroupBody required 请求体
   * @response 200 SaveUserGroupResp OK
   * @response 400 GinErrorOutput Bad Request
   */
  createUserGroup: async <Request extends CreateUserGroupRequest = CreateUserGroupRequest, ResponseData = UserGroupDetail>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/bkmonitor/user-groups')(params, config),
  /**
   * 获取告警组详情
   *
   * @method GET
   * @path /workspaces/{workspaceID}/bkmonitor/user-groups/{groupID}
   * @tag bkintegrations-bkmonitor
   * @param workspaceID path string required 工作空间 ID
   * @param groupID path number required 告警组 ID
   * @response 200 GetUserGroupResp OK
   * @response 400 GinErrorOutput Bad Request
   */
  getUserGroup: async <Request extends GetUserGroupRequest = GetUserGroupRequest, ResponseData = UserGroupDetail>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/bkmonitor/user-groups/{groupID}')(params, config),
  /**
   * 更新告警组
   *
   * @method PUT
   * @path /workspaces/{workspaceID}/bkmonitor/user-groups/{groupID}
   * @tag bkintegrations-bkmonitor
   * @param workspaceID path string required 工作空间 ID
   * @param groupID path number required 告警组 ID
   * @param body body SaveUserGroupBody required 请求体
   * @response 200 SaveUserGroupResp OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateUserGroup: async <Request extends UpdateUserGroupRequest = UpdateUserGroupRequest, ResponseData = UserGroupDetail>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/workspaces/{workspaceID}/bkmonitor/user-groups/{groupID}')(params, config),
  /**
   * 删除告警组
   *
   * @method DELETE
   * @path /workspaces/{workspaceID}/bkmonitor/user-groups/{groupID}
   * @tag bkintegrations-bkmonitor
   * @param workspaceID path string required 工作空间 ID
   * @param groupID path number required 告警组 ID
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteUserGroup: async <Request extends DeleteUserGroupRequest = DeleteUserGroupRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/workspaces/{workspaceID}/bkmonitor/user-groups/{groupID}')(params, config),
};
