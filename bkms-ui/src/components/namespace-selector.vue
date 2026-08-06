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
    :key="selectedValue"
    v-model="selectedValue"
    filterable
    :input-search="false"
    :loading="loading"
    :multiple="multiple"
    multiple-mode="tag"
  >
    <Select.Option
      v-for="item in namespaceList"
      :id="item.name"
      :key="item.name"
      :label="item.name"
    >
    </Select.Option>
  </Select>
</template>
<script setup lang="ts">
  import { onMounted, ref, watch } from 'vue';

  import { Select } from 'bkui-vue';
  import { ApiServerService } from '~/api/modules/bkmsserver';

  import type { Namespace } from '~/@types/bcs';

  interface IProps {
    clusterId: string;
    multiple?: boolean;
    projectID: string;
    value: string | string[];
  }

  const props = defineProps<IProps>();

  const emits = defineEmits(['update:value']);

  const selectedValue = ref<string | string[]>(props.value);

  const namespaceList = ref<Namespace[]>([]);

  const loading = ref(false);
  async function getData() {
    if (!props.projectID || !props.clusterId) return;
    loading.value = true;
    const result = await ApiServerService.ListNamespacesByCluster(
      {
        projectID: props.projectID,
        clusterID: props.clusterId,
      },
      { validateCode: false },
    ).catch(() => []);
    namespaceList.value = result || [];
    loading.value = false;
  }

  watch([() => props.clusterId, () => props.projectID], async () => {
    await getData();
  });
  watch(
    () => selectedValue.value,
    val => {
      emits('update:value', val);
    },
  );
  watch(
    () => props.value,
    () => {
      selectedValue.value = props.value;
    },
  );

  onMounted(async () => {
    await getData();
  });
</script>
