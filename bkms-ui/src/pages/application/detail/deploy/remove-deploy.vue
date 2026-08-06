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
    v-model:is-show="isShow"
    :width="600"
    @closed="handleClose"
    @confirm="handleRemoveDeploy"
  >
    <template #header>
      <div class="flex flex-col items-center">
        <SvgIcon
          :height="42"
          icon="bkms-icon-tishi"
          :width="42"
        />
        <span class="text-[#313238] text-[20px] leading-[32px] text-center mt-[18px]">
          {{ $t('确认移除部署') }} ?
        </span>
      </div>
    </template>
    <div class="mb-[16px] px-[16px] py-[12px] bg-[#F5F6FA] rounded-[2px]">
      <div class="font-bold mb-[6px]">
        {{ $t('将移除应用在「{0}」的全部实例，移除后', [trpcDeployStore.curEnvItem?.displayName ?? '']) }}:
      </div>
      <div>- {{ $t('所有相关的 Pod 将被删除') }}</div>
      <div>- {{ $t('应用服务将立即停止') }}</div>
      <div>- {{ $t('数据可能会丢失（ 如未持久化 ）') }}</div>
    </div>
    <i18n-t keypath="该操作不可撤销，请输入环境名称：{0} 进行确认">
      <span
        class="font-bold text-[#EA3636] px-[3px] cursor-pointer hover:bg-[#FFEBEB]"
        @click="copyText(currentEnvName)"
      >
        {{ currentEnvName }}
      </span>
    </i18n-t>
    <Form
      ref="formRef"
      class="mt-[8px]"
      form-type="vertical"
      :model="formData"
      :rules="rules"
    >
      <Form.FormItem
        property="confirmName"
        required
      >
        <Input
          v-model.trim="formData.confirmName"
          clearable
          :placeholder="$t('请输入{0}', [$t('环境名称')])"
        />
      </Form.FormItem>
    </Form>
    <template #footer>
      <div class="flex justify-center">
        <Button
          class="mr-[8px]"
          :disabled="!currentEnvName || formData.confirmName !== currentEnvName"
          :loading="confirmLoading"
          theme="danger"
          @click="handleRemoveDeploy"
        >
          {{ $t('删除') }}
        </Button>
        <Button
          :loading="confirmLoading"
          @click="handleClose"
        >
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { computed, ref } from 'vue';

  import { Button, Dialog, Form, Input } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { copyText } from '~/common/util';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import { type DeployableAppType, useDeployAPIs } from './use-deploy';

  const { t } = useI18n();

  const isShow = defineModel<boolean>('isShow');
  const emits = defineEmits(['update']);

  const trpcDeployStore = useTrpcDeployStore();
  const appDetailStore = useAppDetail();
  const confirmLoading = ref(false);
  const currentEnvName = computed(() => trpcDeployStore.curEnvItem?.name ?? '');
  const formData = ref({
    confirmName: '',
  });
  const formRef = ref<InstanceType<typeof Form> | null>(null);
  const rules = ref({
    confirmName: [
      {
        trigger: 'blur',
        required: true,
        message: t('请输入{0}', [t('环境名称')]),
      },
      {
        trigger: 'blur',
        validator: (value: string) => value === currentEnvName.value,
        message: t('{0}填写错误', [t('环境名称')]),
      },
    ],
  });

  function handleClose() {
    isShow.value = false;
    formData.value.confirmName = '';
    formRef.value?.clearValidate?.();
  }

  async function handleRemoveDeploy() {
    try {
      const result = await formRef.value
        ?.validate()
        .then(() => true)
        .catch(() => false);

      if (!result) {
        return;
      }

      // 根据应用类型获取对应的部署 API
      const deployAPIs = useDeployAPIs(appDetailStore.appType as DeployableAppType);
      // 移除部署
      confirmLoading.value = true;
      await deployAPIs.deleteDeploy({
        appID: appDetailStore.appID,
        envName: trpcDeployStore.curEnvItem?.name ?? '',
      });
      handleClose();
      emits('update');
    } catch (err) {
      console.error(err);
    } finally {
      confirmLoading.value = false;
    }
  }
</script>

<style lang="postcss" scoped>
  :deep(.bk-modal-body) {
    .bk-modal-header {
      .bk-dialog-header {
        padding-top: 48px;
      }
    }
    .bk-modal-footer {
      .bk-dialog-footer {
        border: none;
        background-color: unset;
        padding-top: 0;
        padding-bottom: 24px;
      }
    }
  }
</style>
