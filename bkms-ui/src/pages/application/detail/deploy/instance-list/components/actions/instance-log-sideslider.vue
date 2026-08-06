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
    class="instance-log-sideslider"
    :width="1200"
    @closed="handleClosed"
  >
    <template #header>
      <div class="flex items-center">
        <span>{{ $t('实例日志') }}</span>
        <Divider
          class="mx-[12px]"
          direction="vertical"
        />
        <span class="text-[#979BA5] text-[14px] leading-[24px]">{{ instance?.id || '' }}</span>
      </div>
    </template>
    <template #default>
      <InstanceLog
        is-custom-modules
        :loading="loading"
        :logs="logs"
        :modules="modules"
        @download="handleDownloadLog"
        @refresh="fetchLogs"
        @update:active-module="handleUpdateActiveModule"
      />
    </template>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { ref, watch } from 'vue';

  import { Divider, Sideslider } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { type AppInstanceOutputObj, type LogEntryOutputObj } from '~/@types/v1/instance';
  import { InstanceService } from '~/api/modules/v1';
  import { useAppDetail } from '~/stores/app-detail';

  import { downloadInstanceLog, RECENT_RESTART_LOG } from '../../../use-deploy';
  import InstanceLog from '../instance-log.vue';

  import type { IModule } from '../instance-log.vue';

  const props = defineProps<{
    envName: string;
    instance: AppInstanceOutputObj | null;
  }>();

  const visible = defineModel<boolean>('visible', { default: false });

  const { t } = useI18n();
  const appDetailStore = useAppDetail();

  const loading = ref(false);
  const logs = ref<LogEntryOutputObj[]>([]);
  const activeModule = ref('realtime');

  const modules = ref<IModule[]>([
    {
      text: t('实时日志'),
      value: 'realtime',
      useDefaultContent: true,
    },
    {
      text: t('最近一次重启日志'),
      value: RECENT_RESTART_LOG,
      useDefaultContent: true,
    },
  ]);

  async function fetchLogs() {
    if (!props.instance || !props.envName) return;

    loading.value = true;
    try {
      logs.value = await InstanceService.listAppInstanceLogs({
        appID: appDetailStore.appID,
        envName: props.envName,
        instanceID: props.instance?.id ?? '',
        tailLines: 2000,
        previous: activeModule.value === RECENT_RESTART_LOG,
      });
    } catch (_) {
      logs.value = [];
    } finally {
      loading.value = false;
    }
  }

  function handleClosed() {
    logs.value = [];
    activeModule.value = 'realtime';
  }

  function handleDownloadLog() {
    if (!props.instance || !props.envName) return;

    downloadInstanceLog({
      appID: appDetailStore.appID,
      envName: props.envName,
      instanceID: props.instance?.id ?? '',
      previous: activeModule.value === RECENT_RESTART_LOG,
    });
  }

  function handleUpdateActiveModule(value: string) {
    if (activeModule.value === value) return;
    activeModule.value = value;
    fetchLogs();
  }

  watch(visible, show => {
    if (show && props.instance) {
      fetchLogs();
    }
  });
</script>

<style lang="postcss" scoped>
  .instance-log-sideslider :deep(.bk-modal-content) {
    height: calc(100% - 52px) !important;
    overflow: hidden !important;
    scrollbar-gutter: auto !important;
    > div {
      height: 100%;
      .bk-sideslider-content {
        height: 100%;
      }
    }
  }
  .instance-log-sideslider :deep(.bk-sideslider-header) {
    border-bottom: none;
  }
</style>
