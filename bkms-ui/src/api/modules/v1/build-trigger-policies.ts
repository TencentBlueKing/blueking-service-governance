/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListBuildTriggerPoliciesRequest, PolicyListOutputObjs, CreateBuildTriggerPolicyRequest, PolicyOutputObj, HandleBuildTriggerPolicyCallbackRequest, CallbackResultOutputObj, CheckBuildTriggerPolicyConflictRequest, ConflictCheckOutputObj, UpdateBuildTriggerPolicyRequest, DeleteBuildTriggerPolicyRequest, ListBuildTriggerPolicyRecordsRequest, PaginatedTriggerRecordOutputObjs, PatchBuildTriggerPolicyStatusRequest } from '~/@types/v1/build-trigger-policies';

export const BuildTriggerPoliciesService = {
  /**
   * 获取应用的触发策略列表
   *
   * @method GET
   * @path /apps/{appID}/build-trigger-policies
   * @tag build-trigger-policies
   * @param appID path string required 应用 ID
   * @response 200 ListPoliciesOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  listBuildTriggerPolicies: async <Request extends ListBuildTriggerPoliciesRequest = ListBuildTriggerPoliciesRequest, ResponseData = PolicyListOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/build-trigger-policies')(params, config),
  /**
   * 新增触发策略
   *
   * @method POST
   * @path /apps/{appID}/build-trigger-policies
   * @tag build-trigger-policies
   * @param appID path string required 应用 ID
   * @param body body PolicyFormInput required 触发策略表单
   * @response 200 PolicyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  createBuildTriggerPolicy: async <Request extends CreateBuildTriggerPolicyRequest = CreateBuildTriggerPolicyRequest, ResponseData = PolicyOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/build-trigger-policies')(params, config),
  /**
   * 接收构建触发回调
   *
   * @method POST
   * @path /apps/{appID}/build-trigger-policies/callback
   * @tag build-trigger-policies
   * @param appID path string required 应用 ID
   * @param X-Bkms-Build-Trigger-Token header string required 应用独享的回调凭证
   * @param body body CallbackEventInput required 回调事件
   * @response 200 CallbackOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 401 GinErrorOutput Unauthorized
   * @response 404 GinErrorOutput Not Found
   * @response 429 GinErrorOutput Too Many Requests
   */
  handleBuildTriggerPolicyCallback: async <Request extends HandleBuildTriggerPolicyCallbackRequest = HandleBuildTriggerPolicyCallbackRequest, ResponseData = CallbackResultOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/build-trigger-policies/callback')(params, config),
  /**
   * 预检触发策略的重叠冲突
   *
   * @method POST
   * @path /apps/{appID}/build-trigger-policies/conflict-check
   * @tag build-trigger-policies
   * @param appID path string required 应用 ID
   * @param body body ConflictCheckInput required 冲突预检参数
   * @response 200 ConflictCheckOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  checkBuildTriggerPolicyConflict: async <Request extends CheckBuildTriggerPolicyConflictRequest = CheckBuildTriggerPolicyConflictRequest, ResponseData = ConflictCheckOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/build-trigger-policies/conflict-check')(params, config),
  /**
   * 更新触发策略
   *
   * @method PUT
   * @path /apps/{appID}/build-trigger-policies/{policyID}
   * @tag build-trigger-policies
   * @param appID path string required 应用 ID
   * @param policyID path string required 触发策略 ID
   * @param body body PolicyFormInput required 触发策略表单
   * @response 200 PolicyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  updateBuildTriggerPolicy: async <Request extends UpdateBuildTriggerPolicyRequest = UpdateBuildTriggerPolicyRequest, ResponseData = PolicyOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/build-trigger-policies/{policyID}')(params, config),
  /**
   * 删除触发策略
   *
   * @method DELETE
   * @path /apps/{appID}/build-trigger-policies/{policyID}
   * @tag build-trigger-policies
   * @param appID path string required 应用 ID
   * @param policyID path string required 触发策略 ID
   * @response 204 unknown No Content
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  deleteBuildTriggerPolicy: async <Request extends DeleteBuildTriggerPolicyRequest = DeleteBuildTriggerPolicyRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/build-trigger-policies/{policyID}')(params, config),
  /**
   * 获取触发策略的触发记录列表
   *
   * @method GET
   * @path /apps/{appID}/build-trigger-policies/{policyID}/records
   * @tag build-trigger-policies
   * @param appID path string required 应用 ID
   * @param policyID path string required 触发策略 ID
   * @param result query string 结果筛选：built / skipped / failed，留空表示不筛选
   * @param page query number required 分页参数：页码，从 1 开始
   * @param pageSize query number required 分页参数：每页数量，支持 5/10/20/50/100
   * @response 200 ListTriggerRecordsOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  listBuildTriggerPolicyRecords: async <Request extends ListBuildTriggerPolicyRecordsRequest = ListBuildTriggerPolicyRecordsRequest, ResponseData = PaginatedTriggerRecordOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/build-trigger-policies/{policyID}/records')(params, config),
  /**
   * 启用或停用触发策略
   *
   * @method PATCH
   * @path /apps/{appID}/build-trigger-policies/{policyID}/status
   * @tag build-trigger-policies
   * @param appID path string required 应用 ID
   * @param policyID path string required 触发策略 ID
   * @param body body PatchPolicyStatusInput required 启停参数
   * @response 200 PolicyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  patchBuildTriggerPolicyStatus: async <Request extends PatchBuildTriggerPolicyStatusRequest = PatchBuildTriggerPolicyStatusRequest, ResponseData = PolicyOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.patch<Request, ResponseData>('/apps/{appID}/build-trigger-policies/{policyID}/status')(params, config),
};
