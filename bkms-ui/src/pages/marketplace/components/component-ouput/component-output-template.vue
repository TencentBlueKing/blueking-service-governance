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
      <!-- Patch 已有的工作负载 -->
      <ResourceListCard
        ref="patchCardRef"
        v-model:items="patches"
        :add-button-text="$t('添加 Patch 路径')"
        class="pb-[12px] mb-[24px]"
        :disable-remove="disableRemove"
        :editor-title="$t('输出模板')"
        @add="handleAddPatch"
      >
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
            <span
              class="text-[12px] text-[#979BA5] truncate min-w-0"
              :title="$t('（将内容合并到已有的工作负载中）')"
            >
              {{ $t('（将内容合并到已有的工作负载中）') }}
            </span>
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
        <!-- <template #header>
        <FlexRow class="w-full">
          <template #left>
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
              <span class="text-[12px] text-[#979BA5] truncate min-w-0">
                {{ $t('（将内容合并到已有的工作负载中）') }}
              </span>
            </div>
          </template>
          <template #right>
            <slot name="patch-header-right">
              <Select
                v-model="workloadType"
                :clearable="false"
              >
                <template #prefix>
                  <span
                    class="px-[8px] text-[#63656E] border-r-[#c4c6cc] border-r text-[12px] bg-[#FAFBFD] leading-[32px]"
                  >
                    {{ $t('工作负载类型') }}
                  </span>
                </template>
                <Select.Option
                  name="GameDeployment"
                  value="GameDeployment"
                />
              </Select>
            </slot>
          </template>
        </FlexRow>
      </template> -->
        <template #item-header-left="{ item }">
          <Input
            class="flex-1 h-[32px]"
            :model-value="getPatchPath(item)"
            :placeholder="$t('请输入 Patch 路径，如 {0}', ['spec.replicas、metadata.labels'])"
            @update:model-value="handlePatchPathUpdate(item, String($event))"
          >
            <template #prefix>
              <span
                class="px-[12px] bg-[#FAFBFD] text-[#313238] text-[12px] h-full flex items-center border-r border-[#DCDEE5] whitespace-nowrap"
              >
                {{ $t('Patch 路径') }}
              </span>
            </template>
          </Input>
        </template>
      </ResourceListCard>

      <!-- 新建 K8s 资源 -->
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

  import { Input, Select, Tag } from 'bkui-vue';
  import yaml from 'js-yaml';

  import ResourceListCard from './resource-list-card.vue';

  import type { IResourceListItem } from './resource-list-card.vue';
  import type { ComponentDefOutputFormInput } from '~/@types/v1/component-defs';

  export interface IK8sResourceItemData {
    content: string;
  }

  export interface IPatchItemData {
    content: string;
    path: string;
  }

  export interface OutputTemplateData {
    k8sResources: (IK8sResourceItemData & { id: number })[];
    patches: (IPatchItemData & { id: number })[];
  }

  /** 输出模板默认数据 */
  const DEFAULT_OUTPUT_TEMPLATE = {
    patcher: {
      'spec.template.spec': {
        terminationGracePeriodSeconds: '{{ .graceSeconds }}',
      },
    },
    spec: [],
  };

  interface IProps {
    /** 初始输出模板数据，传入后自动初始化 */
    initialData?: ComponentDefOutputFormInput;
  }

  const props = withDefaults(defineProps<IProps>(), {
    initialData: undefined,
  });

  const patches = ref<OutputTemplateData['patches']>([]);
  const k8sResources = ref<OutputTemplateData['k8sResources']>([]);

  // id 计数器
  let patchIdCounter = 0;
  let k8sResourceIdCounter = 0;

  const workloadType = ref('GameDeployment');
  const patchCardRef = ref<InstanceType<typeof ResourceListCard>>();
  const k8sCardRef = ref<InstanceType<typeof ResourceListCard>>();

  /** Patch 和 K8s 总数 ≤ 1 时禁用删除，保证至少保留一个资源 */
  const disableRemove = computed(() => patches.value.length + k8sResources.value.length <= 1);

  function createK8sResourceItem(): IK8sResourceItemData & { id: number } {
    return {
      id: ++k8sResourceIdCounter,
      content: '',
    };
  }

  function createPatchItem(): IPatchItemData & { id: number } {
    return {
      id: ++patchIdCounter,
      path: '',
      content: '',
    };
  }

  function getDefaultOutputInitData(): Omit<ComponentDefOutputFormInput, 'name'> {
    const patcher = Object.entries(DEFAULT_OUTPUT_TEMPLATE.patcher).map(([path, patch]) => ({
      path,
      patch: unquoteTemplateExpressions(yaml.dump(patch)),
    }));
    const spec = DEFAULT_OUTPUT_TEMPLATE.spec.map(s => unquoteTemplateExpressions(yaml.dump(s)));
    return { patcher, spec };
  }

  /** 获取输出模板表单数据（提交用） */
  function getOutputForm(): Omit<ComponentDefOutputFormInput, 'name'> {
    return {
      patcher: patches.value.map(p => ({ path: p.path, patch: p.content })),
      spec: k8sResources.value.map(r => r.content),
    };
  }

  function getPatchPath(item: IResourceListItem): string {
    return (item as IPatchItemData & { id: number }).path;
  }

  function handleAddK8sResource() {
    k8sResources.value.push(createK8sResourceItem());
  }

  function handleAddPatch() {
    patches.value.push(createPatchItem());
  }

  function handlePatchPathUpdate(item: IResourceListItem, val: string) {
    (item as IPatchItemData & { id: number }).path = val;
  }

  /** 重置 id 计数器并初始化数据，无参数时使用默认模板 */
  function init(data?: ComponentDefOutputFormInput) {
    const initData = data ?? getDefaultOutputInitData();
    patchIdCounter = 0;
    k8sResourceIdCounter = 0;
    patches.value = initData.patcher?.length
      ? initData.patcher.map(item => ({ id: ++patchIdCounter, path: item.path ?? '', content: item.patch ?? '' }))
      : [];
    k8sResources.value = initData.spec?.length
      ? initData.spec.map(content => ({ id: ++k8sResourceIdCounter, content }))
      : [];
  }

  /** 校验所有子编辑器是否有错误 */
  function isValid(): boolean {
    const patchValid = patchCardRef.value?.isValid() ?? true;
    const k8sValid = k8sCardRef.value?.isValid() ?? true;
    return patchValid && k8sValid;
  }

  /**
   * 移除 YAML 字符串中 Go 模板表达式（{{ ... }}）两侧的引号
   * js-yaml 会对以 { 开头的值加引号，导致数字类型字段渲染后仍是字符串
   */
  function unquoteTemplateExpressions(yamlStr: string): string {
    return yamlStr.replace(/["'](\{\{[\s\S]*?\}\})["']/g, '$1');
  }

  // 监听 initialData 变化自动初始化
  watch(
    () => props.initialData,
    val => {
      init(val);
    },
    { immediate: true },
  );

  defineExpose({ isValid, getOutputForm });
</script>
