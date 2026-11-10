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
    v-model:is-show="visible"
    :before-close="handleBeforeClose"
    quick-close
    render-directive="if"
    :width="960"
    @closed="handleClosed"
  >
    <template #header>
      <div
        class="flex items-center"
        data-testid="strategy-sideslider-header"
      >
        <span>{{ title }}</span>
        <template v-if="headerName">
          <span class="mx-[12px] h-[12px] w-px shrink-0 bg-[#dcdee5]"></span>
          <span class="text-[14px] leading-[22px] text-[#979ba5]">{{ headerName }}</span>
          <Tag
            v-if="showBuiltinTag"
            class="ml-[8px]"
            size="small"
          >
            {{ t('平台内置') }}
          </Tag>
        </template>
      </div>
    </template>

    <div
      ref="contentRef"
      class="flex h-full flex-col gap-[24px] overflow-y-auto px-[24px] pb-[16px] pt-[20px]"
    >
      <Alert
        class="shrink-0"
        theme="info"
        :title="t('策略保存在当前空间，空间下所有应用都可以选用')"
      />

      <BkmsContent
        v-model:collapsed="basicCollapsed"
        :collapsible="true"
        :title="t('基本信息')"
      >
        <Form
          ref="basicFormRef"
          class="basic-info-form mt-[24px]"
          form-type="vertical"
          :model="formModel"
          :rules="basicFormRules"
        >
          <Form.FormItem
            class="basic-info-form-item"
            :label="t('策略名称')"
            property="name"
            required
          >
            <Input
              ref="nameInputRef"
              v-model.trim="formModel.name"
              clearable
              :disabled="isView"
              :maxlength="NAME_MAX_LENGTH"
              :placeholder="t('请输入策略名称，1 ~ 32 个字符')"
              show-clear-only-hover
              @blur="handleNameBlur"
            />
          </Form.FormItem>

          <Form.FormItem
            class="basic-info-form-item"
            :label="t('策略说明')"
            property="description"
          >
            <Input
              ref="descriptionInputRef"
              v-model.trim="formModel.description"
              :disabled="isView"
              :maxlength="DESCRIPTION_MAX_LENGTH"
              :placeholder="t('例如：生产环境先放 1 个实例观察，再按 25% / 50% / 全量放量')"
              show-word-limit
              type="textarea"
            />
          </Form.FormItem>
        </Form>
      </BkmsContent>

      <BkmsContent
        v-model:collapsed="batchCollapsed"
        :collapsible="true"
        :title="t('发布批次')"
      >
        <div class="flex flex-col gap-[8px]">
          <div class="flex items-center gap-[4px] text-[12px] leading-[20px] mt-[24px]">
            <span class="font-700 text-[#313238]">{{ t('发布批次') }}</span>
            <span class="text-[#ea3636]">*</span>
            <span class="text-[#979ba5]"> （{{ t('更新实例会换镜像；调整流量只改已更新实例的北极星权重') }}） </span>
          </div>

          <div class="flex items-start gap-[16px] max-lg:flex-col">
            <StrategyBatchList
              :active-key="activeBatchKey"
              :batches="formModel.batches"
              :errors="batchErrors"
              :readonly="isView"
              @add="addBatch"
              @remove="removeBatch"
              @select="handleSelectBatch"
            />
            <StrategyBatchDetail
              v-if="activeBatch"
              v-model="activeBatchModel"
              :error="batchErrorDetails[activeBatch.key] ?? batchErrors[activeBatch.key]"
              :index="activeBatchIndex"
              :readonly="isView"
            />
          </div>

          <div
            v-if="batchesEmptyError"
            class="text-[12px] leading-[20px] text-[#ea3636]"
          >
            {{ batchesEmptyError }}
          </div>
        </div>
      </BkmsContent>
    </div>

    <template #footer>
      <div class="flex gap-[8px]">
        <template v-if="isView">
          <Button
            theme="primary"
            @click="handleClone"
          >
            {{ t('克隆为空间策略') }}
          </Button>
          <Button @click="handlePreview">
            {{ t('预览策略') }}
          </Button>
          <Button @click="handleCancel">{{ t('关闭') }}</Button>
        </template>
        <template v-else>
          <Button
            :loading="saving"
            theme="primary"
            @click="handleSubmit"
          >
            {{ t('提交') }}
          </Button>
          <Button @click="handlePreview">
            {{ t('预览策略') }}
          </Button>
          <Button @click="handleCancel">{{ t('取消') }}</Button>
        </template>
      </div>
    </template>
  </Sideslider>

  <StrategyPreviewDialog
    v-model:is-show="previewVisible"
    :batches="formModel.batches"
    :name="formModel.name"
    :show-submit="!isView"
    @submit="handleSubmit"
  />
</template>

<script setup lang="ts">
  import { computed, nextTick, reactive, ref, watch } from 'vue';

  import { Alert, Button, Form, Input, Message, Sideslider, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { ReleaseStrategiesService } from '~/api/modules/v1/release-strategies';
  import BkmsContent from '~/components/bkms-content.vue';
  import { useSpaceStore } from '~/stores/space';

  import useStrategyFormDirty from '../use-strategy-form-dirty';
  import {
    createEmptyBatch,
    DESCRIPTION_MAX_LENGTH,
    generateCloneName,
    NAME_MAX_LENGTH,
    STRATEGY_NAME_CONFLICT_PATTERN,
    toBatchForms,
    toBatchInputs,
    translate,
    validateStrategyForm,
    validateStrategyName,
  } from '../utils';
  import StrategyBatchDetail from './strategy-batch-detail.vue';
  import StrategyBatchList from './strategy-batch-list.vue';
  import StrategyPreviewDialog from './strategy-preview-dialog.vue';

  import type {
    StrategyBatchForm,
    StrategyFormData,
    StrategyFormValidateOptions,
    StrategyFormValidation,
  } from '../utils';
  import type { ReleaseStrategyOutputObj } from '~/@types/v1/release-strategies';

  interface Props {
    /** 当前空间已有策略名称：用于克隆时生成不重名的默认名称，以及名称重名的前端同步校验 */
    existingNames?: string[];
    /** 打开场景：新建 / 编辑 / 查看 / 克隆 */
    mode: 'clone' | 'create' | 'edit' | 'view';
    /** 编辑/查看/克隆时传入的策略行数据，新建为空 */
    strategy?: ReleaseStrategyOutputObj;
  }

  const props = withDefaults(defineProps<Props>(), {
    existingNames: () => [],
    strategy: undefined,
  });

  const emit = defineEmits<{
    /** 携带保存后的策略，供调用方（如部署抽屉）直接选中新建的策略 */
    success: [strategy?: ReleaseStrategyOutputObj];
  }>();

  const visible = defineModel<boolean>('isShow', { default: false });

  const { t } = useI18n();
  const spaceStore = useSpaceStore();

  const formModel = reactive<StrategyFormData>({
    name: '',
    description: '',
    batches: [],
  });

  const saving = ref(false);
  const batchesEmptyError = ref('');
  /** 批次行级错误（卡片简短文案） */
  const batchErrors = ref<Record<string, string>>({});
  /** 批次行级错误详情（详情面板文案，可携带上一批累计值） */
  const batchErrorDetails = ref<Record<string, string>>({});
  const activeBatchKey = ref('');
  /** 抽屉内容容器：定位错误时在其内部按批次 key 查找卡片 */
  const contentRef = ref<HTMLElement | null>(null);
  const basicFormRef = ref<InstanceType<typeof Form> | null>(null);
  /** 后端返回的名称类错误（如重名）：接口信息优先于前端同步校验，回填后立即展示 */
  const apiNameError = ref('');
  const nameInputRef = ref<InstanceType<typeof Input> | null>(null);
  const descriptionInputRef = ref<InstanceType<typeof Input> | null>(null);
  const basicCollapsed = ref(false);
  const batchCollapsed = ref(false);
  const previewVisible = ref(false);
  /** 查看态点击「克隆为空间策略」进入克隆态时置 true：提交成功后提示「已克隆为空间策略」而非「策略已创建」 */
  const clonedFromView = ref(false);

  /**
   * 当前场景：打开时取 props.mode，查看态点击「克隆为空间策略」会在抽屉内切到克隆态，
   * 因此独立于 props 维护，避免为了切换表单反向修改父组件的打开场景。
   */
  const mode = ref<Props['mode']>(props.mode);

  const isView = computed(() => mode.value === 'view');
  const isEdit = computed(() => mode.value === 'edit');

  /**
   * 名称唯一性校验上下文：列表数据在前端同步校验，后端仍会保底校验。
   * 编辑态排除策略自身名称（未改名不算重复）；克隆必须使用新名称，故不排除。
   */
  const nameValidateOptions = computed<StrategyFormValidateOptions>(() => ({
    existingNames: props.existingNames,
    selfName: isEdit.value ? (props.strategy?.name ?? '') : '',
  }));

  /**
   * bkui 的 validator 声明类型只有 boolean，但运行时支持返回字符串作为错误文案
   * （内部 `typeof data === 'string'` 即 reject）。注意空串同样会被判定为失败并标红，
   * 因此这里统一把「无错误（空串）」转成 true，只在有文案时返回字符串。
   */
  function toFormValidator(validate: (value: string) => string): (value: unknown) => boolean | Promise<boolean> {
    return (value => validate(value == null ? '' : String(value)) || true) as (
      value: unknown,
    ) => boolean | Promise<boolean>;
  }

  /**
   * 基本信息表单校验规则：复用 utils 中的纯函数，保证「失焦即时校验」与「提交校验」口径一致。
   * required 由规则自身声明，覆盖 FormItem 内置的空值规则，保证空值文案与 utils 一致。
   */
  const basicFormRules = computed(() => ({
    name: [
      {
        required: true,
        trigger: 'blur',
        validator: toFormValidator((value: string) => {
          if (apiNameError.value) return apiNameError.value;
          const error = validateStrategyName(value, nameValidateOptions.value);
          return error ? t(error) : '';
        }),
      },
    ],
    description: [
      {
        trigger: 'blur',
        validator: toFormValidator((value: string) =>
          (value ?? '').length > DESCRIPTION_MAX_LENGTH ? t('策略说明不能超过 100 个字符') : '',
        ),
      },
    ],
  }));

  /** 表单一有改动，取消 / 关闭（含右上角 ×、遮罩、ESC）前先确认是否离开 */
  const { confirmBox, resetDirty } = useStrategyFormDirty(formModel);

  const title = computed(() => {
    const titleMap: Record<Props['mode'], string> = {
      create: t('新建策略'),
      edit: t('编辑策略'),
      view: t('查看策略'),
      clone: t('克隆策略'),
    };
    return titleMap[mode.value];
  });

  /** 标题副标题：查看/编辑/克隆时展示策略名称，新建时无名称不展示 */
  const headerName = computed(() => {
    if (mode.value === 'create') return '';
    return props.strategy?.name?.trim() ?? '';
  });

  /** 平台内置策略只读，标题后追加标签便于识别来源 */
  const showBuiltinTag = computed(() => Boolean(props.strategy?.isBuiltin));

  const activeBatchIndex = computed(() => formModel.batches.findIndex(batch => batch.key === activeBatchKey.value));

  const activeBatch = computed<StrategyBatchForm | undefined>(() => formModel.batches[activeBatchIndex.value]);

  /** 详情面板的双向绑定：批次对象本身来自 formModel.batches，写回时定位替换保持引用一致 */
  const activeBatchModel = computed<StrategyBatchForm>({
    get: () => formModel.batches[activeBatchIndex.value] as StrategyBatchForm,
    set: value => {
      const index = formModel.batches.findIndex(batch => batch.key === value.key);
      if (index > -1) formModel.batches[index] = value;
    },
  });

  function addBatch() {
    const batch = createEmptyBatch();
    formModel.batches.push(batch);
    activeBatchKey.value = batch.key;
  }

  /**
   * 重置/回填表单：克隆时按「原名_克隆NN」预填名称，便于直接提交且不与已有策略重名。
   * resetDirty 放在末尾，等回填产生的深层变更全部触发后再清除离开确认标记，避免误弹确认。
   */
  function fillForm(targetMode: Props['mode'] = props.mode) {
    mode.value = targetMode;
    basicFormRef.value?.clearValidate();
    batchesEmptyError.value = '';
    batchErrors.value = {};
    batchErrorDetails.value = {};
    // 后端名称错误只针对本次提交的策略，回填另一条策略时必须清掉，否则合法名称也会被旧错误挡住
    apiNameError.value = '';
    basicCollapsed.value = false;
    batchCollapsed.value = false;
    previewVisible.value = false;

    const strategy = props.strategy;
    if (!strategy) {
      formModel.name = '';
      formModel.description = '';
      formModel.batches = [createEmptyBatch()];
      activeBatchKey.value = formModel.batches[0]?.key ?? '';
      resetDirty();
      return;
    }

    formModel.name =
      targetMode === 'clone' ? generateCloneName(strategy.name, props.existingNames) : (strategy.name ?? '');
    formModel.description = strategy.description ?? '';
    const batches = toBatchForms(strategy.batches);
    formModel.batches = batches.length > 0 ? batches : [createEmptyBatch()];
    activeBatchKey.value = formModel.batches[0]?.key ?? '';
    resetDirty();
  }

  /**
   * 关闭前确认：取消按钮、右上角 ×、遮罩/ESC 关闭统一走这里，
   * 表单有改动时先弹「确认离开」；查看态只读，无需确认。
   */
  async function handleBeforeClose(): Promise<boolean> {
    if (isView.value) return true;
    const confirmed = await confirmBox();
    // 关闭被取消时一并收起预览弹窗，避免「表单还在、预览已关」的状态残留
    if (!confirmed) previewVisible.value = false;
    return confirmed;
  }

  async function handleCancel() {
    if (!(await handleBeforeClose())) return;
    visible.value = false;
  }

  /**
   * 查看态 → 克隆为空间策略：复用原策略的全部配置切到克隆表单，原策略不受影响；
   * 后续提交走新建（create），取消则直接关闭抽屉，不产生新策略。
   */
  function handleClone() {
    if (!props.strategy) return;
    clonedFromView.value = true;
    fillForm('clone');
  }

  function handleClosed() {
    batchErrors.value = {};
    batchErrorDetails.value = {};
    // 抽屉跨策略复用，关闭时清掉后端名称错误，避免残留到下一次打开
    apiNameError.value = '';
    batchesEmptyError.value = '';
    clonedFromView.value = false;
  }

  /** 名称失焦：先清掉后端返回的重名文案，再交给 Form 重新校验，避免旧错误一直挂着 */
  function handleNameBlur() {
    apiNameError.value = '';
    // 校验失败会 reject，错误已渲染到 FormItem，这里吞掉避免未捕获的 Promise
    void basicFormRef.value?.validate('name')?.catch(() => {});
  }

  async function handlePreview() {
    // 查看态数据来自接口且不可编辑，无需前置校验拦截
    if (!isView.value && !(await validateAndFocus())) return;
    previewVisible.value = true;
  }

  function handleSelectBatch(key: string) {
    activeBatchKey.value = key;
  }

  async function handleSubmit() {
    if (!(await validateAndFocus())) return;
    if (!spaceStore.currentSpace) return;

    saving.value = true;
    try {
      const payload = {
        name: formModel.name.trim(),
        description: formModel.description,
        batches: toBatchInputs(formModel.batches),
      };
      // 关闭默认错误弹窗，名称重复等 400 错误在表单内联提示
      const config = { interceptorErr: false };
      let saved: ReleaseStrategyOutputObj | undefined;
      if (isEdit.value && props.strategy?.id) {
        saved = await ReleaseStrategiesService.updateReleaseStrategy(
          { workspaceID: spaceStore.currentSpace, strategyID: props.strategy.id, ...payload },
          config,
        );
        Message({ theme: 'success', message: t('策略已更新') });
      } else {
        saved = await ReleaseStrategiesService.createReleaseStrategy(
          { workspaceID: spaceStore.currentSpace, ...payload },
          config,
        );
        Message({
          theme: 'success',
          message: clonedFromView.value ? t('已克隆为空间策略「{0}」', [saved?.name]) : t('策略已创建'),
        });
      }
      visible.value = false;
      emit('success', saved);
    } catch (error) {
      console.error(error);
      const apiError = error as { error?: { message?: string } };
      const message = apiError?.error?.message || t('请求异常');
      // 仅精确匹配名称冲突（后端文案为 "release strategy name already exists" 等已存在表述），
      // 不能宽松匹配 name，否则 envName invalid 等其他含 name 的报错会被误挂到「策略名称」项下
      if (STRATEGY_NAME_CONFLICT_PATTERN.test(message)) {
        // 后端重名文案内联展示在名称项下，与前端校验一致
        apiNameError.value = message;
        await nextTick();
        void basicFormRef.value?.validate('name')?.catch(() => {});
      } else {
        Message({ theme: 'error', message });
      }
    } finally {
      saving.value = false;
    }
  }

  /**
   * 定位第一个错误：按表单视觉顺序（策略名称 → 策略说明 → 首个出错批次）展开分组并滚动到可视区。
   * 字段级错误优先聚焦输入框；批次错误则选中该批次，由详情面板展示具体错误。
   */
  async function locateFirstError(validation: StrategyFormValidation, hasBasicError: boolean) {
    const firstErrorBatchKey = formModel.batches.find(batch => validation.batchErrors[batch.key])?.key;

    if (hasBasicError && validation.nameError) {
      basicCollapsed.value = false;
      await nextTick();
      nameInputRef.value?.focus();
      scrollToElement(nameInputRef.value?.$el);
      return;
    }

    if (hasBasicError && validation.descriptionError) {
      basicCollapsed.value = false;
      await nextTick();
      descriptionInputRef.value?.focus();
      scrollToElement(descriptionInputRef.value?.$el);
      return;
    }

    if (firstErrorBatchKey) {
      batchCollapsed.value = false;
      activeBatchKey.value = firstErrorBatchKey;
      await nextTick();
      // 批次列表自身是滚动容器，按 key 精确定位到出错卡片
      const card = contentRef.value?.querySelector<HTMLElement>(`[data-batch-key="${firstErrorBatchKey}"]`);
      scrollToElement(card);
      return;
    }

    if (validation.batchesEmptyError) batchCollapsed.value = false;
  }

  /** 删除指定批次：至少保留一批；删的是当前选中项时优先选中后一个，末位时回退到前一个 */
  function removeBatch(key: string) {
    // 至少保留一个批次：全量由系统在末批确认后自动补齐，无需用户单独配置
    if (formModel.batches.length <= 1) return;
    const index = formModel.batches.findIndex(batch => batch.key === key);
    if (index < 0) return;
    const target = formModel.batches[index] as StrategyBatchForm;

    formModel.batches.splice(index, 1);
    delete batchErrors.value[target.key];
    delete batchErrorDetails.value[target.key];
    if (activeBatchKey.value !== key) return;

    const next = formModel.batches[index] ?? formModel.batches[index - 1];
    activeBatchKey.value = next?.key ?? '';
  }

  /** jsdom 未实现 scrollIntoView，需先判断可调用性，避免单测环境抛错 */
  function scrollToElement(element: HTMLElement | null | undefined) {
    if (typeof element?.scrollIntoView === 'function') {
      element.scrollIntoView({ block: 'center', behavior: 'smooth' });
    }
  }

  /**
   * 提交与预览共用的唯一校验入口：
   * 基本信息走 Form 校验（错误展示在 FormItem 下）→ 批次错误回填 → 展开分组 → 定位到第一个错误。
   */
  async function validateAndFocus(): Promise<boolean> {
    // 统一从 utils 取一次结果：批次错误仍由抽屉内联展示，字段错误交给 Form 渲染
    const validation = validateStrategyForm(formModel, nameValidateOptions.value);
    batchesEmptyError.value = validation.batchesEmptyError ? t(validation.batchesEmptyError) : '';
    batchErrors.value = Object.fromEntries(
      Object.entries(validation.batchErrors).map(([key, value]) => [key, t(value)]),
    );
    batchErrorDetails.value = Object.fromEntries(
      Object.entries(validation.batchErrorDetails).map(([key, { key: text, params }]) => [
        key,
        translate(t, text, params),
      ]),
    );

    // Form 校验会同步渲染字段错误，用 try/catch 兜住 reject，避免未捕获的 Promise
    let basicValid = true;
    try {
      await basicFormRef.value?.validate();
    } catch {
      basicValid = false;
    }

    if (basicValid && validation.valid) return true;

    await nextTick();
    locateFirstError(validation, !basicValid);
    return false;
  }

  watch(visible, isShow => {
    if (isShow) fillForm();
  });
</script>

<style lang="postcss" scoped>
  /* 基本信息：名称与说明上下排列，标签在输入框上方；与「发布批次」区块保持 24px 间距 */
  .basic-info-form :deep(.bk-form-item) {
    margin-bottom: 0;
  }

  .basic-info-form :deep(.bk-form-item + .bk-form-item) {
    margin-top: 16px;
  }

  .basic-info-form :deep(.bk-form-label) {
    align-items: flex-end;
    line-height: 20px;
  }

  /* 错误文案改为文档流内展示，避免绝对定位浮在该项与下一项之间 */
  .basic-info-form :deep(.bk-form-error) {
    position: static;
    padding-top: 2px;
    font-size: 12px;
    line-height: 20px;
  }
</style>
