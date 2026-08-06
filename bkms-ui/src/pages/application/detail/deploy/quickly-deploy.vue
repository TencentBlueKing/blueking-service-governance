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
    width="640"
    @closed="handleClosed"
  >
    <template #header>
      <DividerHeader :title="$t('部署应用')">
        <span v-if="appDetailStore.app">
          {{ appDetailStore.app }}
        </span>
        <span v-if="trpcDeployStore.curEnvItem?.name">
          {{ `${$t('环境')}: ${trpcDeployStore.curEnvItem.displayName}` }}
        </span>
      </DividerHeader>
    </template>
    <QuicklyDeployForm
      ref="deployFormRef"
      class="px-[24px] pt-[18px]"
      :effective-replicas="effectiveReplicas"
      :env-name="trpcDeployStore.curEnvItem?.name"
      :env-type="trpcDeployStore.curEnvItem?.type"
      :is-prod-env="isProdEnv"
    />
    <template #footer>
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
    </template>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { nextTick, ref, watch } from 'vue';

  import { Button, Message, Sideslider } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import QuicklyDeployForm from './quickly-deploy-form.vue';

  const isShow = defineModel<boolean>('isShow');
  const emits = defineEmits(['update']);
  const props = defineProps<{
    effectiveReplicas?: number;
    isProdEnv?: boolean;
  }>();

  const { t } = useI18n();
  const trpcDeployStore = useTrpcDeployStore();
  const appDetailStore = useAppDetail();

  const deployFormRef = ref<InstanceType<typeof QuicklyDeployForm>>();
  const confirmLoading = ref(false);

  function handleBeforeClose() {
    return deployFormRef.value?.confirmLeave() ?? Promise.resolve(true);
  }

  async function handleClose() {
    if (await handleBeforeClose()) {
      isShow.value = false;
    }
  }

  function handleClosed() {
    deployFormRef.value?.reset(1);
  }

  async function handleSubmit() {
    try {
      const envName = trpcDeployStore.curEnvItem?.name;
      if (!envName) return;

      confirmLoading.value = true;
      const submitted = await deployFormRef.value?.submit(envName);
      if (!submitted) return;

      Message({
        theme: 'success',
        message: t('操作成功'),
      });
      deployFormRef.value?.reset(1);
      isShow.value = false;
      emits('update');
    } catch (err) {
      console.error(err);
    } finally {
      confirmLoading.value = false;
    }
  }

  // 从 props 获取环境生效的部署规格
  watch(isShow, newVal => {
    if (newVal) {
      nextTick(() => {
        deployFormRef.value?.reset(props.effectiveReplicas || 1);
      });
    }
  });
</script>
