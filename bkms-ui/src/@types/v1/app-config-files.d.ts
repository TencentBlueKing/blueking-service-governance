/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：app-config-files

export interface ListAppConfigFileVersionsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 应用配置文件 ID
   */
  appConfigFileID?: string;
  /**
   * 环境名
   */
  envName?: string;
  /**
   * 文件名
   */
  name?: string;
  /**
   * 版本号
   */
  version?: number;
  /**
   * 创建人
   */
  creator?: string;
  /**
   * 版本描述
   */
  description?: string;
  /**
   * 页码，从 1 开始
   */
  page: number;
  /**
   * 每页数量，仅支持 5/10/20/50/100
   */
  pageSize: number;
}

export type CompareAppConfigFileVersionsRequest = CompareAppConfigFileVersionsInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface GetAppConfigFileVersionRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 版本记录 ID
   */
  id: string;
}

export interface DeleteAppConfigFileVersionRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 版本记录 ID
   */
  id: string;
}

export type RollbackAppConfigFileVersionRequest = RollbackAppConfigFileVersionInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 版本记录 ID
   */
  id: string;
};

export interface ListAppConfigFilesRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 按文件类型过滤，仅展示指定类型（normal/overlay）
   */
  type?: string;
  /**
   * 按环境名称过滤，可选。为空表示不过滤
   */
  envName?: string;
}

export type CreateAppConfigFileRequest = CreateAppConfigFileInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export type UpdateAppConfigFileRequest = UpdateAppConfigFileInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 应用配置文件 ID
   */
  id: string;
};

export interface DeleteAppConfigFileRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 应用配置文件 ID
   */
  id: string;
}

export type UpdateAppConfigFileContentRequest = UpdateAppConfigFileContentInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 应用配置文件 ID
   */
  id: string;
};

export interface GetAppConfigFileDetailsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 应用配置文件 ID
   */
  id: string;
}

export type UpdateAppConfigFileOverlayContentRequest = UpdateAppConfigFileOverlayContentInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 应用配置文件 ID
   */
  id: string;
};

export type PreviewOverlayMergeRequest = PreviewOverlayMergeInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 基础应用配置文件 ID
   */
  id: string;
};

export interface ListAppConfigFileVersionsOutput {
  data?: PaginatedAppConfigFileVersionOutputObjs;
}

export interface CompareAppConfigFileVersionsInput {
  /**
   * 当前版本的 ID
   */
  currentVersionID: string;
  /**
   * 原来版本的 ID
   */
  previousVersionID: string;
}

export interface CompareAppConfigFileVersionsOutput {
  /**
   * 当前版本
   */
  current?: AppConfigFileVersionOutputObj;
  /**
   * 原来版本
   */
  previous?: AppConfigFileVersionOutputObj;
}

export interface GetAppConfigFileVersionOutput {
  data?: AppConfigFileVersionOutputObj;
}

export interface AppConfigFileEmptyOutput {
}

export interface RollbackAppConfigFileVersionInput {
  /**
   * 编辑开始时的当前版本号，用于乐观锁冲突检测
   */
  currentVersion?: number;
  /**
   * 回滚版本描述，为空则由后端自动生成
   */
  description?: string;
}

export interface RollbackAppConfigFileVersionOutput {
  data?: AppConfigFileOutputObj;
}

export interface ListAppConfigFilesOutput {
  /**
   * 应用配置文件列表
   */
  items?: AppConfigFileOutputObj[];
}

export interface CreateAppConfigFileInput {
  /**
   * 基础应用配置文件 ID，仅当 type 是 overlay 时为必填，在 handler 中做业务校验
   */
  baseAppConfigFileID?: string;
  /**
   * 当 contentSourceType 为 bscp 时，bscpConfig 为必填
   */
  bscpConfig?: BSCPAppConfigFileConfig;
  /**
   * 内容来源，可选本地（local）或 bscp
   */
  contentSourceType: "local" | "bscp";
  /**
   * 版本描述
   */
  description?: string;
  /**
   * 环境名称，可选。为空表示应用级别配置，非空表示特定环境的配置
   */
  envName?: string;
  /**
   * 文件格式，可选 yaml 或 taf
   */
  fileFormat: "yaml" | "taf";
  /**
   * 应用配置文件名称，包含大小写字母、数字和符号（_-），长度 1-20 之间
   */
  name: string;
  /**
   * 应用配置文件类型，普通或覆盖层
   */
  type: "normal" | "overlay";
}

export interface CreateAppConfigFileOutput {
  /**
   * 所创建的对象详情
   */
  item?: AppConfigFileOutputObj;
}

export interface UpdateAppConfigFileInput {
  /**
   * 基础应用配置文件 ID，仅当 type 是 overlay 时生效
   */
  baseAppConfigFileID?: string;
  /**
   * 当 contentSourceType 为 bscp 时，bscpConfig 为必填
   */
  bscpConfig?: BSCPAppConfigFileConfig;
  /**
   * 编辑开始时的当前版本号，用于乐观锁冲突检测
   */
  currentVersion?: number;
  /**
   * 版本描述
   */
  description?: string;
  /**
   * 应用配置文件名称，包含大小写字母、数字和符号（_-），长度 1-20 之间
   */
  name: string;
}

export interface UpdateAppConfigFileOutput {
  /**
   * 所修改的对象详情
   */
  item?: AppConfigFileOutputObj;
}

export interface UpdateAppConfigFileContentInput {
  /**
   * 应用配置文件 content
   */
  content?: string;
  /**
   * 编辑开始时的当前版本号，用于乐观锁冲突检测
   */
  currentVersion?: number;
  /**
   * 版本描述
   */
  description?: string;
}

export interface UpdateAppConfigFileContentOutput {
  /**
   * 基于新的 values 内容产生的编排结果
   */
  arrgData?: ValidateArrgValuesYAMLOutputObj;
  /**
   * 完整 values 内容
   */
  compiledContent?: string;
}

export interface GetAppConfigFileDetailsOutput {
  /**
   * 基础文件信息
   */
  baseContentInfo?: BaseContentInfoOutputObj;
  /**
   * 内容，通常在文件类型为 normal 时有值
   */
  content?: string;
  /**
   * 当前生效版本号
   */
  currentVersion?: number;
  /**
   * 可编辑字段，有效值为 "none"、"content"、"overlayContent"
   */
  editableContentField?: string;
  /**
   * 覆盖层内容，通常在文件类型为 overlay 时有值
   */
  overlayContent?: string;
  /**
   * 当前生效版本最后修改时间，RFC3339 格式
   */
  updatedAt?: string;
  /**
   * 当前生效版本最后修改人
   */
  updater?: string;
}

export interface UpdateAppConfigFileOverlayContentInput {
  /**
   * 编辑开始时的当前版本号，用于乐观锁冲突检测
   */
  currentVersion?: number;
  /**
   * 版本描述
   */
  description?: string;
  /**
   * 应用配置文件 overlayContent
   */
  overlayContent?: string;
}

export interface PreviewOverlayMergeInput {
  /**
   * 覆盖内容（YAML 格式）
   */
  overlayContent?: string;
}

export interface PreviewOverlayMergeOutput {
  /**
   * 合并后的完整配置内容
   */
  data?: string;
}

export interface BaseContentInfoOutputObj {
  /**
   * 基础文件内容
   */
  content?: string;
  /**
   * 基础文件的内容来源
   */
  holderContentSourceType?: string;
  /**
   * 基础文件 ID
   */
  holderID?: string;
  /**
   * 基础文件名称，仅供展示用
   */
  holderName?: string;
  /**
   * 基础文件是否是另一个文件
   */
  isFromAnotherFile?: boolean;
}

export interface ValidateArrgValuesYAMLOutputObj {
  ingressDomain?: ArrgResultItemOutputObj;
  workloadImage?: ArrgResultItemOutputObj;
}

export interface ArrgResultItemOutputObj {
  /**
   * 如果状态为 skipped，本字段将提供具体原因
   */
  skippedReason?: string;
  /**
   * 编排状态，可能是 configured 或 skipped
   */
  status?: string;
}

export interface AppConfigFileOutputObj {
  /**
   * 基础应用配置文件 ID，可能为空
   */
  baseAppConfigFileID?: string;
  /**
   * 仅当 contentSourceType 为 bscp 时，bscpConfig 才有值
   */
  bscpConfig?: BSCPAppConfigFileConfig;
  /**
   * 文件内容来源
   */
  contentSourceType?: string;
  /**
   * 当前生效版本号
   */
  currentVersion?: number;
  /**
   * 环境名称，为空表示应用级别配置，非空表示特定环境的配置
   */
  envName?: string;
  /**
   * 文件格式
   */
  fileFormat?: string;
  /**
   * 应用配置文件 ID
   */
  id?: string;
  /**
   * 文件名称
   */
  name?: string;
  /**
   * 文件类型
   */
  type?: string;
  /**
   * 当前生效版本最后修改时间，RFC3339 格式
   */
  updatedAt?: string;
  /**
   * 当前生效版本最后修改人
   */
  updater?: string;
}

export interface BSCPAppConfigFileConfig {
  /**
   * BSCP 业务 ID
   */
  bizID: string;
  /**
   * BSCP 配置 ID
   */
  id: string;
  /**
   * BSCP 服务 ID
   */
  serviceID: string;
}

export interface AppConfigFileVersionOutputObj {
  /**
   * 所属应用配置文件 ID
   */
  appConfigFileID?: string;
  /**
   * 应用 ID
   */
  appID?: string;
  /**
   * base 文件 ID
   */
  baseAppConfigFileID?: string;
  /**
   * base 文件版本号
   */
  baseVersion?: number;
  /**
   * BSCP 配置引用
   */
  bscpConfig?: BSCPAppConfigFileConfig;
  /**
   * 普通内容
   */
  content?: string;
  /**
   * 内容来源
   */
  contentSourceType?: string;
  /**
   * 创建时间，RFC3339 格式
   */
  createdAt?: string;
  /**
   * 创建人
   */
  creator?: string;
  /**
   * 删除时间，RFC3339 格式
   */
  deletedAt?: string;
  /**
   * 删除人
   */
  deleter?: string;
  /**
   * 版本描述
   */
  description?: string;
  /**
   * 环境名
   */
  envName?: string;
  /**
   * 文件格式
   */
  fileFormat?: string;
  /**
   * 版本记录 ID
   */
  id?: string;
  /**
   * 是否软删除
   */
  isDeleted?: boolean;
  /**
   * 文件名
   */
  name?: string;
  /**
   * 版本操作类型
   */
  operationType?: string;
  /**
   * 覆盖内容
   */
  overlayContent?: string;
  /**
   * 若为 rollback，表示回滚来源版本号
   */
  rollbackFromVersion?: number;
  /**
   * 文件类型
   */
  type?: string;
  /**
   * 版本号
   */
  version?: number;
}

export interface PaginatedAppConfigFileVersionOutputObjs {
  /**
   * 结果数量
   */
  count?: number;
  /**
   * 查询结果
   */
  results?: AppConfigFileVersionOutputObj[];
}
