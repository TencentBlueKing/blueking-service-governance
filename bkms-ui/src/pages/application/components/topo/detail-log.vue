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
  <div class="h-full p-[24px]">
    <InstanceLog
      is-custom-modules
      :loading="loading"
      :logs="logs"
      :modules="modules"
      @download="handleDownloadLog"
      @refresh="fetchLogs"
      @update:active-module="handleUpdateActiveModule"
    />
  </div>
</template>

<script lang="ts" setup>
  import { onMounted, ref, watch } from 'vue';

  import { useI18n } from 'vue-i18n';
  import { LogEntry } from '~/@types/instance';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import InstanceLog, {
    type IModule,
  } from '~/pages/application/detail/deploy/instance-list/components/instance-log.vue';

  import { downloadInstanceLog, RECENT_RESTART_LOG } from '../../detail/deploy/use-deploy';

  const props = defineProps<{
    appId: string;
    envName: string;
    nodeName: string;
  }>();

  const { t } = useI18n();

  const loading = ref(false);
  const logs = ref<LogEntry[]>([]);

  async function fetchLogs() {
    if (!props.nodeName || !props.appId || !props.envName) return;
    loading.value = true;
    try {
      logs.value = await ApiServerService.ListAppInstanceLogs({
        appID: props.appId,
        envName: props.envName,
        instanceID: props.nodeName,
        tailLines: 2000,
        previous: activeModule.value === RECENT_RESTART_LOG,
      });
    } catch (_) {
      logs.value = [];
    } finally {
      loading.value = false;
    }
  }

  // 下载实例日志
  function handleDownloadLog() {
    if (!props.nodeName || !props.appId || !props.envName) return;

    downloadInstanceLog({
      appID: props.appId,
      envName: props.envName,
      instanceID: props.nodeName,
      previous: activeModule.value === RECENT_RESTART_LOG,
    });
  }

  // 模块
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
  function handleUpdateActiveModule(value: string) {
    if (activeModule.value === value) return;
    activeModule.value = value;
    fetchLogs();
  }

  watch(
    () => props.nodeName,
    newName => {
      if (newName) {
        fetchLogs();
      }
    },
  );

  onMounted(() => {
    if (props.nodeName) {
      fetchLogs();
    }
  });
</script>
