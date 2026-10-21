/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：release-strategies

export interface ListReleaseStrategiesRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export type CreateReleaseStrategyRequest = ReleaseStrategyInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
};

export interface GetReleaseStrategyRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 发布策略 ID
   */
  strategyID: string;
}

export type UpdateReleaseStrategyRequest = ReleaseStrategyInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 发布策略 ID
   */
  strategyID: string;
};

export interface DeleteReleaseStrategyRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 发布策略 ID
   */
  strategyID: string;
}

export interface ListReleaseStrategiesOutput {
  /**
   * 空间内全部发布策略，按创建时间升序，不分页
   */
  data?: ReleaseStrategyOutputObj[];
}

export interface ReleaseStrategyInput {
  /**
   * 批次列表，1 ~ 10 个；不校验跨批次递增
   */
  batches: BatchInput[];
  /**
   * 策略描述，不超过 200 个字符
   */
  description?: string;
  /**
   * 策略名称，1 ~ 32 个字符，同一空间内唯一
   */
  name: string;
}

export interface ReleaseStrategyOutput {
  /**
   * 发布策略
   */
  data?: ReleaseStrategyOutputObj;
}

export interface EmptyOutput {
}

export interface ReleaseStrategyOutputObj {
  /**
   * 批次列表
   */
  batches?: BatchOutputObj[];
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 创建人
   */
  creator?: string;
  /**
   * 策略描述
   */
  description?: string;
  /**
   * 策略 ID
   */
  id?: string;
  /**
   * 策略名称
   */
  name?: string;
  /**
   * 最近更新时间
   */
  updatedAt?: string;
  /**
   * 最近更新人
   */
  updater?: string;
  /**
   * 所属工作空间 ID
   */
  workspaceID?: string;
}

export interface BatchOutputObj {
  /**
   * 本批完成后的暂停方式：none / duration / permanent
   */
  pauseMode?: string;
  /**
   * 定时暂停时长（秒），pauseMode 不为 duration 时为 0
   */
  pauseSeconds?: number;
  /**
   * 本批"累计更新到"的实例数：正整数或百分比
   */
  target?: string;
}

export interface BatchInput {
  /**
   * 本批完成后的暂停方式：none 不暂停 / duration 定时暂停 / permanent 永久暂停（需人工继续）
   */
  pauseMode: "none" | "duration" | "permanent";
  /**
   * 定时暂停时长（秒），仅 pauseMode 为 duration 时必填且只能在此时填写，范围 60 ~ 86400
   */
  pauseSeconds?: number;
  /**
   * 本批"累计更新到"的实例数：正整数（如 "1"）或百分比（如 "25%"），百分比范围 1% ~ 100%
   */
  target: string;
}
