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

<!-- 环境视角新版环境选择器 -->
<template>
  <Popover
    ref="popoverRef"
    :arrow="false"
    ext-cls="c-env-select-v2-popover"
    placement="bottom-start"
    theme="light"
    trigger="click"
    :width="626"
    @after-hidden="handlePopoverHidden"
    @after-show="handlePopoverShow"
  >
    <div
      v-bind="$attrs"
      :class="[
        'flex items-center min-w-[280px] h-[32px] border border-[#c4c6cc] rounded-[2px] bg-[#FFF] cursor-pointer overflow-hidden transition-border-color duration-200 hover:border-[#979BA5]',
        { '!border-[#3a84ff] shadow-[0_0_3px_0_#a3c5fd]': isPopoverVisible },
      ]"
    >
      <div class="flex items-center h-full px-[10px] border-r-[1px] border-[#c4c6cc]">{{ $t('环境视角') }}</div>
      <div class="flex items-center flex-1 min-w-0 px-[8px]">
        <span
          v-if="selectedEnvItem"
          class="inline-flex items-center gap-[4px] min-w-0 text-[12px] leading-[22px] text-[#4D4F56]"
        >
          <span class="truncate">{{ selectedEnvItem.displayName }}</span>
          <Tag
            v-if="selectedEnvItem?.type && envTypeMap[selectedEnvItem.type]"
            class="shrink-0"
            size="small"
            :theme="envTypeMap[selectedEnvItem.type]?.theme || ''"
          >
            {{ envTypeMap[selectedEnvItem.type]?.name || '' }}
          </Tag>
          <Tag
            v-if="isFeatureEnv(selectedEnvItem)"
            class="bg-[#E2F5F7] text-[#3A9EAA] shrink-0"
            size="small"
          >
            {{ $t('特性') }}
          </Tag>
        </span>
        <span
          v-else-if="isDefaultSelected"
          class="text-[12px] leading-[22px] text-[#4D4F56]"
        >
          {{ $t('默认配置') }}
        </span>
        <span
          v-else
          class="text-[14px] leading-[22px] text-[#C4C6CC]"
        >
          {{ $t('请选择') }}
        </span>
      </div>
      <AngleDownLine
        :class="['text-[#979BA5] mr-[10px] transition-transform duration-200', { 'rotate-180': isPopoverVisible }]"
      />
    </div>

    <!-- 下拉面板 -->
    <template #content>
      <div class="min-h-[100px]">
        <div class="flex items-center mb-[8px]">
          <Input
            v-model.trim="searchKeyword"
            behavior="simplicity"
            class="flex-1 mt-[8px]"
            clearable
            :placeholder="$t('请输入关键词')"
          >
            <template #prefix>
              <div class="flex items-center justify-center text-[#979BA5]">
                <Search class="ml-[2px] text-[16px]" />
              </div>
            </template>
          </Input>
          <div
            class="h-[40px] flex items-center border-l-[1px] border-b-[1px] border-[#DCDEE5] ml-[-1px] pl-[8px] pb-[2px]"
          >
            <Checkbox
              v-model="onlyModified"
              class="shrink-0"
            >
              <span class="text-[12px] text-[#63656E]">{{ $t('仅显示已修改环境') }}</span>
            </Checkbox>
          </div>
        </div>

        <!-- 分组列表 -->
        <div
          v-if="filteredGroups.length"
          class="flex gap-[8px]"
        >
          <div
            v-for="group in filteredGroups"
            :key="group.type"
            class="flex-1 min-w-0"
          >
            <!-- 分组标题 -->
            <div class="h-[32px] flex items-center px-[8px] bg-[#f5f7fa]">
              <Tag :theme="envTypeMap[group.type]?.theme || ''">
                {{ group.label }}
              </Tag>
            </div>
            <!-- 环境项列表 -->
            <div class="env-list-scroll max-h-[224px] overflow-y-auto">
              <div
                v-for="env in group.envs"
                :key="env.name"
                :class="[
                  'flex flex-col justify-center h-[42px] px-[8px] cursor-pointer text-[12px] text-[#4D4F56] transition-bg-color duration-150',
                  { 'feature-env-child': env.isFeatureChild },
                  { '!bg-[#E1ECFF] !text-[#3a84ff]': isSelected(env) },
                  { 'hover:bg-[#F5F7FA]': !isSelected(env) },
                ]"
                @click="handleSelectEnv(env)"
              >
                <span
                  v-if="env.isFeatureChild"
                  class="feature-env-branch"
                ></span>
                <div class="flex items-center min-w-0">
                  <OverflowTitle
                    class="min-w-0"
                    type="tips"
                  >
                    {{ env.displayName || '--' }}
                  </OverflowTitle>
                  <Tag
                    v-if="isFeatureEnv(env)"
                    class="bg-[#E2F5F7] text-[#3A9EAA] ml-[4px] shrink-0"
                    size="small"
                  >
                    {{ $t('特性') }}
                  </Tag>
                </div>
                <!-- 已修改/未修改状态（默认配置不显示） -->
                <span
                  v-if="env.type !== 'default'"
                  :class="['text-[10px] shrink-0', isModified(env.name) ? 'text-[#F8B64F]' : 'text-[#C4C6CC]']"
                >
                  {{ isModified(env.name) ? $t('已修改') : $t('未修改') }}
                </span>
              </div>
              <!-- 空状态 -->
              <div
                v-if="group.envs.length === 0"
                class="py-[16px] px-[12px] text-[12px] text-[#c4c6cc] text-center"
              >
                {{ searchKeyword.trim() ? $t('无匹配数据') : $t('暂无数据') }}
              </div>
            </div>
          </div>
        </div>
        <!-- 全局空状态：仅显示已修改环境但无匹配数据 -->
        <div
          v-else
          class="py-[16px] px-[12px] text-[12px] text-[#c4c6cc] text-center"
        >
          {{ $t('无匹配数据') }}
        </div>
      </div>
    </template>
  </Popover>
</template>

<script lang="ts" setup>
  import { computed, ref } from 'vue';

  import { Checkbox, Input, OverflowTitle, Popover, Tag } from 'bkui-vue';
  import { AngleDownLine, Search } from 'bkui-vue/lib/icon';
  import { buildStandardEnvMap, envTypeMap, getFeatureSourceEnv } from '~/composables/use-env-manager';

  import type { EnvOutput } from '~/@types/v1/env';

  interface Emits {
    (e: 'change', envName: string): void;
  }

  interface IProps {
    /** 环境列表（不含默认配置） */
    envList: EnvOutput[];
    /** 选中的值（环境 name，默认配置传 '__default__' 或空字符串） */
    modelValue?: string;
    /** 已修改的环境 name 列表 */
    modifiedEnvNames?: string[];
  }

  defineOptions({ inheritAttrs: false });

  const props = withDefaults(defineProps<IProps>(), {
    modelValue: '__default__',
    modifiedEnvNames: () => [],
  });
  const emits = defineEmits<Emits>();

  const popoverRef = ref<InstanceType<typeof Popover> | null>(null);
  const isPopoverVisible = ref(false);
  const searchKeyword = ref('');
  const onlyModified = ref(false);

  const envTypeOrder = ['development', 'test', 'production'];
  const FEATURE_ENV_KIND = 'feature';

  type GroupEnvItem = EnvOutput & {
    displayName: string;
    isFeatureChild?: boolean;
    name: string;
    type: string;
  };

  interface GroupItem {
    envs: GroupEnvItem[];
    isDefault?: boolean;
    label: string;
    type: string;
  }

  /** 是否选中默认配置 */
  const isDefaultSelected = computed(() => props.modelValue === '' || props.modelValue === '__default__');

  /** 当前选中的环境项 */
  const selectedEnvItem = computed(() => {
    if (isDefaultSelected.value) return null;
    return props.envList.find(item => item.name === props.modelValue);
  });

  function isFeatureEnv(env?: EnvOutput) {
    return env?.kind === FEATURE_ENV_KIND;
  }

  function isKeywordMatched(env: EnvOutput, keyword: string) {
    if (!keyword) return true;
    return env.displayName?.toLowerCase().includes(keyword) || env.name?.toLowerCase().includes(keyword);
  }

  function isModifiedVisible(env: EnvOutput) {
    if (!onlyModified.value) return true;
    return !!env.name && props.modifiedEnvNames.includes(env.name);
  }

  /** 分组数据（含过滤） */
  const filteredGroups = computed(() => {
    const keyword = searchKeyword.value.trim().toLowerCase();
    const groups: GroupItem[] = [];

    // 默认分组（仅在非过滤状态下显示）
    if (!onlyModified.value && !keyword) {
      groups.push({
        type: 'default',
        label: '默认',
        envs: [{ name: '__default__', displayName: '默认配置', type: 'default' }],
        isDefault: true,
      });
    }

    const standardEnvs = props.envList.filter(env => !isFeatureEnv(env));
    const standardEnvMap = buildStandardEnvMap(standardEnvs);
    const featureEnvMap = new Map<EnvOutput, EnvOutput[]>();
    const orphanFeatureEnvs: EnvOutput[] = [];

    props.envList.filter(isFeatureEnv).forEach(env => {
      const sourceEnv = getFeatureSourceEnv(env, standardEnvMap);
      if (sourceEnv) {
        const list = featureEnvMap.get(sourceEnv) || [];
        list.push(env);
        featureEnvMap.set(sourceEnv, list);
      } else {
        orphanFeatureEnvs.push(env);
      }
    });

    // 常规分组
    envTypeOrder.forEach(type => {
      const typeConfig = envTypeMap[type];
      const envs: GroupEnvItem[] = [];

      standardEnvs
        .filter(env => env.type === type)
        .forEach(env => {
          const sourceMatched = isKeywordMatched(env, keyword);
          const children = featureEnvMap.get(env) || [];
          const visibleChildren = children.filter(child => {
            if (!isModifiedVisible(child)) return false;
            if (!keyword) return true;
            return sourceMatched || isKeywordMatched(child, keyword);
          });
          const sourceVisible = isModifiedVisible(env) && (!keyword || sourceMatched);
          const shouldShowSource = sourceVisible || (isModifiedVisible(env) && visibleChildren.length > 0);

          if (shouldShowSource) {
            envs.push({
              ...env,
              displayName: env.displayName ?? '',
              name: env.name ?? '',
              type: env.type ?? '',
            });
          }
          envs.push(
            ...visibleChildren.map(child => ({
              ...child,
              displayName: child.displayName ?? '',
              isFeatureChild: shouldShowSource,
              name: child.name ?? '',
              type: child.type ?? '',
            })),
          );
        });

      orphanFeatureEnvs
        .filter(env => env.type === type)
        .filter(env => isModifiedVisible(env) && (!keyword || isKeywordMatched(env, keyword)))
        .forEach(env => {
          envs.push({
            ...env,
            displayName: env.displayName ?? '',
            name: env.name ?? '',
            type: env.type ?? '',
          });
        });

      groups.push({
        type,
        label: typeConfig?.name || type,
        envs,
      });
    });

    // 过滤空分组（仅显示已修改环境时）
    return groups.filter(group => group.isDefault || !onlyModified.value || group.envs.length > 0);
  });

  function handlePopoverHidden() {
    isPopoverVisible.value = false;
    searchKeyword.value = '';
    onlyModified.value = false;
  }

  function handlePopoverShow() {
    isPopoverVisible.value = true;
  }

  /** 选择环境 */
  function handleSelectEnv(env: GroupEnvItem) {
    emits('change', env.name);
    popoverRef.value?.hide();
  }

  /** 判断环境是否已修改 */
  function isModified(envName: string): boolean {
    return props.modifiedEnvNames.includes(envName);
  }

  /** 判断环境是否选中 */
  function isSelected(env: GroupEnvItem): boolean {
    return env.name === props.modelValue;
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
  /* 覆盖 Popover 面板样式 */
  :deep(.bk-popover-reference) {
    outline: none;
  }
  .feature-env-child {
    position: relative;
    padding-left: 28px;
  }
  .feature-env-branch {
    position: absolute;
    top: 0;
    left: 21px;
    width: 16px;
    height: 16px;
    pointer-events: none;

    &::before {
      position: absolute;
      top: 0;
      left: -12px;
      width: 16px;
      height: 16px;
      border-left: 1px solid #dcdee5;
      border-bottom: 1px solid #dcdee5;
      border-bottom-left-radius: 8px;
      content: '';
    }
  }
</style>
<style lang="postcss">
  .c-env-select-v2-popover {
    padding: 0 8px 8px 8px !important;
    border-radius: 2px !important;
    border: none;
    box-shadow: 0 2px 4px 0 #1919290d !important;

    .env-list-scroll {
      &::-webkit-scrollbar {
        width: 6px;
        height: 6px;
      }

      &::-webkit-scrollbar-thumb {
        background: #dcdee5;
        box-shadow: inset 0 0 6px #cccccc4d;

        &:hover {
          background: #dcdee5;
        }
      }
    }
  }
</style>
