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
            :disabled="props.disabled || getGroupEnvValues(group.envs).length === 0"
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
          :class="isEnvDisabled(env) && !isEnvSelected(env) ? 'opacity-60 cursor-not-allowed' : ''"
          @click="handleEnvItemClick(env)"
        >
          <div class="flex flex-1 min-w-0 items-center">
            <ColorIcon
              v-if="props.showDeployIcon"
              class="shrink-0 mr-[8px]"
              :icon="getEnvStatusIcon(env)"
              :size="14"
            />
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
              :disabled="(!isEnvSelected(env) && isEnvStatusDisabled(env)) || props.disabled || !getEnvValue(env)"
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
  import { computed, ref, watch } from 'vue';

  import { Checkbox, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { AppService } from '~/api/modules/v1/app';
  import ColorIcon from '~/components/color-icon.vue';
  import { useDeployStatusMap } from '~/composables/use-deploy-status';
  import { envTypeTagClassMap } from '~/composables/use-env-manager';
  import { useAppDetail } from '~/stores/app-detail';

  import type { AppDeployedEnvOutputObj } from '~/@types/v1/app';
  import type { EnvOutput } from '~/@types/v1/env';

  const props = withDefaults(
    defineProps<{
      /** 整体禁用（只读态） */
      disabled?: boolean;
      /** 这些状态的环境不可勾选且不计入全选，如 ['NotReady'] */
      disabledStatuses?: string[];
      envList: EnvOutput[];
      /** 是否显示环境部署状态图标（为 true 时才会发起部署状态请求） */
      showDeployIcon?: boolean;
      /** v-model 值的来源字段：name（默认，兼容现有调用方）或 id */
      valueKey?: 'id' | 'name';
    }>(),
    {
      disabled: false,
      disabledStatuses: () => [],
      showDeployIcon: false,
      valueKey: 'name',
    },
  );

  const modelValue = defineModel<string[]>({ default: () => [] });
  const { t } = useI18n();
  const { getDeployStatusInfo } = useDeployStatusMap();
  const appDetailStore = useAppDetail();
  const FEATURE_ENV_KIND = 'feature';

  /** 环境名称 -> 部署状态 映射（仅 showDeployIcon 时请求） */
  const appDeployStatusMap = ref<Map<string, AppDeployedEnvOutputObj>>(new Map());

  /** 获取当前应用在各环境的部署状态 */
  async function fetchDeployStatuses(appID: string) {
    const res = await AppService.getAppDeployStatuses({ appID }).catch(() => []);
    const list = (res || []) as AppDeployedEnvOutputObj[];
    appDeployStatusMap.value = new Map(list.filter(item => item.name).map(item => [item.name!, item]));
  }

  /** 仅当开启图标且存在 appID 时才发起部署状态请求 */
  watch(
    () => [props.showDeployIcon, appDetailStore.appID] as const,
    ([showDeployIcon, appID]) => {
      if (showDeployIcon && appID) {
        fetchDeployStatuses(appID);
      } else {
        appDeployStatusMap.value = new Map();
      }
    },
    { immediate: true },
  );

  /** 根据环境获取部署状态对应的 ColorIcon 图标名（与 env-select-panel.vue 保持一致） */
  function getEnvStatusIcon(env: EnvOutput): string {
    const deployStatus = env.name ? appDeployStatusMap.value.get(env.name)?.deployStatus : undefined;
    if (!deployStatus) return 'status-unknown';
    return getDeployStatusInfo(appDetailStore.appType || null, deployStatus).icon || 'status-unknown';
  }

  // 组件可用环境分组展示顺序
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

  /** 环境对应的 v-model 值 */
  function getEnvValue(env: EnvOutput): string | undefined {
    return props.valueKey === 'id' ? env.id : env.name;
  }

  // 获取分组内可用于提交的环境值列表（排除禁用状态）
  function getGroupEnvValues(envs: EnvOutput[]) {
    return envs
      .filter(env => !isEnvDisabled(env))
      .map(env => getEnvValue(env))
      .filter((value): value is string => !!value);
  }

  // 点击环境条目时切换勾选状态
  function handleEnvItemClick(env: EnvOutput) {
    if (props.disabled || !getEnvValue(env)) return;
    // 状态禁用的环境（如 NotReady）不允许新增，但允许取消已选
    if (isEnvStatusDisabled(env) && !isEnvSelected(env)) return;
    handleEnvSelect(env, !isEnvSelected(env));
  }

  // 切换单个环境勾选状态，并同步到 v-model
  function handleEnvSelect(env: EnvOutput, checked: boolean) {
    const envValue = getEnvValue(env);
    if (!envValue) return;
    // 选中操作受状态禁用限制；取消操作不受限制（允许移除已选的禁用环境）
    if (checked && isEnvStatusDisabled(env)) return;
    const selectedValues = modelValue.value || [];
    if (checked) {
      modelValue.value = selectedValues.includes(envValue) ? selectedValues : [...selectedValues, envValue];
      return;
    }

    modelValue.value = selectedValues.filter(value => value !== envValue);
  }

  // 切换分组全选状态，只影响当前分组内的环境
  function handleGroupSelectAll(envs: EnvOutput[], checked: boolean) {
    if (props.disabled) return;
    const groupEnvValues = getGroupEnvValues(envs);
    const selectedValues = modelValue.value || [];
    if (checked) {
      modelValue.value = [...new Set([...selectedValues, ...groupEnvValues])];
      return;
    }

    const groupEnvValueSet = new Set(groupEnvValues);
    modelValue.value = selectedValues.filter(value => !groupEnvValueSet.has(value));
  }

  /** 环境是否整体禁用（只读态或状态命中 disabledStatuses） */
  function isEnvDisabled(env: EnvOutput) {
    return props.disabled || isEnvStatusDisabled(env);
  }

  // 判断单个环境是否已选中
  function isEnvSelected(env: EnvOutput) {
    const envValue = getEnvValue(env);
    return !!envValue && (modelValue.value || []).includes(envValue);
  }

  /** 环境状态是否被禁用（如 NotReady），命中时不可新增但可取消已选 */
  function isEnvStatusDisabled(env: EnvOutput) {
    return props.disabledStatuses.includes(env.status || '');
  }

  function isFeatureEnv(env: EnvOutput) {
    return env.kind === FEATURE_ENV_KIND;
  }

  // 判断分组内可选环境是否已全部选中
  function isGroupAllSelected(envs: EnvOutput[]) {
    const groupEnvValues = getGroupEnvValues(envs);
    const selectedValues = modelValue.value || [];
    return groupEnvValues.length > 0 && groupEnvValues.every(value => selectedValues.includes(value));
  }

  // 判断分组是否处于半选状态
  function isGroupIndeterminate(envs: EnvOutput[]) {
    const groupEnvValues = getGroupEnvValues(envs);
    const selectedValues = modelValue.value || [];
    const selectedCount = groupEnvValues.filter(value => selectedValues.includes(value)).length;
    return selectedCount > 0 && selectedCount < groupEnvValues.length;
  }
</script>
