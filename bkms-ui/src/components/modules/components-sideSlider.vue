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
  <Sideslider
    v-model:is-show="localValue"
    :before-close="handleBeforeClose"
    :width="isCollaspeAside ? 680 : 1400"
    @hidden="handleClose"
  >
    <template #header>
      <FlexRow class="w-full px-[12px]">
        <template #left>
          <DividerHeader
            :show-divider="!!data?.name"
            :title="title || (data ? $t('编辑组件实例') : $t('添加组件实例'))"
            :title-size="16"
          >
            <span v-if="data?.name">
              {{ data.name }}
            </span>
          </DividerHeader>
        </template>
        <template #right>
          <IconTextButton
            :active="!isCollaspeAside"
            class="text-[12px] p-[10px]"
            icon="bkms-icon bkms-icon-variable"
            :text="$t('环境变量')"
            @click="handleToggleisCollaspeAside"
          />
        </template>
      </FlexRow>
    </template>
    <CollapsibleAsideLayout
      v-model:is-collapsed="isCollaspeAside"
      :layout-config="{
        viewportOffset: collapsiableAsideLayoutOffset,
      }"
      :max="640"
      @after-resize="handleRefreshEnvVar"
    >
      <template #main>
        <div class="px-[24px] py-[18px] pb-[32px] min-w-[630px]">
          <template v-if="currentStep === 1">
            <Popover
              ref="popoverRef"
              ext-cls="!shadow-none"
              :is-show="isShow"
              :offset="{
                crossAxis: 20,
                mainAxis: 36,
              }"
              placement="left"
              theme="light"
              trigger="manual"
              width="480"
              :z-index="6000"
            >
              <!-- 占位 -->
              <div class="h-[1px] w-[10px] fixed top-[60px]"></div>
              <!-- 左侧气泡框 -->
              <template #content>
                <div
                  v-if="!curInstanceObj?.properties"
                  class="h-[96vh] overflow-hidden p-[4px] flex flex-col"
                >
                  <FlexRow class="mt-[14px]">
                    <template #left>
                      <h2 class="text-[14px] font-bold flex items-center">
                        <span class="mr-[6px]">{{ $t('请选择一个组件') }}</span>
                        <i
                          class="bkms-icon bkms-icon-refresh cursor-pointer text-[#3A84FF]"
                          @click="handleRefresh"
                        ></i>
                      </h2>
                    </template>
                    <template #right>
                      <Input
                        v-model.trim="searchValue"
                        class="w-[200px]"
                        clearable
                        :placeholder="createPlaceholder({ labels: ['组件'] })"
                        type="search"
                      />
                    </template>
                  </FlexRow>
                  <Tab
                    v-model:active="active"
                    v-bkloading="{ loading }"
                    class="overflow-hidden"
                    type="unborder-card"
                  >
                    <Tab.TabPanel
                      :label="$t('全部')"
                      name="All"
                    ></Tab.TabPanel>
                    <!-- <Tab.TabPanel
                    v-if="moduleType === 'app'"
                    :label="$t('部署')"
                    name="Deploy"
                  ></Tab.TabPanel>
                  <Tab.TabPanel
                    v-if="moduleType !== 'space'"
                    :label="$t('策略')"
                    name="Strategy"
                  ></Tab.TabPanel>
                  <Tab.TabPanel
                    v-if="moduleType !== 'space'"
                    :label="$t('存储')"
                    name="Storage"
                  ></Tab.TabPanel>
                  <Tab.TabPanel
                    v-if="moduleType !== 'space'"
                    :label="$t('其他')"
                    name="Component"
                  ></Tab.TabPanel> -->
                    <!-- <Alert
                    v-show="active === 'Deploy'"
                    class="mt-[16px]"
                    theme="info"
                    :title="$t('应用必须且只有一个部署类组件，无法新增部署组件。')"
                  /> -->
                    <component
                      :is="componentItemMap[moduleType]"
                      v-for="(item, index) in curComponentList"
                      :key="`${index}-${item?.name}`"
                      :active="item.name === curComponent?.name"
                      :class="['mt-[16px]', { 'bg-[#F0F5FF] !border-[#3A84FF]': activeIndex === index }]"
                      :data="item"
                      :disabled="disabledFn?.(item)"
                      :disabled-text="$t('应用必须且只有一个部署类组件，无法新增部署组件。')"
                      :env-name-map="envNameMap"
                      @click="activeIndex = index"
                      @selected="com => handleSelected(index, com)"
                    />
                  </Tab>
                  <Exception
                    v-show="curComponentList.length === 0"
                    :description="$t('暂无数据')"
                    scene="part"
                    type="empty"
                  />
                </div>
                <div
                  v-else
                  class="h-[96vh] overflow-hidden p-[4px] flex flex-col min-h-[0px]"
                >
                  <div class="flex items-center text-[14px]">
                    <h2 class="font-bold">{{ $t('实例详情') }}</h2>
                    <Divider
                      direction="vertical"
                      type="solid"
                    />
                    <span class="text-[#979BA5]">{{ curInstanceObj?.name || '--' }}</span>
                  </div>
                  <div class="mt-[24px] flex-1 overflow-auto">
                    <ToggleCard
                      content-class="overflow-visible"
                      :name="$t('组件配置')"
                      type="normal"
                    >
                      <Form
                        ref="formRef"
                        form-type="vertical"
                        :model="instanceDetail"
                      >
                        <Form.FormItem
                          :label="$t('实例名称')"
                          property="name"
                          required
                        >
                          <Input
                            v-model="instanceDetail.name"
                            disabled
                          />
                        </Form.FormItem>
                        <Form.FormItem
                          v-for="(prop, index) in propertyList"
                          :key="`${index}-${prop.name}`"
                          class="form-item-properties"
                          :property="`properties.[${prop.name}]`"
                          required
                        >
                          <template #label>
                            <template v-if="prop.description">
                              <span>
                                {{ prop.name }}
                              </span>
                              （
                              <OverflowTitle type="tips">
                                {{ prop.description }}
                              </OverflowTitle>
                              ）
                            </template>
                            <template v-else>
                              {{ prop.name }}
                            </template>
                          </template>
                          <DynamicInput
                            v-model="instanceDetail.properties[prop.name]"
                            disabled
                            :select-options="handleSelectOptions(prop)"
                            :type="prop.type as InputType"
                          />
                        </Form.FormItem>
                      </Form>
                    </ToggleCard>
                  </div>
                </div>
              </template>
            </Popover>
            <p class="mb-[8px] text-[14px]">{{ $t('组件') }}<i class="text-[#EA3636] pl-[6px]">*</i></p>
            <Button
              v-if="!curComponent"
              theme="primary"
              @click="handleShowPopover"
            >
              <Plus
                class="mr-[4px]"
                :height="24"
                :width="24"
              />
              {{ $t('添加组件实例') }}
            </Button>
            <template v-else>
              <Input
                class="box-content"
                disabled
                :model-value="curComponent?.displayName || curComponent?.name || '--'"
              >
                <template #suffix>
                  <Button
                    class="!min-w-[60px] rounded-[0_2px_2px_0] text-[12px]"
                    :disabled="!!data"
                    theme="primary"
                    @click="handleReSelect"
                  >
                    {{ $t('重选') }}
                  </Button>
                </template>
              </Input>
              <Radio.Group
                v-if="moduleType === 'app'"
                v-model="isUseInstance"
                class="w-full mt-[24px]"
                type="card"
              >
                <Radio.Button :label="false">
                  <UnderLineTips :description="$t('为应用创建独立的组件配置，所有环境都生效')">
                    {{ $t('自定义配置新实例') }}
                  </UnderLineTips>
                </Radio.Button>
                <Radio.Button
                  :disabled="!curInstanceList.length"
                  :label="true"
                >
                  <UnderLineTips :description="$t('使用空间上已配置好的组件')">
                    {{ $t('引用空间配置的实例') }}
                    <span
                      :class="[
                        'text-[#979BA5] rounded-[8px] px-[8px] transition',
                        isUseInstance ? 'bg-[#3A84FF] text-[#fff]' : 'bg-[#F0F1F5]',
                      ]"
                    >
                      {{ curInstanceList.length }}
                    </span>
                  </UnderLineTips>
                </Radio.Button>
              </Radio.Group>
              <!-- 表单 -->
              <template v-if="!isUseInstance">
                <ToggleCard
                  class="mt-[24px]"
                  content-class="overflow-visible"
                  :name="$t('组件配置')"
                  type="normal"
                >
                  <Form
                    ref="formRef"
                    form-type="vertical"
                    :model="formModel"
                  >
                    <Form.FormItem
                      :label="$t('实例名称')"
                      property="name"
                      required
                      :rules="instanceNameRules"
                    >
                      <Input
                        v-model.trim="formModel.name"
                        v-bk-tooltips="{
                          content: nameConfig?.message,
                          disabled: !nameConfig?.disabled,
                        }"
                        clearable
                        :disabled="nameConfig?.disabled"
                        :placeholder="$t('请输入 20 个字符以内的小写字母、数字和中划线，以小写字母开头')"
                      />
                    </Form.FormItem>
                    <Form.FormItem
                      v-for="(prop, index) in propertyList"
                      :key="`${index}-${prop.name}`"
                      class="form-item-properties"
                      :property="`properties.${prop.name}`"
                      required
                    >
                      <template #label>
                        <template v-if="prop.description">
                          <span>
                            {{ prop.name }}
                          </span>
                          （
                          <OverflowTitle type="tips">
                            {{ prop.description }}
                          </OverflowTitle>
                          ）
                        </template>
                        <template v-else>
                          {{ prop.name }}
                        </template>
                      </template>
                      <DynamicInput
                        v-model="formModel.properties[prop.name]"
                        :select-options="handleSelectOptions(prop)"
                        :type="prop.type as InputType"
                      />
                    </Form.FormItem>
                  </Form>
                </ToggleCard>
                <ToggleCard
                  v-if="moduleType === 'space'"
                  class="mt-[24px]"
                  content-class="!p-0 !mt-[12px]"
                  :name="$t('组件可用环境')"
                  type="normal"
                >
                  <Radio.Group
                    v-model="componentScoped"
                    class="flex flex-col ml-[12px]"
                    @change="handleScopedChange"
                  >
                    <Radio label="global">{{ $t('所有环境') }}</Radio>
                    <Radio
                      class="!ml-0 mt-[10px]"
                      label="environment"
                    >
                      {{ $t('可用环境') }}
                    </Radio>
                    <EnvGroupSelector
                      v-show="componentScoped === 'environment'"
                      v-model="scopedEnvs"
                      class="mt-[12px]"
                      :env-list="envList"
                    />
                  </Radio.Group>
                </ToggleCard>
              </template>
              <!-- 引用空间组件实例 -->
              <template v-else>
                <ToggleCard
                  class="mt-[24px]"
                  :content-class="'pt-[6px]'"
                  :name="$t('选择实例')"
                  type="normal"
                >
                  <Alert
                    class="mb-[12px]"
                    theme="warning"
                    :title="$t('请确保组件已对目标环境可用，否则将会影响用户部署')"
                  />
                  <FlexRow class="mb-[12px]">
                    <template #left>
                      <span>{{ $t('已选择') }} : </span>
                      <span class="font-bold"> {{ curInstanceObj?.name || '--' }} </span>
                    </template>
                    <template #right>
                      <div class="flex items-center gap-[12px]">
                        <div
                          v-for="type in envTypes"
                          :key="type.label"
                          class="flex items-center"
                        >
                          <span :class="['size-[10px] border-1', type.theme]"></span>
                          <span class="ml-[4px]">{{ type.label }}</span>
                        </div>
                      </div>
                    </template>
                  </FlexRow>
                  <Table
                    ref="instanceTableRef"
                    class="mb-[10px]"
                    :data="curInstanceList"
                    :pagination="count > 10 ? pagination : null"
                    :radio-config="{
                      trigger: 'row',
                    }"
                    :row-class-name="getInstanceRowClass"
                    :show-overflow="false"
                    @page-limit-change="pageSizeChange"
                    @page-value-change="pageChange"
                    @radio-change="handleSelectInstance"
                  >
                    <template #empty>
                      <TableException />
                    </template>
                    <TableColumn
                      type="radio"
                      width="40"
                    />
                    <TableColumn
                      field="name"
                      :label="$t('实例名称')"
                      min-width="150"
                    >
                      <template #default="{ row }">
                        {{ row.name || '--' }}
                      </template>
                    </TableColumn>
                    <!-- 可用环境 -->
                    <TableColumn
                      field="scopeEnvNames"
                      :label="$t('可用环境')"
                      min-width="180"
                    >
                      <template #default="{ row }">
                        <div
                          v-if="row.scopeType === 'global'"
                          class="flex items-center gap-[4px]"
                        >
                          <Tag>{{ $t('所有环境') }}</Tag>
                        </div>
                        <div
                          v-else
                          class="flex items-center gap-[4px] flex-wrap"
                        >
                          <Tag
                            v-for="(env, index) in row.scopeEnvNames"
                            :key="index"
                            :class="getEnvTagClass(env)"
                          >
                            {{ envNameMap[env]?.displayName || '--' }}
                          </Tag>
                        </div>
                      </template>
                    </TableColumn>
                  </Table>
                  <p
                    v-if="isInstanceMissing && !instanceDetail.type"
                    class="text-[#EA3636] is-error"
                  >
                    {{ instanceTips }}
                  </p>
                </ToggleCard>
              </template>
            </template>
          </template>
          <template v-if="currentStep === 2">
            <div class="bg-[#F1F2F6] text-[313238] font-bold text-[14px] px-[16px] py-[6px]">
              {{ $t('试运行结果预览') }}
            </div>
            <PatchPreviewCard
              v-for="(item, index) in trialRunResult?.patchPreview || []"
              :key="index"
              :base-manifest="item.baseManifest"
              class="mt-[16px]"
              :patched-manifest="item.patchedManifest"
              :target-kind="item.targetKind"
            />
            <ResourceCard
              v-for="(item, index) in trialRunResult?.resources || []"
              :key="index"
              class="mt-[16px]"
              :kind="item?.kind || '--'"
              :manifest="item.manifest"
            />
          </template>
        </div>
      </template>
      <template #aside>
        <ViewDefaultEnvVars
          ref="envVarRef"
          :custom-request-fn="handleGetVarEnv"
          :env-list="envList"
        />
      </template>
    </CollapsibleAsideLayout>
    <template
      v-if="curComponent"
      #footer
    >
      <div class="py-[8px] !mt-0 flex items-center gap-[8px]">
        <template v-if="currentStep === 1">
          <Button
            :loading="isTrialRunning"
            theme="primary"
            @click="handleTrialRun"
            >{{ $t('试运行') }}</Button
          >
          <Button
            class="ml-[8px]"
            :disabled="isTrialRunning"
            @click="handleCancel"
            >{{ $t('取消') }}</Button
          >
        </template>
        <template v-else-if="currentStep === 2">
          <Button
            :disabled="btnLoading"
            @click="handlePrevStep"
            >{{ $t('上一步') }}</Button
          >
          <Button
            :loading="btnLoading"
            theme="primary"
            @click="handleSubmit"
            >{{ $t('提交') }}</Button
          >
          <Button
            class="ml-[8px]"
            :disabled="btnLoading"
            @click="handleCancel"
            >{{ $t('取消') }}</Button
          >
        </template>
      </div>
    </template>
  </Sideslider>
</template>
<script setup lang="ts">
  import type { Ref } from 'vue';
  import { computed, nextTick, onBeforeMount, reactive, ref, shallowRef, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import {
    Alert,
    Button,
    Divider,
    Exception,
    Form,
    Input,
    OverflowTitle,
    Popover,
    Radio,
    Sideslider,
    Tab,
    Tag,
  } from 'bkui-vue';
  import { Plus } from 'bkui-vue/lib/icon';
  import { cloneDeep } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import {
    ComponentDefsService,
    ComponentInstsService,
    EnvvarsService,
    WorkspaceComponentsService,
  } from '~/api/modules/v1';
  import { BKMS_REGEX } from '~/common/const';
  import EnvGroupSelector from '~/components/env-group-selector.vue';
  import ComponentItem from '~/components/modules/component-item.vue';
  import SpaceComponentItem from '~/components/modules/space-component-item.vue';
  import useDebouncedRef from '~/composables/use-debounce';
  import useEnvManager, { envTypeTagClassMap } from '~/composables/use-env-manager';
  import { useFocusOnErrorField } from '~/composables/use-focus-on-error-field';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import usePageConf from '~/composables/use-page';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import PatchPreviewCard from '~/pages/marketplace/components/component-ouput/patch-preview-card.vue';
  import ResourceCard from '~/pages/marketplace/components/component-ouput/resource-card.vue';
  import { useAppDetail } from '~/stores/app-detail';
  import { useSpaceStore } from '~/stores/space';

  import UnderLineTips from '../under-line-tips.vue';

  import type { InputType } from '../dynamic-input.vue';
  import type { ComponentOutputObj } from '~/@types/v1/app';
  import type {
    ComponentDefOutputObj,
    ListComponentDefsRequest,
    PropertyDefOutputObj,
  } from '~/@types/v1/component-defs';
  import type { EnvOutput } from '~/@types/v1/env';
  import type { WorkspaceComponentOutputObj } from '~/@types/v1/workspace-components';

  export type ComponentData = ComponentOutputObj | WorkspaceComponentOutputObj;
  export type ScopedType = 'environment' | 'global';

  type ComponentFormValue = Omit<ComponentOutputObj, 'properties' | 'scopeEnvNames' | 'scopeType'> & {
    properties: Record<string, DynamicInputValue>;
  };
  type DynamicInputValue = boolean | number | string;

  interface IEmits {
    (e: 'update:modelValue', value: boolean): void;
    (
      e: 'submit',
      data: {
        data: ComponentData;
        isUseInstance: boolean;
        scoped?: { envs: string[]; type: ScopedType };
      },
    ): void;
    (e: 'close'): void;
  }

  type IProps = {
    btnLoading?: boolean; // 按钮加载状态
    data?: ComponentOutputObj & { scoped?: { envs: string[]; type: ScopedType } };
    disabledFn?: (item: ComponentDefOutputObj) => boolean;
    modelValue?: boolean;
    moduleType?: 'app' | 'space';
    nameConfig?: {
      disabled: boolean;
      message: string;
    };
    title?: string;
  };
  type NamedPropertyDef = PropertyDefOutputObj & { name: string };

  interface PatchPreviewItem {
    baseManifest: string;
    patchedManifest: string;
    targetKind: string;
  }
  interface ResourceItem {
    apiVersion: string;
    kind: string;
    manifest: string;
    name: string;
  }
  type SelectOption = { label: string; value: string };
  interface TrialRunResponse {
    patchPreview: PatchPreviewItem[];
    resources: ResourceItem[];
  }
  /** 不支持分类的参数 */
  type UnSupportCategoryParams = Omit<ListComponentDefsRequest, 'category'>;

  const props = withDefaults(defineProps<IProps>(), {
    btnLoading: false,
    data: undefined,
    disabledFn: undefined,
    modelValue: false,
    title: '',
    moduleType: 'app',
  });

  const emits = defineEmits<IEmits>();

  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();
  const appDetailStore = useAppDetail();

  const isCollaspeAside = ref(true);
  const envVarRef = ref();
  const componentItemMap = shallowRef({
    app: ComponentItem,
    space: SpaceComponentItem,
  });
  const localValue = ref(props.modelValue); // 显示/隐藏
  const isShow = ref<boolean>(false); // 是否显示组件列表
  const popoverRef = ref();
  const spaceStore = useSpaceStore();

  // 试运行相关状态
  const currentStep = ref(1); // 1: 表单填写页, 2: 试运行结果页
  const trialRunResult = ref<TrialRunResponse>();
  const isTrialRunning = ref(false);

  const { focusOnErrorField } = useFocusOnErrorField();

  const { envList, handleGetEnvList } = useEnvManager();
  const envTypes = ref([
    {
      label: t('开发环境'),
      theme: envTypeTagClassMap.development,
    },
    {
      label: t('测试环境'),
      theme: envTypeTagClassMap.test,
    },
    {
      label: t('预发布环境'),
      theme: envTypeTagClassMap.staging,
    },
    {
      label: t('生产环境'),
      theme: envTypeTagClassMap.production,
    },
  ]);

  function getInstanceRowClass({ row }: { row: WorkspaceComponentOutputObj }) {
    return row.name === curInstanceObj.value?.name ? 'bg-[#F0F5FF]' : '';
  }

  /**
   * 由于sideslider footer仅在curComponent有值时展示，添加组件实例时，未选中组件footer不展示
   * offset计算规则：
   *   - 100: header + footer的高度（curComponent有值时）
   *   - 52:  header的高度（curComponent无值时）
   * 调整原因：当footer不存在时需将offset调整为52，避免footer吸底样式的异常展示行为
   */
  const collapsiableAsideLayoutOffset = computed(() => (curComponent.value ? 100 : 52));

  // 侧栏关闭前回调
  function handleClose() {
    isShow.value = false; // 隐藏组件列表
    isUseInstance.value = false; // 重置是否使用已有实例
    curComponent.value = undefined; // 重置当前选中的组件
    isInstanceMissing.value = false; // 重置验证状态
    isCollaspeAside.value = true; // 重置为折叠状态
    scopedEnvs.value = []; // 重置空间组件可用环境
    curInstanceObj.value = undefined; // 重置当前选中的实例对象

    Object.assign(formModel, defaultFormValue); // 重置表单数据
    Object.assign(instanceDetail, defaultFormValue); // 重置实例详情
    formRef.value?.clearValidate(); // 重置表单验证

    // 重置试运行相关状态
    currentStep.value = 1;
    trialRunResult.value = undefined;
    isTrialRunning.value = false;

    emits('close');
    return true;
  }

  /**
   * @description: 重新选择
   */
  function handleReSelect() {
    isShow.value = true;
    isUseInstance.value = false;
    curInstanceObj.value = undefined;
  }

  /**
   * @description: 添加组件
   */
  function handleShowPopover() {
    isShow.value = true;
    isUseInstance.value = false;
  }

  let refreshed = false;
  function handleRefreshEnvVar() {
    envVarRef.value?.reRefreshTable();
  }

  function handleToggleisCollaspeAside() {
    isCollaspeAside.value = !isCollaspeAside.value;
    if (!isCollaspeAside.value && !refreshed) {
      setTimeout(() => {
        envVarRef.value?.reRefreshTable();
        refreshed = true;
      }, 300);
    }
    popoverRef.value?.resetPopover();
  }

  const searchValue = useDebouncedRef('', 300) as Ref<string>; // 搜索值
  const active = ref<string>(''); // 当前选中的组件类型
  const loading = ref<boolean>(false);
  const componentList = ref<ComponentDefOutputObj[]>([]); // 组件列表
  const curComponent = ref<ComponentDefOutputObj>(); // 当前选中的组件
  const activeIndex = ref<number>(-1);
  // 当前选中组件的实例列表
  const curInstanceList = computed<WorkspaceComponentOutputObj[]>(() =>
    instanceList.value.reduce((acc, cur) => {
      if (cur.type === curComponent.value?.name) {
        acc.push(cur);
      }
      return acc;
    }, [] as WorkspaceComponentOutputObj[]),
  );

  const isUseInstance = ref<boolean>(false); // 是否使用已有实例

  /**
   * @description: 前端筛选
   */
  const curComponentList = computed<ComponentDefOutputObj[]>(() =>
    componentList.value.filter(item => {
      let result;
      if (!searchValue.value) {
        result = true;
      } else {
        const name = item?.name?.toLowerCase?.() ?? '';
        const displayName = item?.displayName?.toLowerCase?.() ?? '';
        const version = item?.version?.toLowerCase?.() ?? '';
        const search = searchValue.value.toLowerCase();
        const description = item?.description?.toLowerCase?.() ?? '';
        result =
          name.includes(search) ||
          displayName.includes(search) ||
          description.includes(search) ||
          version.includes(search);
      }

      return result;
    }),
  );

  /**
   * @description: 刷新
   */
  async function handleRefresh() {
    loading.value = true;
    await handleGetInstance();
    loading.value = false;
  }
  /**
   * @description: 选择组件
   * @param {ComponentDefOutputObj} com
   */
  function handleSelected(index: number, com: ComponentDefOutputObj) {
    withPausedWatch(() => {
      curComponent.value = com;
      formModel.type = com.name; // name 对应 type
      handleResetProperty(com);

      isShow.value = false;
      if (props.moduleType === 'app') {
        // 重置实例
        isUseInstance.value = false;
      }
      activeIndex.value = index;
    });
  }

  // 默认值
  const defaultFormValue: ComponentFormValue = {
    type: '',
    name: '',
    version: '',
    properties: {},
    refWorkspaceCompName: '',
  };
  const componentScoped = ref<ScopedType>('global');
  const scopedEnvs = ref<string[]>([]);
  const formModel = reactive<ComponentFormValue>({ ...defaultFormValue }); // 表单数据
  const instanceDetail = reactive<ComponentFormValue>({ ...defaultFormValue }); // 实例详情

  // 使用 useLeaveConfirm hook 管理表单变化检测
  const { confirmBox, forceCleanDirtyTag, withPausedWatch } = useLeaveConfirm(formModel);

  const formRef = ref<InstanceType<typeof Form>>(); // 表单实例
  const isInstanceMissing = ref<boolean>(false);
  const instanceTips = computed(() => {
    if (!curInstanceObj.value?.name) {
      return t('请选择实例');
    }
    if (!instanceDetail.type) {
      return t('未找到实例');
    }
    return '';
  });

  // 实例名称校验规则
  const instanceNameRules = [
    {
      validator: () => BKMS_REGEX.instanceNameRegex.test(formModel.name || ''),
      message: t('只能包含小写字母、数字及中划线，必须以字母开头、以字母或数字结尾，长度限制20位以内'),
      trigger: 'blur',
    },
  ];

  /**
   * @description: 取消
   */
  async function handleCancel() {
    if (await handleBeforeClose()) {
      localValue.value = false;
      isShow.value = false;
    }
  }

  const instanceTableRef = ref();
  function handlePrevStep() {
    currentStep.value = 1;
    // 回显已选中的实例
    nextTick(() => {
      const row = curInstanceList.value.find(item => item.name === instanceDetail.name);
      instanceTableRef.value?.getVxeTableInstance()?.setRadioRow(row);
    });
  }

  /**
   * @description: 提交
   */
  async function handleSubmit() {
    let data = {} as ComponentOutputObj | WorkspaceComponentOutputObj;
    if (isUseInstance.value && props.moduleType === 'app' && curInstanceObj.value?.name) {
      data = curInstanceObj.value;
    } else {
      data = formModel;
    }

    const scoped =
      props.moduleType === 'space'
        ? {
            type: componentScoped.value,
            envs: componentScoped.value === 'environment' ? scopedEnvs.value : [],
          }
        : undefined;
    forceCleanDirtyTag(() => {
      emits('submit', {
        data: data,
        isUseInstance: isUseInstance.value,
        scoped,
      });
    });
  }

  async function handleTrialRun() {
    try {
      // 使用空间上已配置好的组件 校验
      if (isUseInstance.value && props.moduleType === 'app') {
        if (!curInstanceObj.value?.name) {
          isInstanceMissing.value = true;
          await nextTick();
          focusOnErrorField();
          return;
        }
      } else {
        const valid = await formRef.value?.validate().catch(() => false);
        focusOnErrorField();
        if (!valid) return;
      }

      isTrialRunning.value = true;
      trialRunResult.value = await ComponentInstsService.previewComponentInst(
        {
          type: formModel.type!,
          properties: formModel.properties,
        },
        { needRes: true },
      );
      currentStep.value = 2;
    } finally {
      isTrialRunning.value = false;
    }
  }

  const instanceList = shallowRef<WorkspaceComponentOutputObj[]>([]);
  const count = computed(() => curInstanceList.value.length);
  const { pagination, pageChange, pageSizeChange } = usePageConf(
    curInstanceList,
    {
      current: 1,
      limit: 10,
      remote: false,
    },
    count,
  );

  /**
   * @description: 获取组件列表
   */
  async function handleGetInstance() {
    componentList.value = await ComponentDefsService.listComponentDefs({
      scopeWorkspaceID: spaceStore.currentSpace,
    } as UnSupportCategoryParams).catch(() => []);
  }

  const envNameMap = ref<Record<string, EnvOutput>>({});

  function getDynamicInputValue(value: unknown): DynamicInputValue {
    if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
      return value;
    }
    if (value === undefined || value === null) {
      return '';
    }
    return JSON.stringify(value);
  }

  function getEnvTagClass(env: string) {
    const envType = envNameMap.value[env]?.type;
    return envType ? envTypeTagClassMap[envType] : '';
  }

  function getFormProperties(properties?: Record<string, unknown>) {
    return Object.entries(properties || {}).reduce(
      (acc, [key, value]) => {
        acc[key] = getDynamicInputValue(value);
        return acc;
      },
      {} as Record<string, DynamicInputValue>,
    );
  }

  /**
   * @description: 处理下拉框选项
   */
  function handleSelectOptions(val: PropertyDefOutputObj): SelectOption[] {
    if (val.type !== 'SELECT') return [];
    return (val.options || []).filter(
      (option): option is SelectOption => typeof option.label === 'string' && typeof option.value === 'string',
    );
  }

  // 属性列表
  const propertyList = ref<NamedPropertyDef[]>([]);

  /**
   * @description: 获取空间组件实例
   */
  async function getComponentInstanceList() {
    instanceList.value = await WorkspaceComponentsService.listWorkspaceComponents({
      workspaceID: spaceStore.currentSpace,
    }).catch(() => []);
  }
  /**
   * @description: 获取组件属性列表
   */
  // function handleGetPropertyList(component: ComponentDefOutputObj) {
  //   propertyList.value = (component?.properties || []) as PropertyDefOutputObj[];
  // }

  /**
   * @description: 重置组件属性和表单参数
   * @param {ComponentDefOutputObj} component
   */
  function handleResetProperty(component?: ComponentDefOutputObj) {
    propertyList.value = ((component?.properties || []) as PropertyDefOutputObj[]).filter(
      (item): item is NamedPropertyDef => !!item.name,
    );
    formModel.properties = {};
    propertyList.value.forEach(item => {
      // 组件下拉框类型字段约束 value：aaa|bbb|ccc，新增组件时，value需要默认选中第一项
      if (!props.data && item.type === 'SELECT' && item.defaultValue && typeof item.defaultValue === 'string') {
        const [firstValue] = item.defaultValue.split('|');
        formModel.properties[item.name] = firstValue;
      } else {
        formModel.properties[item.name] = getDynamicInputValue(item.defaultValue);
      }
    });
  }

  const curInstanceObj = ref<WorkspaceComponentOutputObj>();

  function handleBeforeClose() {
    return confirmBox();
  }

  // 获取应用环境变量
  function handleGetVarEnv(env: string) {
    if (props.moduleType === 'space') {
      const envID = envList.value.find(item => item.name === env)?.id;
      if (!envID) return Promise.resolve([]);
      return EnvvarsService.listEnvAvailableEnvVars({ envID });
    } else {
      return EnvvarsService.listAppEnvVars({
        appID: appDetailStore.appID,
        envName: env,
      });
    }
  }

  /**
   * 组件可用环境变化时触发
   */
  function handleScopedChange() {
    scopedEnvs.value = [];
  }

  /**
   * @description: 当选中空间组件实例时触发
   */
  function handleSelectInstance({ row }: { row: WorkspaceComponentOutputObj }) {
    curInstanceObj.value = row;

    if (curInstanceObj.value?.name) {
      isInstanceMissing.value = false;
    }
    if (!row?.properties) {
      isShow.value = false;
      return;
    }
    instanceDetail.name = row.name;
    instanceDetail.type = row.type;
    instanceDetail.version = row.version;
    instanceDetail.properties = getFormProperties(row.properties);
    isShow.value = true;
  }

  watch(
    () => props.modelValue,
    val => {
      localValue.value = val;
      if (props.data && val) {
        // 编辑态
        const data = props.data;
        const com = componentList.value.find(item => item.name === data?.type);
        if (com) {
          // 暂停监听，避免接口初始化触发 isDirty
          withPausedWatch(() => {
            curComponent.value = com;
            handleResetProperty(com);
            Object.assign(formModel, {
              ...cloneDeep(data),
              properties: getFormProperties(data.properties),
            });
          });
        }
        if (props.moduleType === 'app') {
          isUseInstance.value = !!props.data?.refWorkspaceCompName;
        }

        componentScoped.value = props.data?.scoped?.type || 'global';
        scopedEnvs.value = [...(props.data?.scoped?.envs || [])];
      }

      // 默认显示 popover
      if (val && !props.data) {
        setTimeout(() => {
          isShow.value = true;
        }, 200);
      }
    },
  );
  watch(localValue, val => {
    emits('update:modelValue', val);
  });
  watch(isUseInstance, val => {
    if (!val) {
      // 防止新增态点击重选时 popover 消失
      if (props.data) {
        isShow.value = false;
      }
      return;
    }
  });
  watch(curComponent, (_, oldVal) => {
    withPausedWatch(() => {
      // 重置版本和实例名称
      if (oldVal) {
        formModel.version = '';
        formModel.name = '';
      }
    });
  });

  onBeforeMount(async () => {
    await handleGetInstance();
    await handleGetEnvList(); // 获取环境列表
    await getComponentInstanceList();
    const tempEnvNameMap: Record<string, EnvOutput> = {};
    envList.value.forEach(item => {
      if (item.name) {
        tempEnvNameMap[item.name] = item;
      }
    });
    envNameMap.value = tempEnvNameMap;
  });
</script>
<style scoped lang="postcss">
  :deep(.bk-tab-content) {
    padding: 0 8px 0 0 !important;
    overflow: auto;

    .bk-tab-panel {
      height: 0px !important;
    }

    &::-webkit-scrollbar {
      width: 4px;
    }
  }
  :deep(.bk-sideslider-footer) {
    margin-top: 0px;
  }
  :deep(.bk-exception-part .bk-exception-img) {
    width: 320px;
    height: 160px;
  }

  /** 不预留滚动条位置，避免侧栏有类似padding的效果 */
  :deep(.bk-modal-content) {
    scrollbar-gutter: auto !important;
  }

  :deep(.bk-form-item) {
    &.form-item-properties {
      .bk-form-label {
        display: flex;
        & > span {
          display: flex;
          max-width: calc(100% - 18px);
          align-items: center;
          gap: 4px;
        }
        .position-relative {
          min-width: 0;
        }
        &::after {
          position: unset !important;
        }
      }

      &:last-child {
        margin-bottom: 0;
      }
    }
  }
</style>
