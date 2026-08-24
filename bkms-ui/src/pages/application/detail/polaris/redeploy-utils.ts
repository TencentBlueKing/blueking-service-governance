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

import { i18n } from '~/modules/i18n';

import type { PolarisConfigOutputObj } from '~/@types/v1/polaris-config';

const POLARIS_TOKEN_MASK = '****';

/** 北极星配置重新部署时的变更项，用于展示新旧值对比 */
export interface PolarisRedeployChange {
  /** 字段标识，如 'servicePort' / 'polarisToken' */
  key: string;
  /** 字段中文标签 */
  label: string;
  /** 当前待部署的新值 */
  newValue?: number | string;
  /** 已部署的旧值，未部署时为空 */
  oldValue?: number | string;
}

/** 格式化配置值用于展示：undefined 或空字符串显示为 '--' */
export function formatPolarisRedeployValue(value?: number | string) {
  return value === undefined || value === '' ? '--' : String(value);
}

/**
 * 获取指定环境下的北极星配置变更列表，用于重新部署时的差异提示
 * @param config 北极星配置对象
 * @param envName 环境名称
 * @returns 变更项数组；环境不在作用域内时返回空数组
 */
export function getPolarisRedeployChanges(config: PolarisConfigOutputObj, envName: string): PolarisRedeployChange[] {
  const state = config.envStates?.[envName];
  const appliedFields = state?.appliedFields ?? null;
  const scopeEnvNames = config.scopeEnvNames || [];
  const inScope = scopeEnvNames.includes(envName);

  if (!inScope) return [];

  // 当前作用域内但没有部署快照，说明关键字段尚未在该环境生效。
  if (!hasAppliedFields(appliedFields)) {
    return buildNotDeployedChanges(config);
  }

  const changes: PolarisRedeployChange[] = [];
  if (String(appliedFields?.servicePort ?? '') !== String(config.servicePort ?? '')) {
    changes.push({
      key: 'servicePort',
      label: i18n.global.t('服务端口'),
      oldValue: appliedFields?.servicePort ?? '--',
      newValue: config.servicePort ?? '--',
    });
  }
  if (
    state?.polarisTokenChanged === true &&
    String(appliedFields?.polarisToken ?? '') !== String(config.polarisToken ?? '')
  ) {
    changes.push({
      key: 'polarisToken',
      label: i18n.global.t('北极星Token'),
      oldValue: POLARIS_TOKEN_MASK,
      newValue: POLARIS_TOKEN_MASK,
    });
  }
  return changes;
}

/** 当环境尚未部署时，构建仅包含新值的变更列表（无旧值可对比） */
function buildNotDeployedChanges(config: PolarisConfigOutputObj): PolarisRedeployChange[] {
  return [
    {
      key: 'servicePort',
      label: i18n.global.t('服务端口'),
      newValue: config.servicePort ?? '--',
    },
    {
      key: 'polarisToken',
      label: i18n.global.t('北极星Token'),
      newValue: POLARIS_TOKEN_MASK,
    },
  ];
}

/** 判断 appliedFields 是否有效（非空且有字段），用于判断环境是否已部署过北极星配置 */
function hasAppliedFields(
  fields: NonNullable<PolarisConfigOutputObj['envStates']>[string]['appliedFields'] | null | undefined,
) {
  return !!fields && Object.keys(fields).length > 0;
}
