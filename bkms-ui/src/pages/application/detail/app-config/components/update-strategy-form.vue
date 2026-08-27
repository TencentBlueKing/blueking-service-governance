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
    :collapsible="true"
    :editing="isEditing"
    :show-edit-icon="!isEditing"
    @edit="handleEdit"
  >
    <template #title>
      <div class="flex items-center">
        <div class="text-[14px]">{{ $t('更新策略') }}</div>
        <EnvScopeTag :current-env="currentEnv" />
      </div>
    </template>
    <div class="bg-[#FFF] p-[16px]">
      <Form
        :ref="appSpecSection.formRef"
        form-type="vertical"
        :model="formModel"
        :rules="rules"
      >
        <!-- 查看态：以只读方式展示当前配置值 -->
        <template v-if="!isEditing">
          <div class="grid grid-cols-2 gap-[12px] gap-y-2">
            <FieldItem :container-height="20">
              <template #field>
                <ModifiedFieldLabel
                  :label="`${$t('最大超出数量')} (maxSurge)`"
                  :modified="isFieldOverridden('maxSurge')"
                />
              </template>
              <template #value>
                <span
                  v-bk-tooltips="{
                    content: getDefaultValueTip('maxSurge'),
                    disabled: !getDefaultValueTip('maxSurge'),
                  }"
                  :class="[FIELD_VALUE_BASE_CLASS, { [FIELD_VALUE_MODIFIED_CLASS]: getDefaultValueTip('maxSurge') }]"
                >
                  {{ formModel.maxSurge || '--' }}
                </span>
              </template>
            </FieldItem>
            <FieldItem
              :container-height="20"
              :field-width="210"
            >
              <template #field>
                <ModifiedFieldLabel
                  :label="`${$t('最大不可用数量')} (maxUnavailable)`"
                  :modified="isFieldOverridden('maxUnavailable')"
                />
              </template>
              <template #value>
                <span
                  v-bk-tooltips="{
                    content: getDefaultValueTip('maxUnavailable'),
                    disabled: !getDefaultValueTip('maxUnavailable'),
                  }"
                  :class="[
                    FIELD_VALUE_BASE_CLASS,
                    { [FIELD_VALUE_MODIFIED_CLASS]: getDefaultValueTip('maxUnavailable') },
                  ]"
                >
                  {{ formModel.maxUnavailable || '--' }}
                </span>
              </template>
            </FieldItem>
          </div>
        </template>
        <!-- 编辑态：可输入并保存配置 -->
        <div v-else>
          <div class="flex gap-x-[24px]">
            <!-- maxSurge：滚动更新期间允许超出原定副本数额外创建的新 Pod 最大数量 -->
            <Form.FormItem
              :class="['!w-[400px] !mb-0 relative', { 'field-diff-highlight': shouldShowResetIcon('maxSurge') }]"
              :description="$t('在滚动更新期间,允许超出原定副本数额外创建的新 Pod 的最大数量')"
              :label="`${$t('最大超出数量')} (maxSurge)`"
              property="maxSurge"
              required
            >
              <Input v-model.trim="formModel.maxSurge" />
              <FieldResetHint
                :show-reset="shouldShowResetIcon('maxSurge')"
                :tip="getDefaultValueTip('maxSurge')"
                @reset="handleResetField('maxSurge')"
              />
            </Form.FormItem>
            <!-- maxUnavailable：滚动更新期间允许同时不可用的 Pod 最大数量 -->
            <Form.FormItem
              :class="['!w-[400px] !mb-0 relative', { 'field-diff-highlight': shouldShowResetIcon('maxUnavailable') }]"
              :description="$t('在滚动更新期间，允许同时不可用的 Pod 的最大数量')"
              :label="`${$t('最大不可用数量')} (maxUnavailable)`"
              property="maxUnavailable"
              required
            >
              <Input v-model.trim="formModel.maxUnavailable" />
              <FieldResetHint
                :show-reset="shouldShowResetIcon('maxUnavailable')"
                :tip="getDefaultValueTip('maxUnavailable')"
                @reset="handleResetField('maxUnavailable')"
              />
            </Form.FormItem>
          </div>

          <!-- 操作按钮区域 -->
          <div class="!mb-0 mt-[16px] flex items-center">
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
  import { Button, Form, Input } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import {
    AppSpecUpdateStrategyInput,
    AppSpecUpdateStrategyOutput,
    EnvAppSpecUpdateStrategyInput,
  } from '~/@types/v1/app-spec';
  import { AppSpecService } from '~/api/modules/v1';
  import { BKMS_REGEX } from '~/common/const';
  import BkmsContent from '~/components/bkms-content.vue';
  import FieldItem from '~/components/field-item.vue';

  import EnvScopeTag from './env-scope-tag.vue';
  import FieldResetHint from './field-reset-hint.vue';
  import ModifiedFieldLabel from './modified-field-label.vue';
  import { createFieldAccessors, createFillEnvFormData, useAppSpecSection } from './use-app-spec-section';

  import type { ExtendedEnv } from './types';

  // 查看态字段值样式常量
  const FIELD_VALUE_BASE_CLASS = 'text-[12px] text-[#313238]';
  const FIELD_VALUE_MODIFIED_CLASS = 'border-b border-dashed border-b-[#313238]';

  type FieldKey = 'maxSurge' | 'maxUnavailable';

  interface Props {
    currentEnv: ExtendedEnv | null;
  }

  const FORM_DEFAULTS: AppSpecUpdateStrategyInput = {
    maxUnavailable: '25%',
    maxSurge: '25%',
  };

  const ALL_FIELD_KEYS: FieldKey[] = ['maxUnavailable', 'maxSurge'];

  const FIELD_ACCESSORS = createFieldAccessors<AppSpecUpdateStrategyInput, FieldKey>(ALL_FIELD_KEYS, FORM_DEFAULTS);

  const props = defineProps<Props>();

  const emit = defineEmits<{
    'env-modified-change': [];
    'loading-change': [value: boolean];
  }>();

  const { t } = useI18n();

  const appSpecSection = useAppSpecSection<
    AppSpecUpdateStrategyInput,
    AppSpecUpdateStrategyOutput,
    EnvAppSpecUpdateStrategyInput,
    FieldKey
  >(
    {
      allFieldKeys: ALL_FIELD_KEYS,
      fieldAccessors: FIELD_ACCESSORS,
      formDefaults: FORM_DEFAULTS,

      // --- API 方法 ---
      fetchDefault: appID => AppSpecService.getAppDefaultAppSpecUpdateStrategy({ appID }),
      fetchEnvEffective: (appID, envName) => AppSpecService.getEnvEffectiveAppSpecUpdateStrategy({ appID, envName }),
      fetchEnvOverride: (appID, envName) =>
        AppSpecService.getEnvAppSpecUpdateStrategy({ appID, envName }, { interceptorErr: false }),
      saveDefault: (appID, payload) =>
        AppSpecService.setAppDefaultAppSpecUpdateStrategy({ appID, appSpecUpdateStrategy: payload }),
      saveEnv: (appID, envName, payload) =>
        AppSpecService.setEnvAppSpecUpdateStrategy({ appID, envName, appSpecUpdateStrategy: payload }),
      deleteEnv: (appID, envName) => AppSpecService.deleteEnvAppSpecUpdateStrategy({ appID, envName }),

      // --- 数据转换 ---
      buildDefaultPayload: model => ({
        maxUnavailable: String(model.maxUnavailable),
        maxSurge: String(model.maxSurge),
      }),

      /** 与默认值相同的字段设为 null（继承默认值），被重置的字段也传 null */
      buildEnvPayload: (model, isFieldDiff, isFieldReset) => ({
        maxUnavailable:
          isFieldDiff('maxUnavailable') && !isFieldReset('maxUnavailable')
            ? String(model.maxUnavailable)
            : (null as unknown as string),
        maxSurge:
          isFieldDiff('maxSurge') && !isFieldReset('maxSurge') ? String(model.maxSurge) : (null as unknown as string),
      }),

      /** 填充环境数据，返回被覆盖的字段列表 */
      fillEnvFormData: createFillEnvFormData<AppSpecUpdateStrategyInput, AppSpecUpdateStrategyOutput, FieldKey>(
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
    handleEnvChange,
    handleResetToDefault,
    handleSave,
  } = appSpecSection;

  /** maxUnavailable / maxSurge 校验：非负整数或百分比 */
  const rules = {
    maxUnavailable: [
      {
        validator: (value: string) => BKMS_REGEX.percentOrNonNegativeIntegerRegex.test(value),
        message: t('请输入非负整数或百分比'),
        trigger: 'blur',
      },
    ],
    maxSurge: [
      {
        validator: (value: string) => BKMS_REGEX.percentOrNonNegativeIntegerRegex.test(value),
        message: t('请输入非负整数或百分比'),
        trigger: 'blur',
      },
    ],
  };

  const shouldShowResetIcon = (field: FieldKey) => _shouldShowResetIcon(field, props.currentEnv);
  const isFieldOverridden = (field: FieldKey) => !props.currentEnv?.isDefault && _isFieldOverridden(field);
  const getDefaultValueTip = (field: FieldKey) => {
    if (!isFieldOverridden(field)) return '';
    const defaultValue = getFieldDefaultValue(field);
    return defaultValue !== undefined ? t('默认值：{0}', [defaultValue]) : '';
  };
  const onSave = () => handleSave(props.currentEnv);
  const onResetToDefault = () => handleResetToDefault(props.currentEnv);

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
