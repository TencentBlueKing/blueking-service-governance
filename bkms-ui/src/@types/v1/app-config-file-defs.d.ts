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

export type UpdateHelmFileRequest = UpdateHelmFileInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 配置文件定义 ID
   */
  id: string;
};

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
   * 创建时可选设置环境变量渲染开关；不传时使用 ConfigKind 默认值
   */
  enableEnvVarRender?: boolean;
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
   * 创建时可选设置挂载环境范围；nil = 全环境生效，空数组 = 不挂载任何环境。
   * 环境名格式与路径参数 envName 相同。
   */
  mountedEnvNames?: string[];
  /**
   * 文件名称，仅支持数字、字母、下划线(_)、中划线(-)、点(.)，长度 1-64 之间。
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
  /**
   * EnvContentStrategy: overlay 表示环境内容按补丁与默认配置合并；overwrite 表示环境内容独立生效，不再继承默认配置变更。
   */
  envContentStrategy?: "overlay" | "overwrite";
  isUnifiedConfig?: boolean;
  mountDir?: string;
  mountedEnvNames?: string[];
  /**
   * 文件名称，仅支持数字、字母、下划线(_)、中划线(-)、点(.)，长度 1-64 之间。
   */
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

export interface UpdateHelmFileInput {
  /**
   * overlay 文件必须传当前基础文件 ID，未修改时也传原值。普通文件留空。
   */
  baseAppConfigFileId?: string;
  /**
   * BSCP 来源必须传当前绑定，未修改时也传原值。本地来源留空。
   */
  bscpConfig?: BSCPConfigInput;
  /**
   * 编辑开始时的当前版本号，用于乐观锁冲突检测。
   */
  currentVersion?: number;
  /**
   * 版本描述。基础文件或 BSCP 发生变化时写入版本，只改名称不写版本。
   */
  description?: string;
  /**
   * 文件名称，仅支持数字、字母、下划线(_)、中划线(-)、点(.)，长度 1-64 之间。
   */
  name: string;
}

export interface UpdateHelmFileOutput {
  item?: DefDetailObj;
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

export interface DefDetailObj {
  baseAppConfigFileId?: string;
  baseContentInfo?: BaseContentInfoObj;
  bscpConfig?: BSCPConfigObj;
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
  envContentStrategy?: string;
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

export interface BSCPConfigObj {
  bizID?: string;
  id?: string;
  serviceID?: string;
}

export interface BSCPConfigInput {
  bizID: string;
  id: string;
  serviceID: string;
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
  envContentStrategy?: string;
  id?: string;
  isUnifiedConfig?: boolean;
  mountDir?: string;
  mountedEnvNames?: string[];
  name?: string;
}
