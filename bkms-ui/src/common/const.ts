/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

export const STORAGE_VERSION = '0.0.1';
export const STORAGE_KEY = '_pinia_storage';
export const BKMS_REGEX = {
  // 名称类型正则校验
  nameRegex: /^[a-z]+[-a-z0-9]*[a-z0-9]$/,
  IDRegex: /^[a-z][a-z0-9-]$/,
  appNameRegex: /^[a-z][a-z0-9-]{1,20}$/,
  envNameRegex: /^[a-z][a-z0-9-]{0,19}$/,
  envDisplayNameRegex: /^.{1,32}$/,
  fileNameRegex: /^[a-zA-Z0-9_-]{1,20}$/,
  spaceNameRegex: /^[a-z][a-z0-9-]{1,27}$/,
  spaceDisplayNameRegex: /^.{1,32}$/,
  instanceNameRegex: /^[a-z][a-z0-9-]{0,18}[a-z0-9]$/,
  serviceNameRegex: /^[a-z]([-a-z0-9]{0,61}[a-z0-9])?$/,
  laneNameRegex: /^[a-zA-Z0-9]([-_.a-zA-Z0-9]{0,61}[a-zA-Z0-9])?$/,
  componentNameRegex: /^[a-zA-Z][a-zA-Z0-9-]{0,18}[a-zA-Z0-9]$/,
  instanceKeyRegex: /^[a-zA-Z][a-zA-Z0-9_]{0,19}$/,
  instanceKeyNoLimitRegex: /^[a-zA-Z][a-zA-Z0-9_]*$/,
  envVarKeyRegex: /^[A-Za-z_][A-Za-z0-9_]*$/,
  polarisServiceNameRegex: /^[a-zA-Z0-9._-]{1,128}$/,
  kubernetesMetadataNameRegex: /^([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]$/,
  kubernetesMetadataPrefixRegex: /^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$/,
  kubernetesLabelValueRegex: /^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$/,
  // 正整数正则
  positiveIntegerRegex: /^[1-9]\d*$/,
  // 0%-100%或非负整数正则（允许0）
  percentOrNonNegativeIntegerRegex: /^(0|[1-9]\d*|([0-9]|[1-9]\d?|100)%)$/,
};

/**
 * 阈值为「次数」单位的监控指标集合（如 Pod 重启次数），
 * 其余监控指标默认使用百分比（%）单位
 */
export const COUNT_UNIT_METRICS = new Set(['kube_pod_container_status_restarts_total']);

/**
 * 阈值为「次数」单位的策略码集合，与 COUNT_UNIT_METRICS 对应，
 * 用于表单中根据所选 strategyCode 判断阈值单位
 */
export const COUNT_UNIT_STRATEGY_CODES = new Set(['pod_restart_frequent']);

// 文档地址常量
export const DOC_LINKS = {
  // 接入指引
  ACCESS_GUIDE: '/p/4017296948',
  // bkms-cli 使用文档
  BKMS_CLI: '/p/4017324213',
  // tRPC 开发模式文档
  TRPC_DEV_MODE: '/p/4017348583',
  // 流水线构建操作指引
  PIPELINE_BUILD_GUIDE: '/p/4017315972',
  // APM 观测配置指引 - tRPC Go
  APM_GUIDE_TRPC_GO: '/p/4013675212',
  // APM 观测配置指引 - tRPC C++
  APM_GUIDE_TRPC_CPP: '/p/4015427850',
  // APM 观测配置指引 - TAF
  APM_GUIDE_TAF: '/p/4013675229',
  // 扩缩容稳定性
  SCALE_STABILITY: '/p/1015455438#%E6%89%A9%E7%BC%A9%E5%AE%B9%E7%A8%B3%E5%AE%9A%E6%80%A7',
};
