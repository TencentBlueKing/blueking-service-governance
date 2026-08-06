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
    :title="$t('网络接入')"
  >
    <template #title>
      <span>{{ $t('网络接入') }}</span>
      <EnvScopeTag :current-env="currentEnv" />
    </template>

    <div class="bg-[#FFF] p-[16px] !text-[12px] text-[#4D4F56]">
      <div class="flex items-center">
        <PopConfirm
          :cancel-text="$t('取消')"
          :confirm-config="{ theme: 'primary' }"
          :confirm-text="$t('确认')"
          :content="enabled ? $t('禁用后，需要重新部署应用才能生效。') : $t('启用后，需要重新部署应用才能生效。')"
          :popover-options="{ disabled: loading || saving }"
          :title="enabled ? $t('确认禁用 VPC 网络？') : $t('确认启用 VPC 网络？')"
          trigger="click"
          :width="280"
          @confirm="handleConfirmToggle"
        >
          <Switcher
            class="mr-[12px]"
            :disabled="loading || saving"
            theme="primary"
            :value="enabled"
          />
        </PopConfirm>
        <span class="font-bold text-[#313238] mr-[4px]">{{ $t('VPC 网络（underlay）') }}</span>
        <span>{{ $t('为容器分配 VPC IP，可以腾讯自研云（IDC）直接互通') }}</span>
        <Tag
          v-if="isCustom"
          v-bk-tooltips="{ content: defaultValueTip }"
          class="ml-[12px]"
          size="small"
          theme="warning"
        >
          {{ $t('自定义') }}
        </Tag>
      </div>

      <p class="mt-[14px]">
        <i18n-t keypath="开启后部署时自动为工作负载注入 annotation {0}。">
          <Tag>{{ NETWORK_ANNOTATION }}</Tag>
        </i18n-t>
      </p>
    </div>
  </BkmsContent>
</template>

<script lang="ts" setup>
  import { computed, ref } from 'vue';

  import { Message, PopConfirm, Switcher, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { AppSpecService } from '~/api/modules/v1';
  import BkmsContent from '~/components/bkms-content.vue';
  import { useAppDetail } from '~/stores/app-detail';

  import EnvScopeTag from './env-scope-tag.vue';

  import type { ExtendedEnv } from './types';

  interface Props {
    currentEnv: ExtendedEnv | null;
  }

  const props = defineProps<Props>();

  const emit = defineEmits<{
    'env-modified-change': [];
  }>();

  const { t } = useI18n();
  const appDetailStore = useAppDetail();

  /** 开启 VPC 网络后注入到工作负载的 Kubernetes annotation */
  const NETWORK_ANNOTATION = 'tke.cloud.tencent.com/networks: tke-route-eni';

  // VPC 网络是否已启用
  const enabled = ref(false);
  // 默认环境的 VPC 网络配置
  const defaultEnabled = ref(false);
  const defaultLoaded = ref(false);
  // 是否正在加载网络配置
  const loading = ref(true);
  // 是否正在保存（切换开关状态）
  const saving = ref(false);

  const isCustom = computed(
    () =>
      !!props.currentEnv &&
      !props.currentEnv.isDefault &&
      defaultLoaded.value &&
      enabled.value !== defaultEnabled.value,
  );
  const defaultValueTip = computed(() => (defaultEnabled.value ? t('默认值为开启') : t('默认值为关闭')));

  /** 获取默认环境配置；请求失败不影响当前环境配置的展示。 */
  async function fetchDefaultEnabled(): Promise<boolean | null> {
    try {
      const data = await AppSpecService.getAppDefaultAppSpecTkeRouteEni({
        appID: appDetailStore.appID,
      });
      return data?.enabled ?? false;
    } catch {
      return null;
    }
  }

  /** 切换 VPC 网络开关：默认环境修改默认配置，非默认环境修改对应环境配置 */
  async function handleConfirmToggle() {
    if (!props.currentEnv || loading.value || saving.value) return;

    const targetEnabled = !enabled.value;
    saving.value = true;

    try {
      if (props.currentEnv.isDefault) {
        await AppSpecService.setAppDefaultAppSpecTkeRouteEni({
          appID: appDetailStore.appID,
          enabled: targetEnabled,
        });
      } else {
        await AppSpecService.setEnvAppSpecTkeRouteEni({
          appID: appDetailStore.appID,
          envName: props.currentEnv.name,
          enabled: targetEnabled,
        });
      }

      enabled.value = targetEnabled;
      Message({ theme: 'success', message: t('操作成功') });

      if (!props.currentEnv.isDefault) {
        emit('env-modified-change');
      }
    } finally {
      saving.value = false;
    }
  }

  /** 环境切换：加载目标环境最终生效的网络接入配置。 */
  async function handleEnvChange(env: ExtendedEnv) {
    if (saving.value) return false;

    loading.value = true;
    enabled.value = false;
    defaultEnabled.value = false;
    defaultLoaded.value = false;

    try {
      // 默认环境直接获取默认配置
      if (env.isDefault) {
        const data = await AppSpecService.getAppDefaultAppSpecTkeRouteEni({
          appID: appDetailStore.appID,
        });
        enabled.value = data?.enabled ?? false;
      } else {
        // 非默认环境获取环境级配置和默认配置
        const [data, loadedDefaultEnabled] = await Promise.all([
          AppSpecService.getEnvEffectiveAppSpecTkeRouteEni({
            appID: appDetailStore.appID,
            envName: env.name,
          }),
          fetchDefaultEnabled(),
        ]);

        enabled.value = data?.enabled ?? false;
        if (loadedDefaultEnabled !== null) {
          defaultEnabled.value = loadedDefaultEnabled;
          defaultLoaded.value = true;
        }
      }
    } catch {
      enabled.value = false;
    } finally {
      loading.value = false;
    }
    return true;
  }

  function isSaving() {
    return saving.value;
  }

  defineExpose({
    handleEnvChange,
    isSaving,
    loading,
  });
</script>

<style lang="postcss" scoped>
  :deep(.bkms-content-title) {
    background-color: #eaebf0;
  }
</style>
