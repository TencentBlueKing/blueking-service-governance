<!--
 - TencentBlueKing is pleased to support the open source community by making
 - 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 - Copyright (C) Tencent. All rights reserved.
 - Licensed under the MIT License (the "License"); you may not use this file except
 - in compliance with the License. You may obtain a copy of the License at
 -
 -  http://opensource.org/licenses/MIT
 -
 - Unless required by applicable law or agreed to in writing, software distributed under
 - the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 - either express or implied. See the License for the specific language governing permissions and
 - limitations under the License.
 -
 - We undertake not to change the open source license (MIT license) applicable
 - to the current version of the project delivered to anyone in the future.
-->

<template>
  <div
    ref="editorRef"
    class="w-full h-full"
  >
    <!-- 工具栏 -->
    <div :class="['h-[40px] px-[16px] rounded-t-sm ms-editor', lightTheme ? 'bg-[#DCDEE5]' : 'bg-[#2E2E2E]']">
      <div class="flex items-center justify-between h-full">
        <div
          :class="['flex-1 ms-editor-title text-[14px] leading-none', lightTheme ? 'text-[#313238]' : 'text-[#C4C6CC]']"
        >
          <slot name="title">
            <div v-if="!isDiff">
              {{ title }}
            </div>
            <!-- diff 模式标题 -->
            <div
              v-else
              class="flex items-center h-full"
            >
              <div class="flex-1">{{ title }}</div>
              <div class="flex-1 pl-[18px]">{{ targetTitle }}</div>
            </div>
          </slot>
        </div>
        <div class="flex items-center ms-editor-tools">
          <slot
            :editor="editor"
            name="tools"
          ></slot>
          <IconButton
            v-if="showCopy"
            :desc="$t('复制')"
            @click="handleCopyContent"
          >
            <template #icon>
              <Copy
                color="#979BA5"
                height="16"
                :title="$t('复制')"
                width="16"
              />
            </template>
          </IconButton>
          <IconButton
            class="ml-[8px]"
            :desc="isFullscreen ? $t('退出全屏') : $t('全屏')"
            :icon="isFullscreen ? 'bkms-icon bkms-icon-un-full-screen-2' : 'bkms-icon bkms-icon-filliscreen-line'"
            @click="toggleFullscreen"
          />
        </div>
      </div>
    </div>
    <div
      ref="editorContentRef"
      class="h-[calc(100%-40px)] rounded-b-sm"
    ></div>
  </div>
</template>
<script lang="ts" setup>
  import type { PropType } from 'vue';
  import { computed, onBeforeMount, onMounted, ref, watch } from 'vue';

  import { Copy } from 'bkui-vue/lib/icon';
  import { isEqual } from 'lodash-es';
  import * as monaco from 'monaco-editor';
  import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker.js?worker';
  import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker.js?worker';
  import { copyText, validateYAML } from '~/common/util';

  import type { IMonacoEditorErrorMarkerItem } from '~/common/util';

  const props = defineProps({
    title: String,
    targetTitle: String, // diff模式二列标题
    modelValue: { type: String, default: () => '' }, // 编辑器值
    isDiff: { type: Boolean, default: () => false }, // 开启diff模式
    original: { type: String, default: () => '' }, // 只有在diff模式下有效, diff数据
    lang: { type: String, default: 'yaml' },
    theme: { type: String as PropType<'hc-black' | 'hc-light' | 'vs' | 'vs-dark'>, default: 'vs-dark' },
    readonly: { type: Boolean, default: false },
    showCopy: { type: Boolean, default: true }, // 是否显示复制按钮
    variables: { type: Array as PropType<string[]>, default: () => [] }, // 变量列表
    errorLines: { type: Array as PropType<Partial<IMonacoEditorErrorMarkerItem>[]>, default: () => [] }, // 错误行信息
    /** 自定义校验函数数组，每个函数接收编辑器值，返回错误标记列表 */
    validator: { type: Array as PropType<((value: string) => IMonacoEditorErrorMarkerItem[])[]>, default: () => [] },
    options: {
      type: Object as PropType<
        monaco.editor.IDiffEditorConstructionOptions | monaco.editor.IStandaloneEditorConstructionOptions
      >,
      default: () => ({}),
    },
    /** 全屏目标元素 ref，传入后全屏将作用于该元素而不是编辑器本身 */
    fullscreenTargetRef: { type: Object as PropType<HTMLElement | null>, default: null },
  });

  const emits = defineEmits<{
    (e: 'change' | 'update:modelValue', value: string): void;
    (e: 'error', data: IMonacoEditorErrorMarkerItem[]): void;
    (e: 'diff-stats', stats: { added: number; deleted: number }): void;
    (e: 'fullscreen', isFullscreen: boolean): void;
  }>();

  window.MonacoEnvironment = {
    getWorker(_workerId: string, label: string) {
      if (label === 'json') {
        return new jsonWorker();
      }

      return new editorWorker();
    },
  };

  const editorRef = ref<HTMLElement>();
  const editorContentRef = ref<HTMLElement>();

  const localValue = ref(props.modelValue);
  const localErrorLines = ref<IMonacoEditorErrorMarkerItem[]>([]);

  const lightTheme = computed(() => ['vs', 'hc-light'].includes(props.theme));

  // 全屏
  const isFullscreen = ref(false);
  // 复制
  function handleCopyContent() {
    copyText(localValue.value);
  }

  function handleFullscreenChange() {
    isFullscreen.value = !!document.fullscreenElement;
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        emits('fullscreen', isFullscreen.value);
      });
    });
  }

  function toggleFullscreen() {
    // 优先使用 fullscreenTargetRef，否则使用编辑器根元素
    const target = props.fullscreenTargetRef || editorRef.value;
    if (!document.fullscreenElement) {
      target?.requestFullscreen();
    } else {
      document.exitFullscreen();
    }
  }

  // 编辑器
  let editor: monaco.editor.IStandaloneCodeEditor | monaco.editor.IStandaloneDiffEditor | null = null;
  let editorVariableProvide: monaco.IDisposable | null = null;
  // 销毁编辑器
  function destroyMonaco() {
    editor?.dispose();
    editorVariableProvide?.dispose();
    editor = null;
    editorVariableProvide = null;
  }
  // 获取编辑器model
  function getModels() {
    if (!editor) return [];

    if (props.isDiff) {
      const diffModel = (editor as monaco.editor.IStandaloneDiffEditor)?.getModel();
      return [diffModel?.modified, diffModel?.original];
    }

    return [(editor as monaco.editor.IStandaloneCodeEditor)?.getModel()];
  }

  // 获取值
  function getValue() {
    if (!editor) return;
    if (props.isDiff) {
      return (editor as monaco.editor.IStandaloneDiffEditor)?.getModel()?.modified?.getValue();
    }
    return (editor as monaco.editor.IStandaloneCodeEditor)?.getValue();
  }

  // 内容变更事件
  function handleContentChange() {
    const code = getValue() || '';
    localValue.value = code;
    emits('update:modelValue', localValue.value);
    emits('change', localValue.value);
  }

  // 变量联想功能
  function initAutoCompletion() {
    editorVariableProvide = monaco.languages.registerCompletionItemProvider(props.lang, {
      triggerCharacters: ['{'], // 触发自动补全的字符
      provideCompletionItems(model: monaco.editor.ITextModel, position: monaco.Position) {
        const textBeforePosition = model.getValueInRange({
          startLineNumber: position.lineNumber,
          startColumn: 1,
          endLineNumber: position.lineNumber,
          endColumn: position.column,
        });
        // 根据当前的文本内容和光标位置，返回自动补全的候选项列表
        const variableSuggestions = props.variables?.map((item: string) => ({
          label: item, // 候选项的显示文本
          kind: monaco.languages.CompletionItemKind.Variable, // 候选项的类型
          insertText: item, // 插入光标后的文本
          range: new monaco.Range(position.lineNumber, position.column, position.lineNumber, position.column),
        }));

        if (textBeforePosition === '{{') {
          return {
            suggestions: [...variableSuggestions],
          };
        }
        return {
          suggestions: [],
        };
      },
      resolveCompletionItem: (item: monaco.languages.CompletionItem) => item,
    });
  }

  // 初始化编辑器
  function initMonaco() {
    if (!editorContentRef.value) {
      console.warn('editor el is null');
      return;
    }
    destroyMonaco();

    initAutoCompletion();
    const opt: monaco.editor.IStandaloneEditorConstructionOptions = {
      value: props.modelValue,
      language: props.lang,
      theme: props.theme,
      minimap: {
        enabled: false,
      },
      readOnly: props.readonly,
      automaticLayout: true,
      scrollbar: {
        alwaysConsumeMouseWheel: false,
      },
      contextmenu: false,
      tabSize: 2,
      wordWrap: 'on',
      scrollBeyondLastLine: false,
      fixedOverflowWidgets: true,
      overflowWidgetsDomNode: document.body,
      ...props.options,
    };
    if (props.isDiff) {
      // diff 模式
      const originalModel = monaco.editor.createModel(props.original, props.lang);
      const modifiedModel = monaco.editor.createModel(props.modelValue, props.lang);

      editor = monaco.editor.createDiffEditor(editorContentRef.value, {
        ...opt,
        originalEditable: false,
      });
      editor.setModel({
        original: originalModel,
        modified: modifiedModel,
      });
      (editor as monaco.editor.IStandaloneDiffEditor)?.getModel()?.modified?.onDidChangeContent(() => {
        handleContentChange();
      });
      // 监听 diff 更新，计算新增/删除行数
      (editor as monaco.editor.IStandaloneDiffEditor).onDidUpdateDiff(() => {
        const diffEditor = editor as monaco.editor.IStandaloneDiffEditor;
        const changes = (editor as monaco.editor.IStandaloneDiffEditor).getLineChanges();
        if (!changes) return;
        let added = 0;
        let deleted = 0;
        for (const change of changes) {
          if (change.originalEndLineNumber >= change.originalStartLineNumber) {
            deleted += change.originalEndLineNumber - change.originalStartLineNumber + 1;
          }
          if (change.modifiedEndLineNumber >= change.modifiedStartLineNumber) {
            added += change.modifiedEndLineNumber - change.modifiedStartLineNumber + 1;
          }
        }
        emits('diff-stats', { added, deleted });

        // 自动滚动到第一个差异行（modified 侧的行号）
        if (changes.length > 0) {
          const firstChange = changes[0];
          const targetLine = firstChange.modifiedStartLineNumber;
          if (targetLine > 0) {
            diffEditor.revealLineNearTop(targetLine, monaco.editor.ScrollType.Smooth);
          }
        }
      });
    } else {
      // 普通模式
      editor = monaco.editor.create(editorContentRef.value, opt);
      (editor as monaco.editor.IStandaloneCodeEditor)?.onDidChangeModelContent(() => {
        handleContentChange();
      });
    }
  }

  // 将某一行滚动到编辑器中心并选中
  // function setLineRevealAndSelected(lineNumber: number) {
  //   const models = getModels();
  //   const lineContent = models[0]?.getLineContent(lineNumber);
  //   if (!lineContent) return;

  //   const range = new monaco.Range(lineNumber, 1, lineNumber, lineContent.length + 1);
  //   editor?.revealLineInCenter(lineNumber);
  //   editor?.setSelection(range);
  // };

  // 容器大小变化时重新调整编辑器布局
  function layout() {
    editor?.layout();
  }

  // 返回滚动条顶部
  function scrollToTop() {
    editor?.revealLineNearTop(0);
  }

  // 添加错误行(外部定义)
  function setErrorLines() {
    if (props.errorLines?.length) return;
    // 创建错误标记列表
    const markers = props.errorLines
      .filter(item => item.startLineNumber !== undefined && item.message !== undefined) // 过滤无效的异常标记
      .map(item =>
        Object.assign(
          {
            startLineNumber: 0,
            endLineNumber: 0,
            startColumn: 1,
            endColumn: 200,
            message: '',
            severity: monaco.MarkerSeverity.Error,
          },
          item,
        ),
      );
    // 设置编辑器模型的标记(只设置能修改的model)
    const models = getModels();
    if (models[0]) {
      monaco.editor.setModelMarkers(models[0], 'error', markers);
    }
  }

  // 设置位置
  function setPosition(offset: number) {
    const models = getModels();
    const pos = models[0]?.getPositionAt(offset);
    if (!pos) return;

    editor?.revealPositionNearTop(pos);
  }

  // 设置值
  function setValue(value: string) {
    if (!editor) return;
    if (props.isDiff) {
      return (editor as monaco.editor.IStandaloneDiffEditor).getModel()?.modified?.setValue(value);
    }
    return (editor as monaco.editor.IStandaloneCodeEditor)?.setValue(value);
  }

  // 校验（需在编辑器初始化后调用，否则无法设置错误标记到 model）
  function validate() {
    // 编辑器未初始化时跳过，initMonaco 末尾会再次调用
    if (!editor) return false;

    // 校验不同语言格式（空内容时不跑 YAML 校验）
    const yamlErrors = localValue.value && props.lang === 'yaml' ? validateYAML(localValue.value) : [];
    // 自定义校验（调用每个 validator 函数，传入当前值）
    const customErrors = props.validator.flatMap(fn => fn(localValue.value));
    localErrorLines.value = [...yamlErrors, ...customErrors];

    // 设置编辑器模型的标记(只设置能修改的model)
    const models = getModels();
    if (models[0]) {
      monaco.editor.setModelMarkers(models[0], 'error', localErrorLines.value);
    }

    return !!localErrorLines.value.length;
  }

  watch(
    () => props.original,
    (newValue, oldValue) => {
      if (!editor || !props.isDiff || newValue === oldValue) return;

      (editor as monaco.editor.IStandaloneDiffEditor)?.getModel()?.original?.setValue(props.original);
    },
  );

  watch(
    () => props.modelValue,
    (newValue, oldValue) => {
      if (newValue === oldValue || localValue.value === newValue) return;
      setValue(props.modelValue);
    },
  );

  watch(
    () => props.lang,
    val => {
      const models = getModels();
      models.forEach(model => {
        if (model) {
          monaco.editor.setModelLanguage(model, val);
          // 清空错误行
          monaco.editor.setModelMarkers(model, 'error', []);
        }
      });
    },
  );

  watch(
    () => props.readonly,
    val => {
      editor?.updateOptions({ readOnly: val });
    },
  );

  watch(
    () => props.options,
    val => {
      editor?.updateOptions(val);
    },
  );

  // 异常行标记
  watch(
    () => props.errorLines,
    () => {
      setErrorLines();
    },
    { immediate: true },
  );

  // 变量联想
  watch(
    () => props.variables,
    val => {
      if (!Array.isArray(val) || !val.length) return;
    },
    { immediate: true },
  );

  // 校验（initMonaco 末尾已触发首次校验，此处只需响应内容变化）
  watch(localValue, validate);

  watch(localErrorLines, (newValue, oldValue) => {
    if (isEqual(newValue, oldValue)) return;

    emits('error', localErrorLines.value);
  });

  onMounted(() => {
    document.addEventListener('fullscreenchange', handleFullscreenChange);
    initMonaco();
  });

  onBeforeMount(() => {
    document.removeEventListener('fullscreenchange', handleFullscreenChange);
    destroyMonaco();
  });

  defineExpose({
    getValue,
    setValue,
    layout,
    scrollToTop,
    setPosition,
    validate,
  });
</script>
