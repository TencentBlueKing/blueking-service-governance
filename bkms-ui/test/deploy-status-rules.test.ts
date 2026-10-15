/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License is
 * distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) that applies to the current version
 * of the project delivered to anyone in the future.
 */

import { describe, expect, it } from 'vitest';

import {
  resolveInstanceListHasDeployRecord,
  shouldResetDeployStatusOnEnvChange,
} from '~/pages/application/detail/deploy/deploy-status-rules';

import type { LatestDeployStatus } from '~/@types/v1/deploy';
import type { EnvOutput } from '~/@types/v1/env';

/**
 * 覆盖部署页实例列表的两条状态规则（源码构建中空表问题修复）：
 *   1. build 阶段实例列表恒视为「有部署记录」——构建不删旧实例，列表不能被清空；
 *   2. 环境选择器 refresh 重抛当前环境时不重置部署状态——只有真正切换环境才重置。
 */

describe('resolveInstanceListHasDeployRecord', () => {
  it('build 阶段即使 hasDeployRecord 为 false 也视为有部署记录（构建中实例列表不得被清空）', () => {
    const buildWithoutRecord: LatestDeployStatus = { stage: 'build', status: 'running', hasDeployRecord: false };
    expect(resolveInstanceListHasDeployRecord(buildWithoutRecord)).toBe(true);
  });

  it('build 阶段 hasDeployRecord 缺失（后端聚合未落库）同样视为有部署记录', () => {
    const buildRecordMissingField: LatestDeployStatus = { stage: 'build', status: 'running' };
    expect(resolveInstanceListHasDeployRecord(buildRecordMissingField)).toBe(true);
  });

  it('build 阶段 hasDeployRecord 为 true 时结果仍为 true', () => {
    const buildWithRecord: LatestDeployStatus = { stage: 'build', status: 'running', hasDeployRecord: true };
    expect(resolveInstanceListHasDeployRecord(buildWithRecord)).toBe(true);
  });

  it('deploy 阶段透传 hasDeployRecord=false（移除部署后仍展示未部署空态，不回归）', () => {
    const uninstalled: LatestDeployStatus = {
      stage: 'deploy',
      status: 'uninstalled',
      hasDeployRecord: false,
    };
    expect(resolveInstanceListHasDeployRecord(uninstalled)).toBe(false);
  });

  it('deploy 阶段透传 hasDeployRecord=true（正常部署后实例列表正常查询，不回归）', () => {
    const deployed: LatestDeployStatus = { stage: 'deploy', status: 'success', hasDeployRecord: true };
    expect(resolveInstanceListHasDeployRecord(deployed)).toBe(true);
  });

  it('stage 缺失时按部署阶段语义透传 hasDeployRecord', () => {
    const noStage: LatestDeployStatus = { status: 'success', hasDeployRecord: false };
    expect(resolveInstanceListHasDeployRecord(noStage)).toBe(false);
  });

  it('latest 状态尚未拉到（null/undefined）时不判定为有部署记录', () => {
    expect(resolveInstanceListHasDeployRecord(null)).toBe(undefined);
    expect(resolveInstanceListHasDeployRecord(undefined)).toBe(undefined);
  });
});

describe('shouldResetDeployStatusOnEnvChange', () => {
  const env = (name: string) => ({ name }) as EnvOutput;

  it('环境选择器 refresh 重抛当前环境（名称未变）时不重置部署状态', () => {
    expect(shouldResetDeployStatusOnEnvChange(env('schnee'), 'schnee')).toBe(false);
  });

  it('切换到不同环境时重置部署状态（不回归）', () => {
    expect(shouldResetDeployStatusOnEnvChange(env('prod'), 'schnee')).toBe(true);
  });

  it('当前环境为空时（首次选择环境）允许重置', () => {
    expect(shouldResetDeployStatusOnEnvChange(env('schnee'), undefined)).toBe(true);
    expect(shouldResetDeployStatusOnEnvChange(env('schnee'), '')).toBe(true);
  });

  it('清空环境选择（env 缺失或 name 为空）时允许重置', () => {
    expect(shouldResetDeployStatusOnEnvChange(undefined, 'schnee')).toBe(true);
    expect(shouldResetDeployStatusOnEnvChange(env(''), 'schnee')).toBe(true);
  });
});
