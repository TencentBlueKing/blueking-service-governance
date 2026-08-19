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
    :title="$t('开发模式')"
  >
    <div class="bg-[#FFF] p-[16px] !text-[12px]">
      <div class="flex items-center">
        <span class="text-[#4D4F56] mr-[16px]">
          {{ isProductionEnv ? $t('开发模式（生产环境不支持）') : $t('开发模式') }}
        </span>
        <div class="flex items-center text-[#4D4F56]">
          <Switcher
            class="mr-[6px]"
            :disabled="isProductionEnv"
            theme="primary"
            :value="isDevMode"
            @change="handleDevModeChange"
          />
          {{ $t('支持通过 bkms-cli 上传二进制的方式热更新服务，更新过程不会重启容器实例。') }}
          <Button
            text
            theme="primary"
            @click="goToTrpcDevModeDoc"
          >
            {{ $t('查看详细文档') }}
            <Share class="ml-[6px]" />
          </Button>
        </div>
      </div>
      <DevModeSteps
        v-if="isDevMode"
        class="mt-[24px] max-w-[960px]"
        :env-name="currentEnv?.name"
      />
    </div>
  </BkmsContent>

  <DevModeDialog
    v-model="showDevModeDialog"
    :env-name="currentEnv?.name"
    :is-enabling="pendingDevModeValue"
    @cancel="showDevModeDialog = false"
    @confirm="handleConfirmDevMode"
  />
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Button, Message, Switcher } from 'bkui-vue';
  import { Share } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { AppSpecService } from '~/api/modules/v1';
  import { DOC_LINKS } from '~/common/const';
  import BkmsContent from '~/components/bkms-content.vue';
  import { useAppDetail } from '~/stores/app-detail';

  import DevModeDialog from './dev-mode-dialog.vue';
  import DevModeSteps from './dev-mode-steps.vue';

  import type { ExtendedEnv } from './types';

  interface Props {
    currentEnv: ExtendedEnv | null;
  }

  const props = defineProps<Props>();

  const emit = defineEmits<{
    'env-modified-change': [];
    'loading-change': [value: boolean];
  }>();

  const { t } = useI18n();
  const appDetailStore = useAppDetail();

  const loading = ref(true);
  const isDevMode = ref(false);
  const showDevModeDialog = ref(false);
  const pendingDevModeValue = ref(false);

  const isProductionEnv = computed(() => props.currentEnv?.type === 'production');

  function goToTrpcDevModeDoc() {
    window.open(`${import.meta.env.BK_DOC_URL}${DOC_LINKS.TRPC_DEV_MODE}`, '_blank');
  }

  async function handleConfirmDevMode() {
    if (!props.currentEnv) {
      showDevModeDialog.value = false;
      return;
    }

    try {
      if (pendingDevModeValue.value) {
        // 开启开发模式
        await AppSpecService.setEnvAppSpecDevMode({
          appID: appDetailStore.appID,
          envName: props.currentEnv.name,
          appSpecDevMode: {
            enabled: true,
          },
        });
      } else {
        // 关闭开发模式
        await AppSpecService.deleteEnvAppSpecDevMode({
          appID: appDetailStore.appID,
          envName: props.currentEnv.name,
        });
      }

      isDevMode.value = pendingDevModeValue.value;
      Message({ theme: 'success', message: t('操作成功') });
      emit('env-modified-change');
    } finally {
      showDevModeDialog.value = false;
    }
  }

  function handleDevModeChange(value: boolean) {
    if (isProductionEnv.value) return;
    pendingDevModeValue.value = value;
    showDevModeDialog.value = true;
  }

  /** 环境切换：加载目标环境的开发模式配置 */
  async function handleEnvChange(env: ExtendedEnv) {
    loading.value = true;

    try {
      if (env.isDefault) {
        isDevMode.value = false;
        return true;
      }
      const envSpec = await AppSpecService.getEnvEffectiveAppSpecDevMode({
        appID: appDetailStore.appID,
        envName: env.name,
      });
      isDevMode.value = envSpec?.enabled ?? false;
    } catch {
      isDevMode.value = false;
    } finally {
      loading.value = false;
    }
    return true;
  }

  watch(loading, val => emit('loading-change', val), { immediate: true });

  defineExpose({
    handleEnvChange,
    loading,
  });
</script>

<style lang="postcss" scoped>
  :deep(.bkms-content-title) {
    background-color: #eaebf0;
  }
</style>
