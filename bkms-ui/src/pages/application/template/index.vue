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
    <ToggleCard
      class="template-panel"
      :name="$t('应用模板')"
      type="normal"
    >
      <div class="template-content">
        <div class="section-heading">
          <h3>{{ $t('开发框架') }}</h3>
          <span>{{ $t('适合已确定使用 tRPC、TAF 等框架的应用；tRPC 选定后再选开发语言') }}</span>
        </div>
        <div class="grid grid-cols-4 gap-[16px]">
          <button
            v-for="item in frameworks"
            :key="item.id"
            :aria-pressed="selected.framework === item.id"
            :class="['template-card', { selected: selected.framework === item.id }]"
            :disabled="item.managed && !isFrameworkAvailable(item.id)"
            type="button"
            @click="selectFramework(item.id)"
          >
            <span
              aria-hidden="true"
              class="template-icon"
            >
              <i
                :class="item.icon"
                :style="{ color: item.color, fontSize: item.iconSize }"
              ></i>
            </span>
            <div class="template-description">
              <div class="template-name">{{ item.name }}</div>
              <div class="template-hint">{{ $t(item.description) }}</div>
            </div>
          </button>
        </div>
        <div
          v-if="frameworkLanguages.length"
          class="framework-languages"
        >
          <span>{{ $t('请选择开发语言') }}</span>
          <Radio.Group
            v-model="selected.language"
            class="framework-language-radio"
            type="capsule"
            @change="remember"
          >
            <Radio.Button
              v-for="language in frameworkLanguages"
              :key="language"
              :label="language"
            >
              {{ languageLabel(language) }}
            </Radio.Button>
          </Radio.Group>
        </div>
        <div class="section-heading mt-[24px]">
          <h3>{{ $t('按语言') }}</h3>
          <span>{{ $t('按开发语言创建通用服务，目前支持 Go、Python、Node.js') }}</span>
        </div>
        <div class="grid grid-cols-4 gap-[16px]">
          <button
            v-for="item in languages"
            :key="item.id"
            :aria-pressed="selected.framework === 'blank' && selected.language === item.id"
            :class="['template-card', { selected: selected.framework === 'blank' && selected.language === item.id }]"
            :disabled="!findRuntime(definitions, 'blank', item.id)"
            type="button"
            @click="selectLanguage(item.id)"
          >
            <span
              aria-hidden="true"
              class="template-icon"
            >
              <component
                :is="item.logo"
                class="block size-[40px] rounded-[4px]"
              />
            </span>
            <div class="template-description">
              <div class="template-name">{{ item.name }}</div>
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
    </ToggleCard>
  </div>
</template>
<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue';

  import { Alert, Button, Radio } from 'bkui-vue';
  import { useRouter } from 'vue-router';
  import GoLogo from '~/assets/lang-icons/go.svg';
  import NodejsLogo from '~/assets/lang-icons/nodejs.svg';
  import PythonLogo from '~/assets/lang-icons/python.svg';
  import { useAppStore } from '~/stores/apps';

  import { findRuntime, useRuntimeDefinitions } from './bkms-app/use-create-app';

  const emit = defineEmits(['update-step', 'update-text', 'next', 'cancel']);
  const router = useRouter();
  const appStore = useAppStore();
  const { definitions, loading, error, load } = useRuntimeDefinitions();
  const frameworks = [
    {
      id: 'trpc',
      name: 'tRPC',
      managed: true,
      icon: 'bkms-icon bkms-icon-trpc',
      iconSize: '14px',
      color: '#1B44C8',
      description: '适用于使用 tRPC 协议的服务',
    },
    {
      id: 'taf',
      name: 'TAF',
      managed: true,
      icon: 'bkms-icon bkms-icon-taf',
      iconSize: '32px',
      color: '#1B44C8',
      description: '采用 TAF 框架实现的服务端应用',
    },
    {
      id: 'helm',
      name: 'Helm Chart',
      managed: false,
      icon: 'bkms-icon bkms-icon-HelmCharts',
      iconSize: '32px',
      color: '#0F1689',
      description: '自定义应用编排，拥有灵活的可定制性',
    },
    {
      id: 'agones',
      name: 'Agones',
      managed: false,
      icon: 'bkms-icon bkms-icon-agones',
      iconSize: '32px',
      color: '#0F1689',
      description: '适用于多人在线游戏的专用服务器编排',
    },
  ];
  const languages = [
    { id: 'go', name: 'Go', logo: GoLogo, description: '高并发 Web / 微服务，如 Gin、Echo' },
    { id: 'python', name: 'Python', logo: PythonLogo, description: '数据服务、脚本任务与 AI 应用常见选择' },
    { id: 'nodejs', name: 'Node.js', logo: NodejsLogo, description: '实时接口、前端聚合层与前后端一体服务' },
  ];
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
  const frameworkLanguages = computed(() => {
    if (selected.value.framework === 'blank') return [];
    return (
      definitions.value?.frameworks
        ?.find(item => item.name === selected.value.framework)
        ?.supportedLanguages?.filter(language => findRuntime(definitions.value, selected.value.framework, language))
        .sort((left, right) => (languageOrder[left] ?? 2) - (languageOrder[right] ?? 2)) || []
    );
  });
  const isHelmLike = computed(() => ['helm', 'agones'].includes(selected.value.framework));
  const canContinue = computed(
    () => isHelmLike.value || !!findRuntime(definitions.value, selected.value.framework, selected.value.language),
  );

  function cancel() {
    router.push({ name: 'app' });
  }
  function isFrameworkAvailable(framework: string) {
    return definitions.value?.frameworks?.some(
      item =>
        item.name === framework &&
        item.supportedLanguages?.some(language => findRuntime(definitions.value, framework, language)),
    );
  }
  function languageLabel(language: string) {
    return (
      ({ go: 'Go', cpp: 'C++', python: 'Python', nodejs: 'Node.js' } as Record<string, string>)[language] || language
    );
  }
  function next() {
    if (!canContinue.value) return;
    remember();
    router.push(
      isHelmLike.value
        ? {
            name: selected.value.framework === 'helm' ? 'createHelmTemplateApp' : 'createAgonesTemplateApp',
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
      type: isHelmLike.value ? selected.value.framework : selected.value.framework === 'blank' ? 'blank' : 'bkmsApp',
    });
  }
  function selectFramework(framework: string) {
    const supported = definitions.value?.frameworks?.find(item => item.name === framework)?.supportedLanguages || [];
    selected.value = { framework, language: supported.includes('go') ? 'go' : supported[0] || '' };
    remember();
  }
  function selectLanguage(language: string) {
    selected.value = { framework: 'blank', language };
    remember();
  }
  onMounted(async () => {
    emit('update-step', 1);
    await load();
    if (!canContinue.value && !error.value) selected.value = { framework: 'trpc', language: 'go' };
    remember();
  });
</script>
<style scoped>
  .template-page {
    @apply mx-auto flex min-h-full w-full max-w-[1400px] justify-center bg-[#F5F7FA] p-[20px];
  }

  .template-panel {
    @apply w-full max-w-[1400px] self-start rounded-[2px] bg-white shadow-[0_2px_4px_0_#1919290d];
  }

  .template-content {
    @apply mx-[16px] mb-[16px] px-[24px] pb-[20px] pt-[16px];
  }

  .section-heading {
    @apply mb-[16px] flex flex-wrap items-baseline gap-x-[12px] gap-y-[4px];
  }

  .section-heading h3 {
    @apply m-0 text-[16px] font-bold leading-[24px] text-[#313238];
  }

  .section-heading > span {
    @apply text-[12px] leading-[20px] text-[#979BA5];
  }

  .template-card {
    @apply flex min-h-[92px] items-center gap-[12px] rounded-[2px] border border-[#DCDEE5] bg-white p-[16px] text-[#313238];
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
    @apply mt-[12px] flex items-center gap-[16px] rounded-[2px] bg-[#F5F7FA] px-[16px] py-[12px] text-[14px] leading-[22px] text-[#63656E];
  }

  :deep(.framework-language-radio.bk-radio-capsule .bk-radio-button-label) {
    font-size: 14px;
  }

  .template-card:hover,
  .template-card.selected {
    @apply border-[#3A84FF] bg-[#F0F5FF];
  }

  .template-card:disabled {
    @apply cursor-not-allowed opacity-50;
  }
</style>
