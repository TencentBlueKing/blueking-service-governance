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

<!-- 环境选择器（环境分类） -->
<template>
  <Popover
    ref="popoverRef"
    :arrow="false"
    ext-cls="c-env-select-popover"
    placement="bottom-start"
    theme="light"
    trigger="click"
    :width="800"
    @after-hidden="handlePopoverHidden"
    @after-show="handlePopoverShow"
  >
    <div
      v-bind="$attrs"
      :class="[
        'flex items-center min-w-[280px] max-w-[626px] border border-[#c4c6cc] rounded-[2px] bg-[#FFF] cursor-pointer overflow-hidden transition-border-color duration-200 hover:border-[#979BA5]',
        { '!border-[#3a84ff] shadow-[0_0_3px_0_#a3c5fd]': isPopoverVisible },
        mode === 'multi' ? 'h-auto min-h-[32px] !w-[626px]' : 'h-[32px]',
      ]"
    >
      <!-- 模式切换区域 -->
      <Select
        v-if="multiSelectable"
        class="max-w-[74px] shrink-0 self-start"
        :clearable="false"
        :model-value="mode"
        :popover-min-width="80"
        @change="handleModeChange"
        @click.stop
      >
        <Select.Option
          id="single"
          :name="$t('环境')"
        />
        <Select.Option
          id="multi"
          :name="$t('多环境')"
        />
      </Select>
      <div
        v-else
        class="flex items-center h-[32px] px-[10px] border-r-[1px] border-[#c4c6cc] shrink-0"
      >
        {{ $t('环境') }}
      </div>
      <!-- 已选环境展示 -->
      <div class="flex items-center flex-1 min-w-0 px-[8px] py-[4px]">
        <!-- 多选模式 -->
        <template v-if="mode === 'multi'">
          <OverflowTags
            v-if="selectedMultiEnvItems.length > 0"
            :key="mode"
            :gap="4"
            :more-tag-width="56"
            :tag-extra-width="24"
            :tags="multiEnvDisplayNames"
          >
            <template #default="{ index }">
              <Tag
                closable
                @click.stop
                @close="handleRemoveEnv(selectedMultiEnvItems[index])"
              >
                <!-- 多环境已选择展示区 -->
                <span class="inline-flex items-center gap-[4px]">
                  <span>{{ selectedMultiEnvItems[index].displayName }}</span>
                  <!-- 环境 Tag -->
                  <Tag
                    v-if="getEnvTypeConfig(selectedMultiEnvItems[index])"
                    :class="getEnvTypeTagClass(selectedMultiEnvItems[index])"
                    size="small"
                  >
                    {{ getEnvTypeConfig(selectedMultiEnvItems[index])?.name || '' }}
                  </Tag>
                  <Tag
                    v-if="isFeatureEnv(selectedMultiEnvItems[index])"
                    class="bg-[#E2F5F7] text-[#3A9EAA]"
                    size="small"
                  >
                    {{ $t('特性') }}
                  </Tag>
                </span>
              </Tag>
            </template>
          </OverflowTags>
          <span
            v-else
            class="text-[12px] leading-[22px] text-[#C4C6CC]"
          >
            {{ $t('请选择') }}
          </span>
        </template>
        <!-- 单选模式 -->
        <template v-else>
          <span
            v-if="selectedEnvItem"
            class="inline-flex items-center gap-[4px] min-w-0 text-[12px] leading-[22px] text-[#4D4F56]"
          >
            <span class="truncate">{{ selectedEnvItem.displayName }}</span>
            <Tag
              v-if="selectedEnvItem?.type && envTypeMap[selectedEnvItem.type]"
              :class="envTypeTagClassMap[selectedEnvItem.type]"
              size="small"
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
            v-else
            class="text-[12px] leading-[22px] text-[#C4C6CC]"
          >
            {{ $t('请选择') }}
          </span>
        </template>
      </div>
      <AngleDownLine
        :class="[
          'text-[#979BA5] mr-[10px] shrink-0 self-start mt-[10px] transition-transform duration-200',
          { 'rotate-180': isPopoverVisible },
        ]"
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
              v-model="onlyDeployed"
              class="shrink-0"
            >
              <span class="text-[12px] text-[#4D4F56]">{{ $t('仅显示已部署环境') }}</span>
            </Checkbox>
          </div>
        </div>
        <!-- 无已部署环境 -->
        <div
          v-if="onlyDeployed && filteredGroups.length === 0"
          class="py-[16px] px-[12px] text-[12px] text-[#c4c6cc] text-center"
        >
          {{ $t('无匹配数据') }}
        </div>
        <!-- 分组列表 -->
        <div
          v-else
          class="flex gap-[8px]"
        >
          <div
            v-for="group in filteredGroups"
            :key="group.type"
            class="flex-1 min-w-0"
          >
            <!-- 分组标题 -->
            <div class="h-[32px] flex items-center justify-between px-[8px] bg-[#f5f7fa]">
              <Tag :class="envTypeTagClassMap[group.type]">
                {{ group.label }}
              </Tag>
              <Checkbox
                v-if="mode === 'multi'"
                v-bk-tooltips="$t('全选')"
                :disabled="getSelectableGroupEnvNames(group.envs).length === 0"
                :indeterminate="isGroupIndeterminate(group.envs)"
                :model-value="isGroupAllSelected(group.envs)"
                @change="handleGroupSelectAll(group.envs, $event)"
                @click.stop
              ></Checkbox>
            </div>
            <!-- 环境项列表 -->
            <div class="env-list-scroll max-h-[224px] overflow-y-auto">
              <div
                v-for="env in group.envs"
                :key="env.name"
                v-bk-tooltips="{
                  content: $t('环境未配置集群资源，无法部署应用'),
                  disabled: env.status !== 'NotReady',
                  placement: 'bottom',
                }"
                :class="[
                  'flex items-center h-[32px] px-[8px] cursor-pointer text-[12px] text-[#4D4F56] transition-bg-color duration-150',
                  { 'feature-env-child': env.isFeatureChild },
                  { '!bg-[#e1ecff] !text-[#3a84ff]': isSelected(env) },
                  { 'cursor-not-allowed opacity-60': env.status === 'NotReady' },
                  { 'hover:bg-[#F5F7FA]': !isSelected(env) && env.status !== 'NotReady' },
                ]"
                @click="handleSelectEnv(env)"
              >
                <span
                  v-if="env.isFeatureChild"
                  class="feature-env-branch"
                ></span>
                <div class="flex items-center flex-1 min-w-0">
                  <span class="mr-[4px] flex shrink-0">
                    <StatusDotIcon
                      :icon="getEnvDeployIcon(env)"
                      :size="12"
                    />
                  </span>
                  <span class="truncate">{{ env.displayName }}</span>
                  <Tag
                    v-if="isFeatureEnv(env)"
                    class="bg-[#E2F5F7] text-[#3A9EAA] ml-[4px] shrink-0"
                    size="small"
                  >
                    {{ $t('特性') }}
                  </Tag>
                </div>
                <Done
                  v-if="mode === 'multi' && isSelected(env)"
                  :height="26"
                  :width="26"
                ></Done>
              </div>
              <div
                v-if="group.envs.length === 0"
                class="py-[16px] px-[12px] text-[12px] text-[#c4c6cc] text-center"
              >
                {{ searchKeyword.trim() ? $t('无匹配数据') : $t('暂无数据') }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </Popover>
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Checkbox, Input, Popover, Select, Tag } from 'bkui-vue';
  import { Done } from 'bkui-vue/lib/icon';
  import { AngleDownLine, Search } from 'bkui-vue/lib/icon';
  import { isEqual } from 'lodash-es';
  import { AppService } from '~/api/modules/v1/app';
  import { EnvService } from '~/api/modules/v1/env';
  import OverflowTags from '~/components/overflow-tags.vue';
  import StatusDotIcon from '~/components/status-dot-icon.vue';
  import { useDeployStatusMap } from '~/composables/use-deploy-status';
  import {
    buildStandardEnvMap,
    envTypeMap,
    envTypeTagClassMap,
    getFeatureSourceEnv,
  } from '~/composables/use-env-manager';
  import { useAppDetail } from '~/stores/app-detail';
  import { useDeployEnvStore } from '~/stores/deploy-env';

  import type { AppDeployedEnvOutputObj } from '~/@types/v1/app';
  import type { EnvOutput } from '~/@types/v1/env';

  interface Emits {
    (e: 'update:modelValue', value: string): void;
    (e: 'update:modelValues', value: string[]): void;
    (e: 'update:item', item?: EnvOutput): void;
    (e: 'update:items', items: EnvOutput[]): void;
    (e: 'update:envList', list: EnvOutput[]): void;
    (e: 'update:deployStatusList', list: AppDeployedEnvOutputObj[]): void;
    (e: 'update:loading', loading: boolean): void;
    (e: 'update:mode', mode: 'multi' | 'single'): void;
  }

  interface EnvFeatureRelations {
    featureEnvMap: Map<EnvOutput, EnvOutput[]>;
    orphanFeatureEnvs: EnvOutput[];
    standardEnvs: EnvOutput[];
  }

  interface EnvGroup {
    envs: EnvSelectItem[];
    label: string;
    type: string;
  }

  type EnvSelectItem = EnvOutput & {
    isFeatureChild?: boolean;
  };

  interface IProps {
    /** 当无选中值时是否自动选中第一个可用环境 */
    initFirstEnvWhenEmpty?: boolean;
    /** 选择模式 */
    mode?: 'multi' | 'single';
    /** 单选模式当前选中环境名 */
    modelValue?: string;
    /** 多选模式当前选中环境名列表 */
    modelValues?: string[];
    /** 是否允许切换单选/多选模式 */
    multiSelectable?: boolean;
    /** 单选值暂未出现在环境列表时，是否保留外部传入值 */
    preserveMissingModelValue?: boolean;
    /** 业务类型标识 */
    type?: string;
  }

  defineOptions({ inheritAttrs: false });

  const props = defineProps<IProps>();
  const emits = defineEmits<Emits>();

  const envStore = useDeployEnvStore();
  const appDetailStore = useAppDetail();
  const { getDeployStatusInfo } = useDeployStatusMap();

  const popoverRef = ref<InstanceType<typeof Popover> | null>(null);
  /** 数据加载状态 */
  const isLoading = ref(false);
  /** 下拉面板是否可见 */
  const isPopoverVisible = ref(false);
  /** 全部环境列表 */
  const envList = ref<EnvOutput[]>([]);
  /** 环境名称到部署状态的映射 */
  const appDeployStatusMap = ref<Map<string, AppDeployedEnvOutputObj>>(new Map());
  /** 搜索关键词 */
  const searchKeyword = ref('');
  /** 是否仅显示已部署环境 */
  const onlyDeployed = ref(false);
  /** 当前选择模式 */
  const mode = ref<'multi' | 'single'>(props.mode || 'single');

  /** 环境类型分组顺序：开发 -> 测试 -> 预发布 -> 生产 */
  const envTypeOrder = ['development', 'test', 'staging', 'production'];
  const FEATURE_ENV_KIND = 'feature';

  /** 单选模式下当前选中的环境对象 */
  const selectedEnvItem = computed(() => envList.value.find(item => item.name === props.modelValue));
  /** 多选模式下当前选中的所有环境对象 */
  const selectedMultiEnvItems = computed(() => {
    if (mode.value !== 'multi') return [];
    return (props.modelValues || [])
      .map(name => envList.value.find(item => item.name === name))
      .filter((item): item is EnvOutput => !!item);
  });
  /** 多选模式的 Tag 名称列表，供 OverflowTags 进行宽度估算 */
  const multiEnvDisplayNames = computed(() => selectedMultiEnvItems.value.map(item => item.displayName ?? ''));
  const normalizedSearchKeyword = computed(() => searchKeyword.value.trim().toLowerCase());
  /** 父子关系仅随环境列表变化，避免搜索或部署过滤变化时重复构建映射 */
  const envFeatureRelations = computed(() => buildEnvFeatureRelations(envList.value));

  /** 将特性环境关联到来源环境，无法关联的特性环境按独立环境处理 */
  function buildEnvFeatureRelations(list: EnvOutput[]): EnvFeatureRelations {
    const standardEnvs = list.filter(env => !isFeatureEnv(env));
    const standardEnvMap = buildStandardEnvMap(standardEnvs);
    const featureEnvMap = new Map<EnvOutput, EnvOutput[]>();
    const orphanFeatureEnvs: EnvOutput[] = [];

    list.filter(isFeatureEnv).forEach(env => {
      const sourceEnv = getFeatureSourceEnv(env, standardEnvMap);
      if (sourceEnv) {
        const children = featureEnvMap.get(sourceEnv) || [];
        children.push(env);
        featureEnvMap.set(sourceEnv, children);
      } else {
        orphanFeatureEnvs.push(env);
      }
    });

    return {
      featureEnvMap,
      orphanFeatureEnvs,
      standardEnvs,
    };
  }

  /** 构建指定环境类型的展示分组 */
  function buildEnvGroup(type: string, relations: EnvFeatureRelations, keyword: string): EnvGroup {
    const typeConfig = envTypeMap[type];
    return {
      type,
      label: typeConfig?.name || type,
      envs: [
        ...getGroupedStandardEnvs(type, relations, keyword),
        ...getVisibleOrphanFeatureEnvs(type, relations, keyword),
      ],
    };
  }

  /** 获取环境类型对应的展示配置 */
  function getEnvTypeConfig(env?: EnvOutput) {
    return env?.type ? envTypeMap[env.type] : undefined;
  }

  /** 获取环境类型对应的无边框 Tag 样式 */
  function getEnvTypeTagClass(env?: EnvOutput) {
    return env?.type ? envTypeTagClassMap[env.type] : '';
  }

  /** 组装指定类型下的标准环境及其可见特性环境 */
  function getGroupedStandardEnvs(type: string, relations: EnvFeatureRelations, keyword: string) {
    const envs: EnvSelectItem[] = [];

    relations.standardEnvs
      .filter(env => env.type === type)
      .forEach(env => {
        const visibleChildren = getVisibleFeatureChildren(env, relations.featureEnvMap, keyword);
        const showSource = shouldShowStandardEnv(env, visibleChildren, keyword);

        if (showSource) {
          envs.push(env);
        }
        // 来源环境未显示时，特性环境按普通条目展示，避免出现无父级的缩进。
        envs.push(...visibleChildren.map(child => ({ ...child, isFeatureChild: showSource })));
      });

    return envs;
  }

  /** 获取来源环境下满足过滤条件的特性环境 */
  function getVisibleFeatureChildren(env: EnvOutput, featureEnvMap: Map<EnvOutput, EnvOutput[]>, keyword: string) {
    const sourceMatched = isKeywordMatched(env, keyword);
    return (featureEnvMap.get(env) || []).filter(child => {
      if (!isEnvDeployedVisible(child)) return false;
      if (!keyword) return true;
      // 搜索命中来源环境时展示其全部可见特性环境；否则仅展示自身命中的特性环境。
      return sourceMatched || isKeywordMatched(child, keyword);
    });
  }

  /** 获取指定类型下满足过滤条件的孤立特性环境 */
  function getVisibleOrphanFeatureEnvs(type: string, relations: EnvFeatureRelations, keyword: string) {
    return relations.orphanFeatureEnvs.filter(env => env.type === type && isEnvFilterMatched(env, keyword));
  }

  /** 判断环境是否满足仅已部署过滤条件 */
  function isEnvDeployedVisible(env: EnvOutput) {
    if (!onlyDeployed.value) return true;
    return !!env.name && appDeployStatusMap.value.get(env.name)?.deployStatus === 'deployed';
  }

  /** 判断环境是否同时满足部署状态和关键词过滤 */
  function isEnvFilterMatched(env: EnvOutput, keyword: string) {
    return isEnvDeployedVisible(env) && (!keyword || isKeywordMatched(env, keyword));
  }

  /** 判断环境是否为特性环境 */
  function isFeatureEnv(env?: EnvOutput) {
    return env?.kind === FEATURE_ENV_KIND;
  }

  /** 判断环境名称或展示名称是否命中关键词 */
  function isKeywordMatched(env: EnvOutput, keyword: string) {
    if (!keyword) return true;
    return env.displayName?.toLowerCase().includes(keyword) || env.name?.toLowerCase().includes(keyword);
  }

  /** 判断过滤后的环境分组是否需要展示 */
  function shouldShowGroup(group: EnvGroup) {
    return !onlyDeployed.value || group.envs.length > 0;
  }

  /** 判断标准环境自身或其特性环境命中时是否展示 */
  function shouldShowStandardEnv(env: EnvOutput, visibleChildren: EnvOutput[], keyword: string) {
    // 子环境命中搜索时补充显示来源环境，但来源环境仍需满足部署过滤条件。
    return isEnvFilterMatched(env, keyword) || (isEnvDeployedVisible(env) && visibleChildren.length > 0);
  }

  /** 将不可用环境稳定地排列到列表末尾 */
  function sortEnvList(list: EnvOutput[]) {
    return [...list].sort((a, b) => {
      const aDisabled = a.status === 'NotReady' ? 1 : 0;
      const bDisabled = b.status === 'NotReady' ? 1 : 0;
      return aDisabled - bDisabled;
    });
  }

  /**
   * 分组后的环境列表（含搜索和仅已部署过滤）
   * 按 envTypeOrder 排序，每个分组包含 type、label、envs 三个字段
   */
  const filteredGroups = computed<EnvGroup[]>(() => {
    const keyword = normalizedSearchKeyword.value;
    return envTypeOrder.map(type => buildEnvGroup(type, envFeatureRelations.value, keyword)).filter(shouldShowGroup);
  });

  /**
   * 多选模式值变更的统一出口
   * 自动过滤 NotReady 环境，并同步选中项与 modelValues
   */
  function emitMultiEnvChange(values: string[], options: { fallbackWhenEmpty?: boolean } = {}) {
    let validValues = values.filter(v => envList.value.some(item => item.name === v && item.status !== 'NotReady'));
    if (options.fallbackWhenEmpty && values.length && validValues.length === 0 && props.initFirstEnvWhenEmpty) {
      const firstEnv = envList.value.find(item => item.status !== 'NotReady');
      validValues = firstEnv?.name ? [firstEnv.name] : [];
    }
    if (!isEqual(validValues, props.modelValues || [])) {
      emits('update:modelValues', validValues);
    }
    const selectedItems = validValues
      .map(name => envList.value.find(item => item.name === name))
      .filter((item): item is EnvOutput => !!item);
    emits('update:items', selectedItems);
  }

  /** 获取当前应用在各环境的部署状态 */
  async function getDeployStatuses() {
    if (!appDetailStore.appID) {
      appDeployStatusMap.value = new Map();
      emits('update:deployStatusList', []);
      return;
    }
    const res = await AppService.getAppDeployStatuses({ appID: appDetailStore.appID }).catch(() => []);
    const list = (res || []) as AppDeployedEnvOutputObj[];
    appDeployStatusMap.value = new Map(list.filter(item => item.name).map(item => [item.name!, item]));
    emits('update:deployStatusList', list);
  }

  /** 根据环境获取部署状态对应的状态图标名 */
  function getEnvDeployIcon(env: EnvOutput): string {
    const deployStatus = env.name ? appDeployStatusMap.value.get(env.name)?.deployStatus : undefined;
    if (!deployStatus) return 'status-unknown';
    return getDeployStatusInfo(appDetailStore.appType || null, deployStatus).icon || 'status-unknown';
  }

  /** 获取环境列表，NotReady 环境排到末尾 */
  async function getEnvList() {
    if (!appDetailStore.appID) {
      envList.value = [];
      emits('update:envList', []);
      return;
    }
    const list = await EnvService.listAppEnvs({
      appID: appDetailStore.appID,
    }).catch(() => []);
    envList.value = sortEnvList(list);
    emits('update:envList', envList.value);
  }

  /** 单选模式的选中值变更处理 */
  function handleEnvChange(env: string) {
    const envItem = envList.value.find(item => item.name === env && item.status !== 'NotReady');
    envStore.updateCurrentEnv(envItem?.name || '');
    emits('update:item', envItem);
    emits('update:modelValue', envItem?.name || '');
  }

  /** 组件初始化：拉取环境列表并根据选择模式初始化选中值 */
  async function handleGetEnvList() {
    isLoading.value = true;
    emits('update:loading', true);
    await getEnvList();
    if (mode.value === 'multi') {
      // 多选模式初始化
      if (props.modelValues?.length) {
        emitMultiEnvChange(props.modelValues, { fallbackWhenEmpty: true });
      } else if (props.initFirstEnvWhenEmpty) {
        const firstEnv = envList.value.find(item => item.status !== 'NotReady');
        if (firstEnv?.name) {
          emitMultiEnvChange([firstEnv.name]);
        }
      }
    } else {
      // 单选模式初始化
      if (props.modelValue) {
        const selectedEnv = envList.value.find(item => item.name === props.modelValue);
        if (!selectedEnv && props.preserveMissingModelValue) {
          emits('update:modelValue', props.modelValue);
        } else {
          handleEnvChange(props.modelValue);
        }
      } else if (props.initFirstEnvWhenEmpty) {
        const currentEnvExists = envStore.currentEnv && envList.value.some(item => item.name === envStore.currentEnv);
        const env = currentEnvExists ? envStore.currentEnv : envList.value[0]?.name || '';
        if (env) handleEnvChange(env);
      }
    }
    isLoading.value = false;
    emits('update:loading', false);
  }

  /** 监听 appID 变化，重新拉取该应用在各环境的部署状态 */
  watch(
    () => appDetailStore.appID,
    async appID => {
      if (appID) {
        await Promise.all([getDeployStatuses(), handleGetEnvList()]);
      } else {
        appDeployStatusMap.value = new Map();
        envList.value = [];
        isLoading.value = false;
        emits('update:deployStatusList', []);
        emits('update:envList', []);
        emits('update:loading', false);
      }
    },
    { immediate: true },
  );

  /** 获取分组内可选环境名称列表（排除 NotReady 状态） */
  function getSelectableGroupEnvNames(envs: EnvOutput[]) {
    return envs
      .filter(env => env.status !== 'NotReady')
      .map(env => env.name)
      .filter((name): name is string => !!name);
  }

  /** 分组全选/取消全选：选中时追加，取消时移除该分组内所有可选环境 */
  function handleGroupSelectAll(envs: EnvOutput[], checked: boolean) {
    const selectableEnvNames = getSelectableGroupEnvNames(envs);
    const currentValues = props.modelValues || [];
    if (checked) {
      emitMultiEnvChange([...currentValues, ...selectableEnvNames.filter(name => !currentValues.includes(name))]);
    } else {
      const selectableEnvNameSet = new Set(selectableEnvNames);
      emitMultiEnvChange(currentValues.filter(name => !selectableEnvNameSet.has(name)));
    }
  }

  /** 切换环境选择模式 */
  function handleModeChange(newMode: 'multi' | 'single') {
    if (newMode === mode.value) return;
    mode.value = newMode;
    emits('update:mode', newMode);
    if (newMode === 'multi') {
      // 单选 → 多选：将当前值转为数组
      const currentVal = props.modelValue;
      emitMultiEnvChange(currentVal ? [currentVal] : []);
    } else {
      // 多选 → 单选：取第一项
      const firstVal = props.modelValues?.[0] || '';
      handleEnvChange(firstVal);
    }
  }

  /** 下拉面板关闭时重置搜索关键词 */
  function handlePopoverHidden() {
    isPopoverVisible.value = false;
    searchKeyword.value = '';
  }

  /** 下拉面板打开时标记可见状态 */
  function handlePopoverShow() {
    isPopoverVisible.value = true;
  }

  /** 多选模式下通过 Tag 删除已选环境 */
  function handleRemoveEnv(env: EnvOutput) {
    if (!env.name) return;
    const currentValues = [...(props.modelValues || [])];
    const index = currentValues.indexOf(env.name);
    if (index > -1) {
      currentValues.splice(index, 1);
      emitMultiEnvChange(currentValues);
    }
  }

  /** 点击环境项：单选直接选中并关闭面板；多选 toggle 选中/取消 */
  function handleSelectEnv(env: EnvOutput) {
    if (env.status === 'NotReady' || !env.name) return;
    if (mode.value === 'multi') {
      // 多选模式：toggle 选中/取消，不关闭 popover
      const currentValues = [...(props.modelValues || [])];
      const index = currentValues.indexOf(env.name);
      if (index > -1) {
        currentValues.splice(index, 1);
      } else {
        currentValues.push(env.name);
      }
      emitMultiEnvChange(currentValues);
    } else {
      // 单选模式：保持原有逻辑
      envStore.updateCurrentEnv(env.name);
      emits('update:item', env);
      emits('update:modelValue', env.name);
      popoverRef.value?.hide();
    }
  }

  /** 判断分组内所有可选环境是否全部选中 */
  function isGroupAllSelected(envs: EnvOutput[]) {
    const selectableEnvNames = getSelectableGroupEnvNames(envs);
    return selectableEnvNames.length > 0 && selectableEnvNames.every(name => (props.modelValues || []).includes(name));
  }

  /** 判断分组是否处于半选状态（部分可选环境已选中） */
  function isGroupIndeterminate(envs: EnvOutput[]) {
    const selectableEnvNames = getSelectableGroupEnvNames(envs);
    const selectedCount = selectableEnvNames.filter(name => (props.modelValues || []).includes(name)).length;
    return selectedCount > 0 && selectedCount < selectableEnvNames.length;
  }

  /** 判断指定环境是否处于选中状态 */
  function isSelected(env: EnvOutput) {
    if (mode.value === 'multi') {
      return env.name !== undefined && (props.modelValues || []).includes(env.name);
    }
    return env.name === props.modelValue;
  }

  /** 同步多选模式的有效值：在 envList 加载完成或外部 props 变化后重新校验 */
  function syncValidMultiEnvValues() {
    if (mode.value !== 'multi' || envList.value.length === 0 || !props.modelValues?.length) return;
    emitMultiEnvChange(props.modelValues, { fallbackWhenEmpty: true });
  }

  // 当 multiSelectable 变为 false 时强制回到单选模式
  watch(
    () => props.multiSelectable,
    val => {
      if (!val && mode.value !== 'single') {
        handleModeChange('single');
      }
    },
    { immediate: true },
  );

  /** 外部 mode prop 变化时同步内部模式状态 */
  watch(
    () => props.mode,
    val => {
      if (val && val !== mode.value) {
        mode.value = val;
      }
      syncValidMultiEnvValues();
    },
  );

  /** 外部 modelValues 变化时重新校验多选有效值 */
  watch(
    () => props.modelValues,
    () => {
      syncValidMultiEnvValues();
    },
    { deep: true },
  );

  defineExpose({
    refreshDeployStatuses: getDeployStatuses,
  });
</script>

<style lang="postcss" scoped>
  :deep(.bk-select-trigger) {
    height: 32px;
    border: none !important;
    border-radius: 0 !important;
    border-right: 1px solid #c4c6cc !important;
    background: transparent !important;
    .bk-input {
      border: none !important;
      box-shadow: none !important;
    }
    .bk-select-tag {
      font-size: 12px;
      color: #63656e;
    }
  }
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
    padding-left: 31px;
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
      left: -8px;
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
  .c-env-select-popover {
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
