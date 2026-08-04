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
    render-directive="if"
    width="640"
    @closed="handleClosed"
  >
    <template #header>
      <DividerHeader>
        <template #title>
          <span class="text-[16px]">{{ $t('特性部署') }}</span>
        </template>
        <span v-if="appDetailStore.app">
          {{ appDetailStore.app }}
        </span>
        <span v-if="sourceEnv?.displayName || sourceEnv?.name">
          {{ `${$t('环境')}: ${sourceEnv.displayName || sourceEnv.name}` }}
        </span>
      </DividerHeader>
    </template>
    <div class="px-[24px] pt-[24px]">
      <Alert
        class="mb-[16px]"
        theme="info"
        :title="$t('基于来源环境快速创建一个隔离的特性环境，适合灰度测试等场景。特性环境仅对当前应用可见。')"
      />
      <Form
        ref="formRef"
        form-type="vertical"
        :model="formModel"
        :rules="rules"
      >
        <Form.FormItem
          :label="$t('来源环境')"
          property="sourceEnvID"
          required
        >
          <Select
            v-model="formModel.sourceEnvID"
            :clearable="false"
            :disabled="envListLoading"
            :filterable="false"
          >
            <Select.Option
              v-for="env in availableEnvList"
              :id="env.id"
              :key="env.id"
              :name="env.displayName || env.name"
              :value="env.id"
            >
              <span class="inline-flex items-center gap-[8px]">
                <span>{{ env.displayName || env.name }}</span>
                <Tag
                  v-if="env.type && envTypeMap[env.type]"
                  :class="envTypeTagClassMap[env.type]"
                  size="small"
                >
                  {{ envTypeMap[env.type]?.name || '' }}
                </Tag>
              </span>
            </Select.Option>
          </Select>
          <div class="mt-[8px] text-[12px] leading-[20px] text-[#979BA5]">
            {{ $t('特性环境会继承来源环境的配置，后续也可以在“应用配置”页面修改特性环境配置。') }}
          </div>
        </Form.FormItem>

        <Form.FormItem
          :label="$t('特性环境展示名称')"
          property="displayName"
          required
        >
          <Input
            v-model.trim="formModel.displayName"
            :placeholder="$t('请输入 1-32 字符的环境展示名')"
          />
        </Form.FormItem>
      </Form>
      <QuicklyDeployForm
        ref="deployFormRef"
        class="mt-[24px]"
        :effective-replicas="effectiveReplicas"
        :env-name="sourceEnv?.name"
        :env-type="sourceEnv?.type"
      />
      <div class="mt-[32px]">
        <Button
          class="mr-[10px]"
          :loading="confirmLoading"
          theme="primary"
          @click="handleSubmit"
        >
          {{ $t('部署') }}
        </Button>
        <Button
          :loading="confirmLoading"
          @click="handleClose"
        >
          {{ $t('取消') }}
        </Button>
      </div>
    </div>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, nextTick, reactive, ref, watch } from 'vue';

  import { Alert, Button, Form, Input, Message, Select, Sideslider, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { EnvOutput } from '~/@types/v1/env';
  import { EnvService } from '~/api/modules/v1';
  import { BKMS_REGEX } from '~/common/const';
  import DividerHeader from '~/components/divider-header.vue';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import QuicklyDeployForm from './quickly-deploy-form.vue';

  const isShow = defineModel<boolean>('isShow');
  defineProps<{
    effectiveReplicas?: number;
  }>();
  const emits = defineEmits<{
    (e: 'env-created', env?: EnvOutput): void;
    (e: 'update', env?: EnvOutput): void;
  }>();

  const { t } = useI18n();
  const appDetailStore = useAppDetail();
  const trpcDeployStore = useTrpcDeployStore();

  // 表单相关状态
  const formRef = ref();
  const deployFormRef = ref<InstanceType<typeof QuicklyDeployForm>>();
  const confirmLoading = ref(false);
  const envListLoading = ref(false);
  const envList = ref<EnvOutput[]>([]);
  // 已创建的特性环境缓存，避免重复创建
  const createdFeatureEnv = ref<EnvOutput>();
  const createdFeatureEnvKey = ref('');
  const formModel = reactive<{
    displayName: string;
    sourceEnvID: string;
  }>({
    displayName: '',
    sourceEnvID: '',
  });

  // 可选环境列表（过滤掉未就绪和非标准环境）
  const availableEnvList = computed(() =>
    envList.value.filter(env => env.id && env.status !== 'NotReady' && (env.kind || 'standard') === 'standard'),
  );
  // 当前选中的来源环境
  const sourceEnv = computed(() => availableEnvList.value.find(env => env.id === formModel.sourceEnvID));
  // 用于判断是否已创建过相同参数的特性环境
  const featureFormKey = computed(() => `${formModel.sourceEnvID}:${formModel.displayName.trim()}`);
  const {
    confirmBox: confirmFeatureFormLeave,
    forceCleanDirtyTag: markFeatureFormClean,
    withPausedWatch: withFeatureFormPausedWatch,
  } = useLeaveConfirm(formModel);

  // 表单校验规则
  const rules = {
    sourceEnvID: [
      {
        message: t('请选择来源环境'),
        trigger: 'change',
        validator: (value: string) => !!value,
      },
    ],
    displayName: [
      {
        validator: () => BKMS_REGEX.envDisplayNameRegex.test(formModel.displayName || ''),
        message: t('请输入 1-32 字符的环境展示名'),
        trigger: 'blur',
      },
    ],
  };

  // 关闭前确认：检查主表单和部署表单是否有未保存内容
  async function handleBeforeClose() {
    if (!(await confirmFeatureFormLeave())) return false;
    return deployFormRef.value?.confirmLeave() ?? Promise.resolve(true);
  }

  async function handleClose() {
    if (await handleBeforeClose()) {
      isShow.value = false;
    }
  }

  // 面板关闭后重置表单
  function handleClosed() {
    formRef.value?.clearValidate?.();
    handleInitForm();
  }

  // 获取当前应用的环境列表
  async function handleGetEnvList() {
    if (!appDetailStore.appID) {
      envList.value = [];
      return;
    }
    envListLoading.value = true;
    try {
      const list = await EnvService.listAppEnvs({
        appID: appDetailStore.appID,
      });
      // 未就绪的环境排到列表末尾
      envList.value = list.sort((a, b) => {
        const aDisabled = a.status === 'NotReady' ? 1 : 0;
        const bDisabled = b.status === 'NotReady' ? 1 : 0;
        return aDisabled - bDisabled;
      });
    } catch (err) {
      console.error(err);
      envList.value = [];
    } finally {
      envListLoading.value = false;
    }
  }

  // 初始化表单：默认选中当前环境或第一个可用环境
  function handleInitForm() {
    const currentEnv = availableEnvList.value.find(env => env.name === trpcDeployStore.curEnvItem?.name);
    withFeatureFormPausedWatch(() => {
      formModel.sourceEnvID = currentEnv?.id || availableEnvList.value[0]?.id || '';
      formModel.displayName = '';
    });
    createdFeatureEnv.value = undefined;
    createdFeatureEnvKey.value = '';
    markFeatureFormClean(() => nextTick(resetDeployForm));
  }

  // 提交部署：创建特性环境并执行部署
  async function handleSubmit() {
    // 校验主表单
    const valid = await formRef.value?.validate().catch(() => false);
    if (!valid || !appDetailStore.appID) return;
    // 校验部署表单
    const deployFormValid = await deployFormRef.value?.validate();
    if (!deployFormValid) return;

    confirmLoading.value = true;
    let createdEnv: EnvOutput | undefined;
    let hasEmittedCreated = false;
    try {
      // 复用已创建的特性环境，避免重复请求
      if (createdFeatureEnv.value && createdFeatureEnvKey.value === featureFormKey.value) {
        createdEnv = createdFeatureEnv.value;
      } else {
        createdEnv = await EnvService.createFeatureEnv({
          appID: appDetailStore.appID,
          displayName: formModel.displayName,
          sourceEnvID: formModel.sourceEnvID,
        });
        createdFeatureEnv.value = createdEnv;
        createdFeatureEnvKey.value = featureFormKey.value;
        emits('env-created', createdEnv);
        hasEmittedCreated = true;
      }

      if (!createdEnv?.name) {
        Message({
          theme: 'error',
          message: t('无法获取特性环境标识'),
        });
        return;
      }

      // 提交部署表单
      const submitted = await deployFormRef.value?.submit(createdEnv.name);
      if (!submitted) return;

      Message({
        theme: 'success',
        message: t('操作成功'),
      });
      markFeatureFormClean();
      isShow.value = false;
      emits('update', createdEnv);
    } catch (err) {
      console.error(err);
      // 即使后续步骤失败，也通知外层已创建的环境信息
      if (createdEnv && !hasEmittedCreated) {
        emits('env-created', createdEnv);
      }
    } finally {
      confirmLoading.value = false;
    }
  }

  // 重置部署子表单
  async function resetDeployForm() {
    deployFormRef.value?.reset();
  }

  // 面板打开时加载环境列表并初始化表单
  watch(isShow, async val => {
    if (val) {
      await handleGetEnvList();
      handleInitForm();
    }
  });

  // 来源环境变化时重置部署表单
  watch(
    () => sourceEnv.value?.name,
    () => {
      if (isShow.value) {
        nextTick(resetDeployForm);
      }
    },
  );
</script>
