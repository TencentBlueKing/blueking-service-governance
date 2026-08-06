/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListOperationRecordFilterOptionsRequest, OperationRecordFilterOptionsOutputObj, ListOperationRecordsRequest, PaginatedOperationRecordOutputObj } from '~/@types/v1/operation-audit';

export const OperationAuditService = {
  /**
   * 获取操作记录筛选选项
   *
   * @method GET
   * @path /operation-records/filter-options
   * @tag operation-audit
   * @response 200 ListOperationRecordFilterOptionsOutput OK
   * @response 500 GinErrorOutput Internal Server Error
   */
  listOperationRecordFilterOptions: async <Request extends ListOperationRecordFilterOptionsRequest = ListOperationRecordFilterOptionsRequest, ResponseData = OperationRecordFilterOptionsOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/operation-records/filter-options')(params, config),
  /**
   * 获取操作审计记录列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/operation-records
   * @tag operation-audit
   * @param workspaceID path string required 工作空间 ID
   * @param appID query string 可选分组参数：AppID
   * @param envName query string 可选分组参数：环境名称，如：dev，prod
   * @param startedAt query string 可选过滤参数：开始时间，RFC3339
   * @param endedAt query string 可选过滤参数：结束时间，RFC3339
   * @param operationType query string 可选过滤参数：操作类型，如：create, update, delete
   * @param resourceType query string 可选过滤参数：资源类型，如：workspace, app, env
   * @param result query string 可选过滤参数：结果，如：success, failed
   * @param username query string 可选过滤参数：操作人用户名
   * @param page query number required 分页参数：页码，从 1 开始
   * @param pageSize query number required 分页参数：每页数量，支持 5/10/20/50/100
   * @response 200 ListOperationRecordsOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  listOperationRecords: async <Request extends ListOperationRecordsRequest = ListOperationRecordsRequest, ResponseData = PaginatedOperationRecordOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/operation-records')(params, config),
};
