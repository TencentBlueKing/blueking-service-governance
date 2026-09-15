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
  <Popover
    ref="popoverRef"
    :arrow="false"
    ext-cls="c-env-detail-select-popover"
    placement="bottom-start"
    theme="light"
    trigger="click"
    :width="800"
    @after-hidden="handlePopoverHidden"
    @after-show="isPopoverVisible = true"
  >
    <div
      class="flex items-center justify-between w-full h-full bg-[#F0F1F5] overflow-hidden group text-[#4D4F56] text-[12px] cursor-pointer rounded-[2px] hover:bg-[#EAEBF0]"
    >
      <div class="flex items-center min-w-0 overflow-hidden px-[8px] gap-[4px]">
        <span class="whitespace-nowrap truncate">{{ currentEnvLabel }}</span>
        <Tag
          v-if="currentEnvTypeConfig?.name"
          class="shrink-0"
          :class="envTypeTagClassMap[props.currentEnvType || '']"
          size="small"
        >
          {{ currentEnvTypeConfig.name }}
        </Tag>
      </div>
      <AngleDownFill
        :class="['mr-[5px] text-[#C4C6CC] group-hover:text-[#979BA5]', isPopoverVisible ? '' : 'rotate-180']"
      />
    </div>
    <template #content>
      <div class="min-h-[100px]">
        <Input
          v-model.trim="searchKeyword"
          behavior="simplicity"
          class="mt-[8px]"
          clearable
          :placeholder="$t('请输入环境名称')"
        >
          <template #prefix>
            <Search class="ml-[2px] text-[16px] text-[#979BA5]" />
          </template>
        </Input>
        <div
          v-if="visibleEnvGroups.length"
          class="grid grid-cols-4 gap-[8px] mt-[8px]"
        >
          <div
            v-for="group in visibleEnvGroups"
            :key="group.type"
            class="min-w-0"
          >
            <div class="flex items-center h-[32px] px-[8px] bg-[#F5F7FA]">
              <Tag :class="envTypeTagClassMap[group.type]">
                {{ group.label }}
              </Tag>
            </div>
            <div class="env-list-scroll max-h-[224px] overflow-y-auto">
              <div
                v-for="env in group.envs"
                :key="env.id"
                :class="[
                  'flex items-center h-[32px] px-[8px] cursor-pointer text-[12px] text-[#4D4F56] transition-bg-color duration-150',
                  { '!bg-[#E1ECFF] !text-[#3A84FF]': env.id === props.modelValue },
                  { 'hover:bg-[#F5F7FA]': env.id !== props.modelValue },
                ]"
                @click="handleSelect(env.id)"
              >
                <span class="truncate">{{ env.displayName || env.name }}</span>
              </div>
            </div>
          </div>
        </div>
        <div
          v-else
          class="py-[16px] text-center text-[12px] text-[#C4C6CC]"
        >
          {{ $t('无匹配数据') }}
          <Button
            v-if="props.listError"
            class="mt-[4px]"
            text
            @click="emits('retry')"
          >
            {{ $t('重试') }}
          </Button>
        </div>
      </div>
      <div class="grid grid-cols-2 h-[40px] -mx-[8px] mt-[8px] border-t border-[#DCDEE5]">
        <div
          class="flex h-full items-center justify-center gap-[5px] text-[12px] text-[#4D4F56] cursor-pointer hover:bg-[#F0F1F5]"
          @click="handleCreate"
        >
          <Plus
            height="16px"
            width="16px"
          />
          <span>{{ $t('新建环境') }}</span>
        </div>
        <div
          class="flex h-full items-center justify-center text-[12px] text-[#4D4F56] cursor-pointer hover:bg-[#F0F1F5]"
          @click="handleBack"
        >
          {{ $t('返回列表') }}
        </div>
      </div>
    </template>
  </Popover>
</template>

<script lang="ts" setup>
  import { computed, ref } from 'vue';

  import { Button, Input, Popover, Tag } from 'bkui-vue';
  import { AngleDownFill, Plus, Search } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';

  import type { EnvOutput } from '~/@types/v1/env';

  interface Emits {
    (e: 'back'): void;
    (e: 'create'): void;
    (e: 'retry'): void;
    (e: 'select', envId: string): void;
  }

  interface Props {
    currentEnvName?: string;
    currentEnvType?: string;
    envList: EnvOutput[];
    listError?: boolean;
    modelValue?: string;
  }

  const props = withDefaults(defineProps<Props>(), {
    currentEnvName: '',
    currentEnvType: '',
    listError: false,
    modelValue: '',
  });
  const emits = defineEmits<Emits>();
  const { t } = useI18n();

  const popoverRef = ref<InstanceType<typeof Popover> | null>(null);
  const isPopoverVisible = ref(false);
  const searchKeyword = ref('');
  const envTypeOrder = ['development', 'test', 'staging', 'production'];
  const currentEnvLabel = computed(() => props.currentEnvName || t('请选择环境'));
  const currentEnvTypeConfig = computed(() => {
    const type = props.currentEnvType;
    return type && envTypeMap[type] ? envTypeMap[type] : undefined;
  });
  /** 只展示至少含有一个环境的分类；搜索后同样隐藏空分类。 */
  const visibleEnvGroups = computed(() => {
    const keyword = searchKeyword.value.toLowerCase();
    return envTypeOrder
      .map(type => ({
        type,
        label: envTypeMap[type]?.name || type,
        envs: props.envList.filter(env => {
          if (env.type !== type) return false;
          if (!keyword) return true;
          return [env.name, env.displayName].some(name => name?.toLowerCase().includes(keyword));
        }),
      }))
      .filter(group => group.envs.length > 0);
  });

  function handleBack() {
    popoverRef.value?.hide();
    emits('back');
  }

  function handleCreate() {
    popoverRef.value?.hide();
    emits('create');
  }

  function handlePopoverHidden() {
    isPopoverVisible.value = false;
    searchKeyword.value = '';
  }

  function handleSelect(envId?: string) {
    if (!envId) return;
    popoverRef.value?.hide();
    emits('select', envId);
  }
</script>

<style lang="postcss" scoped>
  :deep(.bk-input.is-simplicity) {
    border-bottom-color: #dcdee5 !important;
    border-radius: 0 !important;

    &:hover {
      background-color: transparent !important;
    }

    .bk-input--text {
      background-color: transparent !important;
    }
  }
  :deep(.bk-popover-reference) {
    display: block;
    height: 100%;
  }
</style>

<style lang="postcss">
  .c-env-detail-select-popover {
    padding: 0 8px !important;
    border: none;
    border-radius: 2px !important;
    box-shadow: 0 2px 4px 0 #1919290d !important;

    .env-list-scroll {
      &::-webkit-scrollbar {
        width: 6px;
        height: 6px;
      }

      &::-webkit-scrollbar-thumb {
        background: #dcdee5;
      }
    }
  }
</style>
