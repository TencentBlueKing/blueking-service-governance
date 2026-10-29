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

import { shallowMount } from '@vue/test-utils';
import { MarkerSeverity } from 'monaco-editor';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import MsEditor from '~/components/monaco-editor/ms-editor.vue';

import { i18nGlobalMocks } from './helpers/mock-i18n';

import type { IMonacoEditorErrorMarkerItem } from '~/common/util';

const mocks = vi.hoisted(() => ({
  value: '',
  onChange: () => {},
  setMarkers: vi.fn(),
  setLanguage: vi.fn(),
}));

// 保留真实 YAML 解析与组件逻辑，只替换依赖浏览器布局和 worker 的 Monaco 实例。
vi.mock('monaco-editor', () => {
  const model = {
    getValue: () => mocks.value,
    setValue(value: string) {
      mocks.value = value;
      mocks.onChange();
    },
  };
  return {
    MarkerSeverity: { Error: 8 },
    languages: { registerCompletionItemProvider: () => ({ dispose() {} }) },
    editor: {
      create: (_element: HTMLElement, options: { value: string }) => {
        mocks.value = options.value;
        return {
          ...model,
          getModel: () => model,
          onDidChangeModelContent(listener: () => void) {
            mocks.onChange = listener;
          },
          dispose() {},
          updateOptions() {},
        };
      },
      setModelMarkers: mocks.setMarkers,
      setModelLanguage: mocks.setLanguage,
    },
  };
});
vi.mock('~/modules/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }));
vi.mock('monaco-editor/esm/vs/editor/editor.worker.js?worker', () => ({ default: class EditorWorker {} }));
vi.mock('monaco-editor/esm/vs/language/json/json.worker.js?worker', () => ({ default: class JsonWorker {} }));
vi.mock('monaco-editor/esm/vs/language/css/css.worker.js?worker', () => ({ default: class CssWorker {} }));
vi.mock('monaco-editor/esm/vs/language/html/html.worker.js?worker', () => ({ default: class HtmlWorker {} }));
vi.mock('monaco-editor/esm/vs/language/typescript/ts.worker.js?worker', () => ({ default: class TsWorker {} }));

let wrapper: ReturnType<typeof shallowMount<typeof MsEditor>>;

function lastErrors() {
  return wrapper.emitted('error')!.at(-1)![0];
}

function mountEditor(modelValue: string, lang = 'yaml') {
  wrapper = shallowMount(MsEditor, { props: { modelValue, lang }, global: { mocks: i18nGlobalMocks } });
  return wrapper;
}

beforeEach(() => vi.clearAllMocks());
afterEach(() => wrapper?.unmount());

describe('编辑器 YAML 校验状态', () => {
  it('初始化时校验已有内容并设置错误标记', () => {
    mountEditor('key: [invalid');
    expect(lastErrors()).toHaveLength(1);
    expect(mocks.setMarkers).toHaveBeenLastCalledWith(expect.anything(), 'error', lastErrors());
  });

  it('重新载入相同错误内容时仍发送校验结果', () => {
    mountEditor('key: [invalid');
    const errors = lastErrors();
    const count = wrapper.emitted('error')!.length;
    wrapper.vm.setValue('key: [invalid');
    expect(wrapper.emitted('error')).toHaveLength(count + 1);
    expect(lastErrors()).toEqual(errors);
  });

  it('内容不变时切换语言会新增或清除 YAML 错误', async () => {
    mountEditor('key: [invalid', 'plaintext');
    expect(lastErrors()).toEqual([]);
    await wrapper.setProps({ lang: 'yaml' });
    expect(lastErrors()).toHaveLength(1);
    await wrapper.setProps({ lang: 'json' });
    expect(lastErrors()).toEqual([]);
    expect(mocks.setMarkers).toHaveBeenLastCalledWith(expect.anything(), 'error', []);
    await wrapper.setProps({ lang: 'yaml' });
    expect(lastErrors()).toHaveLength(1);
  });

  it('修正或清空内容后立即恢复校验通过', () => {
    mountEditor('key: [invalid');
    wrapper.vm.setValue('key: value');
    expect(lastErrors()).toEqual([]);
    expect(wrapper.vm.validate()).toBe(false);
    wrapper.vm.setValue('key: [invalid');
    expect(wrapper.vm.validate()).toBe(true);
    wrapper.vm.setValue('');
    expect(lastErrors()).toEqual([]);
    expect(wrapper.vm.validate()).toBe(false);
  });

  it('保存时校验模型中的最新内容', () => {
    mountEditor('key: value');
    mocks.value = 'key: [invalid';
    expect(wrapper.vm.validate()).toBe(true);
    expect(lastErrors()).toHaveLength(1);
  });

  it.each(['json', 'xml', 'ini', 'shell', 'plaintext'])('%s 不执行 YAML 校验', lang => {
    mountEditor('key: [invalid', lang);
    expect(wrapper.vm.validate()).toBe(false);
    expect(lastErrors()).toEqual([]);
  });

  it('初始化时保留已有的自定义校验', () => {
    const marker: IMonacoEditorErrorMarkerItem = {
      severity: MarkerSeverity.Error,
      message: 'required',
      startLineNumber: 1,
      startColumn: 1,
      endLineNumber: 1,
      endColumn: 1,
    };
    wrapper = shallowMount(MsEditor, {
      props: { modelValue: '', validator: [() => [marker]] },
      global: { mocks: i18nGlobalMocks },
    });
    expect(wrapper.vm.validate()).toBe(true);
    expect(lastErrors()).toEqual([marker]);
  });
});
