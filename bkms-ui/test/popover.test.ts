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

import { cleanup, fireEvent, render, screen } from '@testing-library/vue';
import { Popover } from 'bkui-vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

async function renderPopover(name: string, trigger = 'hover') {
  render(Popover, {
    props: { renderDirective: 'show', popoverDelay: 100, trigger },
    slots: { default: `<button>${name}</button>`, content: `<div>${name}内容</div>` },
  });
  await vi.advanceTimersByTimeAsync(100);
  const reference = screen.getByRole('button', { name });
  const content = screen.getByText(`${name}内容`).closest('.bk-pop2-content') as HTMLElement;
  return { reference, content };
}

beforeEach(() => vi.useFakeTimers());
afterEach(() => {
  cleanup();
  vi.clearAllTimers();
  vi.useRealTimers();
});

describe('Popover 悬浮隐藏', () => {
  it('从触发元素进入内容后快速移出，仍应在隐藏延迟后收起', async () => {
    const { reference, content } = await renderPopover('环境标签');
    await fireEvent.mouseEnter(reference);
    await vi.advanceTimersByTimeAsync(100);
    expect(content.style.display).toBe('block');

    await fireEvent.mouseLeave(reference);
    await fireEvent.mouseEnter(content);
    await vi.advanceTimersByTimeAsync(10);
    await fireEvent.mouseLeave(content);
    await vi.advanceTimersByTimeAsync(200);

    expect(content.style.display).toBe('none');
  });

  it('快速离开上一行内容并悬浮下一行，不应残留两个浮层', async () => {
    const first = await renderPopover('第一行');
    const second = await renderPopover('第二行');
    await fireEvent.mouseEnter(first.reference);
    await vi.advanceTimersByTimeAsync(100);
    await fireEvent.mouseLeave(first.reference);
    await fireEvent.mouseEnter(first.content);
    await vi.advanceTimersByTimeAsync(10);
    await fireEvent.mouseLeave(first.content);
    await fireEvent.mouseEnter(second.reference);
    await vi.advanceTimersByTimeAsync(200);

    expect(first.content.style.display).toBe('none');
    expect(second.content.style.display).toBe('block');
  });

  it('停留在内容区域时应取消触发元素的待隐藏任务', async () => {
    const { reference, content } = await renderPopover('环境明细');
    await fireEvent.mouseEnter(reference);
    await vi.advanceTimersByTimeAsync(100);
    await fireEvent.mouseLeave(reference);
    await fireEvent.mouseEnter(content);
    await vi.advanceTimersByTimeAsync(200);
    expect(content.style.display).toBe('block');

    await fireEvent.mouseLeave(content);
    await vi.advanceTimersByTimeAsync(100);
    expect(content.style.display).toBe('none');
  });

  it('click 模式不应因移出内容而收起', async () => {
    const { reference, content } = await renderPopover('点击标签', 'click');
    await fireEvent.click(reference);
    await vi.advanceTimersByTimeAsync(100);
    await fireEvent.mouseEnter(content);
    await fireEvent.mouseLeave(content);
    await vi.advanceTimersByTimeAsync(200);

    expect(content.style.display).toBe('block');
  });
});
