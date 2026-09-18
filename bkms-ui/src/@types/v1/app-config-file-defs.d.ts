/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：app-config-file-defs

export type CreateAppConfigFileDefRequest = CreateDefInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface ListDefaultFilesWithDefRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export interface GetAppConfigFileDefDetailRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 配置文件定义 ID
   */
  id: string;
  /**
   * 环境名称，为空返回默认文件
   */
  envName?: string;
}

export type UpdateAppConfigFileDefRequest = UpdateDefInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 配置文件定义 ID
   */
  id: string;
};

export interface DeleteAppConfigFileDefRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 配置文件定义 ID
   */
  id: string;
}

export type UpdateAppConfigFileDefContentRequest = UpdateContentInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 配置文件定义 ID
   */
  id: string;
  /**
   * 环境名称，为空更新默认文件
   */
  envName?: string;
};

export interface ListAppConfigFileDefEnvInstancesRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 配置文件定义 ID
   */
  id: string;
}

export interface ResetAppConfigFileDefEnvToDefaultRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 配置文件定义 ID
   */
  id: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface GetMountPreviewRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName?: string;
}

export interface CreateDefInput {
  /**
   * overlay 文件引用的基础文件 ID（fileType=overlay 时必填）
   */
  baseAppConfigFileId?: string;
  /**
   * BSCP 来源配置（contentSourceType=bscp 时必填）
   */
  bscpConfig?: BSCPConfigInput;
  /**
   * 配置种类：framework / plain
   */
  configKind: "framework" | "plain";
  /**
   * 初始内容（可选，仅 local 来源）
   */
  content?: string;
  /**
   * 内容来源：local / bscp
   */
  contentSourceType: "local" | "bscp";
  /**
   * 版本描述
   */
  description?: string;
  /**
   * 文件格式
   */
  fileFormat: "yaml" | "taf";
  /**
   * 文件类型：normal / overlay
   */
  fileType: "normal" | "overlay";
  /**
   * 容器内挂载目录（plain 必填）
   */
  mountDir?: string;
  /**
   * 文件名称
   */
  name: string;
}

export interface CreateDefOutput {
  item?: DefDetailObj;
}

export interface ListDefsOutput {
  items?: DefDetailObj[];
}

export interface GetDefDetailOutput {
  item?: DefDetailObj;
}

export interface UpdateDefInput {
  enableEnvVarRender?: boolean;
  isUnifiedConfig?: boolean;
  mountDir?: string;
  mountedEnvNames?: string[];
  name?: string;
}

export interface UpdateDefOutput {
  item?: DefSummaryObj;
}

export interface DeleteDefOutput {
}

export interface UpdateContentInput {
  content?: string;
  currentVersion?: number;
  description?: string;
}

export interface UpdateContentOutput {
  content?: string;
  currentVersion?: number;
  fileId?: string;
}

export interface ListEnvInstancesOutput {
  items?: EnvInstanceObj[];
}

export interface ResetEnvOutput {
}

export interface MountPreviewOutput {
  items?: MountPreviewItemObj[];
}

export interface MountPreviewItemObj {
  configKind?: string;
  contentSource?: string;
  defId?: string;
  hasEnvFile?: boolean;
  mountDir?: string;
  name?: string;
}

export interface EnvInstanceObj {
  content?: string;
  currentVersion?: number;
  envName?: string;
  fileId?: string;
  overlayContent?: string;
  type?: string;
  updatedAt?: string;
  updater?: string;
}

export interface DefSummaryObj {
  configKind?: string;
  createdAt?: string;
  creator?: string;
  enableEnvVarRender?: boolean;
  id?: string;
  isUnifiedConfig?: boolean;
  mountDir?: string;
  mountedEnvNames?: string[];
  name?: string;
}

export interface DefDetailObj {
  baseAppConfigFileId?: string;
  baseContentInfo?: BaseContentInfoObj;
  configKind?: string;
  content?: string;
  contentSourceType?: string;
  createdAt?: string;
  creator?: string;
  currentVersion?: number;
  /**
   * EditableContentField 前端可编辑的字段（"content" / "overlayContent" / "none"）。
   */
  editableContentField?: string;
  enableEnvVarRender?: boolean;
  fileFormat?: string;
  fileId?: string;
  fileType?: string;
  /**
   * HasEnvInstance 指定环境是否有独立实例（仅按环境查询时有意义）。
   */
  hasEnvInstance?: boolean;
  id?: string;
  isUnifiedConfig?: boolean;
  mountDir?: string;
  mountedEnvNames?: string[];
  name?: string;
  overlayContent?: string;
  updatedAt?: string;
  updater?: string;
}

export interface BaseContentInfoObj {
  content?: string;
  holderContentSourceType?: string;
  holderId?: string;
  holderName?: string;
  isFromAnotherFile?: boolean;
}

export interface BSCPConfigInput {
  bizID: string;
  id: string;
  serviceID: string;
}
