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
    filterable
    :loading="loading"
    :value="modelValue"
    @change="handleProjectChange"
  >
    <template #prefix>
      <span class="px-[10px] text-[#63656E] border-r-[#c4c6cc] border-r leading-[30px]">
        {{ $t('工蜂') }}
      </span>
    </template>
    <Select.Option
      v-for="p in projectList"
      :key="p.id"
      :name="p.httpUrl"
      :value="p.httpUrl"
    />
  </Select>
</template>
<script setup lang="ts">
  import { onMounted, ref } from 'vue';

  import { Select } from 'bkui-vue';
  import { getGitProjects } from '~/api/modules/custom';

  interface IProject {
    httpUrl: string;
    id: string;
    lastActivity: number;
    name: string;
    nameWithNameSpace: string;
    sshUrl: string;
  }

  interface IProps {
    modelValue: string;
    workspace: string;
  }

  const props = defineProps<IProps>();

  const emits = defineEmits(['update:modelValue']);

  // 代码库&代码库别名
  const projectList = ref<IProject[]>([]);
  const loading = ref<boolean>(false);
  async function getProjects() {
    if (!props.workspace) return;
    loading.value = true;
    const res = (await getGitProjects({
      projectId: props.workspace,
    }).catch(() => ({ data: { project: [] } }))) as { project: IProject[] };
    projectList.value = res?.project || [];
    loading.value = false;
  }
  function handleProjectChange(url: string) {
    const project = projectList.value.find(p => p.httpUrl === url);
    emits('update:modelValue', project?.httpUrl);
  }

  onMounted(() => {
    getProjects();
  });
</script>
