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
    v-model="projectCode"
    :disabled="disabled"
    filterable
    :remote-method="remoteSearch"
  >
    <Select.Option
      v-for="item in projectList"
      :id="item.code"
      :key="item.code"
      :name="`${item.name}（${item.code}）`"
    >
      <span class="flex-1">{{ item.name }}（{{ item.code }}）</span>
    </Select.Option>
  </Select>
</template>
<script lang="ts" setup>
  import type { Ref } from 'vue';
  import { onMounted, ref, watch } from 'vue';

  import { Select } from 'bkui-vue';
  import useDebouncedRef from '~/composables/use-debounce';

  import useProject from './use-project';

  const props = defineProps<{
    disabled?: boolean;
    projectCode: string;
  }>();

  const emits = defineEmits(['change']);

  const projectCode = ref(props.projectCode);
  const searchKey = useDebouncedRef('', 600) as Ref<string>;

  const { projectList, handleInitProjectList } = useProject(searchKey);

  function getData() {
    return projectList.value;
  }

  // 远程搜索
  const selectRef = ref<InstanceType<typeof Select>>();
  const remoteSearch = (key: string) => {
    if (selectRef.value) {
      selectRef.value.searchLoading = false;
      searchKey.value = key;
    }
  };
  watch(searchKey, async () => {
    if (selectRef.value) {
      selectRef.value.searchLoading = true;
      await handleInitProjectList();
      selectRef.value.searchLoading = false;
    }
  });

  watch(projectCode, val => {
    emits('change', val);
  });

  // 监听父组件传入的 projectCode 变化
  watch(
    () => props.projectCode,
    val => {
      projectCode.value = val;
    },
  );

  onMounted(async () => {
    if (!props?.disabled) {
      await handleInitProjectList();
    }
  });

  defineExpose({
    getData,
  });
</script>
