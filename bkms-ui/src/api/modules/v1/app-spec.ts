/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { GetAppDefaultAppSpecAnnotationsRequest, AppSpecAnnotationsOutput, SetAppDefaultAppSpecAnnotationsRequest, EmptyOutput, GetAppDefaultAppSpecLabelsRequest, AppSpecLabelsOutput, SetAppDefaultAppSpecLabelsRequest, GetAppDefaultAppSpecLifecycleRequest, AppSpecLifecycleOutput, SetAppDefaultAppSpecLifecycleRequest, GetAppDefaultAppSpecProbeRequest, AppSpecProbeOutput, SetAppDefaultAppSpecProbeRequest, GetAppDefaultAppSpecResourcesRequest, AppSpecResourcesOutput, SetAppDefaultAppSpecResourcesRequest, GetAppDefaultAppSpecTkeRouteEniRequest, AppSpecTkeRouteEniOutput, SetAppDefaultAppSpecTkeRouteEniRequest, GetAppDefaultAppSpecUpdateStrategyRequest, AppSpecUpdateStrategyOutput, SetAppDefaultAppSpecUpdateStrategyRequest, GetAppSpecOverviewRequest, AppSpecOverviewOutput, GetEnvAppSpecAnnotationsRequest, SetEnvAppSpecAnnotationsRequest, DeleteEnvAppSpecAnnotationsRequest, GetEnvEffectiveAppSpecAnnotationsRequest, GetEnvAppSpecDevModeRequest, AppSpecDevModeOutput, SetEnvAppSpecDevModeRequest, DeleteEnvAppSpecDevModeRequest, GetEnvEffectiveAppSpecDevModeRequest, GetEnvAppSpecLabelsRequest, SetEnvAppSpecLabelsRequest, DeleteEnvAppSpecLabelsRequest, GetEnvEffectiveAppSpecLabelsRequest, GetEnvAppSpecLifecycleRequest, SetEnvAppSpecLifecycleRequest, DeleteEnvAppSpecLifecycleRequest, GetEnvEffectiveAppSpecLifecycleRequest, GetEnvAppSpecProbeRequest, SetEnvAppSpecProbeRequest, DeleteEnvAppSpecProbeRequest, GetEnvEffectiveAppSpecProbeRequest, DeleteEnvAppSpecProbeByTypeRequest, GetEnvAppSpecResourcesRequest, SetEnvAppSpecResourcesRequest, DeleteEnvAppSpecResourcesRequest, GetEnvEffectiveAppSpecResourcesRequest, GetEnvAppSpecTkeRouteEniRequest, SetEnvAppSpecTkeRouteEniRequest, DeleteEnvAppSpecTkeRouteEniRequest, GetEnvEffectiveAppSpecTkeRouteEniRequest, GetEnvAppSpecUpdateStrategyRequest, SetEnvAppSpecUpdateStrategyRequest, DeleteEnvAppSpecUpdateStrategyRequest, GetEnvEffectiveAppSpecUpdateStrategyRequest, ListWorkspaceAppSpecDevModeRulesRequest, DevModeRuleOutputObj, CreateWorkspaceAppSpecDevModeRuleRequest, UpdateWorkspaceAppSpecDevModeRuleRequest, DeleteWorkspaceAppSpecDevModeRuleRequest, ListWorkspaceAppSpecResourcesRulesRequest, ResourcesRuleOutputObj, CreateWorkspaceAppSpecResourcesRuleRequest, UpdateWorkspaceAppSpecResourcesRuleRequest, DeleteWorkspaceAppSpecResourcesRuleRequest } from '~/@types/v1/app-spec';

export const AppSpecService = {
  /**
   * 获取应用默认 annotations section 配置
   *
   * @method GET
   * @path /apps/{appID}/app-spec/default-annotations
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @response 200 AppSpecAnnotationsSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getAppDefaultAppSpecAnnotations: async <Request extends GetAppDefaultAppSpecAnnotationsRequest = GetAppDefaultAppSpecAnnotationsRequest, ResponseData = AppSpecAnnotationsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/app-spec/default-annotations')(params, config),
  /**
   * 设置应用默认 annotations section 配置
   *
   * @method PUT
   * @path /apps/{appID}/app-spec/default-annotations
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param body body AppSpecAnnotationsInput required annotations 配置
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  setAppDefaultAppSpecAnnotations: async <Request extends SetAppDefaultAppSpecAnnotationsRequest = SetAppDefaultAppSpecAnnotationsRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/app-spec/default-annotations')(params, config),
  /**
   * 获取应用默认 labels section 配置
   *
   * @method GET
   * @path /apps/{appID}/app-spec/default-labels
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @response 200 AppSpecLabelsSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getAppDefaultAppSpecLabels: async <Request extends GetAppDefaultAppSpecLabelsRequest = GetAppDefaultAppSpecLabelsRequest, ResponseData = AppSpecLabelsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/app-spec/default-labels')(params, config),
  /**
   * 设置应用默认 labels section 配置
   *
   * @method PUT
   * @path /apps/{appID}/app-spec/default-labels
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param body body AppSpecLabelsInput required labels 配置
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  setAppDefaultAppSpecLabels: async <Request extends SetAppDefaultAppSpecLabelsRequest = SetAppDefaultAppSpecLabelsRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/app-spec/default-labels')(params, config),
  /**
   * 获取应用默认 lifecycle section 配置
   *
   * @method GET
   * @path /apps/{appID}/app-spec/default-lifecycle
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @response 200 AppSpecLifecycleSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getAppDefaultAppSpecLifecycle: async <Request extends GetAppDefaultAppSpecLifecycleRequest = GetAppDefaultAppSpecLifecycleRequest, ResponseData = AppSpecLifecycleOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/app-spec/default-lifecycle')(params, config),
  /**
   * 设置应用默认 lifecycle section 配置
   *
   * @method PUT
   * @path /apps/{appID}/app-spec/default-lifecycle
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param body body SetAppDefaultAppSpecLifecycleInput required lifecycle 配置
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  setAppDefaultAppSpecLifecycle: async <Request extends SetAppDefaultAppSpecLifecycleRequest = SetAppDefaultAppSpecLifecycleRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/app-spec/default-lifecycle')(params, config),
  /**
   * 获取应用默认 probe section 配置
   *
   * @method GET
   * @path /apps/{appID}/app-spec/default-probe
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @response 200 AppSpecProbeSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getAppDefaultAppSpecProbe: async <Request extends GetAppDefaultAppSpecProbeRequest = GetAppDefaultAppSpecProbeRequest, ResponseData = AppSpecProbeOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/app-spec/default-probe')(params, config),
  /**
   * 设置应用默认 probe section 配置
   *
   * @method PUT
   * @path /apps/{appID}/app-spec/default-probe
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param body body SetAppDefaultAppSpecProbeInput required probe 配置
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  setAppDefaultAppSpecProbe: async <Request extends SetAppDefaultAppSpecProbeRequest = SetAppDefaultAppSpecProbeRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/app-spec/default-probe')(params, config),
  /**
   * 获取应用默认 resources section 配置
   *
   * @method GET
   * @path /apps/{appID}/app-spec/default-resources
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @response 200 AppSpecResourcesSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getAppDefaultAppSpecResources: async <Request extends GetAppDefaultAppSpecResourcesRequest = GetAppDefaultAppSpecResourcesRequest, ResponseData = AppSpecResourcesOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/app-spec/default-resources')(params, config),
  /**
   * 设置应用默认 resources section 配置
   *
   * @method PUT
   * @path /apps/{appID}/app-spec/default-resources
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param body body SetAppDefaultAppSpecResourcesInput required resources 配置
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  setAppDefaultAppSpecResources: async <Request extends SetAppDefaultAppSpecResourcesRequest = SetAppDefaultAppSpecResourcesRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/app-spec/default-resources')(params, config),
  /**
   * 获取应用默认 tkeRouteEni section 配置
   *
   * @method GET
   * @path /apps/{appID}/app-spec/default-tke-route-eni
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @response 200 AppSpecTkeRouteEniSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getAppDefaultAppSpecTkeRouteEni: async <Request extends GetAppDefaultAppSpecTkeRouteEniRequest = GetAppDefaultAppSpecTkeRouteEniRequest, ResponseData = AppSpecTkeRouteEniOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/app-spec/default-tke-route-eni')(params, config),
  /**
   * 设置应用默认 tkeRouteEni section 配置
   *
   * @method PUT
   * @path /apps/{appID}/app-spec/default-tke-route-eni
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param body body AppSpecTkeRouteEniInput required tkeRouteEni 配置
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  setAppDefaultAppSpecTkeRouteEni: async <Request extends SetAppDefaultAppSpecTkeRouteEniRequest = SetAppDefaultAppSpecTkeRouteEniRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/app-spec/default-tke-route-eni')(params, config),
  /**
   * 获取应用默认 updateStrategy section 配置
   *
   * @method GET
   * @path /apps/{appID}/app-spec/default-update-strategy
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @response 200 AppSpecUpdateStrategySectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getAppDefaultAppSpecUpdateStrategy: async <Request extends GetAppDefaultAppSpecUpdateStrategyRequest = GetAppDefaultAppSpecUpdateStrategyRequest, ResponseData = AppSpecUpdateStrategyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/app-spec/default-update-strategy')(params, config),
  /**
   * 设置应用默认 updateStrategy section 配置
   *
   * @method PUT
   * @path /apps/{appID}/app-spec/default-update-strategy
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param body body SetAppDefaultAppSpecUpdateStrategyInput required updateStrategy 配置
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  setAppDefaultAppSpecUpdateStrategy: async <Request extends SetAppDefaultAppSpecUpdateStrategyRequest = SetAppDefaultAppSpecUpdateStrategyRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/app-spec/default-update-strategy')(params, config),
  /**
   * 获取应用 AppSpec 配置概况
   *
   * @method GET
   * @path /apps/{appID}/app-spec/overview
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @response 200 GetAppSpecOverviewOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getAppSpecOverview: async <Request extends GetAppSpecOverviewRequest = GetAppSpecOverviewRequest, ResponseData = AppSpecOverviewOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/app-spec/overview')(params, config),
  /**
   * 获取应用环境 annotations section 覆盖配置
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/app-spec/annotations
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 AppSpecAnnotationsSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvAppSpecAnnotations: async <Request extends GetEnvAppSpecAnnotationsRequest = GetEnvAppSpecAnnotationsRequest, ResponseData = AppSpecAnnotationsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/annotations')(params, config),
  /**
   * 设置应用环境 annotations section 覆盖配置
   *
   * @method PUT
   * @path /apps/{appID}/envs/{envName}/app-spec/annotations
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param body body AppSpecAnnotationsInput required annotations 配置
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  setEnvAppSpecAnnotations: async <Request extends SetEnvAppSpecAnnotationsRequest = SetEnvAppSpecAnnotationsRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/annotations')(params, config),
  /**
   * 删除应用环境 annotations section 覆盖配置
   *
   * @method DELETE
   * @path /apps/{appID}/envs/{envName}/app-spec/annotations
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  deleteEnvAppSpecAnnotations: async <Request extends DeleteEnvAppSpecAnnotationsRequest = DeleteEnvAppSpecAnnotationsRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/annotations')(params, config),
  /**
   * 获取应用环境最终生效的 annotations section 配置
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/app-spec/annotations/effective
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 AppSpecAnnotationsSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvEffectiveAppSpecAnnotations: async <Request extends GetEnvEffectiveAppSpecAnnotationsRequest = GetEnvEffectiveAppSpecAnnotationsRequest, ResponseData = AppSpecAnnotationsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/annotations/effective')(params, config),
  /**
   * 获取应用环境 devMode section 覆盖配置
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/app-spec/dev-mode
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 AppSpecDevModeSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvAppSpecDevMode: async <Request extends GetEnvAppSpecDevModeRequest = GetEnvAppSpecDevModeRequest, ResponseData = AppSpecDevModeOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/dev-mode')(params, config),
  /**
   * 设置应用环境 devMode section 覆盖配置
   *
   * @method PUT
   * @path /apps/{appID}/envs/{envName}/app-spec/dev-mode
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param body body SetEnvAppSpecDevModeInput required devMode 配置
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  setEnvAppSpecDevMode: async <Request extends SetEnvAppSpecDevModeRequest = SetEnvAppSpecDevModeRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/dev-mode')(params, config),
  /**
   * 删除应用环境 devMode section 覆盖配置
   *
   * @method DELETE
   * @path /apps/{appID}/envs/{envName}/app-spec/dev-mode
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  deleteEnvAppSpecDevMode: async <Request extends DeleteEnvAppSpecDevModeRequest = DeleteEnvAppSpecDevModeRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/dev-mode')(params, config),
  /**
   * 获取应用环境最终生效的 devMode section 配置
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/app-spec/dev-mode/effective
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 AppSpecDevModeSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvEffectiveAppSpecDevMode: async <Request extends GetEnvEffectiveAppSpecDevModeRequest = GetEnvEffectiveAppSpecDevModeRequest, ResponseData = AppSpecDevModeOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/dev-mode/effective')(params, config),
  /**
   * 获取应用环境 labels section 覆盖配置
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/app-spec/labels
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 AppSpecLabelsSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvAppSpecLabels: async <Request extends GetEnvAppSpecLabelsRequest = GetEnvAppSpecLabelsRequest, ResponseData = AppSpecLabelsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/labels')(params, config),
  /**
   * 设置应用环境 labels section 覆盖配置
   *
   * @method PUT
   * @path /apps/{appID}/envs/{envName}/app-spec/labels
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param body body AppSpecLabelsInput required labels 配置
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  setEnvAppSpecLabels: async <Request extends SetEnvAppSpecLabelsRequest = SetEnvAppSpecLabelsRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/labels')(params, config),
  /**
   * 删除应用环境 labels section 覆盖配置
   *
   * @method DELETE
   * @path /apps/{appID}/envs/{envName}/app-spec/labels
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  deleteEnvAppSpecLabels: async <Request extends DeleteEnvAppSpecLabelsRequest = DeleteEnvAppSpecLabelsRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/labels')(params, config),
  /**
   * 获取应用环境最终生效的 labels section 配置
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/app-spec/labels/effective
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 AppSpecLabelsSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvEffectiveAppSpecLabels: async <Request extends GetEnvEffectiveAppSpecLabelsRequest = GetEnvEffectiveAppSpecLabelsRequest, ResponseData = AppSpecLabelsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/labels/effective')(params, config),
  /**
   * 获取应用环境 lifecycle section 覆盖配置
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/app-spec/lifecycle
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 AppSpecLifecycleSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvAppSpecLifecycle: async <Request extends GetEnvAppSpecLifecycleRequest = GetEnvAppSpecLifecycleRequest, ResponseData = AppSpecLifecycleOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/lifecycle')(params, config),
  /**
   * 设置应用环境 lifecycle section 覆盖配置
   *
   * @method PUT
   * @path /apps/{appID}/envs/{envName}/app-spec/lifecycle
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param body body SetEnvAppSpecLifecycleInput required lifecycle 配置
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  setEnvAppSpecLifecycle: async <Request extends SetEnvAppSpecLifecycleRequest = SetEnvAppSpecLifecycleRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/lifecycle')(params, config),
  /**
   * 删除应用环境 lifecycle section 覆盖配置
   *
   * @method DELETE
   * @path /apps/{appID}/envs/{envName}/app-spec/lifecycle
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  deleteEnvAppSpecLifecycle: async <Request extends DeleteEnvAppSpecLifecycleRequest = DeleteEnvAppSpecLifecycleRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/lifecycle')(params, config),
  /**
   * 获取应用环境最终生效的 lifecycle section 配置
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/app-spec/lifecycle/effective
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 AppSpecLifecycleSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvEffectiveAppSpecLifecycle: async <Request extends GetEnvEffectiveAppSpecLifecycleRequest = GetEnvEffectiveAppSpecLifecycleRequest, ResponseData = AppSpecLifecycleOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/lifecycle/effective')(params, config),
  /**
   * 获取应用环境 probe section 覆盖配置
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/app-spec/probe
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 AppSpecProbeSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvAppSpecProbe: async <Request extends GetEnvAppSpecProbeRequest = GetEnvAppSpecProbeRequest, ResponseData = AppSpecProbeOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/probe')(params, config),
  /**
   * 设置应用环境 probe section 覆盖配置
   *
   * @method PUT
   * @path /apps/{appID}/envs/{envName}/app-spec/probe
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param body body SetEnvAppSpecProbeInput required probe 配置
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  setEnvAppSpecProbe: async <Request extends SetEnvAppSpecProbeRequest = SetEnvAppSpecProbeRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/probe')(params, config),
  /**
   * 删除应用环境 probe section 覆盖配置
   *
   * @method DELETE
   * @path /apps/{appID}/envs/{envName}/app-spec/probe
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  deleteEnvAppSpecProbe: async <Request extends DeleteEnvAppSpecProbeRequest = DeleteEnvAppSpecProbeRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/probe')(params, config),
  /**
   * 获取应用环境最终生效的 probe section 配置
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/app-spec/probe/effective
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 AppSpecProbeSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvEffectiveAppSpecProbe: async <Request extends GetEnvEffectiveAppSpecProbeRequest = GetEnvEffectiveAppSpecProbeRequest, ResponseData = AppSpecProbeOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/probe/effective')(params, config),
  /**
   * 删除应用环境下指定类型的探针配置
   *
   * @method DELETE
   * @path /apps/{appID}/envs/{envName}/app-spec/probe/{probeType}
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param probeType path string required 探针类型，可选 liveness、readiness、startup
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  deleteEnvAppSpecProbeByType: async <Request extends DeleteEnvAppSpecProbeByTypeRequest = DeleteEnvAppSpecProbeByTypeRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/probe/{probeType}')(params, config),
  /**
   * 获取应用环境 resources section 覆盖配置
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/app-spec/resources
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 AppSpecResourcesSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvAppSpecResources: async <Request extends GetEnvAppSpecResourcesRequest = GetEnvAppSpecResourcesRequest, ResponseData = AppSpecResourcesOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/resources')(params, config),
  /**
   * 设置应用环境 resources section 覆盖配置
   *
   * @method PUT
   * @path /apps/{appID}/envs/{envName}/app-spec/resources
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param body body SetEnvAppSpecResourcesInput required resources 配置
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  setEnvAppSpecResources: async <Request extends SetEnvAppSpecResourcesRequest = SetEnvAppSpecResourcesRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/resources')(params, config),
  /**
   * 删除应用环境 resources section 覆盖配置
   *
   * @method DELETE
   * @path /apps/{appID}/envs/{envName}/app-spec/resources
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  deleteEnvAppSpecResources: async <Request extends DeleteEnvAppSpecResourcesRequest = DeleteEnvAppSpecResourcesRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/resources')(params, config),
  /**
   * 获取应用环境最终生效的 resources section 配置
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/app-spec/resources/effective
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 AppSpecResourcesSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvEffectiveAppSpecResources: async <Request extends GetEnvEffectiveAppSpecResourcesRequest = GetEnvEffectiveAppSpecResourcesRequest, ResponseData = AppSpecResourcesOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/resources/effective')(params, config),
  /**
   * 获取应用环境 tkeRouteEni section 覆盖配置
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/app-spec/tke-route-eni
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 AppSpecTkeRouteEniSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvAppSpecTkeRouteEni: async <Request extends GetEnvAppSpecTkeRouteEniRequest = GetEnvAppSpecTkeRouteEniRequest, ResponseData = AppSpecTkeRouteEniOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/tke-route-eni')(params, config),
  /**
   * 设置应用环境 tkeRouteEni section 覆盖配置
   *
   * @method PUT
   * @path /apps/{appID}/envs/{envName}/app-spec/tke-route-eni
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param body body AppSpecTkeRouteEniInput required tkeRouteEni 配置
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  setEnvAppSpecTkeRouteEni: async <Request extends SetEnvAppSpecTkeRouteEniRequest = SetEnvAppSpecTkeRouteEniRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/tke-route-eni')(params, config),
  /**
   * 删除应用环境 tkeRouteEni section 覆盖配置
   *
   * @method DELETE
   * @path /apps/{appID}/envs/{envName}/app-spec/tke-route-eni
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  deleteEnvAppSpecTkeRouteEni: async <Request extends DeleteEnvAppSpecTkeRouteEniRequest = DeleteEnvAppSpecTkeRouteEniRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/tke-route-eni')(params, config),
  /**
   * 获取应用环境最终生效的 tkeRouteEni section 配置
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/app-spec/tke-route-eni/effective
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 AppSpecTkeRouteEniSectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvEffectiveAppSpecTkeRouteEni: async <Request extends GetEnvEffectiveAppSpecTkeRouteEniRequest = GetEnvEffectiveAppSpecTkeRouteEniRequest, ResponseData = AppSpecTkeRouteEniOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/tke-route-eni/effective')(params, config),
  /**
   * 获取应用环境 updateStrategy section 覆盖配置
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/app-spec/update-strategy
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 AppSpecUpdateStrategySectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvAppSpecUpdateStrategy: async <Request extends GetEnvAppSpecUpdateStrategyRequest = GetEnvAppSpecUpdateStrategyRequest, ResponseData = AppSpecUpdateStrategyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/update-strategy')(params, config),
  /**
   * 设置应用环境 updateStrategy section 覆盖配置
   *
   * @method PUT
   * @path /apps/{appID}/envs/{envName}/app-spec/update-strategy
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param body body SetEnvAppSpecUpdateStrategyInput required updateStrategy 配置
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  setEnvAppSpecUpdateStrategy: async <Request extends SetEnvAppSpecUpdateStrategyRequest = SetEnvAppSpecUpdateStrategyRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/update-strategy')(params, config),
  /**
   * 删除应用环境 updateStrategy section 覆盖配置
   *
   * @method DELETE
   * @path /apps/{appID}/envs/{envName}/app-spec/update-strategy
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  deleteEnvAppSpecUpdateStrategy: async <Request extends DeleteEnvAppSpecUpdateStrategyRequest = DeleteEnvAppSpecUpdateStrategyRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/update-strategy')(params, config),
  /**
   * 获取应用环境最终生效的 updateStrategy section 配置
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/app-spec/update-strategy/effective
   * @tag app-spec
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 AppSpecUpdateStrategySectionOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getEnvEffectiveAppSpecUpdateStrategy: async <Request extends GetEnvEffectiveAppSpecUpdateStrategyRequest = GetEnvEffectiveAppSpecUpdateStrategyRequest, ResponseData = AppSpecUpdateStrategyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/app-spec/update-strategy/effective')(params, config),
  /**
   * 查询工作空间开发模式默认配置规则
   *
   * @method GET
   * @path /workspaces/{workspaceID}/app-spec/dev-mode
   * @tag app-spec
   * @param workspaceID path string required 工作空间 ID
   * @response 200 ListDevModeRulesOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  listWorkspaceAppSpecDevModeRules: async <Request extends ListWorkspaceAppSpecDevModeRulesRequest = ListWorkspaceAppSpecDevModeRulesRequest, ResponseData = DevModeRuleOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/app-spec/dev-mode')(params, config),
  /**
   * 新增工作空间开发模式默认配置规则
   *
   * @method POST
   * @path /workspaces/{workspaceID}/app-spec/dev-mode
   * @tag app-spec
   * @param workspaceID path string required 工作空间 ID
   * @param body body DevModeRuleInput required 开发模式默认配置规则
   * @response 200 DevModeRuleOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  createWorkspaceAppSpecDevModeRule: async <Request extends CreateWorkspaceAppSpecDevModeRuleRequest = CreateWorkspaceAppSpecDevModeRuleRequest, ResponseData = DevModeRuleOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/app-spec/dev-mode')(params, config),
  /**
   * 编辑工作空间开发模式默认配置规则
   *
   * @method PUT
   * @path /workspaces/{workspaceID}/app-spec/dev-mode/{ruleID}
   * @tag app-spec
   * @param workspaceID path string required 工作空间 ID
   * @param ruleID path string required 规则 ID
   * @param body body DevModeRuleInput required 开发模式默认配置规则
   * @response 200 DevModeRuleOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  updateWorkspaceAppSpecDevModeRule: async <Request extends UpdateWorkspaceAppSpecDevModeRuleRequest = UpdateWorkspaceAppSpecDevModeRuleRequest, ResponseData = DevModeRuleOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/workspaces/{workspaceID}/app-spec/dev-mode/{ruleID}')(params, config),
  /**
   * 删除工作空间开发模式默认配置规则
   *
   * @method DELETE
   * @path /workspaces/{workspaceID}/app-spec/dev-mode/{ruleID}
   * @tag app-spec
   * @param workspaceID path string required 工作空间 ID
   * @param ruleID path string required 规则 ID
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  deleteWorkspaceAppSpecDevModeRule: async <Request extends DeleteWorkspaceAppSpecDevModeRuleRequest = DeleteWorkspaceAppSpecDevModeRuleRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/workspaces/{workspaceID}/app-spec/dev-mode/{ruleID}')(params, config),
  /**
   * 查询工作空间资源规格默认配置规则
   *
   * @method GET
   * @path /workspaces/{workspaceID}/app-spec/resources
   * @tag app-spec
   * @param workspaceID path string required 工作空间 ID
   * @response 200 ListResourcesRulesOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  listWorkspaceAppSpecResourcesRules: async <Request extends ListWorkspaceAppSpecResourcesRulesRequest = ListWorkspaceAppSpecResourcesRulesRequest, ResponseData = ResourcesRuleOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/app-spec/resources')(params, config),
  /**
   * 新增工作空间资源规格默认配置规则
   *
   * @method POST
   * @path /workspaces/{workspaceID}/app-spec/resources
   * @tag app-spec
   * @param workspaceID path string required 工作空间 ID
   * @param body body ResourcesRuleInput required 资源规格默认配置规则
   * @response 200 ResourcesRuleOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  createWorkspaceAppSpecResourcesRule: async <Request extends CreateWorkspaceAppSpecResourcesRuleRequest = CreateWorkspaceAppSpecResourcesRuleRequest, ResponseData = ResourcesRuleOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/app-spec/resources')(params, config),
  /**
   * 编辑工作空间资源规格默认配置规则
   *
   * @method PUT
   * @path /workspaces/{workspaceID}/app-spec/resources/{ruleID}
   * @tag app-spec
   * @param workspaceID path string required 工作空间 ID
   * @param ruleID path string required 规则 ID
   * @param body body ResourcesRuleInput required 资源规格默认配置规则
   * @response 200 ResourcesRuleOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  updateWorkspaceAppSpecResourcesRule: async <Request extends UpdateWorkspaceAppSpecResourcesRuleRequest = UpdateWorkspaceAppSpecResourcesRuleRequest, ResponseData = ResourcesRuleOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/workspaces/{workspaceID}/app-spec/resources/{ruleID}')(params, config),
  /**
   * 删除工作空间资源规格默认配置规则
   *
   * @method DELETE
   * @path /workspaces/{workspaceID}/app-spec/resources/{ruleID}
   * @tag app-spec
   * @param workspaceID path string required 工作空间 ID
   * @param ruleID path string required 规则 ID
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  deleteWorkspaceAppSpecResourcesRule: async <Request extends DeleteWorkspaceAppSpecResourcesRuleRequest = DeleteWorkspaceAppSpecResourcesRuleRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/workspaces/{workspaceID}/app-spec/resources/{ruleID}')(params, config),
};
