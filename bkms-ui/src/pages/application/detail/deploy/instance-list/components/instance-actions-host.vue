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
  <WeightDialog
    v-model:visible="weightVisible"
    :env-name="weightEnvName"
    :instance="weightInstance"
  />
  <GrayDialog
    v-model:visible="grayVisible"
    :env-display-name="grayEnvDisplayName"
    :env-name="grayEnvName"
    :instance-ids="grayInstanceIds"
    :instances="grayInstances"
  />
  <InstanceLogSideslider
    v-model:visible="logVisible"
    :env-name="logEnvName"
    :instance="logInstance"
  />
  <ManagementCommandSideslider
    v-model:visible="commandVisible"
    :env-display-name="commandEnvDisplayName"
    :instance-ids="commandInstanceIds"
  />
  <MonitorSideSlider
    v-model:is-show="monitorVisible"
    :env-name="monitorEnvName"
    :env-names="monitorEnvNames"
    :initial-selection="monitorSelection"
  />
</template>

<script lang="ts" setup>
  import { ref, watch } from 'vue';

  import { type AppInstanceOutputObj } from '~/@types/v1/instance';

  import MonitorSideSlider from '../../monitor-sideslider.vue';
  import GrayDialog from './actions/gray-dialog.vue';
  import InstanceLogSideslider from './actions/instance-log-sideslider.vue';
  import ManagementCommandSideslider from './actions/management-command-sideslider.vue';
  import WeightDialog from './actions/weight-dialog.vue';

  const weightVisible = ref(false);
  const weightEnvName = ref('');
  const weightInstance = ref<AppInstanceOutputObj | null>(null);

  const grayVisible = ref(false);
  const grayEnvName = ref('');
  const grayEnvDisplayName = ref('');
  const grayInstances = ref<AppInstanceOutputObj[]>([]);
  const grayInstanceIds = ref<string[] | undefined>(undefined);

  const logVisible = ref(false);
  const logEnvName = ref('');
  const logInstance = ref<AppInstanceOutputObj | null>(null);

  const commandVisible = ref(false);
  const commandEnvDisplayName = ref('');
  const commandInstanceIds = ref<string[]>([]);

  const monitorVisible = ref(false);
  const monitorSelection = ref<Record<string, string[]>>();
  const monitorEnvName = ref<string>();
  const monitorEnvNames = ref<string[]>();

  function openCommand(instanceIds: string[], envDisplayName?: string) {
    commandInstanceIds.value = [...instanceIds];
    commandEnvDisplayName.value = envDisplayName || '';
    commandVisible.value = true;
  }

  function openGray(options: {
    envDisplayName?: string;
    envName: string;
    instanceIds?: string[];
    instances: AppInstanceOutputObj[];
  }) {
    grayEnvName.value = options.envName;
    grayEnvDisplayName.value = options.envDisplayName || '';
    grayInstances.value = options.instances;
    grayInstanceIds.value = options.instanceIds;
    grayVisible.value = true;
  }

  function openLog(instance: AppInstanceOutputObj, envName: string) {
    logEnvName.value = envName;
    logInstance.value = instance;
    logVisible.value = true;
  }

  function openMonitor(initialSelection?: Record<string, string[]>, envName?: string, envNames?: string[]) {
    monitorSelection.value = initialSelection;
    monitorEnvName.value = envName;
    monitorEnvNames.value = envNames;
    monitorVisible.value = true;
  }

  function openWeight(instance: AppInstanceOutputObj, envName: string) {
    weightEnvName.value = envName;
    weightInstance.value = instance;
    weightVisible.value = true;
  }

  watch(commandVisible, visible => {
    if (!visible) {
      commandInstanceIds.value = [];
      commandEnvDisplayName.value = '';
    }
  });

  defineExpose({
    openCommand,
    openGray,
    openLog,
    openMonitor,
    openWeight,
  });
</script>
