/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：operation-audit

export interface ListOperationRecordFilterOptionsRequest {
}

export interface ListOperationRecordsRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 可选分组参数：AppID
   */
  appID?: string;
  /**
   * 可选分组参数：环境名称，如：dev，prod
   */
  envName?: string;
  /**
   * 可选过滤参数：开始时间，RFC3339
   */
  startedAt?: string;
  /**
   * 可选过滤参数：结束时间，RFC3339
   */
  endedAt?: string;
  /**
   * 可选过滤参数：操作类型，如：create, update, delete
   */
  operationType?: string;
  /**
   * 可选过滤参数：资源类型，如：workspace, app, env
   */
  resourceType?: string;
  /**
   * 可选过滤参数：结果，如：success, failed
   */
  result?: string;
  /**
   * 可选过滤参数：操作人用户名
   */
  username?: string;
  /**
   * 分页参数：页码，从 1 开始
   */
  page: number;
  /**
   * 分页参数：每页数量，支持 5/10/20/50/100
   */
  pageSize: number;
}

export interface ListOperationRecordFilterOptionsOutput {
  data?: OperationRecordFilterOptionsOutputObj;
}

export interface ListOperationRecordsOutput {
  data?: PaginatedOperationRecordOutputObj;
}

export interface PaginatedOperationRecordOutputObj {
  /**
   * 结果数量
   */
  count?: string;
  /**
   * 查询结果
   */
  results?: OperationRecordOutputObj[];
}

export interface OperationRecordOutputObj {
  /**
   * 访问类型，如：web, api
   */
  accessType?: string;
  /**
   * 资源属性，如：name
   */
  attribute?: string;
  /**
   * 资源属性展示用名称，如：名称
   */
  attributeDisplayName?: string;
  /**
   * 操作时间
   */
  createdAt?: string;
  /**
   * 操作数据（前后数据对比）
   */
  data?: OperationDataOutputObj;
  /**
   * 操作分组（聚合数据，用于关联到特定的分类）
   */
  group?: OperationGroupOutputObj;
  /**
   * 操作类型，如：create, update, delete
   */
  operationType?: string;
  /**
   * 操作类型展示用名称，如：创建、更新、删除
   */
  operationTypeDisplayName?: string;
  /**
   * 资源 ID
   */
  resourceID?: string;
  /**
   * 资源类型，如：workspace, app, env
   */
  resourceType?: string;
  /**
   * 资源类型展示用名称，如：工作空间、应用、环境
   */
  resourceTypeDisplayName?: string;
  /**
   * 操作结果，如：success, failed
   */
  result?: string;
  /**
   * 操作人用户名
   */
  username?: string;
}

export interface OperationDataOutputObj {
  /**
   * 变更后数据
   */
  after?: string;
  /**
   * 变更前数据
   */
  before?: string;
}

export interface OperationGroupOutputObj {
  appID?: string;
  /**
   * 环境展示名称，环境已删除或记录未关联环境时为空
   */
  envDisplayName?: string;
  /**
   * 环境类别（standard/feature），环境已删除或记录未关联环境时为空
   */
  envKind?: string;
  envName?: string;
  workspaceID?: string;
}

export interface OperationRecordFilterOptionsOutputObj {
  /**
   * 操作结果选项
   */
  operationResults?: FilterOptionOutputObj[];
  /**
   * 操作类型选项
   */
  operationTypes?: FilterOptionOutputObj[];
  /**
   * 资源类型选项
   */
  resourceTypes?: FilterOptionOutputObj[];
}

export interface FilterOptionOutputObj {
  displayName?: string;
  value?: string;
}
