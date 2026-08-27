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
  <BkmsContent
    class="shadow-[0_2px_4px_0_#1919290d]"
    :collapsible="true"
    :editing="isEditing"
    :show-edit-icon="!isEditing"
    @edit="handleEdit"
  >
    <!-- 标题带 Tooltip 提示 Request/Limit 含义 -->
    <template #title>
      <UnderLineTips
        class="line-height-[18px]"
        :description="$t('Request为最小值，Limit为最大值')"
        :placement="PlacementEnum.RIGHT"
      >
        {{ $t('资源规格') }}
      </UnderLineTips>
      <EnvScopeTag :current-env="currentEnv" />
    </template>
    <div class="bg-[#FFF] p-[16px]">
      <Form
        :ref="appSpecSection.formRef"
        form-type="vertical"
        :model="formModel"
        :rules="rules"
      >
        <!-- 查看态：以只读方式展示当前资源配置 -->
        <template v-if="!isEditing">
          <div class="grid grid-cols-2 gap-[12px] gap-y-2">
            <!-- 实例数（副本数）：占满两列 -->
            <FieldItem
              class="col-span-2"
              :container-height="20"
            >
              <template #field>
                <ModifiedFieldLabel
                  :label="$t('实例数')"
                  :modified="isFieldOverridden('replicas')"
                />
              </template>
              <template #value>
                <!-- 自动扩缩容展示 -->
                <template v-if="isAutoScaleEnabled">
                  <span :class="[FIELD_VALUE_BASE_CLASS, 'inline-flex items-center gap-[6px]']">
                    {{ autoScaleReplicasText }}
                    <AutoScaleTag
                      :enabled="isAutoScaleEnabled"
                      :status="autoScaleConfig?.status"
                    />
                  </span>
                </template>
                <template v-else>
                  <span
                    v-bk-tooltips="{
                      content: getDefaultValueTip('replicas'),
                      disabled: !getDefaultValueTip('replicas'),
                    }"
                    :class="[FIELD_VALUE_BASE_CLASS, { [FIELD_VALUE_MODIFIED_CLASS]: getDefaultValueTip('replicas') }]"
                  >
                    {{ formModel.replicas }}
                  </span>
                </template>
              </template>
            </FieldItem>
            <!-- CPU Requests -->
            <FieldItem :container-height="20">
              <template #field>
                <ModifiedFieldLabel
                  :label="`CPU ${$t('预留')} (Requests)`"
                  :modified="isFieldOverridden('cpuRequests')"
                />
              </template>
              <template #value>
                <span
                  v-bk-tooltips="{
                    content: getDefaultValueTip('cpuRequests'),
                    disabled: !getDefaultValueTip('cpuRequests'),
                  }"
                  :class="[FIELD_VALUE_BASE_CLASS, { [FIELD_VALUE_MODIFIED_CLASS]: getDefaultValueTip('cpuRequests') }]"
                >
                  {{ formModel.cpuRequests ? $t('{0} 核', [getCoreCount(formModel.cpuRequests)]) : '--' }}
                </span>
              </template>
            </FieldItem>
            <!-- CPU Limits -->
            <FieldItem :container-height="20">
              <template #field>
                <ModifiedFieldLabel
                  :label="`CPU ${$t('限制')} (Limits)`"
                  :modified="isFieldOverridden('cpuLimits')"
                />
              </template>
              <template #value>
                <span
                  v-bk-tooltips="{
                    content: getDefaultValueTip('cpuLimits'),
                    disabled: !getDefaultValueTip('cpuLimits'),
                  }"
                  :class="[FIELD_VALUE_BASE_CLASS, { [FIELD_VALUE_MODIFIED_CLASS]: getDefaultValueTip('cpuLimits') }]"
                >
                  {{ formModel.cpuLimits ? $t('{0} 核', [getCoreCount(formModel.cpuLimits)]) : '--' }}
                </span>
              </template>
            </FieldItem>
            <!-- 内存 Requests -->
            <FieldItem :container-height="20">
              <template #field>
                <ModifiedFieldLabel
                  :label="`${$t('内存预留')} (Requests)`"
                  :modified="isFieldOverridden('memoryRequests')"
                />
              </template>
              <template #value>
                <span
                  v-bk-tooltips="{
                    content: getDefaultValueTip('memoryRequests'),
                    disabled: !getDefaultValueTip('memoryRequests'),
                  }"
                  :class="[
                    FIELD_VALUE_BASE_CLASS,
                    { [FIELD_VALUE_MODIFIED_CLASS]: getDefaultValueTip('memoryRequests') },
                  ]"
                >
                  {{ formModel.memoryRequests || '--' }}
                </span>
              </template>
            </FieldItem>
            <!-- 内存 Limits -->
            <FieldItem :container-height="20">
              <template #field>
                <ModifiedFieldLabel
                  :label="`${$t('内存限制')} (Limits)`"
                  :modified="isFieldOverridden('memoryLimits')"
                />
              </template>
              <template #value>
                <span
                  v-bk-tooltips="{
                    content: getDefaultValueTip('memoryLimits'),
                    disabled: !getDefaultValueTip('memoryLimits'),
                  }"
                  :class="[
                    FIELD_VALUE_BASE_CLASS,
                    { [FIELD_VALUE_MODIFIED_CLASS]: getDefaultValueTip('memoryLimits') },
                  ]"
                >
                  {{ formModel.memoryLimits || '--' }}
                </span>
              </template>
            </FieldItem>
          </div>
        </template>
        <!-- 编辑态：可输入并保存资源配置 -->
        <div v-else>
          <!-- 实例数（副本数） -->
          <Form.FormItem
            :class="[
              '!text-[#4D4F56] !w-[400px] relative',
              { 'field-diff-highlight': shouldShowResetIcon('replicas') },
            ]"
            :label="$t('实例数')"
            property="replicas"
            :required="!isAutoScaleEnabled"
          >
            <!-- 自动扩缩容，只读 -->
            <Input
              v-if="isAutoScaleEnabled"
              disabled
              :model-value="autoScaleReplicasText"
            />
            <Input
              v-else
              v-model.trim="formModel.replicas"
              :min="1"
              :precision="0"
              type="number"
            />
            <FieldResetHint
              :show-reset="!isAutoScaleEnabled && shouldShowResetIcon('replicas')"
              :tip="getDefaultValueTip('replicas')"
              @reset="handleResetField('replicas')"
            />
          </Form.FormItem>

          <!-- CPU 资源配置：Requests 和 Limits 并排显示 -->
          <div class="flex gap-x-[24px]">
            <Form.FormItem
              :class="['!w-[400px] relative', { 'field-diff-highlight': shouldShowResetIcon('cpuRequests') }]"
              label="CPU 预留 (Requests)"
              property="cpuRequests"
            >
              <Select v-model="formModel.cpuRequests">
                <Select.Option
                  v-for="option in getCpuOptions(formModel.cpuRequests)"
                  :key="option"
                  :label="$t('{0} 核', [getCoreCount(option)])"
                  :value="option"
                />
              </Select>
              <FieldResetHint
                :show-reset="shouldShowResetIcon('cpuRequests')"
                :tip="getDefaultValueTip('cpuRequests')"
                @reset="handleResetField('cpuRequests')"
              />
            </Form.FormItem>
            <Form.FormItem
              :class="['!w-[400px] relative', { 'field-diff-highlight': shouldShowResetIcon('cpuLimits') }]"
              label="CPU 限制 (Limits)"
              property="cpuLimits"
            >
              <Select v-model="formModel.cpuLimits">
                <Select.Option
                  v-for="option in getCpuOptions(formModel.cpuLimits)"
                  :key="option"
                  :label="$t('{0} 核', [getCoreCount(option)])"
                  :value="option"
                />
              </Select>
              <FieldResetHint
                :show-reset="shouldShowResetIcon('cpuLimits')"
                :tip="getDefaultValueTip('cpuLimits')"
                @reset="handleResetField('cpuLimits')"
              />
            </Form.FormItem>
          </div>

          <!-- 内存资源配置：Requests 和 Limits 并排显示 -->
          <div class="flex gap-x-[24px]">
            <Form.FormItem
              :class="['!w-[400px] relative', { 'field-diff-highlight': shouldShowResetIcon('memoryRequests') }]"
              label="内存预留 (Requests)"
              property="memoryRequests"
            >
              <Select v-model="formModel.memoryRequests">
                <Select.Option
                  v-for="option in MEMORY_OPTIONS"
                  :key="option"
                  :label="option"
                  :value="option"
                />
              </Select>
              <FieldResetHint
                :show-reset="shouldShowResetIcon('memoryRequests')"
                :tip="getDefaultValueTip('memoryRequests')"
                @reset="handleResetField('memoryRequests')"
              />
            </Form.FormItem>
            <Form.FormItem
              :class="['!w-[400px] relative', { 'field-diff-highlight': shouldShowResetIcon('memoryLimits') }]"
              label="内存限制 (Limits)"
              property="memoryLimits"
            >
              <Select v-model="formModel.memoryLimits">
                <Select.Option
                  v-for="option in MEMORY_OPTIONS"
                  :key="option"
                  :label="option"
                  :value="option"
                />
              </Select>
              <FieldResetHint
                :show-reset="shouldShowResetIcon('memoryLimits')"
                :tip="getDefaultValueTip('memoryLimits')"
                @reset="handleResetField('memoryLimits')"
              />
            </Form.FormItem>
          </div>

          <!-- 操作按钮区域 -->
          <div class="!mb-0 flex items-center">
            <Button
              :loading="saving"
              theme="primary"
              @click="onSave"
            >
              {{ $t('保存') }}
            </Button>
            <Button
              class="ml-[8px]"
              @click="handleCancelEdit"
            >
              {{ $t('取消') }}
            </Button>
            <!-- 仅非默认环境显示"恢复默认配置"按钮 -->
            <Button
              v-if="currentEnv && !currentEnv.isDefault"
              class="ml-[8px]"
              :loading="resetting"
              @click="onResetToDefault"
            >
              {{ $t('恢复默认配置') }}
            </Button>
          </div>
        </div>
      </Form>
    </div>
  </BkmsContent>
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Button, Form, Input, Select } from 'bkui-vue';
  import { PlacementEnum } from 'bkui-vue/lib/shared';
  import { useI18n } from 'vue-i18n';
  import { AppSpecResourcesInput, AppSpecResourcesOutput, EnvAppSpecResourcesInput } from '~/@types/v1/app-spec';
  import { AppSpecService } from '~/api/modules/v1';
  import BkmsContent from '~/components/bkms-content.vue';
  import FieldItem from '~/components/field-item.vue';
  import UnderLineTips from '~/components/under-line-tips.vue';
  import { useGPAConfigPolling } from '~/composables/use-gpa-config-polling';
  import AutoScaleTag from '~/pages/application/detail/components/auto-scale-tag.vue';
  import { useAppDetail } from '~/stores/app-detail';

  import EnvScopeTag from './env-scope-tag.vue';
  import FieldResetHint from './field-reset-hint.vue';
  import ModifiedFieldLabel from './modified-field-label.vue';
  import { createFieldAccessors, createFillEnvFormData, pickFields, useAppSpecSection } from './use-app-spec-section';

  import type { ExtendedEnv } from './types';

  // 查看态字段值样式常量
  const FIELD_VALUE_BASE_CLASS = 'text-[12px] text-[#313238]';
  const FIELD_VALUE_MODIFIED_CLASS = 'border-b border-dashed border-b-[#313238]';

  type FieldKey = 'cpuLimits' | 'cpuRequests' | 'memoryLimits' | 'memoryRequests' | 'replicas';

  interface Props {
    currentEnv: ExtendedEnv | null;
  }

  const FORM_DEFAULTS: AppSpecResourcesInput = {
    replicas: 1,
    cpuRequests: '',
    cpuLimits: '',
    memoryRequests: '',
    memoryLimits: '',
  };

  const ALL_FIELD_KEYS: FieldKey[] = ['replicas', 'cpuRequests', 'cpuLimits', 'memoryRequests', 'memoryLimits'];

  const FIELD_ACCESSORS = createFieldAccessors<AppSpecResourcesInput, FieldKey>(ALL_FIELD_KEYS, FORM_DEFAULTS);

  const CPU_OPTIONS = ['0.1', '0.2', '0.5', '1', '2', '4', '8', '16', '32'];
  const MEMORY_OPTIONS = ['256Mi', '512Mi', '1Gi', '2Gi', '4Gi', '8Gi', '16Gi', '32Gi', '64Gi'];

  const props = defineProps<Props>();

  const emit = defineEmits<{
    'env-modified-change': [];
    'loading-change': [value: boolean];
  }>();

  const { t } = useI18n();
  const appDetailStore = useAppDetail();
  const autoScaleEnv = ref<ExtendedEnv | null>(props.currentEnv);
  const {
    config: autoScaleConfig,
    enabled: isAutoScaleEnabled,
    refresh: refreshAutoScaleConfig,
    updatePolling: updateAutoScalePolling,
  } = useGPAConfigPolling({
    active: () => !!autoScaleEnv.value && !autoScaleEnv.value.isDefault,
    appID: () => appDetailStore.appID,
    envName: () => autoScaleEnv.value?.name,
  });
  const autoScaleReplicasText = computed(() => {
    const { minReplicas, maxReplicas } = autoScaleConfig.value || {};
    if (typeof minReplicas !== 'number' || typeof maxReplicas !== 'number') return '--';
    return `${minReplicas} ～ ${maxReplicas}`;
  });

  const appSpecSection = useAppSpecSection<
    AppSpecResourcesInput,
    AppSpecResourcesOutput,
    EnvAppSpecResourcesInput,
    FieldKey
  >(
    {
      allFieldKeys: ALL_FIELD_KEYS,
      fieldAccessors: FIELD_ACCESSORS,
      formDefaults: FORM_DEFAULTS,

      // --- API 方法 ---
      fetchDefault: appID => AppSpecService.getAppDefaultAppSpecResources({ appID }),
      fetchEnvEffective: (appID, envName) => AppSpecService.getEnvEffectiveAppSpecResources({ appID, envName }),
      fetchEnvOverride: (appID, envName) =>
        AppSpecService.getEnvAppSpecResources({ appID, envName }, { interceptorErr: false }),
      saveDefault: (appID, payload) =>
        AppSpecService.setAppDefaultAppSpecResources({ appID, appSpecResources: payload }),
      saveEnv: (appID, envName, payload) =>
        AppSpecService.setEnvAppSpecResources({ appID, envName, appSpecResources: payload }),
      deleteEnv: (appID, envName) => AppSpecService.deleteEnvAppSpecResources({ appID, envName }),

      // --- 数据转换 ---
      buildDefaultPayload: model => pickFields(model, ALL_FIELD_KEYS),

      /** 每个字段独立判断：仅当该字段自身有差异且未被重置时才传值，否则传 null */
      buildEnvPayload: (model, isFieldDiff, isFieldReset) => {
        return {
          replicas:
            !isAutoScaleEnabled.value && isFieldDiff('replicas') && !isFieldReset('replicas')
              ? model.replicas
              : (null as unknown as number),
          cpuRequests:
            isFieldDiff('cpuRequests') && !isFieldReset('cpuRequests')
              ? model.cpuRequests
              : (null as unknown as string),
          cpuLimits:
            isFieldDiff('cpuLimits') && !isFieldReset('cpuLimits') ? model.cpuLimits : (null as unknown as string),
          memoryRequests:
            isFieldDiff('memoryRequests') && !isFieldReset('memoryRequests')
              ? model.memoryRequests
              : (null as unknown as string),
          memoryLimits:
            isFieldDiff('memoryLimits') && !isFieldReset('memoryLimits')
              ? model.memoryLimits
              : (null as unknown as string),
        };
      },

      /** 填充环境数据，返回被覆盖的字段列表 */
      fillEnvFormData: createFillEnvFormData<AppSpecResourcesInput, AppSpecResourcesOutput, FieldKey>(
        ALL_FIELD_KEYS,
        FORM_DEFAULTS,
      ),
    },
    emit,
  );

  const {
    saving,
    resetting,
    loading,
    isEditing,
    formModel,
    isFieldOverridden: _isFieldOverridden,
    shouldShowResetIcon: _shouldShowResetIcon,
    getFieldDefaultValue,
    handleResetField,
    handleCancelEdit,
    handleEdit,
    handleEnvChange: handleAppSpecEnvChange,
    handleResetToDefault,
    handleSave,
  } = appSpecSection;

  /** 内存值（如 '256Mi', '2Gi'）转 MiB，用于 Limits >= Requests 校验 */
  const parseMemoryToMiB = (value: string): number => {
    const match = value.match(/^(\d+(?:\.\d+)?)(Mi|Gi)$/);
    if (!match) return 0;
    const num = parseFloat(match[1]);
    return match[2] === 'Gi' ? num * 1024 : num;
  };

  /** CPU 值转核数；兼容 Kubernetes 毫核格式（如 200m = 0.2 核）。 */
  const getCoreCount = (value: string): number => {
    const milliCoreMatch = value.match(/^(\d+(?:\.\d+)?)m$/);
    return milliCoreMatch ? Number(milliCoreMatch[1]) / 1000 : Number(value);
  };

  /** 将 CLI 写入的自定义 CPU 值加入下拉框，并过滤与其等量的预设项。 */
  const getCpuOptions = (currentValue: string): string[] => {
    if (!currentValue || CPU_OPTIONS.includes(currentValue)) return CPU_OPTIONS;

    const currentCoreCount = getCoreCount(currentValue);
    return [currentValue, ...CPU_OPTIONS.filter(option => getCoreCount(option) !== currentCoreCount)];
  };

  const rules = {
    replicas: [
      {
        validator: (value: number) => isAutoScaleEnabled.value || value >= 1,
        message: t('实例数不能小于1'),
        trigger: 'blur',
      },
    ],
    cpuRequests: [
      {
        validator: () => formModel.value.cpuRequests.length > 0,
        message: `CPU Requests${t('不能为空')}`,
        trigger: 'change',
      },
    ],
    cpuLimits: [
      {
        validator: () => formModel.value.cpuLimits.length > 0,
        message: `CPU Limits${t('不能为空')}`,
        trigger: 'change',
      },
      {
        validator: () => getCoreCount(formModel.value.cpuLimits) >= getCoreCount(formModel.value.cpuRequests),
        message: t('{0} Limits 不能小于 Requests', [t('CPU')]),
        trigger: 'change',
      },
    ],
    memoryRequests: [
      {
        validator: () => formModel.value.memoryRequests.length > 0,
        message: `${t('内存')} Requests${t('不能为空')}`,
        trigger: 'change',
      },
    ],
    memoryLimits: [
      {
        validator: () => formModel.value.memoryLimits.length > 0,
        message: `${t('内存')} Limits${t('不能为空')}`,
        trigger: 'change',
      },
      {
        validator: () =>
          parseMemoryToMiB(formModel.value.memoryLimits) >= parseMemoryToMiB(formModel.value.memoryRequests),
        message: t('{0} Limits 不能小于 Requests', [t('内存')]),
        trigger: 'change',
      },
    ],
  };

  /** GPA 自动扩缩容接管实例数时，资源规格页不再展示实例数的环境覆盖标识。 */
  const shouldShowResetIcon = (field: FieldKey) =>
    field === 'replicas' && isAutoScaleEnabled.value ? false : _shouldShowResetIcon(field, props.currentEnv);

  /** 判断字段是否被当前环境覆盖；自动扩缩容开启时实例数展示 GPA 区间，不展示 AppSpec 覆盖态。 */
  const isFieldOverridden = (field: FieldKey) =>
    field === 'replicas' && isAutoScaleEnabled.value
      ? false
      : !props.currentEnv?.isDefault && _isFieldOverridden(field);

  /** 环境切换时先沿用资源规格原有加载流程，再同步 GPA 状态避免跨环境残留。 */
  async function handleEnvChange(env: ExtendedEnv) {
    const confirmed = await handleAppSpecEnvChange(env);
    if (confirmed === false) return false;

    autoScaleEnv.value = env;
    await refreshAutoScaleConfig();
    return true;
  }

  /** 保存完成后刷新 GPA 状态，确保自动扩缩容标签和区间跟随当前环境最新配置。 */
  async function onSave() {
    const success = await handleSave(props.currentEnv);
    if (success) {
      autoScaleEnv.value = props.currentEnv;
      await refreshAutoScaleConfig();
    }
  }

  /** 获取字段默认值提示；自动扩缩容开启时实例数不再展示 AppSpec 默认值。 */
  const getDefaultValueTip = (field: FieldKey) => {
    if (field === 'replicas' && isAutoScaleEnabled.value) return '';
    if (!isFieldOverridden(field)) return '';
    const defaultValue = getFieldDefaultValue(field);
    if (defaultValue === undefined) return '';

    // 针对不同字段类型格式化默认值显示
    if (field === 'cpuRequests' || field === 'cpuLimits') {
      return t('默认值：{0}', [t('{0} 核', [getCoreCount(defaultValue as string)])]);
    }
    return t('默认值：{0}', [defaultValue]);
  };

  /** 恢复当前环境资源规格为默认配置。 */
  const onResetToDefault = () => handleResetToDefault(props.currentEnv);

  watch(
    () => props.currentEnv,
    env => {
      autoScaleEnv.value = env;
    },
    { immediate: true },
  );

  watch(isAutoScaleEnabled, enabled => updateAutoScalePolling(enabled), { immediate: true });

  defineExpose({
    handleEnvChange,
    loading,
  });
</script>

<style lang="postcss" scoped>
  :deep(.bkms-content-title) {
    background-color: #eaebf0;
  }

  :deep(.bk-form-label) {
    color: #4d4f56;
  }

  :deep(.field-diff-highlight) {
    .bk-form-label {
      border-left: 3px solid #ff9c01 !important;
      padding-left: 6px !important;
    }
  }
</style>
