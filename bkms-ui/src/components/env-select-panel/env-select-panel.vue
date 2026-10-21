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
    :width="panelWidth"
    @after-hidden="handlePopoverHidden"
    @after-show="handlePopoverShow"
  >
    <div
      v-bind="$attrs"
      :class="[
        'flex items-center min-w-[280px] max-w-[626px] border border-[#c4c6cc] rounded-[2px] bg-[#FFF] cursor-pointer overflow-hidden transition-border-color duration-200 hover:border-[#979BA5]',
        { '!border-[#3a84ff] shadow-[0_0_3px_0_#a3c5fd]': isPopoverVisible },
        mode === 'multi' ? 'h-auto min-h-[32px] !w-[626px]' : 'h-[32px]',
        { '!max-w-none': !showEnvPrefix },
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
        v-else-if="showEnvPrefix"
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
      <div
        v-if="enableVisibleEnvSettings && isEditingVisibleEnvs"
        class="px-[8px] pt-[8px]"
      >
        <div class="text-[14px] leading-[22px] font-medium text-[#313238]">{{ $t('设置可见环境') }}</div>
        <p class="mt-[4px] mb-[8px] text-[12px] leading-[20px] text-[#979BA5]">
          {{
            $t(
              '特性环境始终可见，无需设置。可仅勾选需要在「部署管理」页面环境下拉列表中显示的环境；其他页面不受此设置影响，仍会显示全部环境。',
            )
          }}
        </p>
        <div class="env-list-scroll max-h-[320px] overflow-y-auto">
          <EnvGroupSelector
            v-model="visibleEnvDraft"
            :class="columns === 2 ? '!grid-cols-2' : ''"
            :deploy-statuses="appDeployStatuses"
            :disabled="visibleEnvsSaving"
            :env-list="standardEnvList"
            :get-env-disabled-reason="getVisibleEnvDisabledReason"
            show-deploy-icon
          />
        </div>
      </div>
      <div
        v-else
        class="min-h-[100px]"
      >
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
            v-if="showOnlyDeployedFilter"
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
          class="gap-[8px]"
          :class="columns === 2 ? 'grid grid-cols-2' : 'flex'"
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
                  content: getEnvUnavailableReason(env),
                  disabled: !isEnvDisabled(env),
                  placement: 'bottom',
                }"
                :class="[
                  'flex items-center h-[32px] px-[8px] cursor-pointer text-[12px] text-[#4D4F56] transition-bg-color duration-150',
                  { 'feature-env-child': env.isFeatureChild },
                  { '!bg-[#e1ecff] !text-[#3a84ff]': isSelected(env) },
                  { 'cursor-not-allowed opacity-60': isEnvDisabled(env) },
                  { 'hover:bg-[#F5F7FA]': !isSelected(env) && !isEnvDisabled(env) },
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
      <!-- 编辑态会替换操作按钮，避免同一次点击被误判为弹层外部点击。 -->
      <div
        v-if="enableVisibleEnvSettings"
        class="flex items-center justify-between border-t border-[#DCDEE5] mt-[8px] px-[8px] pt-[8px]"
        @click.stop
      >
        <template v-if="isEditingVisibleEnvs">
          <Button
            :disabled="visibleEnvsSaving || allStandardEnvsSelected"
            text
            theme="primary"
            @click="visibleEnvDraft = [...allStandardEnvNames]"
            >{{ $t('恢复全部') }}</Button
          >
          <div class="flex gap-[8px]">
            <Button
              :disabled="standardEnvList.length > 0 && !visibleEnvDraft.length"
              :loading="visibleEnvsSaving"
              size="small"
              theme="primary"
              @click="handleSaveVisibleEnvs"
              >{{ $t('保存') }}</Button
            >
            <Button
              :disabled="appDetailStore.savingVisibleEnvs"
              size="small"
              @click="endVisibleEnvEdit"
              >{{ $t('取消') }}</Button
            >
          </div>
        </template>
        <template v-else>
          <span class="text-[12px] text-[#979BA5]">{{
            isRestricted && filterVisibleEnvs
              ? $t('已按可见环境筛选，共 {0} 个', [selectableEnvList.length])
              : $t('显示全部环境')
          }}</span>
          <Button
            :disabled="
              isLoading ||
              appDetailStore.loading ||
              visibleEnvsSaving ||
              appDetailStore.appDetail?.id !== appDetailStore.appID
            "
            text
            theme="primary"
            @click="handleEditVisibleEnvs"
            >{{ $t('设置可见环境') }}</Button
          >
        </template>
      </div>
    </template>
  </Popover>
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Button, Checkbox, Input, Message, Popover, Select, Tag } from 'bkui-vue';
  import { Done } from 'bkui-vue/lib/icon';
  import { AngleDownLine, Search } from 'bkui-vue/lib/icon';
  import { isEqual } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { AppService } from '~/api/modules/v1/app';
  import { EnvService } from '~/api/modules/v1/env';
  import EnvGroupSelector from '~/components/env-group-selector.vue';
  import { getRequiredVisibleEnvReasons } from '~/components/env-select-panel/visible-env-rules';
  import OverflowTags from '~/components/overflow-tags.vue';
  import StatusDotIcon from '~/components/status-dot-icon.vue';
  import { useAppVisibleEnvs } from '~/composables/use-app-visible-envs';
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
    /** 下拉面板列数，实例列表为 4 列，表单内收成 2 列 */
    columns?: 2 | 4;
    /** 仅部署相关入口开启应用级名单配置，其他调用方保持原有行为。 */
    enableVisibleEnvSettings?: boolean;
    /** 查看历史时保留完整候选；部署目标及实例列表按名单筛选。 */
    filterVisibleEnvs?: boolean;
    /** 当无选中值时是否自动选中第一个可用环境 */
    initFirstEnvWhenEmpty?: boolean;
    /** 只展示这些 kind 的环境，不传则全部展示 */
    kinds?: string[];
    /** 选择模式 */
    mode?: 'multi' | 'single';
    /** 单选模式当前选中环境名 */
    modelValue?: string;
    /** 多选模式当前选中环境名列表 */
    modelValues?: string[];
    /** 是否允许切换单选/多选模式 */
    multiSelectable?: boolean;
    /** 下拉面板宽度，默认 4 列为 800、2 列为 560 */
    popoverWidth?: number;
    /** 单选值暂未出现在环境列表时，是否保留外部传入值 */
    preserveMissingModelValue?: boolean;
    /** 是否展示触发器左侧的“环境”前缀 */
    showEnvPrefix?: boolean;
    /** 是否展示“仅显示已部署环境”筛选 */
    showOnlyDeployedFilter?: boolean;
    /** 选中环境时是否同步部署页当前环境 */
    syncEnvStore?: boolean;
    /** 业务类型标识 */
    type?: string;
    /** 返回环境的业务禁用原因；有返回值时环境不可新增选择并展示提示 */
    getEnvDisabledReason?: (env: EnvOutput) => string | undefined;
  }

  defineOptions({ inheritAttrs: false });

  const props = withDefaults(defineProps<IProps>(), {
    columns: 4,
    showEnvPrefix: true,
    showOnlyDeployedFilter: true,
    syncEnvStore: true,
  });
  const emits = defineEmits<Emits>();

  const envStore = useDeployEnvStore();
  const appDetailStore = useAppDetail();
  const { t } = useI18n();
  const { getDeployStatusInfo } = useDeployStatusMap();
  const { isEnvVisible, isRestricted, visibleEnvNames } = useAppVisibleEnvs();
  const isEditingVisibleEnvs = ref(false);
  const visibleEnvDraft = ref<string[]>([]);
  const submittingVisibleEnvs = ref(false);
  const visibleEnvsSaving = computed(() => submittingVisibleEnvs.value || appDetailStore.savingVisibleEnvs);
  /** 关闭、取消或切换应用时递增，阻止旧的进入编辑/保存流程继续操作面板。 */
  let visibleEnvEditSession = 0;

  function endVisibleEnvEdit() {
    visibleEnvEditSession += 1;
    submittingVisibleEnvs.value = false;
    isEditingVisibleEnvs.value = false;
    visibleEnvDraft.value = [];
  }

  const popoverRef = ref<InstanceType<typeof Popover> | null>(null);
  /** 数据加载状态 */
  const isLoading = ref(false);
  /** 下拉面板是否可见 */
  const isPopoverVisible = ref(false);
  /** 全部环境列表 */
  const envList = ref<EnvOutput[]>([]);
  /** 由加载入口递增，切换应用时失效；列表写入、选择初始化和 loading 共用此序号。 */
  let envListRequest = 0;
  /** 已完成选择清理的名单版本；加载期间跳过清理时，在加载结束后补做。 */
  let reconciledVisibleEnvsVersion = appDetailStore.visibleEnvsVersion;
  /** 保留所有泳道的原始记录，供可见环境规则判断。 */
  const appDeployStatuses = ref<AppDeployedEnvOutputObj[]>([]);
  /** 从同一份列表派生展示索引，ID 未命中时按名称兜底。 */
  const appDeployStatusMap = computed(() => {
    const byID = new Map<string, AppDeployedEnvOutputObj>();
    const byName = new Map<string, AppDeployedEnvOutputObj>();
    appDeployStatuses.value.forEach(status => {
      if (status.id) byID.set(status.id, status);
      if (status.name) byName.set(status.name, status);
    });
    return { byID, byName };
  });
  /** 仅最新部署状态请求可写入；与编辑会话独立，普通刷新也会递增。 */
  let deployStatusesRequest = 0;
  /** 被刷新顶替的编辑/保存校验等待此结果，避免使用旧状态或静默中断。 */
  let latestDeployStatusesPromise: Promise<boolean>;
  /** 搜索关键词 */
  const searchKeyword = ref('');
  /** 是否仅显示已部署环境 */
  const onlyDeployed = ref(false);
  /** 当前选择模式 */
  const mode = ref<'multi' | 'single'>(props.mode || 'single');

  /** 环境类型分组顺序：开发 -> 测试 -> 预发布 -> 生产 */
  const envTypeOrder = ['development', 'test', 'staging', 'production'];
  const FEATURE_ENV_KIND = 'feature';
  const panelWidth = computed(() => props.popoverWidth || (props.columns === 2 ? 560 : 800));
  const kindEnvList = computed(() => {
    if (!props.kinds?.length) return envList.value;
    return envList.value.filter(env => props.kinds?.includes(env.kind || 'standard'));
  });
  const selectableEnvList = computed(() =>
    props.filterVisibleEnvs ? kindEnvList.value.filter(isEnvVisible) : kindEnvList.value,
  );
  const standardEnvList = computed(() => envList.value.filter(env => (env.kind || 'standard') === 'standard'));
  const requiredVisibleEnvReasons = computed(() =>
    getRequiredVisibleEnvReasons(envList.value, appDeployStatuses.value),
  );
  const allStandardEnvNames = computed(() =>
    standardEnvList.value.map(env => env.name).filter((name): name is string => !!name),
  );
  const requiredVisibleEnvNames = computed(() =>
    allStandardEnvNames.value.filter(name => requiredVisibleEnvReasons.value.has(name)),
  );
  const hasSelectedAllStandardEnvs = (selected: string[]) =>
    selected.length === allStandardEnvNames.value.length &&
    allStandardEnvNames.value.every(name => selected.includes(name));
  const allStandardEnvsSelected = computed(() => hasSelectedAllStandardEnvs(visibleEnvDraft.value));

  /** 单选模式下当前选中的环境对象 */
  // 总览下钻可继续查看隐藏环境，触发器仍需显示当前查看对象。
  const selectedEnvItem = computed(() => kindEnvList.value.find(item => item.name === props.modelValue));
  /** 多选模式下当前选中的所有环境对象 */
  const selectedMultiEnvItems = computed(() => {
    if (mode.value !== 'multi') return [];
    return (props.modelValues || [])
      .map(name => selectableEnvList.value.find(item => item.name === name))
      .filter((item): item is EnvOutput => !!item);
  });
  /** 多选模式的 Tag 名称列表，供 OverflowTags 进行宽度估算 */
  const multiEnvDisplayNames = computed(() => selectedMultiEnvItems.value.map(item => item.displayName ?? ''));
  const normalizedSearchKeyword = computed(() => searchKeyword.value.trim().toLowerCase());
  /** 父子关系仅随环境列表变化，避免搜索或部署过滤变化时重复构建映射 */
  const envFeatureRelations = computed(() => buildEnvFeatureRelations(selectableEnvList.value));

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

  function getEnvDeployStatus(env: EnvOutput) {
    const { byID, byName } = appDeployStatusMap.value;
    return (env.id ? byID.get(env.id) : undefined) || (env.name ? byName.get(env.name) : undefined);
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
    return getEnvDeployStatus(env)?.deployStatus === 'deployed';
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
   * 自动过滤状态或业务禁用环境，并同步选中项与 modelValues
   */
  function emitMultiEnvChange(values: string[], options: { fallbackWhenEmpty?: boolean } = {}) {
    let validValues = values.filter(v => selectableEnvList.value.some(item => item.name === v && !isEnvDisabled(item)));
    if (options.fallbackWhenEmpty && values.length && validValues.length === 0 && props.initFirstEnvWhenEmpty) {
      const firstEnv = selectableEnvList.value.find(item => !isEnvDisabled(item));
      validValues = firstEnv?.name ? [firstEnv.name] : [];
    }
    if (!isEqual(validValues, props.modelValues || [])) {
      emits('update:modelValues', validValues);
    }
    const selectedItems = validValues
      .map(name => selectableEnvList.value.find(item => item.name === name))
      .filter((item): item is EnvOutput => !!item);
    emits('update:items', selectedItems);
  }

  /** 获取当前应用在各环境的部署状态 */
  async function getDeployStatuses(): Promise<boolean> {
    const requestToken = ++deployStatusesRequest;
    const appID = appDetailStore.appID;
    if (!appID) {
      appDeployStatuses.value = [];
      emits('update:deployStatusList', []);
      return false;
    }
    const request = AppService.getAppDeployStatuses({ appID })
      .catch(() => null) // 接口错误由全局拦截器提示。
      .then(res => {
        if (appID !== appDetailStore.appID) return false;
        // 展示刷新可能顶替设置/保存前的校验；等待最新结果，避免操作静默中断。
        if (requestToken !== deployStatusesRequest) return latestDeployStatusesPromise;
        if (res === null) return false;
        const list = (res || []) as AppDeployedEnvOutputObj[];
        appDeployStatuses.value = list;
        emits('update:deployStatusList', list);
        return true;
      });
    latestDeployStatusesPromise = request;
    return request;
  }

  /** 根据环境获取部署状态对应的状态图标名 */
  function getEnvDeployIcon(env: EnvOutput): string {
    const deployStatus = getEnvDeployStatus(env)?.deployStatus;
    if (!deployStatus) return 'status-unknown';
    return getDeployStatusInfo(appDetailStore.appType || null, deployStatus).icon || 'status-unknown';
  }

  /** 获取环境列表，NotReady 环境排到末尾 */
  async function getEnvList(appID: string, request: number) {
    if (!appID) {
      envList.value = [];
      emits('update:envList', []);
      return false;
    }
    const list = await EnvService.listAppEnvs({
      appID,
    }).catch(() => []);
    if (request !== envListRequest || appID !== appDetailStore.appID) return false;
    envList.value = sortEnvList(list);
    emits('update:envList', envList.value);
    return true;
  }

  /** 按入口约定回退：部署表单清空，查看选择器优先缓存环境，再取首个可用环境。 */
  function getFallbackEnvName(preferredName = '') {
    if (!props.initFirstEnvWhenEmpty) return '';
    const preferred = selectableEnvList.value.find(env => env.name === preferredName && !isEnvDisabled(env));
    const first = preferred || selectableEnvList.value.find(env => !isEnvDisabled(env));
    return first?.name || '';
  }

  /** 单选模式的选中值变更处理 */
  function handleEnvChange(env: string) {
    const envItem = kindEnvList.value.find(item => item.name === env && !isEnvDisabled(item));
    if (props.syncEnvStore) {
      envStore.updateCurrentEnv(envItem?.name || '');
    }
    emits('update:item', envItem);
    emits('update:modelValue', envItem?.name || '');
  }

  /** 组件初始化：拉取环境列表并根据选择模式初始化选中值 */
  async function handleGetEnvList() {
    const appID = appDetailStore.appID;
    const request = ++envListRequest;
    isLoading.value = true;
    emits('update:loading', true);
    try {
      const [loaded] = await Promise.all([
        getEnvList(appID, request),
        // 复用当前应用详情，避免辅助刷新失败时清空全局详情及已有的可见名单。
        props.enableVisibleEnvSettings && appDetailStore.appDetail?.id !== appID
          ? appDetailStore.fetchAppDetail()
          : Promise.resolve(),
      ]);
      // 等待应用详情期间也可能发起新一轮加载，初始化选择前再次检查归属。
      if (!loaded || request !== envListRequest || appID !== appDetailStore.appID) return;
      if (mode.value === 'multi') {
        // 多选模式初始化
        if (props.modelValues?.length) {
          emitMultiEnvChange(props.modelValues, { fallbackWhenEmpty: true });
        } else if (props.initFirstEnvWhenEmpty) {
          const firstEnv = selectableEnvList.value.find(item => !isEnvDisabled(item));
          if (firstEnv?.name) {
            emitMultiEnvChange([firstEnv.name]);
          }
        }
      } else {
        initializeSingleSelection();
      }
    } finally {
      if (appID === appDetailStore.appID && request === envListRequest) {
        isLoading.value = false;
        emits('update:loading', false);
        if (reconciledVisibleEnvsVersion !== appDetailStore.visibleEnvsVersion) reconcileVisibleSelection();
      }
    }
  }

  /** 初始化单选；总览下钻等查看入口允许保留名单外的当前环境。 */
  function initializeSingleSelection() {
    const envName = props.modelValue;
    if (!envName) {
      const fallbackName = getFallbackEnvName(envStore.currentEnv);
      if (fallbackName) handleEnvChange(fallbackName);
      return;
    }
    const exists = selectableEnvList.value.some(env => env.name === envName);
    if (!exists && props.preserveMissingModelValue && !kindEnvList.value.some(env => env.name === envName)) {
      emits('update:modelValue', envName);
      return;
    }
    const preserveCurrent = exists || props.preserveMissingModelValue || !props.filterVisibleEnvs;
    handleEnvChange(preserveCurrent ? envName : getFallbackEnvName());
  }

  /** 主动刷新环境列表和部署状态，用于侧栏打开、页面刷新等需要最新数据的交互。 */
  async function refresh() {
    await Promise.all([getDeployStatuses(), handleGetEnvList()]);
  }

  /** 监听 appID 变化，重新拉取该应用在各环境的部署状态 */
  watch(
    () => appDetailStore.appID,
    async appID => {
      endVisibleEnvEdit();
      envListRequest += 1;
      envList.value = [];
      emits('update:envList', []);
      if (appID) {
        await Promise.all([getDeployStatuses(), handleGetEnvList()]);
      } else {
        deployStatusesRequest += 1;
        appDeployStatuses.value = [];
        envList.value = [];
        isLoading.value = false;
        emits('update:deployStatusList', []);
        emits('update:envList', []);
        emits('update:loading', false);
      }
    },
    { immediate: true },
  );

  function ensureRequiredVisibleEnvs() {
    visibleEnvDraft.value = [...new Set([...visibleEnvDraft.value, ...requiredVisibleEnvNames.value])];
  }

  /** 获取环境不可选原因，业务限制优先于未配置集群状态。 */
  function getEnvUnavailableReason(env: EnvOutput) {
    return (
      props.getEnvDisabledReason?.(env) || (env.status === 'NotReady' ? t('环境未配置集群资源，无法部署应用') : '')
    );
  }

  /** 获取分组内可选环境名称列表（排除状态和业务禁用项） */
  function getSelectableGroupEnvNames(envs: EnvOutput[]) {
    return envs
      .filter(env => !isEnvDisabled(env))
      .map(env => env.name)
      .filter((name): name is string => !!name);
  }

  function getVisibleEnvDisabledReason(env: EnvOutput) {
    const reason = env.name && requiredVisibleEnvReasons.value.get(env.name);
    if (reason === 'deployment') {
      return t('该环境已有部署，必须保持可见，不能取消');
    }
    if (reason === 'feature-parent') {
      return t('该环境是特性环境的父环境，必须保持可见，不能取消');
    }
    return '';
  }

  async function handleEditVisibleEnvs() {
    const session = ++visibleEnvEditSession;
    if (!(await getDeployStatuses()) || session !== visibleEnvEditSession || !isPopoverVisible.value) return;
    visibleEnvDraft.value = isRestricted.value ? [...visibleEnvNames.value] : [...allStandardEnvNames.value];
    ensureRequiredVisibleEnvs();
    searchKeyword.value = '';
    isEditingVisibleEnvs.value = true;
  }

  watch(requiredVisibleEnvNames, () => {
    if (isEditingVisibleEnvs.value) ensureRequiredVisibleEnvs();
  });

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
    endVisibleEnvEdit();
    isPopoverVisible.value = false;
    searchKeyword.value = '';
  }

  /** 下拉面板打开时刷新部署状态，避免状态图标和仅已部署过滤使用旧缓存。 */
  function handlePopoverShow() {
    isPopoverVisible.value = true;
    void getDeployStatuses();
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

  async function handleSaveVisibleEnvs() {
    if (visibleEnvsSaving.value || !isEditingVisibleEnvs.value) return;
    const session = visibleEnvEditSession;
    const appID = appDetailStore.appID;
    // 第一次 await 前固定用户选择；关闭编辑态会清空界面草稿。
    const draft = [...visibleEnvDraft.value];
    submittingVisibleEnvs.value = true;
    try {
      if (!(await getDeployStatuses()) || session !== visibleEnvEditSession || appID !== appDetailStore.appID) return;
      // 刷新后的必选环境合并进快照，不从可能已变化的界面草稿取值。
      const selected = [...new Set([...draft, ...requiredVisibleEnvNames.value])];
      // 接口以空名单表示全部可见，编辑态则用全选展示同一状态。
      const result = await appDetailStore.updateVisibleEnvs(hasSelectedAllStandardEnvs(selected) ? [] : selected);
      // PUT 可能已发出；旧会话不能关闭后来打开的面板。
      if (session !== visibleEnvEditSession || result === 'stale') return;
      if (result === 'refresh-failed') {
        Message({ theme: 'warning', message: t('配置已保存，获取最新配置失败，请刷新页面重试') });
      }
      isEditingVisibleEnvs.value = false;
      popoverRef.value?.hide();
    } catch {
      // 请求拦截器展示权限或校验错误；保留草稿便于修改后重试。
    } finally {
      // 关闭/取消会释放本地锁，旧请求不能覆盖新会话的状态。
      if (session === visibleEnvEditSession) submittingVisibleEnvs.value = false;
    }
  }

  /** 点击环境项：单选直接选中并关闭面板；多选 toggle 选中/取消 */
  function handleSelectEnv(env: EnvOutput) {
    if (!env.name || (isEnvDisabled(env) && !isSelected(env))) return;
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
      if (props.syncEnvStore) {
        envStore.updateCurrentEnv(env.name);
      }
      emits('update:item', env);
      emits('update:modelValue', env.name);
      popoverRef.value?.hide();
    }
  }

  /** 环境是否不可新增选择。 */
  function isEnvDisabled(env: EnvOutput) {
    return !!getEnvUnavailableReason(env);
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

  /** 保存名单后清理失效选择；部署表单不自动选中其他目标。 */
  function reconcileVisibleSelection() {
    if (!props.filterVisibleEnvs || isLoading.value) return;
    reconciledVisibleEnvsVersion = appDetailStore.visibleEnvsVersion;
    if (mode.value === 'multi') {
      emitMultiEnvChange(props.modelValues || []);
      return;
    }
    const current = kindEnvList.value.find(env => env.name === props.modelValue);
    if (!current || isEnvVisible(current)) return;
    handleEnvChange(getFallbackEnvName());
  }

  /** 同步多选模式的有效值：在 envList 加载完成或外部 props 变化后重新校验 */
  function syncValidMultiEnvValues() {
    if (mode.value !== 'multi' || envList.value.length === 0 || !props.modelValues?.length) return;
    emitMultiEnvChange(props.modelValues, { fallbackWhenEmpty: true });
  }

  watch(
    () => appDetailStore.visibleEnvsVersion,
    () => {
      if (props.enableVisibleEnvSettings) reconcileVisibleSelection();
    },
  );

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
    refresh,
    refreshEnvList: handleGetEnvList,
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
