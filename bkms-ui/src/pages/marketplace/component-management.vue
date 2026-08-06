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
    v-model:is-show="isShow"
    :before-close="handleBeforeClose"
    quick-close
    render-directive="if"
    :width="isAsideVisible ? SIDE_SLIDER_WIDTH.withRefVarPanel : SIDE_SLIDER_WIDTH.default"
    @closed="handleClose"
  >
    <template #header>
      <span class="align-bottom">
        <span>{{ isEditMode ? t('编辑组件') : t('新建组件') }}</span>
        <template v-if="isEditMode">
          <span class="ml-[12px] mr-[9px] text-[#DCDEE5]">|</span>
          <span class="text-[14px] text-[#979BA5]">
            {{ currentEditComponent?.displayName || currentEditComponent?.name }}
          </span>
        </template>
      </span>
    </template>
    <template #default>
      <template v-if="currentStep === 1">
        <CollapsibleAsideLayout
          v-model:is-collapsed="isRefVarPanelCollapsed"
          :layout-config="{
            viewportOffset: 100,
          }"
          :max="SIDE_SLIDER_WIDTH.withRefVarPanel - SIDE_SLIDER_WIDTH.minWidth"
          :min="360"
          @after-resize="handleRefreshTable"
        >
          <template #main>
            <div class="flex flex-col gap-[24px] px-[24px] py-[18px]">
              <ToggleCard
                :name="$t('基本信息')"
                type="normal"
              >
                <Form
                  ref="formRef"
                  form-type="vertical"
                  :model="formData"
                  :rules="rules"
                >
                  <Form.FormItem
                    class="special-fomr-label"
                    :label="t('组件 ID')"
                    property="name"
                    required
                  >
                    <Input
                      v-model="formData.name"
                      :disabled="isEditMode"
                      :placeholder="t('请输入 2-20 字符的字母、数字、中划线，以字母开头')"
                    />
                  </Form.FormItem>
                  <!-- 是否公开 -->
                  <Form.FormItem
                    class="custom-form-content special-fomr-label"
                    :label="t('使用范围')"
                    property="isPublic"
                  >
                    <Radio.Group v-model="formData.isPublic">
                      <Radio :label="false">
                        <span class="text-[14px]">{{ t('仅本空间可见') }}</span>
                      </Radio>
                      <Radio :label="true">
                        <span class="text-[14px]">{{ t('所有空间可见') }}</span>
                      </Radio>
                    </Radio.Group>
                  </Form.FormItem>
                  <Form.FormItem
                    class="special-fomr-label mb-0"
                    :label="t('描述')"
                    property="description"
                  >
                    <Input
                      v-model="formData.description"
                      :maxlength="100"
                      :placeholder="t('请输入描述')"
                      type="textarea"
                    />
                  </Form.FormItem>
                </Form>
              </ToggleCard>
              <ComponentInputTemplate
                ref="inputTemplateRef"
                :initial-data="inputData"
              />
              <ComponentOutputTemplate
                ref="outputTemplateRef"
                :initial-data="outputData"
              >
                <template #header-right>
                  <Button
                    class="mr-[6px]"
                    text
                    theme="primary"
                    @click="setRefVarPanelVisible(!isAsideVisible)"
                  >
                    <i class="bkms-icon bkms-icon-variable text-[12px] mr-[4px]"></i>
                    <span class="text-[12px]">{{ $t('可引用变量') }}</span>
                  </Button>
                </template>
              </ComponentOutputTemplate>
            </div>
          </template>
          <template #aside>
            <RefVarPanel
              :input-variable-names="inputVariableNames"
              @close="setRefVarPanelVisible(false)"
            />
          </template>
        </CollapsibleAsideLayout>
      </template>
      <template v-if="currentStep === 2">
        <div class="h-[calc(100vh-100px)] flex flex-col gap-[16px] overflow-y-auto px-[24px] py-[18px]">
          <div class="bg-[#F1F2F6] text-[313238] font-bold text-[14px] px-[16px] py-[6px]">
            {{ $t('试运行结果预览') }}
          </div>
          <PatchPreviewCard
            v-for="(item, index) in trialRunResult?.patchPreview || []"
            :key="index"
            :base-manifest="item.baseManifest"
            :patched-manifest="item.patchedManifest"
            :target-kind="item.targetKind"
          />
          <ResourceCard
            v-for="(item, index) in trialRunResult?.resources || []"
            :key="index"
            :kind="item?.kind || '--'"
            :manifest="item.manifest"
          />
        </div>
      </template>
    </template>
    <template #footer>
      <div class="py-[8px] !mt-0 flex items-center gap-[8px]">
        <template v-if="currentStep === 1">
          <Button
            :loading="isTrialRunning"
            theme="primary"
            @click="handleTrialRun"
          >
            <span class="text-[14px]">{{ t('试运行') }}</span>
          </Button>
          <Button
            :disabled="isTrialRunning"
            @click="handleClose"
          >
            <span class="text-[14px]">{{ t('取消') }}</span>
          </Button>
        </template>
        <template v-else-if="currentStep === 2">
          <Button
            :disabled="isLoading"
            @click="currentStep = 1"
          >
            <span class="text-[14px]">{{ t('上一步') }}</span>
          </Button>
          <Button
            :loading="isLoading"
            theme="primary"
            @click="handleCommit"
          >
            <span class="text-[14px]">{{ t('提交') }}</span>
          </Button>
          <Button
            :disabled="isLoading"
            @click="handleClose"
          >
            <span class="text-[14px]">{{ t('取消') }}</span>
          </Button>
        </template>
      </div>
    </template>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, nextTick, provide, ref, watch } from 'vue';

  import { Button, Form, Input, Message, Radio, Sideslider } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { BuiltinVarOutputObj } from '~/@types/v1/component-defs';
  import { ComponentDefsService } from '~/api/modules/v1';
  import { BKMS_REGEX } from '~/common/const';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import ComponentInputTemplate from '~/pages/marketplace/components/component-input/component-input-template.vue';
  import ComponentOutputTemplate from '~/pages/marketplace/components/component-ouput/component-output-template.vue';
  import PatchPreviewCard from '~/pages/marketplace/components/component-ouput/patch-preview-card.vue';
  import ResourceCard from '~/pages/marketplace/components/component-ouput/resource-card.vue';
  import { BUILTIN_VARS_SYMBOL, REFRESH_TABLE_SIGNAL } from '~/pages/marketplace/components/params-table/constants';
  import RefVarPanel from '~/pages/marketplace/components/ref-var-panel.vue';
  import { useSpaceStore } from '~/stores/space';

  import type {
    ComponentDefOutputFormInput,
    ComponentDefOutputObj,
    PropertyDefInput,
  } from '~/@types/v1/component-defs';

  interface IEmit {
    (e: 'refresh'): void;
  }

  interface IProps {
    typeOptions?: ITypeOption[];
  }

  type ITypeOption = {
    label: string;
    value: string;
  };

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

  interface TrialRunResponse {
    patchPreview: PatchPreviewItem[];
    resources: ResourceItem[];
  }

  /** 面板宽度 */
  const SIDE_SLIDER_WIDTH = {
    default: 960,
    withRefVarPanel: 1200,
    minWidth: 650,
  };

  defineProps<IProps>();
  const emits = defineEmits<IEmit>();

  const spaceStore = useSpaceStore();

  // 引入国际化
  const { t } = useI18n();
  const isEditMode = ref(false); // 是否为编辑模式
  const currentEditComponent = ref<ComponentDefOutputObj>(); // 当前编辑的组件数据
  /** 编辑模式下当前组件的 properties，传给 ComponentInputTemplate 初始化 */
  const inputData = ref<PropertyDefInput[]>([]);
  /** 编辑模式下当前组件的输出模板数据，传给 ComponentOutputTemplate 初始化 */
  const outputData = ref<ComponentDefOutputFormInput>();

  const isShow = ref(false);
  const isLoading = ref(false);
  const currentStep = ref(1); // 1: 表单填写页, 2: 试运行结果页
  const trialRunResult = ref<TrialRunResponse>();
  const isTrialRunning = ref(false);

  /** 内置变量列表 */
  const builtinVars = ref<BuiltinVarOutputObj[]>([]);

  const inputTemplateRef = ref<InstanceType<typeof ComponentInputTemplate>>();
  const outputTemplateRef = ref<InstanceType<typeof ComponentOutputTemplate>>();
  const isRefVarPanelCollapsed = ref(true);
  const isAsideVisible = computed(() => !isRefVarPanelCollapsed.value);

  /** 面板切换后通知 ParamsTable 刷新宽度 */
  const refreshTableSignal = ref(0);
  provide(REFRESH_TABLE_SIGNAL, refreshTableSignal);
  provide(BUILTIN_VARS_SYMBOL, builtinVars);

  const inputVariableNames = computed(() => {
    const data = inputTemplateRef.value?.getFormData() ?? [];
    return data.map(row => ({ name: row.name, description: row.description ?? '' })).filter(row => row.name);
  });

  // 保存初始数据，用于离开时对比
  const originInputData = ref('');
  const originOutputData = ref('');

  const formData = ref({
    name: '',
    isPublic: false,
    description: '',
    workloadType: '',
  });
  const { confirmBox, forceCleanDirtyTag, withPausedWatch } = useLeaveConfirm(formData);
  const rules = {
    name: [
      {
        validator: (value: string) => value.length,
        message: t('请输入组件ID'),
        trigger: 'blur',
      },
      {
        validator: (value: string) => BKMS_REGEX.componentNameRegex.test(value),
        message: t('以字母开头，可包含字母、数字、中划线，长度2-20位'),
        trigger: 'blur',
      },
    ],
  };
  // ref
  const formRef = ref();

  function close() {
    withPausedWatch(() => {
      isShow.value = false;
      init();
    });
  }

  // 侧边栏关闭前确认
  function handleBeforeClose(): Promise<boolean> {
    const inputChanged = JSON.stringify(inputTemplateRef.value?.getValue() ?? []) !== originInputData.value;
    const outputChanged = JSON.stringify(outputTemplateRef.value?.getOutputForm()) !== originOutputData.value;
    const subComponentClean = !inputChanged && !outputChanged;
    return confirmBox(true, { validates: [() => subComponentClean] });
  }

  async function handleClose() {
    if (await handleBeforeClose()) {
      close();
    }
  }

  // 提交创建或更新
  async function handleCommit() {
    isLoading.value = true;

    // 公共参数
    const baseParams = {
      compDefName: formData.value.name,
      description: formData.value.description,
      outputForm: { name: formData.value.name, ...outputData.value },
      scopeType: (formData.value.isPublic ? 'global' : 'workspace') as 'global' | 'workspace',
      scopeWorkspaceIDs: formData.value.isPublic ? [] : [spaceStore.currentSpace],
      managedByWorkspaceIDs: [spaceStore.currentSpace],
    } satisfies {
      [key: string]: unknown;
      scopeType: 'global' | 'workspace';
    };

    // 根据模式调用不同接口
    const res = isEditMode.value
      ? await ComponentDefsService.patchComponentDef({
          ...baseParams,
          propertiesInput: {
            properties: inputData.value,
          },
        }).catch(() => false)
      : await ComponentDefsService.createComponentDef({
          ...baseParams,
          properties: inputData.value,
        }).catch(() => false);
    forceCleanDirtyTag(() => {
      if (res === false) {
        isLoading.value = false;
        Message({
          theme: 'danger',
          message: isEditMode.value ? t('组件更新失败') : t('组件新建失败'),
        });
        return;
      }
      isLoading.value = false;
      Message({
        theme: 'success',
        message: isEditMode.value ? t('组件更新成功') : t('组件新建成功'),
      });
      emits('refresh');
      close();
    });
  }

  function handleRefreshTable() {
    setTimeout(() => refreshTableSignal.value++, 300);
  }

  async function handleTrialRun() {
    try {
      if (!(await validateForm())) return;
      isTrialRunning.value = true;
      const outputForm = outputTemplateRef.value?.getOutputForm();
      const properties = inputTemplateRef.value?.getValue() || [];
      trialRunResult.value = await ComponentDefsService.previewComponentDef(
        {
          compDefName: formData.value.name,
          outputForm: { name: formData.value.name, ...outputForm },
          properties,
        },
        { needRes: true },
      );
      inputData.value = properties;
      outputData.value = outputForm;
      currentStep.value = 2;
    } finally {
      isTrialRunning.value = false;
    }
  }

  function init() {
    formData.value = {
      name: '',
      isPublic: false,
      description: '',
      workloadType: '',
    };
    isEditMode.value = false;
    currentEditComponent.value = undefined;
    inputData.value = [];
    outputData.value = undefined;
    originInputData.value = '';
    originOutputData.value = '';
    inputTemplateRef.value?.resetData();
    trialRunResult.value = undefined;
    isTrialRunning.value = false;
  } // 打开弹窗 component: 新建组件, edit: 编辑组件

  function open(typeParam: 'component' | 'edit', componentData?: ComponentDefOutputObj) {
    isShow.value = true;
    isEditMode.value = typeParam === 'edit';
    currentStep.value = 1;

    // 获取内置变量（异步，不阻塞 UI）
    ComponentDefsService.getComponentDefsBuiltinVars()
      .then(res => {
        builtinVars.value = res || [];
      })
      .catch(() => {
        builtinVars.value = [];
      });

    withPausedWatch(() => {
      if (typeParam === 'edit' && componentData) {
        // 编辑模式
        currentEditComponent.value = componentData;
        inputData.value = (componentData.properties || []) as PropertyDefInput[];
        outputData.value =
          componentData.outputForm?.patcher?.length || componentData.outputForm?.spec?.length
            ? {
                patcher: componentData.outputForm?.patcher || [],
                spec: componentData.outputForm?.spec || [],
                name: componentData.outputForm?.name,
              }
            : undefined;
        formData.value = {
          name: componentData.name || '',
          isPublic: componentData.scopeType === 'global',
          description: componentData.description || '',
          workloadType: '',
        };
      } else {
        // 新建模式
        init();
      }
    });
  }

  function setRefVarPanelVisible(visible: boolean) {
    isRefVarPanelCollapsed.value = !visible;
    handleRefreshTable();
  }

  // 表单校验
  async function validateForm() {
    const valid = await formRef.value?.validate().catch(() => false);
    const inputValid = await inputTemplateRef.value?.isValid();
    const outputValid = outputTemplateRef.value?.isValid();
    return !!(valid && inputValid && outputValid);
  }

  // 子组件挂载后快照初始数据（Sideslider render-directive="if" 导致子组件异步挂载，需等 ref 变为实例）
  watch(
    inputTemplateRef,
    ref => {
      if (!ref || !isShow.value) return;
      nextTick(() => {
        originInputData.value = JSON.stringify(ref.getValue() ?? []);
        originOutputData.value = JSON.stringify(outputTemplateRef.value?.getOutputForm() ?? '');
      });
    },
    { flush: 'post' },
  );

  defineExpose({
    open,
    close,
  });
</script>

<style lang="postcss" scoped>
  :deep(.bk-form-label) {
    span {
      font-size: 12px;
    }
  }

  :deep(.custom-form-content) {
    .bk-form-content {
      line-height: 12px;
    }
  }

  :deep(.special-fomr-label) {
    .bk-form-label > span {
      font-size: 14px;
    }
  }

  :deep(.bk-modal-content) {
    scrollbar-gutter: auto !important;
  }

  :deep(.ms-editor) {
    padding-left: 25px;
  }
</style>
