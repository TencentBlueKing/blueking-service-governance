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

/**
 * 场景级测试：实例列表 WeTerm 登录（composable，ROUTES 未收录，本头即场景说明）
 *
 * V = 5（判定节点：入参是否完整、接口成功、接口返回无链接、NOT_IMPLEMENTED、其他错误）
 * 路径：① 成功返回链接 → 跳转 weterm:// 并提示唤起 ② 返回无链接 → 不跳转
 *      ③ NOT_IMPLEMENTED → 部署级提示且不弹通用错误 ④ 其他错误 → 复用统一错误提示
 *      ⑤ 登录进行中连点 → 只发一次请求，结束后可再次发起
 * stub/mock：API / Message / showApiErrorMessage / app-detail store；跳转通过 navigate 注入，
 *            原因是 jsdom 对自定义协议导航无实现，window.location 不可重定义。
 */

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { type AppInstanceOutputObj } from '~/@types/v1/instance';

const createAppInstanceWeTerm = vi.hoisted(() => vi.fn());
const message = vi.hoisted(() => vi.fn());
const showApiErrorMessage = vi.hoisted(() => vi.fn());

vi.mock('~/api/modules/v1', () => ({ InstanceService: { createAppInstanceWeTerm } }));
vi.mock('bkui-vue', () => ({ Message: message }));
vi.mock('~/common/util', () => ({ showApiErrorMessage }));
vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => ({ appID: 'app-1' }) }));
vi.mock('vue-i18n', async () => (await import('./helpers/mock-i18n')).i18nMockFactory());

const { useWeTermLogin } = await import('~/pages/application/detail/deploy/instance-list/composables/use-weterm-login');

const instance: AppInstanceOutputObj = { id: 'pod-1' };
const instanceWithoutId: AppInstanceOutputObj = { id: '' };

function setup() {
  const navigate = vi.fn();
  const { loginWithWeTerm } = useWeTermLogin({ navigate });
  return { loginWithWeTerm, navigate };
}

describe('实例列表：WeTerm 登录', () => {
  beforeEach(() => {
    createAppInstanceWeTerm.mockReset();
    message.mockReset();
    showApiErrorMessage.mockReset();
  });

  it('当接口返回登录链接时，应跳转 weterm:// 并提示唤起客户端', async () => {
    createAppInstanceWeTerm.mockResolvedValue({ url: 'weterm://cluster/ns/pod-1' });
    const { loginWithWeTerm, navigate } = setup();

    await loginWithWeTerm(instance, 'prod');

    expect(createAppInstanceWeTerm).toHaveBeenCalledWith(
      { appID: 'app-1', envName: 'prod', instanceID: 'pod-1' },
      { interceptorErr: false },
    );
    expect(navigate).toHaveBeenCalledWith('weterm://cluster/ns/pod-1');
    expect(message).toHaveBeenCalledWith({
      message: '正在唤起 WeTerm 客户端，如未唤起请先安装 WeTerm 客户端',
    });
  });

  it('当接口未返回链接时，不应跳转也不提示', async () => {
    createAppInstanceWeTerm.mockResolvedValue({});
    const { loginWithWeTerm, navigate } = setup();

    await loginWithWeTerm(instance, 'prod');

    expect(navigate).not.toHaveBeenCalled();
    expect(message).not.toHaveBeenCalled();
  });

  it('当接口返回 NOT_IMPLEMENTED 时，应提示部署未配置且不弹通用错误', async () => {
    createAppInstanceWeTerm.mockRejectedValue({
      error: { code: 'NOT_IMPLEMENTED', message: 'weterm env not configured' },
    });
    const { loginWithWeTerm, navigate } = setup();

    await loginWithWeTerm(instance, 'prod');

    expect(navigate).not.toHaveBeenCalled();
    expect(message).toHaveBeenCalledWith({
      theme: 'warning',
      message: '当前部署未配置 WeTerm 环境，暂不支持 WeTerm 登录',
    });
    expect(showApiErrorMessage).not.toHaveBeenCalled();
  });

  it('当接口返回其他错误时，应复用统一错误提示以保留 traceId', async () => {
    const apiError = { error: { code: 'INTERNAL', message: 'boom', traceId: 'trace-1' } };
    createAppInstanceWeTerm.mockRejectedValue(apiError);
    const { loginWithWeTerm, navigate } = setup();

    await loginWithWeTerm(instance, 'prod');

    expect(navigate).not.toHaveBeenCalled();
    expect(showApiErrorMessage).toHaveBeenCalledWith(apiError);
    expect(message).not.toHaveBeenCalled();
  });

  it('当登录进行中重复点击时，应只发一次请求且结束后可再次发起', async () => {
    let release: (value: unknown) => void = () => {};
    createAppInstanceWeTerm.mockImplementation(
      () =>
        new Promise(resolve => {
          release = resolve;
        }),
    );
    const { loginWithWeTerm, navigate } = setup();

    const first = loginWithWeTerm(instance, 'prod');
    const duplicated = loginWithWeTerm(instance, 'prod');
    await Promise.resolve();
    expect(createAppInstanceWeTerm).toHaveBeenCalledTimes(1);

    release({ url: 'weterm://cluster/ns/pod-1' });
    await Promise.all([first, duplicated]);
    expect(navigate).toHaveBeenCalledTimes(1);

    createAppInstanceWeTerm.mockResolvedValue({ url: 'weterm://cluster/ns/pod-1' });
    await loginWithWeTerm(instance, 'prod');
    expect(createAppInstanceWeTerm).toHaveBeenCalledTimes(2);
  });

  it('当实例缺少 id 或环境名为空时，不应发起请求', async () => {
    const { loginWithWeTerm } = setup();

    await loginWithWeTerm(instanceWithoutId, 'prod');
    await loginWithWeTerm(instance, '');

    expect(createAppInstanceWeTerm).not.toHaveBeenCalled();
  });
});
