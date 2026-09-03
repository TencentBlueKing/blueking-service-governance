/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListEventsRequest, PaginatedEventsOutputObj, ListAppInstancesRequest, PaginatedAppInstancesOutputObj, UpdateAppInstancesRequest, EmptyOutput, ListTrpcAdminCmdsRequest, ListTrpcAdminCmdsOutputObjs, ExecuteTrpcAdminCmdRequest, ExecuteTrpcAdminCmdOutputObjs, BatchDeleteAppInstancesRequest, UpdateAppInstancePolarisRequest, ScaleAppInstancesRequest, ExecuteTafAdminCmdRequest, ExecuteTafAdminCmdOutputObjs, WatchAppInstancesRequest, AppInstanceWatchStreamDoc, ListAppInstanceLogsRequest, LogEntryOutputObj, PortForwardRequest, CreateAppInstanceWebConsoleRequest, WebConsoleInfoOutputObj } from '~/@types/v1/instance';

export const InstanceService = {
  /**
   * 获取指定环境的事件列表
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/events
   * @tag instance
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @param level query string 事件级别（可选过滤参数，可选值：Normal, Warning）
   * @param startedAt query number 起始时间戳
   * @param endedAt query number 结束时间戳
   * @param page query number required 页码，从 1 开始
   * @param pageSize query number required 每页数量
   * @response 200 ListEventsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listEvents: async <Request extends ListEventsRequest = ListEventsRequest, ResponseData = PaginatedEventsOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/events')(params, config),
  /**
   * 获取应用实例列表
   *
   * 北极星拉取失败不阻塞 Pod 输出：polarisInfos 为空数组，与未注册北极星同形，其余字段照常返回。
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/instances
   * @tag instance
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @param all query boolean 为 true 时一次返回全量实例；禁止同时带 page 或 pageSize
   * @param page query number 页码，取值 1-10000；分页模式必填，all=true 时禁止出现
   * @param pageSize query number 每页数量；分页模式必填，all=true 时禁止出现
   * @response 200 ListAppInstancesOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listAppInstances: async <Request extends ListAppInstancesRequest = ListAppInstancesRequest, ResponseData = PaginatedAppInstancesOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/instances')(params, config),
  /**
   * 更新应用实例（支持单/多/全量实例更新）
   *
   * @method PUT
   * @path /apps/{appID}/envs/{envName}/instances
   * @tag instance
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param body body UpdateAppInstancesInput required 更新实例请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateAppInstances: async <Request extends UpdateAppInstancesRequest = UpdateAppInstancesRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/envs/{envName}/instances')(params, config),
  /**
   * 查询 Trpc 管理命令
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/instances/admin-cmds
   * @tag instance
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param instanceIDs query string[] required 实例 ID 列表
   * @response 200 ListTrpcAdminCmdsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listTrpcAdminCmds: async <Request extends ListTrpcAdminCmdsRequest = ListTrpcAdminCmdsRequest, ResponseData = ListTrpcAdminCmdsOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/instances/admin-cmds')(params, config),
  /**
   * 执行 Trpc 管理命令
   *
   * @method POST
   * @path /apps/{appID}/envs/{envName}/instances/admin-cmds
   * @tag instance
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param body body ExecuteTrpcAdminCmdInput required 执行 Trpc 管理命令请求
   * @response 200 ExecuteTrpcAdminCmdOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  executeTrpcAdminCmd: async <Request extends ExecuteTrpcAdminCmdRequest = ExecuteTrpcAdminCmdRequest, ResponseData = ExecuteTrpcAdminCmdOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/envs/{envName}/instances/admin-cmds')(params, config),
  /**
   * 批量删除指定的应用实例，同时缩容副本数量
   *
   * @method POST
   * @path /apps/{appID}/envs/{envName}/instances/operations/batch_delete
   * @tag instance
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param body body BatchDeleteAppInstancesInput required 批量删除实例请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  batchDeleteAppInstances: async <Request extends BatchDeleteAppInstancesRequest = BatchDeleteAppInstancesRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/envs/{envName}/instances/operations/batch_delete')(params, config),
  /**
   * 更新应用实例的北极星注解（权重 / 隔离）
   *
   * @method PUT
   * @path /apps/{appID}/envs/{envName}/instances/operations/polaris
   * @tag instance
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param body body UpdateAppInstancePolarisInput required 更新北极星注解请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateAppInstancePolaris: async <Request extends UpdateAppInstancePolarisRequest = UpdateAppInstancePolarisRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/envs/{envName}/instances/operations/polaris')(params, config),
  /**
   * 扩缩容应用实例数量
   *
   * @method PUT
   * @path /apps/{appID}/envs/{envName}/instances/operations/scale
   * @tag instance
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param body body ScaleAppInstancesInput required 扩缩容请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  scaleAppInstances: async <Request extends ScaleAppInstancesRequest = ScaleAppInstancesRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/envs/{envName}/instances/operations/scale')(params, config),
  /**
   * 执行 TAF 管理命令
   *
   * @method POST
   * @path /apps/{appID}/envs/{envName}/instances/taf-admin-cmds
   * @tag instance
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param body body ExecuteTafAdminCmdInput required 执行 TAF 管理命令请求
   * @response 200 ExecuteTafAdminCmdOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  executeTafAdminCmd: async <Request extends ExecuteTafAdminCmdRequest = ExecuteTafAdminCmdRequest, ResponseData = ExecuteTafAdminCmdOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/envs/{envName}/instances/taf-admin-cmds')(params, config),
  /**
   * 订阅应用实例投影变更
   *
   * SSE 同一条流上推送两类事件，信封不同：
   * 1) Pod 事件 ADDED/MODIFIED/DELETED/ENDED，object 为实例投影；
   * DELETED 只保证 id，ENDED 时 object 为 null；Pod 事件不承载附属数据，polarisInfos 恒为空数组。
   * 2) 附属数据事件 PLUGIN，带 plugin 标识来源（当前仅 polaris），object 为 {id, data}；
   * 约 15s 一轮，仅该实例的附属数据有变化时推送。它不是实例的增删改，前端按 id 覆盖对应行的插件数据即可。
   * 3）附属数据首包取自 List 响应内嵌的 polarisInfos，增量只看 PLUGIN 事件。
   * 4）插件拉取失败时跳过本轮、不推事件也不拆流，页面保留上次已知状态。
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/instances/watch
   * @tag instance
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @param resourceVersion query string required List 成功响应带回的续传位点
   * @response 200 AppInstanceWatchStreamDoc SSE 事件流；每条 data 为 podEvent 或
   * @response 400 GinErrorOutput Bad Request
   * @response 500 GinErrorOutput Internal Server Error
   */
  watchAppInstances: async <Request extends WatchAppInstancesRequest = WatchAppInstancesRequest, ResponseData = AppInstanceWatchStreamDoc>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/instances/watch')(params, config),
  /**
   * 获取应用运行实例（Pod）日志
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/instances/{instanceID}/logs
   * @tag instance
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param instanceID path string required 实例 ID
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @param previous query boolean 是否获取重启前日志
   * @param tailLines query number required 日志行数（从尾部起算），最大 2000
   * @response 200 ListAppInstanceLogsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listAppInstanceLogs: async <Request extends ListAppInstanceLogsRequest = ListAppInstanceLogsRequest, ResponseData = LogEntryOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/instances/{instanceID}/logs')(params, config),
  /**
   * 应用实例端口转发
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/instances/{instanceID}/port-forward/connect
   * @tag instance
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param instanceID path string required 实例 ID
   * @param remotePort query number required 目标 Pod 端口号
   * @param localPort query number required CLI 本地监听端口号
   */
  portForward: async <Request extends PortForwardRequest = PortForwardRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/instances/{instanceID}/port-forward/connect')(params, config),
  /**
   * 创建应用运行实例（Pod）WebConsole
   *
   * @method POST
   * @path /apps/{appID}/envs/{envName}/instances/{instanceID}/web-console
   * @tag instance
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param instanceID path string required 实例 ID
   * @param body body CreateAppInstanceWebConsoleInput 创建 WebConsole 请求
   * @response 200 CreateAppInstanceWebConsoleOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  createAppInstanceWebConsole: async <Request extends CreateAppInstanceWebConsoleRequest = CreateAppInstanceWebConsoleRequest, ResponseData = WebConsoleInfoOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/envs/{envName}/instances/{instanceID}/web-console')(params, config),
};
