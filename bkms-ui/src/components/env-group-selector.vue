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
  <div class="grid grid-cols-[repeat(4,minmax(0,1fr))] gap-[12px]">
    <div
      v-for="group in groupedEnvList"
      :key="group.type"
      class="min-w-0"
    >
      <div :class="['h-[32px] flex items-center justify-between px-[12px]', envTypeTagClassMap[group.type]]">
        <span class="text-[12px] font-bold">
          {{ group.label }}
        </span>
        <div class="flex items-center">
          <span class="text-[12px] text-[#4D4F56] mr-[8px]">{{ $t('全选') }}</span>
          <Checkbox
            class="bg-[#fff]"
            :disabled="getGroupEnvNames(group.envs).length === 0"
            :indeterminate="isGroupIndeterminate(group.envs)"
            :model-value="isGroupAllSelected(group.envs)"
            @change="handleGroupSelectAll(group.envs, $event)"
            @click.stop
          />
        </div>
      </div>
      <div class="flex flex-col gap-[4px] mt-[4px]">
        <div
          v-for="env in group.envs"
          :key="env.id || env.name"
          class="h-[32px] flex items-center px-[12px] bg-[#F5F7FA] cursor-pointer"
          @click="handleEnvItemClick(env)"
        >
          <div class="flex flex-1 min-w-0 items-center">
            <span
              class="min-w-0 truncate text-[12px] text-[#4D4F56]"
              :title="env.displayName || env.name"
            >
              {{ env.displayName || env.name || '--' }}
            </span>
            <Tag
              v-if="isFeatureEnv(env)"
              class="bg-[#E2F5F7] text-[#3A9EAA] ml-[6px] shrink-0"
              size="small"
            >
              {{ $t('特性') }}
            </Tag>
          </div>
          <div
            class="ml-[8px]"
            @click.stop
          >
            <Checkbox
              class="bg-[#fff]"
              :disabled="!env.name"
              :model-value="isEnvSelected(env)"
              @change="handleEnvSelect(env, $event)"
            />
          </div>
        </div>
        <div
          v-if="group.envs.length === 0"
          class="h-[40px] flex items-center justify-center bg-[#F5F7FA] text-[12px] text-[#979BA5]"
        >
          {{ $t('暂无数据') }}
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import { Checkbox, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { envTypeTagClassMap } from '~/composables/use-env-manager';

  import type { EnvOutput } from '~/@types/v1/env';

  const props = defineProps<{
    envList: EnvOutput[];
  }>();

  const modelValue = defineModel<string[]>({ default: () => [] });
  const { t } = useI18n();
  const FEATURE_ENV_KIND = 'feature';

  // 按环境类型分组的可用环境展示顺序
  const envTypeOrder = ['development', 'test', 'staging', 'production'] as const;
  // 环境分组类型，约束分组配置必须覆盖所有展示类型
  type EnvType = (typeof envTypeOrder)[number];

  // 组件可用环境分组面板样式配置
  const envTypePanelMap: Record<
    EnvType,
    {
      label: string;
    }
  > = {
    development: {
      label: t('开发'),
    },
    test: {
      label: t('测试'),
    },
    staging: {
      label: t('预发布'),
    },
    production: {
      label: t('生产'),
    },
  };

  // 按环境类型将接口返回的环境列表归组，用于模板四列展示
  const groupedEnvList = computed(() =>
    envTypeOrder.map(type => ({
      type,
      ...envTypePanelMap[type],
      envs: props.envList.filter(env => env.type === type),
    })),
  );

  // 获取分组内可用于提交的环境名称列表
  function getGroupEnvNames(envs: EnvOutput[]) {
    return envs.map(env => env.name).filter((name): name is string => !!name);
  }

  // 点击环境条目时切换勾选状态
  function handleEnvItemClick(env: EnvOutput) {
    if (!env.name) return;
    handleEnvSelect(env, !isEnvSelected(env));
  }

  // 切换单个环境勾选状态，并同步到 v-model
  function handleEnvSelect(env: EnvOutput, checked: boolean) {
    if (!env.name) return;
    const selectedEnvNames = modelValue.value || [];
    if (checked) {
      modelValue.value = selectedEnvNames.includes(env.name) ? selectedEnvNames : [...selectedEnvNames, env.name];
      return;
    }

    modelValue.value = selectedEnvNames.filter(name => name !== env.name);
  }

  // 切换分组全选状态，只影响当前分组内的环境
  function handleGroupSelectAll(envs: EnvOutput[], checked: boolean) {
    const groupEnvNames = getGroupEnvNames(envs);
    const selectedEnvNames = modelValue.value || [];
    if (checked) {
      modelValue.value = [...new Set([...selectedEnvNames, ...groupEnvNames])];
      return;
    }

    const groupEnvNameSet = new Set(groupEnvNames);
    modelValue.value = selectedEnvNames.filter(name => !groupEnvNameSet.has(name));
  }

  // 判断单个环境是否已选中
  function isEnvSelected(env: EnvOutput) {
    return !!env.name && (modelValue.value || []).includes(env.name);
  }

  function isFeatureEnv(env: EnvOutput) {
    return env.kind === FEATURE_ENV_KIND;
  }

  // 判断分组内可选环境是否已全部选中
  function isGroupAllSelected(envs: EnvOutput[]) {
    const groupEnvNames = getGroupEnvNames(envs);
    const selectedEnvNames = modelValue.value || [];
    return groupEnvNames.length > 0 && groupEnvNames.every(name => selectedEnvNames.includes(name));
  }

  // 判断分组是否处于半选状态
  function isGroupIndeterminate(envs: EnvOutput[]) {
    const groupEnvNames = getGroupEnvNames(envs);
    const selectedEnvNames = modelValue.value || [];
    const selectedCount = groupEnvNames.filter(name => selectedEnvNames.includes(name)).length;
    return selectedCount > 0 && selectedCount < groupEnvNames.length;
  }
</script>
