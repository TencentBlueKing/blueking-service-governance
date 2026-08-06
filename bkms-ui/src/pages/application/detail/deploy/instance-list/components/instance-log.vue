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
  <BkTaskLog
    class="h-full"
    :data="stepData"
    :enable-minimap="false"
    height="100%"
    :is-custom-modules="isCustomModules"
    :loading="loading"
    :modules="defaultModules"
    :title="$t(title)"
    type="default"
    @download="emits('download')"
    @refresh="emits('refresh')"
    @update:active-module="handleUpdateActiveModule"
  />
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import BkTaskLog from '@blueking/task-log';
  import { useI18n } from 'vue-i18n';
  import { LogEntryOutputObj } from '~/@types/v1/instance';

  import '@blueking/task-log/vue3/vue3.css';

  export interface IModule {
    text: string;
    useDefaultContent?: boolean; // 是否使用默认日志模板内容
    value: number | string;
  }

  interface IStep {
    data: LogEntryOutputObj[];
    endTime?: string;
    id: string;
    name: string;
    startTime?: string;
    status?: string;
  }

  interface Props {
    height?: number | string;
    isCustomModules?: boolean;
    loading?: boolean;
    logs?: LogEntryOutputObj[];
    modules?: IModule[];
    /** 日志面板标题，默认显示“实例日志”。 */
    title?: string;
  }
  const props = withDefaults(defineProps<Props>(), {
    logs: () => [],
    loading: false,
    title: '实例日志',
  });

  const { t } = useI18n();

  const emits = defineEmits<{
    download: [];
    refresh: [];
    retry: [data: IStep];
    'update:active-module': [value: string];
  }>();

  const defaultModules = computed(() => props.modules || []);

  const handleUpdateActiveModule = (value: string) => {
    emits('update:active-module', value);
  };

  // 日志数据
  const stepData = computed<IStep>(() => {
    return {
      id: 'instance-log',
      name: t('实例日志'),
      status: 'SUCCESS',
      data: props.logs.map(item => ({
        ...item,
        log: item.content,
      })),
    };
  });
</script>

<style scoped>
  /* 隐藏包含暂不支持icon功能 */
  :deep(span:has(> i.task-log-auto-refresh)) {
    display: none !important;
  }
  /* :deep(.bk-task-log-scroll) {
    white-space: pre !important;
  } */
  :deep(.relative) {
    height: calc(100% - 40px);
  }
</style>
