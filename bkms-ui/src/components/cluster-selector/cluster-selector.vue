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
  <Select
    ref="selectRef"
    v-model="selectedValue"
    :clearable="false"
    filterable
    :input-search="false"
    :loading="loading"
    :multiple="multiple"
    :popover-min-width="400"
    :popover-options="{
      zIndex: 999999,
    }"
  >
    <template
      v-if="trigger"
      #trigger="{ selected }"
    >
      <slot
        name="trigger"
        :selected="selected"
      ></slot>
    </template>
    <Select.Group
      v-for="group in list"
      :key="group.type"
      collapsible
      :label="group.title"
    >
      <Select.Option
        v-for="item in group.list"
        :id="item.id"
        :key="item.id"
        :disabled="diabledList?.includes(item.id || '') && item.id !== selectedValue"
        :name="item.name"
      />
    </Select.Group>
    <template #extension>
      <div
        v-bk-authority="{
          disablePerms: false,
          actionId: 'cluster_create',
          resourceName: projectName,
          permCtx: {
            resource_type: 'project',
            project_id: projectID,
          },
        }"
        :class="[
          'h-full w-full flex justify-center items-center select-none cursor-pointer',
          { 'cursor-not-allowed': !projectCode },
        ]"
        @click="goCreateCluster"
      >
        <span
          v-bk-tooltips="{
            disabled: projectCode,
            content: $t('请先选择容器项目'),
          }"
          class="flex-1 text-center"
          >{{ $t('新增集群') }}</span
        >
      </div>
    </template>
  </Select>
</template>
<script setup lang="ts">
  import { ref, watch } from 'vue';

  import { Select } from 'bkui-vue';

  import type { ClusterSelectorGroup, ClusterType } from './use-cluster-selector';
  // import useClusterSelector from './use-cluster-selector';

  interface IProps {
    clusterType?: ClusterType;
    diabledList?: string[];
    list: ClusterSelectorGroup[];
    loading?: boolean;
    multiple?: boolean;
    projectCode?: string;
    projectID?: string;
    projectName?: string;
    trigger?: boolean;
    value: string | string[];
  }

  const props = defineProps<IProps>();
  const emits = defineEmits(['update:value', 'update:clusterType']);

  const selectedValue = ref(props.value);

  // ref
  const selectRef = ref();

  // const {
  //   loading,
  //   clusterData,
  //   getClusterList,
  // } = useClusterSelector(emits, props.projectID, props.clusterType);

  // watch(() => props.projectID, async () => {
  //   // 项目改变，重新获取集群列表
  //   await getClusterList(props.projectID);
  // });

  function goCreateCluster() {
    if (!props.projectCode) return;
    selectRef.value?.hidePopover?.();
    // const url = new URL(``, window.location.origin);
    window.open(`${window.BK_BCS}/bcs/projects/${props.projectCode}/clusters/create`, '_blank');
  }

  watch(
    () => props.value,
    () => {
      selectedValue.value = props.value;
    },
  );

  // 查找所选集群对应的 type
  function findClusterType(clusterId: string): ClusterType | undefined {
    for (const group of props.list) {
      if (group.list.find(item => item.id === clusterId)) {
        return group.type;
      }
    }
    return undefined;
  }

  watch(
    selectedValue,
    () => {
      emits('update:value', selectedValue.value);
      const clusterType = findClusterType(selectedValue.value as string);
      if (clusterType) {
        emits('update:clusterType', clusterType);
      }
    },
    { immediate: true },
  );
</script>
