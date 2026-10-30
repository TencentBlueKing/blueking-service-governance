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

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useDeployAPIs } from '~/pages/application/detail/deploy/use-deploy';

const request = vi.hoisted(() => vi.fn().mockResolvedValue({}));
vi.mock('~/api/clients', () => ({
  v1Prefix: '/api',
  v1Fetch: Object.fromEntries(
    ['get', 'post', 'put', 'delete'].map(method => [
      method,
      (path: string) => (params: unknown, config: unknown) => request(method, path, params, config),
    ]),
  ),
}));
vi.mock('~/common/util', () => ({ downloadByLink: vi.fn() }));

beforeEach(() => request.mockClear());

describe('平台托管应用部署接口', () => {
  it.each(['trpc', 'taf', 'bkmsApp'] as const)('当用户部署 %s 应用时，应统一调用平台托管部署接口', async type => {
    const api = useDeployAPIs(type);
    const params = { appID: 'app-a', envName: 'test', imageTag: 'v1', replicas: 1 };
    await api.preCheckDeploy?.(params, { needRes: true });
    await api.createDeployDirectly?.(params);
    await api.listDeployRecords({ appID: 'app-a', envName: 'test', page: 1, pageSize: 10 });
    await api.listLatestDeployRecords?.(params);
    await api.deleteDeploy(params);
    await api.buildAndCreateDeploy?.({ appID: 'app-a', envName: 'test', branch: 'main', imageTag: 'v1', replicas: 1 });
    expect(request.mock.calls.map(([method, path]) => [method, path])).toEqual([
      ['get', '/apps/{appID}/envs/{envName}/bkmsapp-deploys/precheck'],
      ['post', '/apps/{appID}/envs/{envName}/bkmsapp-deploys'],
      ['get', '/apps/{appID}/envs/{envName}/bkmsapp-deploys'],
      ['get', '/apps/{appID}/envs/{envName}/bkmsapp-deploys/latest-status'],
      ['delete', '/apps/{appID}/envs/{envName}/bkmsapp-deploys'],
      ['post', '/apps/{appID}/envs/{envName}/build-deploys'],
    ]);
    expect(request).toHaveBeenNthCalledWith(1, 'get', expect.any(String), params, { needRes: true });
  });

  it.each(['helm', 'agones'] as const)('当用户查询或移除 %s 部署时，应继续使用 Helm 接口', async type => {
    const api = useDeployAPIs(type);
    await api.listDeployRecords({ appID: 'app-a', envName: 'test', page: 1, pageSize: 10 });
    await api.deleteDeploy({ appID: 'app-a', envName: 'test', deployID: 'deploy-a' });
    expect(request.mock.calls.map(([method, path]) => [method, path])).toEqual([
      ['get', '/apps/{appID}/envs/{envName}/helm-deploys'],
      ['delete', '/apps/{appID}/envs/{envName}/helm-deploys/{deployID}'],
    ]);
    expect(api.preCheckDeploy).toBeUndefined();
  });
});
