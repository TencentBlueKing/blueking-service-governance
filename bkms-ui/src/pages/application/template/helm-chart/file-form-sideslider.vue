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
    :title="isEdit ? $t('编辑文件') : $t('添加文件')"
    :width="800"
    @hidden="handleHidden"
    @shown="handleShown"
  >
    <div class="px-[24px] pt-[18px]">
      <Form
        ref="formRef"
        form-type="vertical"
        :model="formData"
        :rules="formRules"
      >
        <Form.FormItem
          :label="$t('名称')"
          property="name"
          required
        >
          <Input
            v-model.trim="formData.name"
            :placeholder="$t('包含字母、数字、下划线(_)和连字符(-)，长度 1-20 之间')"
          />
        </Form.FormItem>
        <Form.FormItem
          :class="formData.type === 'overlay' ? 'mb-0' : ''"
          :label="$t('类型')"
          property="type"
          required
        >
          <Radio.Group
            v-model="formData.type"
            class="flex flex-col"
            :disabled="isEdit"
          >
            <Radio label="normal">
              {{ $t('普通') }}
              <span class="ml-[12px] text-[12px] text-[#979BA5]">
                <InfoLine class="mr-[4px] text-[14px] transform translate-y-[2px]" />
                {{ $t('标准的 Helm Chart values YAML 配置文件') }}
              </span>
            </Radio>
            <Radio
              label="overlay"
              style="margin-left: 0"
            >
              {{ $t('覆盖层') }}
              <span class="ml-[12px] text-[12px] text-[#979BA5]">
                <InfoLine class="mr-[4px] text-[14px] transform translate-y-[2px]" />
                {{ $t('覆盖配置片段，与普通 values 文件基于 Patch 算法合并生成完整配置') }}
              </span>
            </Radio>
          </Radio.Group>
        </Form.FormItem>
        <!-- 覆盖层 - 基础 values -->
        <Form.FormItem
          v-if="formData.type === 'overlay'"
          class="bg-[#F5F7FA] px-[24px] py-[16px]"
          :label="$t('基础 values')"
          property="baseAppConfigFileID"
          required
        >
          <Select
            filterable
            :model-value="formData.baseAppConfigFileID"
            @change="handleBaseFileChange"
          >
            <Select.Option
              v-for="item in baseFileOptions"
              :key="item.id"
              :name="item.name"
              :value="item.id"
            />
          </Select>
        </Form.FormItem>
        <Form.FormItem
          :class="formData.contentSourceType === 'bscp' ? 'mb-0' : ''"
          :label="$t('内容来源')"
          property="contentSourceType"
          required
        >
          <Radio.Group
            v-model="formData.contentSourceType"
            class="flex flex-col"
            :disabled="isEdit"
          >
            <Radio label="local">
              {{ $t('本地编辑') }}
              <span class="ml-[12px] text-[12px] text-[#979BA5]">
                <InfoLine class="mr-[4px] text-[14px] transform translate-y-[2px]" />
                {{ $t('文件保存后，请在页面编辑器中填写内容') }}
              </span>
            </Radio>
            <Radio
              label="bscp"
              style="margin-left: 0"
              >{{ $t('服务配置中心（BSCP）') }}</Radio
            >
          </Radio.Group>
        </Form.FormItem>
        <!-- 配置中心 -->
        <template v-if="formData.contentSourceType === 'bscp'">
          <BcspConfigSelector
            ref="bcspSelectorRef"
            v-model="formData.bscpConfig"
            :current-file="currentFile"
            :is-edit="isEdit"
            @change="handleBscpConfigChange"
            @service-not-fully-released="handleBscpServiceNotFullyReleased"
            @yaml-validate="handleBscpYamlValidate"
          />
        </template>
      </Form>
    </div>

    <template #footer>
      <div class="flex items-center">
        <span v-bk-tooltips="{ content: $t('BSCP 配置内容非合法 YAML 格式，无法保存'), disabled: !isSaveDisabled }">
          <Button
            class="mr-[8px]"
            :disabled="loading || isSaveDisabled"
            :loading="loading"
            theme="primary"
            @click="handleSubmit"
          >
            {{ isEdit ? $t('保存') : $t('确定') }}
          </Button>
        </span>
        <Button @click="handleCancel">
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Sideslider>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue';

  import { Button, Form, Input, Radio, Select, Sideslider } from 'bkui-vue';
  import { InfoLine } from 'bkui-vue/lib/icon';
  import { cloneDeep } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import {
    AppConfigFileOutputObj,
    BSCPAppConfigFileConfig,
    CreateAppConfigFileRequest,
  } from '~/@types/v1/app-config-files';
  import { BKMS_REGEX } from '~/common/const';
  import useLeaveConfirm from '~/composables/use-leave-confirm';

  import BcspConfigSelector from './bcsp-config-selector.vue';

  interface Emits {
    (e: 'update:visible', value: boolean): void;
    (e: 'submit', data: FileFormData): void;
    (e: 'cancel'): void;
  }

  // 使用 CreateAppConfigFileRequest 类型，排除 envName 字段，并让部分字段可选
  type FileFormData = Omit<CreateAppConfigFileRequest, 'appID' | 'envName'> & { bscpConfig: BSCPAppConfigFileConfig };

  interface Props {
    baseFileOptions: Array<{ id: string; name: string }>;
    currentFile?: AppConfigFileOutputObj | null;
    isEdit: boolean;
    loading?: boolean;
    visible: boolean;
    workspaceName: string;
  }

  const props = withDefaults(defineProps<Props>(), {
    loading: false,
    currentFile: null,
  });

  const emit = defineEmits<Emits>();

  const { t } = useI18n();

  const formRef = ref<InstanceType<typeof Form>>();
  const bcspSelectorRef = ref<InstanceType<typeof BcspConfigSelector>>();

  // 表单数据
  const defaultFormData: FileFormData = {
    name: '',
    type: 'normal',
    contentSourceType: 'local',
    baseAppConfigFileID: '',
    fileFormat: 'yaml',
    description: '',
    bscpConfig: {
      bizID: '',
      id: '',
      serviceID: '',
    },
  };

  const formData = reactive<FileFormData>({ ...defaultFormData });

  const { confirmBox, forceCleanDirtyTag, withPausedWatch } = useLeaveConfirm(formData);

  // 表单验证规则
  const formRules = {
    name: [
      {
        required: true,
        message: t('请输入文件名称'),
        trigger: 'blur',
      },
      {
        validator: () => BKMS_REGEX.fileNameRegex.test(formData.name || ''),
        message: t('包含字母、数字、下划线(_)和连字符(-)，长度 1-20 之间'),
        trigger: 'blur',
      },
    ],
    type: [
      {
        required: true,
        message: t('请选择文件类型'),
        trigger: 'change',
      },
    ],
    contentSourceType: [
      {
        required: true,
        message: t('请选择内容来源'),
        trigger: 'change',
      },
    ],
  };

  const visible = computed({
    get: () => props.visible,
    set: value => emit('update:visible', value),
  });

  // BSCP 配置项 YAML 合法性
  const isBscpYamlValid = ref(true);

  // BSCP 服务未全量上线状态
  const isBscpServiceNotFullyReleased = ref(false);

  // BSCP 配置项内容非合法 YAML 或服务未全量上线时禁用保存
  const isSaveDisabled = computed(() => {
    if (formData.contentSourceType !== 'bscp') return false;
    if (isBscpServiceNotFullyReleased.value) return true;
    return !isBscpYamlValid.value;
  });

  // 监听弹窗显示状态，弹窗打开且非编辑模式时重置表单
  watch(
    () => props.visible,
    newVisible => {
      if (newVisible && !props.isEdit) {
        withPausedWatch(() => {
          resetForm();
        });
      }
    },
    { immediate: true },
  );

  // 生成版本描述
  function generateDescription(): string {
    if (!props.currentFile) return '';
    const changes: string[] = [];

    if (props.currentFile.name !== formData.name) {
      changes.push(`${t('修改名称')} (${props.currentFile.name} -> ${formData.name})`);
    }

    if (formData.type === 'overlay' && props.currentFile.baseAppConfigFileID !== formData.baseAppConfigFileID) {
      const oldBase = props.baseFileOptions.find(item => item.id === props.currentFile!.baseAppConfigFileID);
      const newBase = props.baseFileOptions.find(item => item.id === formData.baseAppConfigFileID);
      const oldName = oldBase?.name || props.currentFile.baseAppConfigFileID;
      const newName = newBase?.name || formData.baseAppConfigFileID;
      changes.push(`${t('覆盖层')} (${oldName} -> ${newName})`);
    }

    return changes.join('; ');
  }

  // 基础 values 选择变化
  function handleBaseFileChange(val: string) {
    formData.baseAppConfigFileID = val;
  }

  // 侧边栏关闭前确认
  function handleBeforeClose(): Promise<boolean> {
    return confirmBox();
  }

  // BSCP 配置变化
  function handleBscpConfigChange(config: NonNullable<typeof formData.bscpConfig>) {
    formData.bscpConfig = { ...config };
  }

  // BSCP 服务未全量上线状态变化
  function handleBscpServiceNotFullyReleased(isNotFullyReleased: boolean) {
    isBscpServiceNotFullyReleased.value = isNotFullyReleased;
  }

  // BSCP 配置项 YAML 合法性变化
  function handleBscpYamlValidate(isValid: boolean) {
    isBscpYamlValid.value = isValid;
  }

  async function handleCancel() {
    if (await handleBeforeClose()) {
      emit('cancel');
    }
  }

  // 侧边栏隐藏时重置表单
  function handleHidden() {
    withPausedWatch(() => {
      resetForm();
    });
  }

  function handleShown() {
    if (props.currentFile && props.isEdit) {
      const {
        name = '',
        type = 'normal' as const,
        contentSourceType = 'local' as const,
        baseAppConfigFileID = '',
        fileFormat = 'yaml' as const,
        bscpConfig,
      } = props.currentFile;
      const fileData: FileFormData = {
        name,
        type: type as FileFormData['type'],
        contentSourceType: contentSourceType as FileFormData['contentSourceType'],
        baseAppConfigFileID,
        fileFormat: fileFormat as FileFormData['fileFormat'],
        description: '',
        bscpConfig: bscpConfig || {
          bizID: '',
          id: '',
          serviceID: '',
        },
      };
      withPausedWatch(() => {
        Object.assign(formData, fileData);
      });
    }
  }

  // 提交表单
  async function handleSubmit() {
    const mainFormValid = await formRef.value?.validate().catch(() => false);
    if (!mainFormValid) return;

    // 验证服务配置中心表单
    if (formData.contentSourceType === 'bscp') {
      const bcspValid = await bcspSelectorRef.value?.validate().catch(() => false);
      if (!bcspValid) return;
    }
    const submitData = cloneDeep(formData);

    if (props.isEdit) {
      submitData.description = generateDescription();
    }

    // 如果是本地编辑模式，不传递 bscpConfig
    if (formData.contentSourceType === 'local') {
      delete (submitData as Partial<FileFormData>).bscpConfig;
    }
    forceCleanDirtyTag(() => {
      emit('submit', submitData as FileFormData);
    });
  }

  // 重置表单数据和验证状态
  function resetForm() {
    Object.assign(formData, { ...defaultFormData });
    formRef.value?.clearValidate();
    bcspSelectorRef.value?.clearValidate();
    isBscpYamlValid.value = true;
    isBscpServiceNotFullyReleased.value = false;
  }
</script>
