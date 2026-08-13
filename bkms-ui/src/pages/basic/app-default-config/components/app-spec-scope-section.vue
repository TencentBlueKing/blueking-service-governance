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
  <Form.FormItem
    property="envTypes"
    required
  >
    <div class="flex flex-col">
      <!-- 所有环境：本期暂不支持 -->
      <!-- <Radio
        v-if="showAllEnvOption"
        label="all"
      >
        <span class="text-[14px]">{{ $t('所有环境') }}</span>
        <span class="text-[#979BA5] text-[14px]">{{ $t('( 所有环境都生效，包括新增的环境也会生效 )') }}</span>
      </div> -->
      <div>
        <span class="text-[14px] text-[#4D4F56] mr-[4px]">{{ $t('按环境类型') }}</span>
        <span class="text-[#979BA5] text-[14px]">{{ $t('( 针对环境类型生效，包括新增的环境也会生效 )') }}</span>
      </div>
      <!-- 按环境类型：多选；已被占用或不支持的环境分类禁用 -->
      <div
        v-if="scopeType === 'env_type'"
        class="env-type-checkbox-group flex items-center"
      >
        <Checkbox.Group v-model="envTypes">
          <Checkbox
            v-for="envType in envTypeOptions"
            :key="envType.id"
            :disabled="envType.disabled"
            :label="envType.id"
          >
            <span
              v-bk-tooltips="{
                content: envType.tooltip,
                disabled: !envType.tooltip,
              }"
            >
              <Tag
                class="!justify-center"
                :class="[envTypeTagClassMap[envType.id], envType.disabled ? 'cursor-not-allowed' : 'cursor-pointer']"
                @click="handleTagClick(envType)"
              >
                {{ envType.name }}
              </Tag>
            </span>
          </Checkbox>
        </Checkbox.Group>
      </div>
      <!-- 特定环境：本期暂不支持
      <Radio
        class="ml-0!"
        label="specific_envs"
      >
        <span class="text-[14px]">{{ $t('特定环境') }}</span>
        <span class="text-[#979BA5] text-[14px]">{{
          $t('( 针对指定的环境生效，新增同类环境不会自动生效 )')
        }}</span>
      </Radio>
      -->
    </div>
  </Form.FormItem>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import { Checkbox, Form, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';

  import { ALL_ENV_TYPES } from './app-spec-scope';

  import type { AppSpecScopeType } from './app-spec-scope';

  interface Props {
    /** 已被其他规则占用的环境类型 */
    occupiedEnvTypes?: string[];
    /** 是否展示「所有环境」（新增配置项侧滑） */
    showAllEnvOption?: boolean;
    /** 可选环境类型范围 */
    supportedEnvTypes?: string[];
  }

  const props = withDefaults(defineProps<Props>(), {
    occupiedEnvTypes: () => [],
    showAllEnvOption: true,
    supportedEnvTypes: () => ALL_ENV_TYPES,
  });

  const { t } = useI18n();

  const scopeType = defineModel<AppSpecScopeType>('scopeType', { default: 'env_type' });
  const envTypes = defineModel<string[]>('envTypes', { default: () => [] });

  /** 环境类型选项：展示全部类型，不可选的置灰；tooltip 在选项上直接给出 */
  const envTypeOptions = computed(() => {
    const supportedSet = new Set(props.supportedEnvTypes);
    const occupiedSet = new Set(props.occupiedEnvTypes);

    return ALL_ENV_TYPES.map(id => {
      const unsupported = !supportedSet.has(id);
      return {
        id,
        name: envTypeMap[id]?.name || id,
        disabled: unsupported || occupiedSet.has(id),
        // 目前仅开发模式会禁用 production
        tooltip: unsupported && id === 'production' ? t('生产环境不支持使用开发模式') : '',
      };
    });
  });

  /** 点击 Tag 时同步勾选/取消对应环境类型 */
  function handleTagClick(envType: { disabled: boolean; id: string }) {
    if (envType.disabled) return;

    const selected = envTypes.value;
    envTypes.value = selected.includes(envType.id)
      ? selected.filter(id => id !== envType.id)
      : [...selected, envType.id];
  }
</script>

<style lang="postcss" scoped>
  .env-type-checkbox-group {
    :deep(.bk-checkbox ~ .bk-checkbox) {
      margin-left: 32px;
    }

    :deep(.bk-checkbox.is-disabled) {
      cursor: not-allowed;
    }

    :deep(.bk-checkbox.is-disabled .bk-checkbox-input) {
      border-color: #c4c6cc;
    }

    :deep(.bk-checkbox.is-disabled .bk-checkbox-label) {
      cursor: not-allowed;
    }
  }
</style>
