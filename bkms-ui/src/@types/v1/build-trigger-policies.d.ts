/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：build-trigger-policies

export interface ListBuildTriggerPoliciesRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export type CreateBuildTriggerPolicyRequest = PolicyFormInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export type HandleBuildTriggerPolicyCallbackRequest = CallbackEventInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 应用独享的回调凭证
   */
  "X-Bkms-Build-Trigger-Token": string;
};

export type CheckBuildTriggerPolicyConflictRequest = ConflictCheckInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export type UpdateBuildTriggerPolicyRequest = PolicyFormInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 触发策略 ID
   */
  policyID: string;
};

export interface DeleteBuildTriggerPolicyRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 触发策略 ID
   */
  policyID: string;
}

export interface ListBuildTriggerPolicyRecordsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 触发策略 ID
   */
  policyID: string;
  /**
   * 结果筛选：built / skipped / failed，留空表示不筛选
   */
  result?: string;
  /**
   * 分页参数：页码，从 1 开始
   */
  page: number;
  /**
   * 分页参数：每页数量，支持 5/10/20/50/100
   */
  pageSize: number;
}

export type PatchBuildTriggerPolicyStatusRequest = PatchPolicyStatusInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 触发策略 ID
   */
  policyID: string;
};

export interface ListPoliciesOutput {
  /**
   * 触发策略列表
   */
  data?: PolicyListOutputObjs;
}

export interface PolicyFormInput {
  /**
   * 分支匹配方式：eq 等于，prefix 前缀，all 全部
   */
  branchMatchMode: "eq" | "prefix" | "all";
  /**
   * 分支匹配值，多值以英文逗号分隔；匹配方式为 all 时必须留空
   */
  branchMatchValue?: string;
  /**
   * 触发事件，本期仅支持 push（推送分支）
   */
  event: "push";
  /**
   * 策略名称，应用内唯一，由汉字、大小写字母、数字、- 与 _ 组成
   */
  name: string;
  /**
   * 文件路径条件，留空表示全匹配
   */
  pathFilter?: string;
  /**
   * 镜像版本号规则
   */
  versionRule: VersionRuleInput;
}

export interface PolicyOutput {
  /**
   * 触发策略
   */
  data?: PolicyOutputObj;
}

export interface CallbackEventInput {
  /**
   * 推送的分支名，构建时使用该分支而非构建配置中的默认分支
   */
  branch: string;
  /**
   * commit 作者，用于审计
   */
  commitAuthor?: string;
  /**
   * 本次推送的 HEAD commit 哈希
   */
  commitID: string;
  /**
   * 事件类型，本期仅 push
   */
  event: "push";
  /**
   * 事件发生时间
   */
  eventTime?: string;
  /**
   * 触发策略 ID，用于定位策略
   */
  policyID: string;
}

export interface CallbackOutput {
  /**
   * 回调处理结果
   */
  data?: CallbackResultOutputObj;
}

export interface ConflictCheckInput {
  /**
   * 排除的策略 ID，编辑场景下用于排除自身；新建场景留空
   */
  excludeTriggerID?: string;
  /**
   * 待检测的策略表单
   */
  policy: PolicyFormInput;
}

export interface ConflictCheckOutput {
  /**
   * 冲突检测结果
   */
  data?: ConflictCheckOutputObj;
}

export interface ListTriggerRecordsOutput {
  /**
   * 触发记录分页结果
   */
  data?: PaginatedTriggerRecordOutputObjs;
}

export interface PatchPolicyStatusInput {
  /**
   * 是否启用；用指针以区分「未传」与「显式传 false」
   */
  enabled: boolean;
}

export interface PaginatedTriggerRecordOutputObjs {
  /**
   * 记录总数，按当前筛选条件统计
   */
  count?: string;
  /**
   * 当前页结果，按触发时间倒序
   */
  results?: TriggerRecordOutputObj[];
}

export interface TriggerRecordOutputObj {
  /**
   * 归属应用 ID
   */
  appID?: string;
  /**
   * 分支名
   */
  branch?: string;
  /**
   * 结果为 built 时关联的构建号，其余为空
   */
  buildID?: string;
  /**
   * commit 作者
   */
  commitAuthor?: string;
  /**
   * commit 哈希
   */
  commitID?: string;
  /**
   * 事件类型
   */
  event?: string;
  /**
   * 归属策略 ID
   */
  policyID?: string;
  /**
   * 跳过或失败原因，结果为 built 时为空
   */
  reason?: string;
  /**
   * 处理结果：built 已构建，skipped 已跳过，failed 触发失败
   */
  result?: string;
  /**
   * 触发时间
   */
  triggeredAt?: string;
}

export interface ConflictCheckOutputObj {
  /**
   * 发生冲突的已有策略名列表，无冲突时为空数组
   */
  conflictPolicyNames?: string[];
  /**
   * 冲突级别：none 无冲突，warn 软冲突（可保存），error 硬冲突（禁止保存）
   */
  level?: string;
}

export interface CallbackResultOutputObj {
  /**
   * 结果为 built 时的构建号，其余为空
   */
  buildID?: string;
  /**
   * 跳过或失败原因，结果为 built 时为空
   */
  reason?: string;
  /**
   * 处理结果：built 已发起构建，skipped 已跳过，failed 触发失败
   */
  result?: string;
}

export interface PolicyOutputObj {
  /**
   * 所属应用 ID
   */
  appID?: string;
  /**
   * 分支匹配方式
   */
  branchMatchMode?: string;
  /**
   * 分支匹配值
   */
  branchMatchValue?: string;
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 创建人
   */
  creator?: string;
  /**
   * 触发事件
   */
  event?: string;
  /**
   * 流水线与触发器健康状态：unknown / healthy / unauthorized
   */
  health?: string;
  /**
   * 策略 ID
   */
  id?: string;
  /**
   * 策略名称
   */
  name?: string;
  /**
   * 文件路径条件
   */
  pathFilter?: string;
  /**
   * 关联的蓝盾触发专用流水线 ID
   */
  pipelineID?: string;
  /**
   * 启停状态：enabled 生效中，disabled 已停用
   */
  status?: string;
  /**
   * 关联的蓝盾触发器标识
   */
  triggerID?: string;
  /**
   * 更新时间
   */
  updatedAt?: string;
  /**
   * 镜像版本号规则
   */
  versionRule?: VersionRuleOutput;
}

export interface VersionRuleOutput {
  /**
   * 自定义前缀
   */
  prefix?: string;
  /**
   * 版本号规则类型
   */
  type?: string;
  /**
   * 版本号是否拼接分支名
   */
  withBranch?: boolean;
}

export interface VersionRuleInput {
  /**
   * 自定义前缀，仅 custom 类型使用
   */
  prefix?: string;
  /**
   * 版本号规则类型：custom 自定义版本，semver 语义化版本
   */
  type: "custom" | "semver";
  /**
   * 版本号是否拼接分支名，仅 custom 类型使用
   */
  withBranch?: boolean;
}

export interface PolicyListOutputObjs {
  /**
   * 策略总数，生效中与已停用合并计数
   */
  count?: string;
  /**
   * 全部策略
   */
  results?: PolicyOutputObj[];
}
