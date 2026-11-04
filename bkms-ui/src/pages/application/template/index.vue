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
  <div class="template-page">
    <div class="template-panel">
      <div class="template-header">{{ $t('请选择一个应用模板') }}</div>
      <div class="template-content">
        <div class="section-heading">
          <h3>{{ $t('框架应用') }}</h3>
        </div>
        <div class="grid grid-cols-4 gap-[16px]">
          <div
            v-for="item in frameworks"
            :key="item.id"
            :class="[
              'template-card',
              { selected: selected.framework === item.id, disabled: item.managed && !isFrameworkAvailable(item.id) },
            ]"
          >
            <button
              :aria-label="item.label"
              :aria-pressed="selected.framework === item.id"
              class="template-card-select"
              :disabled="item.managed && !isFrameworkAvailable(item.id)"
              type="button"
              @click="selectFramework(item.id)"
            ></button>
            <span
              aria-hidden="true"
              class="template-icon"
            >
              <TypeIcon
                :show-label="false"
                :size="32"
                :type="item.id"
              />
            </span>
            <div class="template-description">
              <div class="template-name">{{ item.label }}</div>
              <div class="template-hint">{{ $t(item.description) }}</div>
              <div
                v-if="getFrameworkLanguages(item.id).length"
                class="framework-languages"
              >
                <span>{{ $t('语言') }}</span>
                <div
                  v-if="getFrameworkLanguages(item.id).length > 1"
                  class="framework-language-options"
                >
                  <button
                    v-for="language in getFrameworkLanguages(item.id)"
                    :key="language"
                    :aria-pressed="selected.framework === item.id && selected.language === language"
                    :class="[
                      'framework-language-option',
                      { selected: selected.framework === item.id && selected.language === language },
                    ]"
                    type="button"
                    @click="selectFramework(item.id, language)"
                  >
                    {{ getAppDisplayLabel(language) }}
                  </button>
                </div>
                <span
                  v-else
                  class="text-[#313238]"
                  >{{ getAppDisplayLabel(getFrameworkLanguages(item.id)[0]) }}</span
                >
              </div>
            </div>
          </div>
        </div>
        <div class="section-heading mt-[28px]">
          <h3>{{ $t('通用应用') }}</h3>
          <span>{{ $t('不限框架，平台提供构建、部署、配置等通用能力') }}</span>
        </div>
        <div class="grid grid-cols-4 gap-[16px]">
          <button
            v-for="item in languages"
            :key="item.id"
            :aria-pressed="selected.framework === APP_FRAMEWORKS.BLANK && selected.language === item.id"
            :class="[
              'template-card',
              { selected: selected.framework === APP_FRAMEWORKS.BLANK && selected.language === item.id },
            ]"
            :disabled="!findRuntime(definitions, APP_FRAMEWORKS.BLANK, item.id)"
            type="button"
            @click="selectLanguage(item.id)"
          >
            <span
              aria-hidden="true"
              class="template-icon"
            >
              <TypeIcon
                :framework="APP_FRAMEWORKS.BLANK"
                :language="item.id"
                :show-label="false"
                :size="40"
                :type="APP_TYPES.BKMS_APP"
              />
            </span>
            <div class="template-description">
              <div class="template-name">{{ item.label }}</div>
              <div class="template-hint">{{ $t(item.description) }}</div>
            </div>
          </button>
        </div>
        <Alert
          v-if="error"
          class="mt-[20px]"
          theme="warning"
        >
          <template #title>
            {{ $t('应用运行时目录加载失败，请重试') }}
            <Button
              text
              theme="primary"
              @click="load"
              >{{ $t('重试') }}</Button
            >
          </template>
        </Alert>
        <div class="mt-[20px] flex justify-center gap-[8px]">
          <Button
            class="min-w-[88px]"
            :disabled="!canContinue"
            :loading="loading && !isHelmLike"
            theme="primary"
            @click="next"
            >{{ $t('下一步') }}</Button
          >
          <Button
            class="min-w-[88px]"
            @click="cancel"
            >{{ $t('取消') }}</Button
          >
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue';

  import { Alert, Button } from 'bkui-vue';
  import { useRouter } from 'vue-router';
  import { APP_FRAMEWORKS, APP_LANGUAGES, APP_TEMPLATES, APP_TYPES } from '~/common/app-types';
  import { getAppDisplayLabel, isHelmLikeAppType } from '~/composables/app-type';
  import TypeIcon from '~/pages/application/components/type-icon.vue';
  import { useAppStore } from '~/stores/apps';

  import { findRuntime, useRuntimeDefinitions } from './bkms-app/use-create-app';

  const emit = defineEmits(['update-step', 'update-text', 'next', 'cancel']);
  const router = useRouter();
  const appStore = useAppStore();
  const { definitions, loading, error, load } = useRuntimeDefinitions();
  const frameworks = APP_TEMPLATES.map(item => ({ ...item, managed: !isHelmLikeAppType(item.id) }));
  const languages = APP_LANGUAGES.filter(item => item.canCreateBlank);
  const legacySelections: Record<string, string> = {
    createTrpcTemplateApp: 'trpc:go',
    createTAFTemplateApp: 'taf:cpp',
    createHelmTemplateApp: 'helm:',
    createAgonesTemplateApp: 'agones:',
  };
  const remembered = legacySelections[appStore.lastAppTemplateID] || appStore.lastAppTemplateID || 'trpc:go';
  const [framework, initialLanguage] = remembered.split(':');
  const selected = ref({ framework, language: initialLanguage || '' });
  const languageOrder: Record<string, number> = { go: 0, cpp: 1 };
  function getFrameworkLanguages(framework: string) {
    return (
      definitions.value?.frameworks
        ?.find(item => item.name === framework)
        ?.supportedLanguages?.filter(language => findRuntime(definitions.value, framework, language))
        .sort((left, right) => (languageOrder[left] ?? 2) - (languageOrder[right] ?? 2)) || []
    );
  }
  const isHelmLike = computed(() => isHelmLikeAppType(selected.value.framework));
  const canContinue = computed(
    () => isHelmLike.value || !!findRuntime(definitions.value, selected.value.framework, selected.value.language),
  );

  function cancel() {
    router.push({ name: 'app' });
  }
  function isFrameworkAvailable(framework: string) {
    return getFrameworkLanguages(framework).length > 0;
  }
  function next() {
    if (!canContinue.value) return;
    remember();
    router.push(
      isHelmLike.value
        ? {
            name: selected.value.framework === APP_TYPES.HELM ? 'createHelmTemplateApp' : 'createAgonesTemplateApp',
          }
        : {
            name: 'createBKMSAppTemplateApp',
            query: { framework: selected.value.framework, language: selected.value.language },
          },
    );
    emit('next');
  }
  function remember() {
    appStore.lastAppTemplateID = `${selected.value.framework}:${selected.value.language}`;
    emit('update-text', {
      type: isHelmLike.value
        ? selected.value.framework
        : selected.value.framework === APP_FRAMEWORKS.BLANK
          ? APP_FRAMEWORKS.BLANK
          : APP_TYPES.BKMS_APP,
    });
  }
  function selectFramework(framework: string, language?: string) {
    if (!language && selected.value.framework === framework) return;
    const supported = getFrameworkLanguages(framework);
    selected.value = { framework, language: language || supported[0] || '' };
    remember();
  }
  function selectLanguage(language: string) {
    selected.value = { framework: APP_FRAMEWORKS.BLANK, language };
    remember();
  }
  onMounted(async () => {
    emit('update-step', 1);
    await load();
    if (!canContinue.value && !error.value) selected.value = { framework: APP_FRAMEWORKS.TRPC, language: 'go' };
    remember();
  });
</script>

<style lang="postcss" scoped>
  .template-page {
    @apply mx-auto flex min-h-full w-full max-w-[1400px] justify-center bg-[#F5F7FA] p-[20px];
  }

  .template-panel {
    @apply w-full max-w-[1400px] self-start overflow-hidden rounded-[2px] bg-white shadow-[0_2px_4px_0_#1919290d];
  }

  .template-header {
    @apply select-none pt-[32px] text-center text-[24px] leading-[32px] text-[#313238];
  }

  .template-content {
    @apply px-[40px] pb-[20px] pt-[24px];
  }

  .section-heading {
    @apply mb-[12px] flex flex-wrap items-center gap-x-[12px] gap-y-[4px];
  }

  .section-heading h3 {
    @apply m-0 text-[14px] font-bold leading-[22px] text-[#313238];
  }

  .section-heading > span {
    @apply text-[12px] leading-[20px] text-[#979BA5];
  }

  .template-card {
    @apply relative flex min-h-[92px] items-center gap-[12px] rounded-[2px] border border-[#DCDEE5] bg-white p-[16px] text-[#313238];
  }

  .template-card-select {
    @apply absolute inset-0 cursor-pointer rounded-[2px] border-0 bg-transparent;
  }

  .template-card-select:focus-visible {
    outline: 2px solid #3a84ff;
    outline-offset: 2px;
  }

  .template-card-select ~ .template-icon,
  .template-card-select ~ .template-description {
    pointer-events: none;
  }

  .template-icon {
    @apply inline-flex size-[40px] shrink-0 items-center justify-center overflow-hidden leading-none;
  }

  .template-description {
    @apply min-w-0 flex-1 text-left;
  }

  .template-name {
    @apply text-[14px] font-bold leading-[22px];
  }

  .template-hint {
    @apply mt-[2px] text-[12px] leading-[18px] text-[#979BA5];
  }

  .framework-languages {
    @apply mt-[8px] flex items-center gap-[8px] text-[12px] leading-[22px] text-[#63656E];
  }

  .framework-language-options {
    @apply relative inline-flex pointer-events-auto;
  }

  .framework-language-option {
    @apply relative h-[24px] min-w-[40px] cursor-pointer whitespace-nowrap border border-[#C4C6CC] bg-white px-[10px] text-[12px] text-[#63656E];
  }

  .framework-language-option + .framework-language-option {
    margin-left: -1px;
  }

  .framework-language-option:first-child {
    @apply rounded-l-[2px];
  }

  .framework-language-option:last-child {
    @apply rounded-r-[2px];
  }

  .framework-language-option:hover {
    @apply text-[#3A84FF];
  }

  .framework-language-option.selected {
    @apply z-[1] border-[#3A84FF] bg-[#E1ECFF] text-[#3A84FF];
  }

  .template-card:hover,
  .template-card.selected {
    @apply border-[#3A84FF] bg-[#F0F5FF];
  }

  .template-card:disabled,
  .template-card.disabled {
    @apply cursor-not-allowed opacity-50;
  }

  .template-card-select:disabled {
    cursor: not-allowed;
  }
</style>
