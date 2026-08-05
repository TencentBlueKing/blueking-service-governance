<template>
  <div>
    <ToggleCard
      content-class="!overflow-y-auto !m-0 pt-[16px]"
      :name="$t('组件输出模板')"
      :stop-propagation="false"
      type="normal"
    >
      <template #header-right>
        <slot name="header-right" />
      </template>

      <CollapseCard class="pb-[12px] mb-[24px]">
        <template #header-left>
          <div class="flex items-center min-w-0">
            <span class="font-bold text-[#313238] shrink-0">
              {{ $t('Patch 已有的工作负载') }}
            </span>
            <Tag
              class="ml-[12px] mr-[4px] shrink-0"
              theme="info"
            >
              {{ $t('工作负载 Patch') }}
            </Tag>
          </div>
        </template>
        <template #header-right>
          <Select
            v-model="workloadType"
            :clearable="false"
          >
            <template #prefix>
              <span class="px-[8px] text-[#63656E] border-r-[#c4c6cc] border-r text-[12px] bg-[#FAFBFD] leading-[32px]">
                {{ $t('工作负载类型') }}
              </span>
            </template>
            <Select.Option
              name="GameDeployment"
              value="GameDeployment"
            />
          </Select>
        </template>

        <Alert
          class="mb-[12px]"
          theme="info"
          :title="$t('仅填写需要修改的字段（如副本数），保存后将合并覆盖到工作负载，未填写的字段保持不变。')"
        />
        <MsEditorPlus
          ref="patchEditorRef"
          :model-value="patchContent"
          :title="$t('Patch 内容（YAML 片段）')"
          :validator="[validatePatchContent]"
          @update:model-value="patchContent = $event"
        />
      </CollapseCard>

      <ResourceListCard
        ref="k8sCardRef"
        v-model:items="k8sResources"
        :add-button-text="$t('添加附加资源')"
        class="mb-[24px] pb-[12px]"
        :disable-remove="disableRemove"
        :editor-title="$t('资源定义')"
        @add="handleAddK8sResource"
      >
        <template #header-left>
          <div class="flex items-center min-w-0">
            <span class="font-bold text-[#313238] shrink-0">
              {{ $t('新建 Kubernetes 资源') }}
            </span>
            <Tag
              class="ml-[12px] mr-[4px] shrink-0"
              theme="success"
            >
              {{ $t('附加资源') }}
            </Tag>
            <span class="text-[12px] text-[#979BA5] truncate min-w-0">
              {{ $t('（将随工作负载一起创建独立的 Kubernetes 资源）') }}
            </span>
          </div>
        </template>
        <template #item-header-left>
          <span class="text-[12px] text-[#313238]">
            {{ $t('资源定义') }}
            （{{ $t('YAML 格式，支持引用上方变量') }}）
            <span class="text-[#EA3636]">*</span>
          </span>
        </template>
      </ResourceListCard>
    </ToggleCard>
  </div>
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Alert, Select, Tag } from 'bkui-vue';
  import * as monaco from 'monaco-editor';
  import { useI18n } from 'vue-i18n';
  import MsEditorPlus from '~/components/monaco-editor/ms-editor-plus.vue';

  import ResourceListCard from './resource-list-card.vue';

  import type { IMonacoEditorErrorMarkerItem } from '~/common/util';

  export interface IK8sResourceItemData {
    content: string;
  }

  export interface OutputTemplateData {
    patchers: string[];
    specs: string[];
  }

  const DEFAULT_OUTPUT_TEMPLATE: OutputTemplateData = {
    patchers: ['spec:\n  template:\n    spec:\n      terminationGracePeriodSeconds: {{ .graceSeconds }}'],
    specs: [],
  };

  /** 单个编辑器中以 YAML 文档分隔符表示多个按顺序执行的 patcher。 */
  const YAML_DOCUMENT_SEPARATOR = /^\s*(?:---|\.\.\.)\s*(?:#.*)?$/m;

  interface IProps {
    initialData?: OutputTemplateData;
  }

  const props = withDefaults(defineProps<IProps>(), {
    initialData: undefined,
  });

  const { t } = useI18n();
  const patchContent = ref('');
  const k8sResources = ref<(IK8sResourceItemData & { id: number })[]>([]);

  let k8sResourceIdCounter = 0;

  const workloadType = ref('GameDeployment');
  const patchEditorRef = ref<InstanceType<typeof MsEditorPlus>>();
  const k8sCardRef = ref<InstanceType<typeof ResourceListCard>>();

  /** Patch 与 K8s 总数至少为 1，保证至少保留一个资源。 */
  const disableRemove = computed(() => !patchContent.value.trim() && k8sResources.value.length <= 1);

  function createK8sResourceItem(): IK8sResourceItemData & { id: number } {
    return {
      id: ++k8sResourceIdCounter,
      content: '',
    };
  }

  /** 获取输出模板表单数据（提交用）。 */
  function getOutputData(): OutputTemplateData {
    return {
      patchers: getPatchers(patchContent.value),
      specs: k8sResources.value.map(resource => resource.content),
    };
  }

  function getPatchers(content: string): string[] {
    return content.split(YAML_DOCUMENT_SEPARATOR).filter(fragment => fragment.trim());
  }

  function handleAddK8sResource() {
    k8sResources.value.push(createK8sResourceItem());
  }

  /** 重置 id 计数器并初始化数据，无参数时使用默认模板 */
  function init(data?: OutputTemplateData) {
    const initData = data ?? DEFAULT_OUTPUT_TEMPLATE;
    k8sResourceIdCounter = 0;
    patchContent.value = initData.patchers?.join('\n---\n') ?? '';
    k8sResources.value = initData.specs?.map(content => ({ id: ++k8sResourceIdCounter, content })) ?? [];
  }

  /** 校验所有子编辑器是否有错误 */
  function isValid(): boolean {
    const patchValid = patchEditorRef.value?.validate() ?? true;
    const k8sValid = k8sCardRef.value?.isValid() ?? true;
    return patchValid && k8sValid;
  }

  function validatePatchContent(value: string): IMonacoEditorErrorMarkerItem[] {
    if (getPatchers(value).length || k8sResources.value.length) return [];
    return [
      {
        severity: monaco.MarkerSeverity.Error,
        message: t('组件输出至少要包含一个资源'),
        startLineNumber: 1,
        startColumn: 1,
        endLineNumber: 1,
        endColumn: 1,
      },
    ];
  }

  watch(
    () => props.initialData,
    data => init(data),
    { immediate: true },
  );

  defineExpose({ isValid, getOutputData });
</script>
