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
  <Dialog
    v-model:is-show="visible"
    quick-close
    render-directive="if"
    :title="isEdit ? $t('编辑环境变量') : $t('新增环境变量')"
    :width="560"
  >
    <div class="px-1">
      <Form
        ref="formRef"
        form-type="vertical"
        :model="formData"
        :rules="formRules"
      >
        <Form.FormItem
          label="Key"
          property="key"
          required
        >
          <Input
            v-model.trim="formData.key"
            :maxlength="128"
            :placeholder="$t('字母或下划线开头，仅允许字母、数字、下划线')"
          />
        </Form.FormItem>

        <Form.FormItem
          :description="$t('敏感环境变量的值将在页面上以脱敏形式展示，只有应用进程内能够获取到这些变量的明文值。')"
          :label="$t('是否敏感')"
        >
          <Switcher
            v-model="formData.isSensitive"
            :disabled="isSensitiveSwitcherDisabled"
            theme="primary"
            @change="handleSensitiveChange"
          />
        </Form.FormItem>

        <Form.FormItem
          label="Value"
          property="value"
        >
          <SensitiveValueInput
            v-if="formData.isSensitive"
            v-model="formData.value"
            v-model:modified="sensitiveValueModified"
            :mode="sensitiveValueInputMode"
            @enter="handleSubmit"
            @reset="clearValueValidate"
          />
          <Input
            v-else
            v-model.trim="formData.value"
            clearable
            :maxlength="4096"
            @enter="handleSubmit"
          />
        </Form.FormItem>

        <!-- 作用域类型（仅新建时可选） -->
        <template v-if="!isEdit">
          <Form.FormItem
            :label="$t('生效环境类型')"
            property="scopeType"
            required
          >
            <Radio.Group
              v-model="formData.scopeType"
              class="flex"
            >
              <Radio.Button
                class="flex-1"
                label="workspace"
                >{{ $t('所有') }}</Radio.Button
              >
              <Radio.Button
                class="flex-1"
                label="envType"
                >{{ $t('指定环境类型') }}</Radio.Button
              >
            </Radio.Group>
            <div
              v-if="formData.scopeType === 'envType'"
              class="mt-[12px] px-[10px] py-[6px] bg-[#F5F7FA] rounded-[2px]"
            >
              <Radio.Group
                v-model="formData.scopeValue"
                class="flex items-center"
              >
                <Radio label="development">{{ $t('开发') }}</Radio>
                <Radio label="test">{{ $t('测试') }}</Radio>
                <Radio label="production">{{ $t('生产') }}</Radio>
              </Radio.Group>
            </div>
          </Form.FormItem>
        </template>

        <!-- 当前作用域展示（编辑模式） -->
        <Form.FormItem
          v-if="isEdit"
          :label="$t('生效环境类型')"
        >
          <Tag :type="scopeDisplay.tagType">{{ scopeDisplay.label }}</Tag>
        </Form.FormItem>

        <!-- 描述 -->
        <Form.FormItem
          :label="$t('描述')"
          property="description"
        >
          <Input
            v-model="formData.description"
            :maxlength="256"
            :placeholder="$t('变量用途说明（非必填）')"
            :rows="2"
            type="textarea"
          />
        </Form.FormItem>
      </Form>
    </div>

    <template #footer>
      <Button
        :loading="submitting"
        theme="primary"
        @click="handleSubmit"
      >
        {{ $t('确定') }}
      </Button>
      <Button
        class="ml-[8px]"
        @click="visible = false"
      >
        {{ $t('取消') }}
      </Button>
    </template>
  </Dialog>
</template>

<script setup lang="ts">
  import { computed, nextTick, reactive, ref, watch } from 'vue';

  import { Button, Dialog, Form, Input, Message, Radio, Switcher, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { ScopedEnvVarOutputObj } from '~/@types/v1/envvars';
  import { EnvvarsService } from '~/api/modules/v1';
  import { BKMS_REGEX } from '~/common/const';
  import SensitiveValueInput from '~/components/editable-variable-table/sensitive-value-input.vue';
  import { getScopeDisplay } from '~/composables/use-scope-display';

  interface Emits {
    (e: 'update:isShow', value: boolean): void;
    (e: 'success'): void;
  }

  interface Props {
    editData?: null | ScopedEnvVarOutputObj;
    isShow: boolean;
    workspaceId: string;
  }

  const props = withDefaults(defineProps<Props>(), {
    editData: null,
  });

  const emit = defineEmits<Emits>();

  const { t } = useI18n();

  interface FormData {
    description: string;
    isSensitive: boolean;
    key: string;
    scopeType: 'envType' | 'workspace';
    scopeValue: string;
    value: string;
  }

  function createDefaultForm(): FormData {
    return {
      key: '',
      value: '',
      isSensitive: false,
      scopeType: 'workspace',
      scopeValue: 'development',
      description: '',
    };
  }

  const visible = computed({
    get: () => props.isShow,
    set: (val: boolean) => emit('update:isShow', val),
  });

  const isEdit = computed(() => !!props.editData);

  const formData = reactive<FormData>(createDefaultForm());
  const formRef = ref<InstanceType<typeof Form>>();
  const submitting = ref(false);
  // 标记已有敏感值是否被重新输入，用于决定更新时是否提交 value。
  const sensitiveValueModified = ref(false);

  /** 编辑模式下的作用域展示 */
  const scopeDisplay = computed(() => {
    if (!props.editData) return getScopeDisplay('workspace', '');
    return getScopeDisplay(props.editData?.scopeType || '', props.editData?.scopeValue || '');
  });

  /** 表单校验规则 */
  const formRules = computed(() => ({
    key: [
      { pattern: BKMS_REGEX.envVarKeyRegex, message: t('字母或下划线开头，仅允许字母、数字、下划线'), trigger: 'blur' },
    ],
  }));

  const isOriginalSensitive = computed(() => isEdit.value && props.editData?.isSensitive === true);
  const isSensitiveSwitcherDisabled = computed(() => isOriginalSensitive.value);
  const isSensitiveValueUnchanged = computed(
    () => isOriginalSensitive.value && formData.isSensitive && !sensitiveValueModified.value,
  );
  const sensitiveValueInputMode = computed<'create' | 'edit'>(() => (isOriginalSensitive.value ? 'edit' : 'create'));

  function clearValueValidate() {
    nextTick(() => {
      formRef.value?.clearValidate('value');
    });
  }

  /** 回填表单（编辑模式）或重置（新建模式） */
  function fillForm() {
    if (!props.editData) {
      resetForm();
      return;
    }

    Object.assign(formData, {
      key: props.editData.key,
      value: props.editData.isSensitive ? '' : props.editData.value || '',
      isSensitive: !!props.editData.isSensitive,
      scopeType: props.editData.scopeType,
      scopeValue: props.editData.scopeValue || '',
      description: props.editData.description || '',
    });
    sensitiveValueModified.value = false;
    clearValueValidate();
  }

  /** 创建环境变量 */
  async function handleCreate() {
    const params = {
      workspaceID: props.workspaceId,
      scopeType: formData.scopeType,
      scopeValue: formData.scopeType === 'workspace' ? '' : formData.scopeValue,
      key: formData.key.trim(),
      value: formData.value,
      isSensitive: formData.isSensitive,
      description: formData.description,
    };

    const success = await EnvvarsService.createScopedEnvVar(params, { validateCode: false })
      .then(() => true)
      .catch(() => false);

    if (success) {
      Message({ delay: 2000, message: t('创建成功'), theme: 'success' });
      emit('success');
      visible.value = false;
    }
  }

  function handleSensitiveChange() {
    if (!isEdit.value) {
      sensitiveValueModified.value = true;
      return;
    }
    sensitiveValueModified.value = formData.isSensitive && props.editData?.isSensitive === false;
  }

  /** 提交表单 */
  async function handleSubmit() {
    const valid = await formRef.value?.validate().catch(() => false);
    if (!valid) return;

    submitting.value = true;
    try {
      if (isEdit.value) {
        await handleUpdate();
      } else {
        await handleCreate();
      }
    } finally {
      submitting.value = false;
    }
  }

  /** 更新环境变量 */
  async function handleUpdate() {
    if (!props.editData) return;

    const params: {
      description: string;
      isSensitive: boolean;
      key: string;
      scopedEnvVarID: string;
      value?: string;
      workspaceID: string;
    } = {
      workspaceID: props.workspaceId,
      scopedEnvVarID: props.editData?.id || '',
      key: formData.key.trim(),
      value: formData.value,
      isSensitive: formData.isSensitive,
      description: formData.description,
    };
    if (isSensitiveValueUnchanged.value) {
      // 未修改已有敏感值时不传 value，避免覆盖服务端保存的密文。
      delete params.value;
    }

    const success = await EnvvarsService.updateScopedEnvVar(params, { validateCode: false })
      .then(() => true)
      .catch(() => false);

    if (success) {
      Message({ delay: 2000, message: t('更新成功'), theme: 'success' });
      emit('success');
      visible.value = false;
    }
  }

  /** 重置表单 */
  function resetForm() {
    Object.assign(formData, createDefaultForm());
    sensitiveValueModified.value = true;
    formRef.value?.clearValidate();
  }

  watch(
    () => props.isShow,
    val => {
      if (val) fillForm();
    },
  );
</script>
