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

import { describe, expect, it } from 'vitest';
import {
  groupDeployedEnvs,
  isDeployedEnvType,
  isFeatureEnvKind,
  resolveDeployedEnvDisplayName,
} from '~/composables/use-deployed-envs';

/**
 * 覆盖制品管理「已部署环境」的分组规则（特性环境适配需求）：
 * 特性环境的 envType 继承自来源环境，「已部署环境」列表达的是环境分类，
 * 因此特性环境与其继承的分类合并计数、不单列「特性」；只部署过特性环境时也要能正常展示。
 */

describe('groupDeployedEnvs', () => {
  it('按环境分类归组，顺序为开发、测试、预发布、生产', () => {
    const groups = groupDeployedEnvs([
      { envKind: 'standard', envName: 'prod', envType: 'production' },
      { envKind: 'standard', envName: 'stag', envType: 'staging' },
      { envKind: 'standard', envName: 'dev', envType: 'development' },
      { envKind: 'standard', envName: 'test', envType: 'test' },
    ]);

    expect(groups.map(group => group.key)).toEqual(['development', 'test', 'staging', 'production']);
    expect(groups.map(group => group.label)).toEqual(['开发', '测试', '预发布', '生产']);
  });

  it('特性环境并入其继承的环境分类，不再单列「特性」分组', () => {
    const groups = groupDeployedEnvs([
      { envDisplayName: '开发', envKind: 'standard', envName: 'dev', envType: 'development' },
      { envDisplayName: '需求联调', envKind: 'feature', envName: 'feat-demo-abcde', envType: 'development' },
      { envDisplayName: '特性生产', envKind: 'feature', envName: 'feat-prod-xyz', envType: 'production' },
    ]);

    expect(groups.map(group => group.key)).toEqual(['development', 'production']);
    expect(groups[0]).toMatchObject({ count: 2, displayNames: ['开发', '需求联调'], label: '开发' });
    expect(groups[1]).toMatchObject({ count: 1, displayNames: ['特性生产'], label: '生产' });
  });

  it('只部署到特性环境时仍能按分类展示，列表不再显示为空', () => {
    const groups = groupDeployedEnvs([
      { envDisplayName: '需求联调', envKind: 'feature', envName: 'feat-demo-abcde', envType: 'development' },
    ]);

    expect(groups).toHaveLength(1);
    expect(groups[0]).toMatchObject({ count: 1, displayNames: ['需求联调'], key: 'development' });
  });

  it('展示名优先取接口返回的 envDisplayName，缺失时回退翻译函数再回退内部名', () => {
    const translate = (envName: string) => (envName === 'feat-demo-abcde' ? '需求联调' : envName);
    const groups = groupDeployedEnvs(
      [
        { envDisplayName: '后端给的展示名', envKind: 'standard', envName: 'dev', envType: 'development' },
        { envKind: 'feature', envName: 'feat-demo-abcde', envType: 'development' },
        { envKind: 'feature', envName: 'feat-unknown', envType: 'development' },
      ],
      translate,
    );

    expect(groups[0].displayNames).toEqual(['后端给的展示名', '需求联调', 'feat-unknown']);
  });

  it('环境类型不属于标准分类时不计入分组', () => {
    expect(groupDeployedEnvs([{ envKind: 'feature', envName: 'feat-x', envType: 'unknown' }])).toEqual([]);
  });

  it('没有部署环境时返回空数组', () => {
    expect(groupDeployedEnvs(undefined)).toEqual([]);
    expect(groupDeployedEnvs([])).toEqual([]);
  });
});

describe('isFeatureEnvKind', () => {
  it('仅 envKind 为 feature 时判定为特性环境，环境类型不参与判断', () => {
    expect(isFeatureEnvKind('feature')).toBe(true);
    expect(isFeatureEnvKind('standard')).toBe(false);
    expect(isFeatureEnvKind(undefined)).toBe(false);
  });
});

describe('isDeployedEnvType', () => {
  it('仅识别开发、测试、预发布、生产四种标准分类', () => {
    expect(isDeployedEnvType('development')).toBe(true);
    expect(isDeployedEnvType('production')).toBe(true);
    expect(isDeployedEnvType('feature')).toBe(false);
    expect(isDeployedEnvType(undefined)).toBe(false);
  });
});

describe('resolveDeployedEnvDisplayName', () => {
  it('环境名为空时展示占位符', () => {
    expect(resolveDeployedEnvDisplayName({})).toBe('--');
  });
});
