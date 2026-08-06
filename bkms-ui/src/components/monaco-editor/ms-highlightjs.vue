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

<!-- eslint-disable vue/no-v-html -->
<template>
  <div
    ref="highlightRef"
    class="flex flex-col border-[1px] border-[#DCDEE5] w-full h-full"
  >
    <!-- 工具栏 -->
    <FlexRow
      :class="[
        'h-[40px] px-[16px] rounded-t-sm border-b-[1px] border-[#DCDEE5]',
        lightTheme ? 'bg-[#F5F7FA]' : 'bg-[#2E2E2E]',
      ]"
    >
      <template #left>
        <span :class="['text-[14px] leading-none', lightTheme ? 'text-[#313238]' : 'text-[#C4C6CC]']">
          <slot name="title">{{ title }}</slot>
        </span>
      </template>
      <template #right>
        <div
          v-if="showTools"
          class="flex items-center"
        >
          <slot name="tools"></slot>
          <IconButton
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
      </template>
    </FlexRow>
    <pre
      v-bkloading="{ loading }"
      :class="[
        className,
        'relative p-[16px] max-w-full h-full flex-1 default-values',
        {
          'overflow-auto': !loading,
        },
      ]"
    ><code v-bk-xss-html="highlightedCode"></code></pre>
  </div>
</template>
<script lang="ts" setup>
  import { computed, onBeforeMount, onMounted, ref, watch } from 'vue';

  import { Copy } from 'bkui-vue/lib/icon';
  import hljs from 'highlight.js/lib/core';
  import yaml from 'highlight.js/lib/languages/yaml';
  import { copyText, escapeHtml } from '~/common/util';

  import 'highlight.js/styles/github.css'; // 浅色主题

  /** 文本高亮标记配置 */
  interface HighlightMark {
    /** 可选的 CSS class 名 */
    className?: string;
    /** 标记颜色，如 '#FF9C01'（仅允许合法 CSS 颜色值） */
    color?: string;
    /**
     * 是否替换所有匹配项，默认 false（仅替换第一次出现）。
     * 设为 true 时，将替换高亮 HTML 中所有匹配的文本片段。
     */
    replaceAll?: boolean;
    /** 需要标记的原始文本（未转义） */
    text: string;
  }

  interface IProps {
    autodetect?: boolean;
    code: string;
    /** 需要特殊着色的文本片段配置，不传则行为不变 */
    highlights?: HighlightMark[];
    ignoreIllegals?: boolean;
    language?: string;
    lightTheme?: boolean;
    loading?: boolean;
    showTools?: boolean;
    title?: string;
  }
  const props = withDefaults(defineProps<IProps>(), {
    language: 'yaml',
    autodetect: true,
    ignoreIllegals: true,
    lightTheme: true,
    showTools: true,
  });

  hljs.registerLanguage('yaml', yaml);

  const language = ref(props.language);
  watch(
    () => props.language,
    newLanguage => {
      language.value = newLanguage;
    },
  );

  const autodetect = computed(() => props.autodetect && !language.value);
  const cannotDetectLanguage = computed(() => !autodetect.value && !hljs.getLanguage(language.value));

  const className = computed((): string => {
    if (cannotDetectLanguage.value) {
      return '';
    }
    return `hljs ${language.value}`;
  });

  watch(
    () => [props.code, autodetect.value],
    () => {
      if (autodetect.value) {
        const result = hljs.highlightAuto(props.code);
        language.value = result.language ?? '';
      }
    },
    { immediate: true },
  );

  /**
   * 对高亮后的 HTML 字符串进行后处理，将 highlights 指定的文本包裹带颜色的 span。
   * 默认仅替换第一次出现；若 mark.replaceAll 为 true，则替换所有出现。
   */
  function applyHighlightMarks(html: string, marks?: HighlightMark[]): string {
    if (!marks?.length) return html;

    /** 校验 CSS 颜色值合法性，防止 XSS 注入 */
    const CSS_COLOR_RE = /^(#[\da-f]{3,8}|[a-z]+|\b(?:rgba?|hsla?)\([\d.,\s%]+\))$/i;

    let result = html;
    for (const mark of marks) {
      // 对目标文本做 HTML 转义，以匹配 hljs 输出中的转义文本
      const escapedText = escapeHtml(mark.text);
      const safeColor = mark.color && CSS_COLOR_RE.test(mark.color) ? mark.color : '';
      const safeClass = mark.className ? escapeHtml(mark.className) : '';
      const styleAttr = safeColor ? ` style="color:${safeColor}"` : '';
      const classAttr = safeClass ? ` class="${safeClass}"` : '';
      const replacement = `<span${styleAttr}${classAttr}>${escapedText}</span>`;
      if (mark.replaceAll) {
        result = result.replaceAll(escapedText, replacement);
      } else {
        result = result.replace(escapedText, replacement);
      }
    }
    return result;
  }

  const highlightedCode = computed((): string => {
    let html: string;

    if (cannotDetectLanguage.value) {
      console.warn(`The language "${language.value}" you specified could not be found.`);
      html = escapeHtml(props.code);
    } else if (autodetect.value) {
      // 自动检测语言
      const result = hljs.highlightAuto(props.code);
      html = result.value;
    } else {
      const result = hljs.highlight(props.code, {
        language: language.value,
        ignoreIllegals: props.ignoreIllegals,
      });
      html = result.value;
    }
    // 后处理：对指定文本片段做颜色标记
    return applyHighlightMarks(html, props.highlights);
  });

  const highlightRef = ref<HTMLElement>();
  // 全屏
  const isFullscreen = ref(false);
  // 复制
  function handleCopyContent() {
    copyText(props.code);
  }
  function handleFullscreenChange() {
    isFullscreen.value = !!document.fullscreenElement;
  }

  function toggleFullscreen() {
    if (!document.fullscreenElement) {
      highlightRef.value?.requestFullscreen();
    } else {
      document.exitFullscreen();
    }
  }

  onMounted(() => {
    document.addEventListener('fullscreenchange', handleFullscreenChange);
  });

  onBeforeMount(() => {
    document.removeEventListener('fullscreenchange', handleFullscreenChange);
  });
</script>

<style scoped>
  :deep(.default-values code + div) {
    pointer-events: none;
  }
</style>
