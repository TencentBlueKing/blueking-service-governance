<!--
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
 -->
<template>
  <div class="flex items-center gap-[8px]">
    <Select
      v-model="selectedId"
      :class="variant === 'text' ? 'framework-template-text-select !w-auto' : 'w-[280px]'"
      :clearable="false"
      :disabled="loading || switching || !templates.length"
      filterable
      :loading="loading"
      :popover-min-width="360"
      :search-placeholder="$t('搜索模板名称')"
      @change="handleChange"
      @toggle="popoverShow = $event"
    >
      <template
        v-if="variant === 'text'"
        #trigger
      >
        <span class="inline-flex cursor-pointer items-center text-[12px] font-normal leading-[20px] text-[#3A84FF]">
          <span>{{ $t('配置模板') }}：{{ currentLabel || $t('请选择') }}</span>
          <AngleDown
            class="ml-[4px] shrink-0 text-[#3A84FF] transition-transform duration-200"
            :class="{ 'rotate-180': popoverShow }"
            :height="20"
            :width="20"
          />
        </span>
      </template>
      <Select.Option
        v-for="item in templates"
        :key="item.id"
        :label="optionLabel(item)"
        :value="item.id"
      />
      <template #extension>
        <div
          class="flex w-full cursor-pointer items-center justify-center px-[12px] py-[8px] text-[#63656E]"
          role="button"
          tabindex="0"
          @click.stop="openTemplateManager"
          @keydown.enter.stop="openTemplateManager"
        >
          <Plus
            :height="20"
            :width="20"
          />
          <span class="text-[12px]">{{ $t('新建模板') }}</span>
        </div>
      </template>
    </Select>
  </div>
</template>

<script setup lang="ts">
  import { computed, h, nextTick, onBeforeUnmount, ref, watch } from 'vue';

  import { InfoBox, Message, Select } from 'bkui-vue';
  import { AngleDown, Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { FrameworkConfigTemplatesService } from '~/api/modules/v1';
  import { cancelRequest } from '~/api/request-queue';

  import type { TemplateOutputObj } from '~/@types/v1/framework-config-templates';

  type FrameworkAppType = 'taf' | 'trpc';
  type TemplateDetail = TemplateOption & { content: string };
  type TemplateOption = TemplateOutputObj & { fileName: string; id: string; name: string };

  function hasTemplateOption(template: TemplateOutputObj): template is TemplateOption {
    return (
      typeof template.fileName === 'string' &&
      template.fileName.length > 0 &&
      typeof template.id === 'string' &&
      template.id.length > 0 &&
      typeof template.name === 'string'
    );
  }

  function isAbortError(error: unknown): boolean {
    return error instanceof DOMException && error.name === 'AbortError';
  }

  const props = withDefaults(
    defineProps<{
      appType: FrameworkAppType;
      language: string;
      variant?: 'field' | 'text';
      workspaceId: string;
    }>(),
    {
      variant: 'field',
    },
  );

  const emit = defineEmits<{
    availability: [available: boolean];
    loading: [loading: boolean];
    select: [template: TemplateDetail];
  }>();

  const { t } = useI18n();
  const router = useRouter();
  const templates = ref<TemplateOption[]>([]);
  const selectedId = ref('');
  const previousId = ref('');
  const loading = ref(false);
  const switching = ref(false);
  const popoverShow = ref(false);
  let loadSeq = 0;
  let isUnmounted = false;

  // 模板详情缓存：避免同一模板重复请求详情接口
  const detailCache = new Map<string, TemplateDetail>();

  // 请求标识：用于通过 cancelRequest 中断上一轮未完成的请求
  let requestTag = '';

  const currentLabel = computed(() => {
    const selected = templates.value.find(item => item.id === selectedId.value);
    return selected ? optionLabel(selected) : '';
  });

  function confirmSwitch() {
    return new Promise<boolean>(resolve => {
      let settled = false;
      const finish = (value: boolean) => {
        if (settled) return;
        settled = true;
        resolve(value);
      };
      InfoBox({
        title: t('确认切换配置模板？'),
        content: h('div', { class: 'py-[12px] text-center leading-[22px]' }, [
          t('切换后，之前填写的框架配置内容将不会保留。'),
        ]),
        headerAlign: 'center',
        footerAlign: 'center',
        confirmText: t('切换'),
        cancelText: t('取消'),
        onCancel: () => finish(false),
        onClose: () => finish(false),
        onConfirm: () => finish(true),
      });
    });
  }

  async function handleChange(value: number | string) {
    const seq = loadSeq;
    const nextId = String(value);
    if (switching.value || !nextId || nextId === previousId.value) return;

    const selected = templates.value.find(item => item.id === nextId);
    if (!selected) {
      rollbackSelection();
      return;
    }

    switching.value = true;
    emit('loading', true);
    try {
      if (!(await confirmSwitch())) {
        if (seq === loadSeq) rollbackSelection();
        return;
      }
      if (seq !== loadSeq) return;

      const detail = await loadTemplateDetail(selected);
      if (seq !== loadSeq) return;
      if (!detail) {
        rollbackSelection();
        return;
      }

      selectedId.value = nextId;
      previousId.value = nextId;
      emit('select', detail);
    } finally {
      if (seq === loadSeq) {
        switching.value = false;
        emit('loading', false);
      }
    }
  }

  async function load() {
    const seq = ++loadSeq;

    // 中断上一轮未完成的请求
    if (requestTag) cancelRequest(requestTag);
    requestTag = `fw-template-${seq}`;

    // 清空详情缓存（workspaceId/appType/language 变化时）
    detailCache.clear();

    emit('availability', false);
    emit('loading', true);
    selectedId.value = '';
    previousId.value = '';
    templates.value = [];
    if (!props.workspaceId) {
      loading.value = false;
      switching.value = false;
      emit('loading', false);
      return;
    }

    loading.value = true;
    switching.value = true;
    try {
      const result = await FrameworkConfigTemplatesService.listWorkspaceFrameworkConfigTemplates(
        {
          workspaceID: props.workspaceId,
          framework: props.appType,
          ...(props.language ? { language: props.language } : {}),
        },
        { id: requestTag },
      );
      if (seq !== loadSeq) return;

      templates.value = (Array.isArray(result) ? result : [])
        .filter(hasTemplateOption)
        .sort((left, right) => Number(Boolean(right.builtin)) - Number(Boolean(left.builtin)));
      emit('availability', templates.value.length > 0);
      const initial = templates.value.find(template => template.isDefault) || templates.value[0];
      if (!initial) return;

      const detail = await loadTemplateDetail(initial);
      if (seq !== loadSeq) return;
      if (!detail) return;

      selectedId.value = initial.id;
      previousId.value = selectedId.value;
      emit('select', detail);
    } catch (error) {
      if (seq === loadSeq && !isAbortError(error)) {
        console.error('加载框架配置模板列表失败', error);
        templates.value = [];
        emit('availability', false);
        Message({ theme: 'warning', message: t('框架配置模板加载失败') });
      }
    } finally {
      if (seq === loadSeq) {
        loading.value = false;
        switching.value = false;
        emit('loading', false);
      }
    }
  }

  async function loadTemplateDetail(template: TemplateOption): Promise<TemplateDetail | undefined> {
    if (!props.workspaceId) return undefined;

    // 命中缓存直接返回
    const cached = detailCache.get(template.id);
    if (cached) return cached;

    try {
      const detail = await FrameworkConfigTemplatesService.getWorkspaceFrameworkConfigTemplate(
        {
          workspaceID: props.workspaceId,
          templateID: template.id,
        },
        { id: requestTag },
      );
      if (typeof detail?.content !== 'string') {
        console.error('框架配置模板内容为空', template.id);
        return undefined;
      }
      const result = { ...template, ...detail, id: template.id, content: detail.content };
      detailCache.set(template.id, result);
      return result;
    } catch (error) {
      if (isAbortError(error)) return undefined;
      console.error('加载框架配置模板详情失败', error);
      Message({ theme: 'warning', message: t('模板详情加载失败，请重试') });
      return undefined;
    }
  }

  function openTemplateManager() {
    const { href } = router.resolve({
      name: 'basicItem',
      params: {
        menuName: 'frameworkTemplates',
        space: props.workspaceId,
      },
    });
    window.open(href, '_blank', 'noopener,noreferrer');
  }

  function optionLabel(template: TemplateOption) {
    return template.name;
  }

  function rollbackSelection() {
    switching.value = true;
    selectedId.value = previousId.value;
    void nextTick(() => {
      if (!isUnmounted) switching.value = false;
    });
  }

  onBeforeUnmount(() => {
    isUnmounted = true;
    loadSeq += 1;
    if (requestTag) cancelRequest(requestTag);
  });

  watch(
    () => [props.workspaceId, props.appType, props.language] as const,
    () => {
      void load();
    },
    { immediate: true },
  );
</script>

<style lang="postcss" scoped>
  .framework-template-text-select:deep(.bk-select-trigger) {
    height: auto;
    min-height: 0;
    border: none;
    background: transparent;
    box-shadow: none;
  }
</style>
