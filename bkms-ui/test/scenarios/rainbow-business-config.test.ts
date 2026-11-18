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

import { nextTick } from 'vue';

import { fireEvent, render, screen } from '@testing-library/vue';
import { flushPromises } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createMemoryHistory, createRouter } from 'vue-router';
import BusinessConfig from '~/pages/application/detail/business-config/index.vue';
import { useAppDetail } from '~/stores/app-detail';

import { i18nGlobalMocks } from '../helpers/mock-i18n';

import type * as BkuiVue from 'bkui-vue';

const mocks = vi.hoisted(() => ({
  delete: vi.fn(),
  get: vi.fn(),
  message: vi.fn(),
  put: vi.fn(),
  showApiErrorMessage: vi.fn(),
}));
const savedUrl = 'https://config.example.test/console/#/project/1?env=prod&tab=config';
const updatedUrl = 'https://config.example.test/console/#/project/2?env=test&tab=config';
const businessConfigPath = '/space/app/demo/trpc/businessConfig';
const disableHint = '关闭后，当前应用将不再嵌入七彩石页面，七彩石上的原有配置不受影响。';

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('~/stores/app-detail', async () => {
  const { reactive } = await import('vue');
  const store = reactive({ appID: '' });
  return { useAppDetail: () => store };
});
vi.mock('~/api/modules/v1/rainbowtconf', () => ({
  RainbowtconfService: {
    deleteRainbowTConf: mocks.delete,
    getRainbowTConf: mocks.get,
    putRainbowTConf: mocks.put,
  },
}));
vi.mock('~/common/util', () => ({ showApiErrorMessage: mocks.showApiErrorMessage }));
vi.mock('bkui-vue', async importOriginal => ({
  ...(await importOriginal<typeof BkuiVue>()),
  Message: mocks.message,
}));

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}

async function openDialog(button = 0) {
  await fireEvent.click(screen.getAllByRole('button', { name: '配置七彩石' })[button]);
  return screen.findByPlaceholderText('请输入');
}

async function openPage(query = 'activeTab=rainbow') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: businessConfigPath, component: BusinessConfig }],
  });
  await router.push(`${businessConfigPath}?${query}`);
  const page = render(BusinessConfig, {
    global: { mocks: i18nGlobalMocks, plugins: [router], directives: { bkloading: () => {}, 'bk-tooltips': () => {} } },
  });
  await flushPromises();
  return { ...page, router };
}

function permissionDenied(developer?: string) {
  return { error: { code: 'IAM_NO_PERMISSION', details: [{ code: 'IAM_NO_PERMISSION', extras: { developer } }] } };
}

async function submit() {
  await fireEvent.click(screen.getByRole('button', { name: '确定' }));
  await flushPromises();
}

function toggleRainbow() {
  return fireEvent.click(screen.getByRole('switch', { name: '启用七彩石' }));
}

describe('业务配置 · 七彩石', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    useAppDetail().appID = 'app-one';
    vi.stubEnv('BK_IAM_URL', 'https://iam.example.test/');
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('通过右上角和空态打开配置，首次保存直接嵌入响应地址', async () => {
    mocks.get.mockResolvedValue({ url: '' });
    mocks.put.mockResolvedValue({ url: savedUrl });
    const { router } = await openPage('activeTab=file-config&keep=1');

    expect(router.currentRoute.value.query).toEqual({ activeTab: 'rainbow', keep: '1' });
    expect(screen.getByText('七彩石', { selector: '.bk-tab-header--active > div' })).toBeVisible();
    expect(screen.getByText('暂未配置七彩石')).toBeVisible();
    expect(await openDialog()).toHaveValue('');
    expect(screen.queryByRole('switch', { name: '启用七彩石' })).toBeNull();
    await fireEvent.click(screen.getByRole('button', { name: '取消' }));
    const input = await openDialog(1);
    await fireEvent.update(input, `  ${savedUrl}  `);
    await submit();

    expect(mocks.put).toHaveBeenCalledExactlyOnceWith({ appID: 'app-one', url: savedUrl }, { interceptorErr: false });
    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', savedUrl);
    expect(screen.queryByText('暂未配置七彩石')).toBeNull();
    expect(mocks.get).toHaveBeenCalledTimes(1);
    expect(await openDialog()).toHaveValue(savedUrl);
    expect(screen.getByRole('switch', { name: '启用七彩石' })).toBeChecked();
  });

  it.each([
    ['   ', '请输入七彩石页面地址'],
    ['http://config.example.test/', '请输入有效的 HTTPS 地址'],
    ['https:///console/', '请输入有效的 HTTPS 地址'],
    ['https://user:password@config.example.test/', '七彩石页面地址不能包含用户名或密码'],
    [`https://config.example.test/${'界'.repeat(2049)}`, '七彩石页面地址不能超过 2048 个字符'],
  ])('阻止提交不符合要求的地址 %#', async (url, message) => {
    mocks.get.mockResolvedValue({ url: savedUrl });
    await openPage();

    const input = await openDialog();
    expect(input).toHaveValue(savedUrl);
    await fireEvent.update(input, url);
    await submit();

    expect(screen.getByRole('alert')).toHaveTextContent(message);
    expect(mocks.put).not.toHaveBeenCalled();
    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', savedUrl);
  });

  it('后端地址校验错误显示在输入框下，编辑后清除并可再次提交', async () => {
    const failure = { error: { code: 'INVALID_ARGUMENT', message: 'invalid url' } };
    mocks.get.mockResolvedValue({ url: savedUrl });
    mocks.put.mockRejectedValueOnce(failure).mockResolvedValueOnce({ url: updatedUrl });
    await openPage();

    const input = await openDialog();
    const rejectedUrl = `${updatedUrl}&invalid=1`;
    await fireEvent.update(input, rejectedUrl);
    await submit();
    expect(screen.getByRole('alert')).toHaveTextContent('七彩石页面地址不合法，请检查后重试');
    expect(mocks.showApiErrorMessage).not.toHaveBeenCalled();
    expect(input).toHaveAttribute('aria-describedby', expect.stringContaining('rainbow-url-error'));
    expect(input.closest('.bk-form-item')).toHaveClass('is-error');
    expect(input).toHaveValue(rejectedUrl);
    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', savedUrl);

    await fireEvent.update(input, updatedUrl);
    expect(screen.queryByRole('alert')).toBeNull();
    expect(input.closest('.bk-form-item')).not.toHaveClass('is-error');
    await submit();
    expect(mocks.put).toHaveBeenLastCalledWith({ appID: 'app-one', url: updatedUrl }, { interceptorErr: false });
    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', updatedUrl);
    expect(mocks.get).toHaveBeenCalledTimes(1);
  });

  it('保存出现一般错误时保留旧 iframe 和输入内容，重试后展示修改后的页面', async () => {
    const failure = { error: { code: 'INTERNAL_SERVER_ERROR', message: 'save failed' } };
    mocks.get.mockResolvedValue({ url: savedUrl });
    mocks.put.mockRejectedValueOnce(failure).mockResolvedValueOnce({ url: updatedUrl });
    await openPage();

    const input = await openDialog();
    await fireEvent.update(input, updatedUrl);
    await submit();
    expect(mocks.showApiErrorMessage).toHaveBeenCalledWith(failure);
    expect(screen.queryByRole('alert')).toBeNull();
    expect(input).toHaveValue(updatedUrl);
    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', savedUrl);

    await submit();
    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', updatedUrl);
    expect(mocks.get).toHaveBeenCalledTimes(1);
  });

  it('查询失败提供重试，加载完成前不显示未配置空态', async () => {
    mocks.get.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce({ url: '' });
    await openPage();

    expect(screen.getByText('七彩石配置加载失败')).toBeVisible();
    expect(screen.queryByText('暂未配置七彩石')).toBeNull();
    expect(screen.getByRole('button', { name: '配置七彩石' })).toBeDisabled();
    await fireEvent.click(screen.getByRole('button', { name: '重试' }));

    expect(await screen.findByText('暂未配置七彩石')).toBeVisible();
  });

  it('查看权限缺失时与其他查询失败一致，展示加载失败且不出现空态和申请入口', async () => {
    mocks.get.mockRejectedValue(permissionDenied());
    await openPage();

    expect(screen.getByText('七彩石配置加载失败')).toBeVisible();
    expect(screen.queryByText('暂未配置七彩石')).toBeNull();
    expect(screen.queryByRole('link', { name: '申请成为开发者' })).toBeNull();
    expect(screen.getByRole('button', { name: '配置七彩石' })).toBeDisabled();
    expect(screen.getByRole('button', { name: '重试' })).toBeVisible();
  });

  it('保存权限缺失时提供申请入口并保留原配置', async () => {
    mocks.get.mockResolvedValue({ url: savedUrl });
    mocks.put.mockRejectedValue(permissionDenied('team/developer'));
    await openPage();

    const input = await openDialog();
    await fireEvent.update(input, updatedUrl);
    await submit();

    const link = screen.getByRole('link', { name: '申请成为开发者' });
    expect(link).toHaveAttribute('href', 'https://iam.example.test/apply-join-user-group?id=team%2Fdeveloper');
    expect(link).toHaveAttribute('target', '_blank');
    expect(input).toHaveValue(updatedUrl);
    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', savedUrl);
  });

  it('关闭开关只修改草稿，重新开启恢复输入，取消后保留原地址', async () => {
    mocks.get.mockResolvedValue({ url: savedUrl });
    await openPage();

    const input = await openDialog();
    await fireEvent.update(input, updatedUrl);
    await toggleRainbow();
    expect(screen.getByRole('switch', { name: '启用七彩石' })).not.toBeChecked();
    expect(screen.getByText(disableHint)).toBeVisible();
    expect(screen.queryByPlaceholderText('请输入')).toBeNull();
    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', savedUrl);
    expect(mocks.delete).not.toHaveBeenCalled();

    await toggleRainbow();
    expect(screen.getByPlaceholderText('请输入')).toHaveValue(updatedUrl);
    expect(screen.queryByText(disableHint)).toBeNull();
    await toggleRainbow();
    await fireEvent.click(screen.getByRole('button', { name: '取消' }));
    expect(mocks.delete).not.toHaveBeenCalled();
    expect(mocks.put).not.toHaveBeenCalled();
    expect(await openDialog()).toHaveValue(savedUrl);
    expect(screen.getByRole('switch', { name: '启用七彩石' })).toBeChecked();
  });

  it('确认关闭后只删除一次并直接展示空态，可以重新配置', async () => {
    // DELETE 返回的空对象没有 data 字段，v1Fetch 解包后得到 undefined。
    const pendingDelete = deferred<undefined>();
    mocks.get.mockResolvedValue({ url: savedUrl });
    mocks.delete.mockReturnValue(pendingDelete.promise);
    mocks.put.mockResolvedValue({ url: updatedUrl });
    await openPage();

    await fireEvent.update(await openDialog(), '不合法的草稿地址');
    await toggleRainbow();
    await submit();
    await submit();
    expect(mocks.delete).toHaveBeenCalledExactlyOnceWith({ appID: 'app-one' }, { interceptorErr: false });
    expect(mocks.put).not.toHaveBeenCalled();
    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', savedUrl);

    pendingDelete.resolve(undefined);
    await flushPromises();
    expect(screen.getByText('暂未配置七彩石')).toBeVisible();
    expect(screen.queryByTitle('七彩石')).toBeNull();
    expect(mocks.get).toHaveBeenCalledTimes(1);
    expect(mocks.message).toHaveBeenCalledWith({ message: '保存成功', theme: 'success' });

    const input = await openDialog(1);
    expect(input).toHaveValue('');
    expect(screen.queryByRole('switch', { name: '启用七彩石' })).toBeNull();
    await fireEvent.update(input, updatedUrl);
    await submit();
    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', updatedUrl);
  });

  it('删除失败保留原 iframe 和关闭草稿，点击确定可以重试', async () => {
    const failure = { error: { code: 'INTERNAL_SERVER_ERROR', message: 'delete failed' } };
    mocks.get.mockResolvedValue({ url: savedUrl });
    mocks.delete.mockRejectedValueOnce(failure).mockResolvedValueOnce(undefined);
    await openPage();

    await openDialog();
    await toggleRainbow();
    await submit();
    expect(mocks.showApiErrorMessage).toHaveBeenCalledWith(failure);
    expect(screen.getByRole('switch', { name: '启用七彩石' })).not.toBeChecked();
    expect(screen.queryByPlaceholderText('请输入')).toBeNull();
    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', savedUrl);

    await submit();
    expect(mocks.delete).toHaveBeenCalledTimes(2);
    expect(screen.getByText('暂未配置七彩石')).toBeVisible();
  });

  it('删除权限缺失沿用申请入口，保留原 iframe 和关闭草稿', async () => {
    mocks.get.mockResolvedValue({ url: savedUrl });
    mocks.delete.mockRejectedValue(permissionDenied('team/developer'));
    await openPage();

    await openDialog();
    await toggleRainbow();
    await submit();
    expect(screen.getByRole('link', { name: '申请成为开发者' })).toHaveAttribute(
      'href',
      'https://iam.example.test/apply-join-user-group?id=team%2Fdeveloper',
    );
    expect(screen.getByRole('switch', { name: '启用七彩石' })).not.toBeChecked();
    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', savedUrl);
    expect(mocks.showApiErrorMessage).not.toHaveBeenCalled();
  });

  it('丢弃切换应用后迟到的查询结果', async () => {
    const oldGet = deferred<{ url: string }>();
    mocks.get.mockReturnValueOnce(oldGet.promise).mockResolvedValueOnce({ url: updatedUrl });
    await openPage();
    expect(screen.queryByText('暂未配置七彩石')).toBeNull();

    useAppDetail().appID = 'app-two';
    await nextTick();
    expect(await screen.findByTitle('七彩石')).toHaveAttribute('src', updatedUrl);
    oldGet.resolve({ url: savedUrl });
    await flushPromises();

    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', updatedUrl);
  });

  it('防止重复提交，同一应用重新打开时等待保存完成后读取最新地址', async () => {
    const oldPut = deferred<{ url: string }>();
    mocks.get.mockResolvedValueOnce({ url: savedUrl }).mockResolvedValueOnce({ url: updatedUrl });
    mocks.put.mockReturnValue(oldPut.promise);
    const page = await openPage();

    const input = await openDialog();
    await fireEvent.update(input, updatedUrl);
    await submit();
    await submit();
    expect(mocks.put).toHaveBeenCalledTimes(1);

    page.unmount();
    await openPage();
    expect(mocks.get).toHaveBeenCalledTimes(1);
    expect(screen.getByRole('button', { name: '配置七彩石' })).toBeDisabled();
    expect(screen.queryByTitle('七彩石')).toBeNull();

    oldPut.resolve({ url: updatedUrl });
    await flushPromises();

    expect(mocks.get).toHaveBeenCalledTimes(2);
    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', updatedUrl);
    expect(mocks.message).not.toHaveBeenCalled();
  });

  it('同一应用重新打开时等待在途删除完成，再读取未配置状态', async () => {
    const pendingDelete = deferred<undefined>();
    mocks.get.mockResolvedValueOnce({ url: savedUrl }).mockResolvedValueOnce({ url: '' });
    mocks.delete.mockReturnValue(pendingDelete.promise);
    const page = await openPage();

    await openDialog();
    await toggleRainbow();
    await submit();
    page.unmount();

    await openPage();
    expect(mocks.get).toHaveBeenCalledTimes(1);
    expect(screen.getByRole('button', { name: '配置七彩石' })).toBeDisabled();

    pendingDelete.resolve(undefined);
    await flushPromises();
    expect(mocks.get).toHaveBeenCalledTimes(2);
    expect(screen.getByText('暂未配置七彩石')).toBeVisible();
    expect(mocks.message).not.toHaveBeenCalled();
  });

  it.each(['put', 'delete'] as const)('离开等待 %s 的页面后，其他应用立即读取自己的地址', async method => {
    const otherAppUrl = 'https://config.example.test/console/#/project/3';
    const pendingWrite = deferred<undefined | { url: string }>();
    mocks.get.mockResolvedValueOnce({ url: savedUrl }).mockResolvedValueOnce({ url: otherAppUrl });
    mocks[method].mockReturnValue(pendingWrite.promise);
    const firstPage = await openPage();
    const input = await openDialog();
    if (method === 'put') {
      await fireEvent.update(input, updatedUrl);
    } else {
      await toggleRainbow();
    }
    await submit();
    firstPage.unmount();

    const waitingPage = await openPage();
    expect(mocks.get).toHaveBeenCalledTimes(1);
    waitingPage.unmount();
    useAppDetail().appID = 'app-two';
    await openPage();
    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', otherAppUrl);

    pendingWrite.resolve(method === 'put' ? { url: updatedUrl } : undefined);
    await flushPromises();
    expect(mocks.get).toHaveBeenCalledTimes(2);
    expect(mocks.get).toHaveBeenLastCalledWith({ appID: 'app-two' }, { interceptorErr: false });
    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', otherAppUrl);
    expect(mocks.message).not.toHaveBeenCalled();
  });

  it('旧页面保存失败后，重新进入同一应用仍能读取并提交', async () => {
    const oldPut = deferred<{ url: string }>();
    mocks.get.mockResolvedValue({ url: savedUrl });
    mocks.put.mockReturnValueOnce(oldPut.promise).mockResolvedValueOnce({ url: updatedUrl });
    const page = await openPage();
    await fireEvent.update(await openDialog(), updatedUrl);
    await submit();
    page.unmount();

    await openPage();
    expect(mocks.get).toHaveBeenCalledTimes(1);
    oldPut.reject(new Error('offline'));
    await flushPromises();
    expect(mocks.get).toHaveBeenCalledTimes(2);
    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', savedUrl);
    expect(mocks.showApiErrorMessage).not.toHaveBeenCalled();
    expect(mocks.message).not.toHaveBeenCalled();

    await fireEvent.update(await openDialog(), updatedUrl);
    await submit();
    expect(mocks.put).toHaveBeenCalledTimes(2);
    expect(screen.getByTitle('七彩石')).toHaveAttribute('src', updatedUrl);
  });
});
