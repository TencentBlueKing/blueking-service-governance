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
  <div class="bg-[#F5F7FA] px-[24px] py-[16px]">
    <Loading :loading="isLoading">
      <Form
        ref="formRef"
        form-type="vertical"
        :model="formData"
        :rules="formRules"
      >
        <Form.FormItem
          :label="$t('业务')"
          property="bizID"
          required
        >
          <Select
            :disabled="isEdit"
            filterable
            :loading="bizLoading"
            :model-value="modelValue.bizID"
            @change="handleBizChange"
          >
            <Select.Option
              v-for="item in bizOptions"
              :key="item.id"
              :name="item.name"
              :value="item.id"
            />
          </Select>
        </Form.FormItem>
        <Form.FormItem
          :description="$t('仅可使用有全部实例上线版本的服务')"
          :label="$t('服务')"
          property="serviceID"
          required
        >
          <Select
            :disabled="isEdit || !modelValue.bizID"
            filterable
            :loading="serviceLoading"
            :model-value="modelValue.serviceID"
            @change="handleServiceChange"
          >
            <Select.Option
              v-for="item in serviceOptions"
              :key="item.id"
              :name="item.name"
              :value="item.id"
            />
          </Select>
        </Form.FormItem>
        <Form.FormItem
          v-if="!isServiceNotFullyReleased"
          :label="$t('配置项')"
          property="id"
          required
        >
          <Select
            :disabled="isEdit || !modelValue.serviceID"
            filterable
            :loading="configLoading"
            :model-value="modelValue.id"
            @change="handleConfigChange"
          >
            <Select.Option
              v-for="item in configOptions"
              :key="item.id"
              :name="item.name"
              :value="item.id"
            />
          </Select>
        </Form.FormItem>
        <!-- 服务未全量上线提示 使用Tag会阻止a标签的默认行为 -->
        <Alert
          v-if="isServiceNotFullyReleased"
          theme="warning"
        >
          {{ $t('服务"{0}"未上线全量版本，请先将服务上线到全部实例。', [currentServiceName]) }}
          <a
            class="text-[#3a84ff]"
            :href="bscpServiceReleaseUrl"
            target="_blank"
            >{{ $t('去上线') }}
            <i class="bkms-icon bkms-icon-jump-link text-[14px]"></i>
          </a>
        </Alert>
      </Form>
      <MsEditor
        v-if="!isServiceNotFullyReleased"
        v-model="configContent"
        class="!h-[360px]"
        :readonly="true"
        :title="$t('{0} 内容（只读，不可修改）', [configOptions.find(c => c.id === modelValue.id)?.name || ''])"
      />
    </Loading>
  </div>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Alert, Form, Loading, Message, Select } from 'bkui-vue';
  import yaml from 'js-yaml';
  import { useI18n } from 'vue-i18n';
  import { ApiServerService } from '~/api/modules/bkmsserver';

  import type { AppConfigFileOutputObj, BSCPAppConfigFileConfig } from '~/@types/v1/app-config-files';
  import type { BSCPBizOutput, BSCPConfigOutput, BSCPServiceOutput } from '~/@types/v1/bkintegrations-bscp';

  interface Emits {
    (e: 'update:modelValue', value: BSCPAppConfigFileConfig): void;
    (e: 'change', value: BSCPAppConfigFileConfig): void;
    (e: 'validate', isValid: boolean): void;
    (e: 'yaml-validate', isValid: boolean): void;
    (e: 'service-not-fully-released', isNotFullyReleased: boolean): void;
  }

  interface Props {
    currentFile?: AppConfigFileOutputObj | null;
    isEdit?: boolean;
    modelValue: BSCPAppConfigFileConfig;
  }

  const props = withDefaults(defineProps<Props>(), {
    isEdit: false,
  });

  const emit = defineEmits<Emits>();
  const { t } = useI18n();

  const formRef = ref<InstanceType<typeof Form>>();
  const formData = computed(() => props.modelValue);

  // bscp 配置内容
  const configContent = ref<string>('');
  const isConfigContentValidYaml = ref(true);

  function validateContentYaml(content: string): boolean {
    try {
      const parsed = yaml.load(content);
      return parsed !== null && parsed !== undefined && typeof parsed === 'object';
    } catch {
      return false;
    }
  }

  watch(
    configContent,
    content => {
      if (!content) {
        isConfigContentValidYaml.value = true;
      } else {
        isConfigContentValidYaml.value = validateContentYaml(content);
      }
      emit('yaml-validate', isConfigContentValidYaml.value);
    },
    { immediate: true },
  );

  const formRules = {
    bizID: [
      {
        required: true,
        message: t('请选择业务'),
        trigger: 'change',
      },
    ],
    serviceID: [
      {
        required: true,
        message: t('请选择服务'),
        trigger: 'change',
      },
    ],
    id: [
      {
        required: true,
        message: t('请选择配置项'),
        trigger: 'change',
      },
    ],
  };

  // 数据选项
  const bizOptions = ref<BSCPBizOutput[]>([]);
  const serviceOptions = ref<BSCPServiceOutput[]>([]);
  const configOptions = ref<BSCPConfigOutput[]>([]);

  // 加载状态
  const isLoading = ref(false);
  const bizLoading = ref(false);
  const serviceLoading = ref(false);
  const configLoading = ref(false);
  const configContentLoading = ref(false);

  // 服务未全量上线状态
  const isServiceNotFullyReleased = ref(false);

  // 当前服务名称
  const currentServiceName = computed(() => {
    const service = serviceOptions.value.find(s => s.id === props.modelValue.serviceID);
    return service?.name || service?.alias || props.modelValue.serviceID;
  });

  // BSCP 服务上线页面 URL
  const bscpServiceReleaseUrl = computed(() => {
    const bscpBaseUrl = import.meta.env.BK_BSCP_URL || '';
    return `${bscpBaseUrl}/space/${props.modelValue.bizID}/service/${props.modelValue.serviceID}`;
  });

  const internalValue = computed({
    get: () => props.modelValue,
    set: value => {
      emit('update:modelValue', value);
      emit('change', value);
    },
  });

  // 业务变化
  async function handleBizChange(bizID: string) {
    const newValue = {
      bizID,
      serviceID: '',
      id: '',
    };
    internalValue.value = newValue;
    isServiceNotFullyReleased.value = false;

    // 清空下级数据
    if (!bizID) {
      serviceOptions.value = [];
      configOptions.value = [];
    }
  }

  // 配置项选择变化
  function handleConfigChange(id: string) {
    const newValue = {
      ...internalValue.value,
      id,
    };
    internalValue.value = newValue;

    loadConfigContent(id);
  }

  // 服务选择变化
  async function handleServiceChange(serviceID: string) {
    const newValue = {
      ...internalValue.value,
      serviceID,
      id: '',
    };
    internalValue.value = newValue;
    isServiceNotFullyReleased.value = false;

    if (!serviceID) {
      configOptions.value = [];
    }
  }

  // 判断是否为 BSCP 服务未全量上线错误
  function isBscpNotFullyReleasedError(err: unknown): boolean {
    if (!err || typeof err !== 'object') return false;
    const errObj = err as Record<string, unknown>;
    const errorObj = errObj.error as Record<string, unknown> | undefined;
    const errorCode = String(errorObj?.code ?? errObj.code ?? '');

    return (
      errorCode === 'BSCP_NOT_FULLY_RELEASED' ||
      (errorCode === 'NOT_FOUND' && JSON.stringify(err).includes('BSCP_NOT_FULLY_RELEASED'))
    );
  }

  // 加载业务列表
  async function loadBizOptions() {
    bizLoading.value = true;
    try {
      const res = await ApiServerService.ListBSCPBizs({});
      bizOptions.value = res;
    } catch {
      bizOptions.value = [];
    } finally {
      bizLoading.value = false;
    }
  }

  // 加载配置项内容
  async function loadConfigContent(id: number | string) {
    configContentLoading.value = true;
    try {
      const res = await ApiServerService.GetBSCPConfig({
        bizID: internalValue.value.bizID,
        serviceID: internalValue.value.serviceID,
        configID: String(id),
      });
      configContent.value = res?.content || '';
    } catch {
      configContent.value = '';
    } finally {
      configContentLoading.value = false;
    }
  }

  // 加载配置项列表
  async function loadConfigOptions() {
    configLoading.value = true;
    isServiceNotFullyReleased.value = false;
    try {
      // interceptorErr: false 拦截错误弹窗
      const res = await ApiServerService.ListBSCPConfigs(
        {
          bizID: internalValue.value.bizID,
          serviceID: internalValue.value.serviceID,
        },
        { interceptorErr: false, needStatus: true },
      );
      configOptions.value = res;
    } catch (err: unknown) {
      configOptions.value = [];
      const apiError = err as Record<string, Record<string, unknown>>;
      if (isBscpNotFullyReleasedError(err)) {
        isServiceNotFullyReleased.value = true;
        // BSCP_NOT_FULLY_RELEASED 不需要弹窗提示，其他异常需要弹窗提示
        return;
      }
      // 兼容其他弹窗提示
      Message({
        theme: 'error',
        actions: [
          {
            id: 'assistant',
            disabled: true, // 不显示助手
          },
        ],
        message: {
          code: apiError.status,
          overview: apiError?.error?.message || window.i18n.t('请求异常'),
          suggestion: '',
          type: 'json',
          details: `${JSON.stringify(apiError?.error || {}, null, 2)}`,
        },
      });
    } finally {
      configLoading.value = false;
    }
  }

  // 加载服务列表
  async function loadServiceOptions() {
    serviceLoading.value = true;
    try {
      const res = await ApiServerService.ListBSCPServices({ bizID: internalValue.value.bizID });
      serviceOptions.value = res;
    } catch {
      serviceOptions.value = [];
    } finally {
      serviceLoading.value = false;
    }
  }

  // 监听业务变化，加载服务列表
  watch(
    () => props.modelValue.bizID,
    (newBizID, oldBizID) => {
      if (newBizID && newBizID !== oldBizID) {
        loadServiceOptions();
      }
    },
  );

  // 监听服务变化，加载配置项列表
  watch(
    () => props.modelValue.serviceID,
    (newServiceID, oldServiceID) => {
      if (newServiceID && newServiceID !== oldServiceID) {
        loadConfigOptions();
      }
    },
  );

  // 监听"服务未全量上线"状态变化，通知父组件
  watch(isServiceNotFullyReleased, val => {
    emit('service-not-fully-released', val);
  });

  // 清除验证
  function clearValidate() {
    formRef.value?.clearValidate();
  }

  async function initializeData() {
    if (!props.isEdit) {
      await loadBizOptions();
      return;
    }
    isLoading.value = true;
    // 编辑：初始化数据
    await Promise.all([
      loadBizOptions(),
      loadServiceOptions(),
      loadConfigOptions(),
      loadConfigContent(props.currentFile?.bscpConfig?.id || ''),
    ]);
    isLoading.value = false;
  }

  // 表单验证方法
  async function validate(): Promise<boolean> {
    try {
      const isValid = await formRef.value?.validate();
      emit('validate', !!isValid);
      return !!isValid;
    } catch {
      emit('validate', false);
      return false;
    }
  }

  initializeData();

  defineExpose({
    validate,
    clearValidate,
  });
</script>
