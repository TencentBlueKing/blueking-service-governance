/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListHelmDeployRecordsRequest, PaginatedHelmDeployRecordOutputObjs, CreateHelmDeployRequest, PreviewHelmDeployRequest, PreviewHelmDeployOutput, RollbackHelmDeployRequest, EmptyOutput, DeleteHelmDeployRequest, PreviewRollbackHelmDeployRequest, ListTafDeployRecordsRequest, PaginatedAppModelDeployRecordsOutputObjs, CreateTafDeployRequest, DeleteTafDeployRequest, PreCheckTafDeployEnvVarsRequest, EnvVarPreCheckOutput, GetLatestTafDeployStatusRequest, LatestDeployStatus, ListTafResourceSnapshotsRequest, PaginatedAppModelResourceSnapshotsOutputObjs, GetTafResourceSnapshotRequest, GetAppModelResourceSnapshotOutput, ListTrpcDeployRecordsRequest, CreateTrpcDeployRequest, DeleteTrpcDeployRequest, PreCheckTrpcDeployEnvVarsRequest, GetLatestTrpcDeployStatusRequest, ListTrpcResourceSnapshotsRequest, GetTrpcResourceSnapshotRequest } from '~/@types/v1/deploy';

export const DeployService = {
  /**
   * 获取 Helm 应用部署记录列表
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/helm-deploys
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @param keyword query string 搜索关键字
   * @param page query number required 分页页码（从 1 开始）
   * @param pageSize query number required 分页大小
   * @response 200 ListHelmDeployRecordsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listHelmDeployRecords: async <Request extends ListHelmDeployRecordsRequest = ListHelmDeployRecordsRequest, ResponseData = PaginatedHelmDeployRecordOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/helm-deploys')(params, config),
  /**
   * 部署 Helm 应用
   *
   * @method POST
   * @path /apps/{appID}/envs/{envName}/helm-deploys
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param body body CreateHelmDeployInput required 部署 Helm 应用请求
   * @response 201 EmptyOutput Created
   * @response 400 GinErrorOutput Bad Request
   */
  createHelmDeploy: async <Request extends CreateHelmDeployRequest = CreateHelmDeployRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/envs/{envName}/helm-deploys')(params, config),
  /**
   * 部署 Helm 应用预览
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/helm-deploys/preview
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param imageTag query string required 目标镜像 TAG
   * @param chartVersion query string required 指定的部署的 Chart 版本
   * @param valuesFileID query string required 部署使用的 ValuesFile ID
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @response 200 PreviewHelmDeployOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  previewHelmDeploy: async <Request extends PreviewHelmDeployRequest = PreviewHelmDeployRequest, ResponseData = PreviewHelmDeployOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/helm-deploys/preview')(params, config),
  /**
   * Helm 回滚到指定的部署版本
   *
   * @method PUT
   * @path /apps/{appID}/envs/{envName}/helm-deploys/{deployID}
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param deployID path string required 部署记录 ID
   * @param body body RollbackHelmDeployInput Helm 回滚请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  rollbackHelmDeploy: async <Request extends RollbackHelmDeployRequest = RollbackHelmDeployRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/envs/{envName}/helm-deploys/{deployID}')(params, config),
  /**
   * 删除 Helm 应用部署
   *
   * @method DELETE
   * @path /apps/{appID}/envs/{envName}/helm-deploys/{deployID}
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param deployID path string required 部署记录 ID
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteHelmDeploy: async <Request extends DeleteHelmDeployRequest = DeleteHelmDeployRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/envs/{envName}/helm-deploys/{deployID}')(params, config),
  /**
   * Helm 部署版本回滚预览
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/helm-deploys/{deployID}/preview
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param deployID path string required 准备回滚到的记录 ID
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @response 200 PreviewHelmDeployOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  previewRollbackHelmDeploy: async <Request extends PreviewRollbackHelmDeployRequest = PreviewRollbackHelmDeployRequest, ResponseData = PreviewHelmDeployOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/helm-deploys/{deployID}/preview')(params, config),
  /**
   * 获取 TAF 应用部署记录列表
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/taf-deploys
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @param keyword query string 搜索关键字
   * @param page query number required 分页页码（从 1 开始）
   * @param pageSize query number required 分页大小
   * @response 200 ListAppModelDeployRecordsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listTafDeployRecords: async <Request extends ListTafDeployRecordsRequest = ListTafDeployRecordsRequest, ResponseData = PaginatedAppModelDeployRecordsOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/taf-deploys')(params, config),
  /**
   * 创建 TAF 应用部署
   *
   * @method POST
   * @path /apps/{appID}/envs/{envName}/taf-deploys
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param body body CreateAppModelDeployInput required 创建 TAF 部署请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  createTafDeploy: async <Request extends CreateTafDeployRequest = CreateTafDeployRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/envs/{envName}/taf-deploys')(params, config),
  /**
   * 删除 TAF 应用部署（下架当前环境最新版本）
   *
   * @method DELETE
   * @path /apps/{appID}/envs/{envName}/taf-deploys
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteTafDeploy: async <Request extends DeleteTafDeployRequest = DeleteTafDeployRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/envs/{envName}/taf-deploys')(params, config),
  /**
   * TAF 部署前环境变量校验
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/taf-deploys/env-var-precheck
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @response 200 EnvVarPreCheckOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  preCheckTafDeployEnvVars: async <Request extends PreCheckTafDeployEnvVarsRequest = PreCheckTafDeployEnvVarsRequest, ResponseData = EnvVarPreCheckOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/taf-deploys/env-var-precheck')(params, config),
  /**
   * 获取 TAF 应用最新一次部署的状态
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/taf-deploys/latest-status
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @response 200 GetLatestAppModelDeployStatusOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getLatestTafDeployStatus: async <Request extends GetLatestTafDeployStatusRequest = GetLatestTafDeployStatusRequest, ResponseData = LatestDeployStatus>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/taf-deploys/latest-status')(params, config),
  /**
   * 获取 TAF 应用资源快照列表
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/taf-deploys/{deployID}/resource-snapshots
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param deployID path string required 部署记录 ID
   * @param page query number required 分页页码（从 1 开始）
   * @param pageSize query number required 分页大小
   * @response 200 ListAppModelResourceSnapshotsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listTafResourceSnapshots: async <Request extends ListTafResourceSnapshotsRequest = ListTafResourceSnapshotsRequest, ResponseData = PaginatedAppModelResourceSnapshotsOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/taf-deploys/{deployID}/resource-snapshots')(params, config),
  /**
   * 获取 TAF 类型应用部署记录下某个资源的快照详情（具体 k8s 资源的 Manifest 等）
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/taf-deploys/{deployID}/resource-snapshots/{snapshotID}
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param deployID path string required 部署记录 ID
   * @param snapshotID path string required 资源清单快照 ID
   * @response 200 GetAppModelResourceSnapshotOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getTafResourceSnapshot: async <Request extends GetTafResourceSnapshotRequest = GetTafResourceSnapshotRequest, ResponseData = GetAppModelResourceSnapshotOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/taf-deploys/{deployID}/resource-snapshots/{snapshotID}')(params, config),
  /**
   * 获取 Trpc 应用部署记录列表
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/trpc-deploys
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @param keyword query string 搜索关键字
   * @param page query number required 分页页码（从 1 开始）
   * @param pageSize query number required 分页大小
   * @response 200 ListAppModelDeployRecordsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listTrpcDeployRecords: async <Request extends ListTrpcDeployRecordsRequest = ListTrpcDeployRecordsRequest, ResponseData = PaginatedAppModelDeployRecordsOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/trpc-deploys')(params, config),
  /**
   * 创建 Trpc 应用部署
   *
   * @method POST
   * @path /apps/{appID}/envs/{envName}/trpc-deploys
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param body body CreateAppModelDeployInput required 创建 Trpc 部署请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  createTrpcDeploy: async <Request extends CreateTrpcDeployRequest = CreateTrpcDeployRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/envs/{envName}/trpc-deploys')(params, config),
  /**
   * 删除 Trpc 应用部署（下架当前环境最新版本）
   *
   * @method DELETE
   * @path /apps/{appID}/envs/{envName}/trpc-deploys
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteTrpcDeploy: async <Request extends DeleteTrpcDeployRequest = DeleteTrpcDeployRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/envs/{envName}/trpc-deploys')(params, config),
  /**
   * Trpc 部署前环境变量校验
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/trpc-deploys/env-var-precheck
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @response 200 EnvVarPreCheckOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  preCheckTrpcDeployEnvVars: async <Request extends PreCheckTrpcDeployEnvVarsRequest = PreCheckTrpcDeployEnvVarsRequest, ResponseData = EnvVarPreCheckOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/trpc-deploys/env-var-precheck')(params, config),
  /**
   * 获取 Trpc 应用最新一次部署的状态
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/trpc-deploys/latest-status
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @response 200 GetLatestAppModelDeployStatusOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getLatestTrpcDeployStatus: async <Request extends GetLatestTrpcDeployStatusRequest = GetLatestTrpcDeployStatusRequest, ResponseData = LatestDeployStatus>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/trpc-deploys/latest-status')(params, config),
  /**
   * 获取 Trpc 应用资源快照列表
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/trpc-deploys/{deployID}/resource-snapshots
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param deployID path string required 部署记录 ID
   * @param page query number required 分页页码（从 1 开始）
   * @param pageSize query number required 分页大小
   * @response 200 ListAppModelResourceSnapshotsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listTrpcResourceSnapshots: async <Request extends ListTrpcResourceSnapshotsRequest = ListTrpcResourceSnapshotsRequest, ResponseData = PaginatedAppModelResourceSnapshotsOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/trpc-deploys/{deployID}/resource-snapshots')(params, config),
  /**
   * 获取 Trpc 类型应用部署记录下某个资源的快照详情（具体 k8s 资源的 Manifest 等）
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/trpc-deploys/{deployID}/resource-snapshots/{snapshotID}
   * @tag deploy
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param deployID path string required 部署记录 ID
   * @param snapshotID path string required 资源清单快照 ID
   * @response 200 GetAppModelResourceSnapshotOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getTrpcResourceSnapshot: async <Request extends GetTrpcResourceSnapshotRequest = GetTrpcResourceSnapshotRequest, ResponseData = GetAppModelResourceSnapshotOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/trpc-deploys/{deployID}/resource-snapshots/{snapshotID}')(params, config),
};
