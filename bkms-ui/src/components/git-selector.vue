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
    filterable
    :loading="loading"
    :model-value="modelValue"
    :remote-method="handleRemoteSearch"
    @change="handleProjectChange"
  >
    <template #prefix>
      <span class="px-[10px] text-[#63656E] border-r-[#c4c6cc] border-r">
        {{ prefix || $t('工蜂Git') }}
      </span>
    </template>
    <Select.Option
      v-for="p in projectList"
      :key="p.id"
      :name="p.url"
      :value="p.url"
    />
    <template #extension>
      <div
        v-if="showOauth"
        class="flex items-center justify-center w-full"
      >
        <Link
          class="text-[12px]"
          :href="oauthURL"
          target="_blank"
          theme="primary"
          @click="selectRef?.hidePopover"
          >{{ $t('前往授权') }}</Link
        >
      </div>
      <Loading
        v-else
        class="w-full"
        :loading="loading"
        size="small"
      >
        <div
          class="flex items-center justify-center w-full cursor-pointer"
          @click="handleRefresh"
        >
          <RightTurnLine class="text-[14px] mr-[6px]" />
          <span class="#4D4F56 text-[12ox]">{{ $t('刷新列表') }}</span>
        </div>
      </Loading>
    </template>
  </Select>
</template>
<script lang="ts" setup>
  import { ref, watch } from 'vue';

  import { Link, Loading, Select } from 'bkui-vue';
  import { RightTurnLine } from 'bkui-vue/lib/icon';
  import { debounce } from 'lodash-es';
  import { ApiServerService } from '~/api/modules/bkmsserver';

  import type { BkCIOAuthGitProjectOutputObj } from '~/@types/bkci';

  interface IEmit {
    (e: 'update:modelValue', value: string): void;
    (e: 'change' | 'init', value: BkCIOAuthGitProjectOutputObj): void;
  }
  interface IProps {
    modelValue?: string;
    prefix?: string;
    workspace: string;
  }
  const props = defineProps<IProps>();
  const emits = defineEmits<IEmit>();

  const projectList = ref<BkCIOAuthGitProjectOutputObj[]>([]);
  const loading = ref<boolean>(false);
  const showOauth = ref<boolean>(false);
  const oauthURL = ref<string>('');
  const searchKeyword = ref<string>('');

  // ref
  const selectRef = ref();

  // 获取授权链接
  async function getBkCIOAuthUrl() {
    const url = await ApiServerService.GetBkCIOAuthUrl({
      workspaceID: props.workspace,
    }).catch(() => '');
    if (url) {
      showOauth.value = true;
      oauthURL.value = url;
    }
  }

  async function getProjects(keyword = '') {
    if (!props.workspace) return [];

    loading.value = true;

    const res = await ApiServerService.ListBkCIOAuthGitProjects({
      workspaceID: props.workspace,
      keyword,
    }).catch(() => []);

    projectList.value = res as BkCIOAuthGitProjectOutputObj[];
    if (res?.length > 0) {
      initializeWithSelectedProject();
    } else if (!keyword) {
      // 只在非搜索情况下检查授权
      getBkCIOAuthUrl();
    }
    loading.value = false;
    return projectList.value;
  }

  const debouncedSearch = debounce((keyword: string) => {
    getProjects(keyword);
  }, 300);

  function handleProjectChange(value: string) {
    const project = projectList.value.find(p => p.url === value);
    if (project) {
      emits('update:modelValue', value);
      emits('change', project);
    } else {
      emits('update:modelValue', '');
    }
  }

  function handleRefresh() {
    searchKeyword.value = '';
    debouncedSearch.cancel();
    getProjects();
  }

  // 远程搜索
  function handleRemoteSearch(keyword: string) {
    searchKeyword.value = keyword;
    debouncedSearch(keyword);
  }

  function initializeWithSelectedProject() {
    if (props.modelValue) {
      const project = projectList.value.find(p => p.url === props.modelValue);
      if (project) {
        emits('init', project);
      }
    }
  }

  watch(
    () => props.workspace,
    () => {
      searchKeyword.value = '';
      debouncedSearch.cancel();
      getProjects();
    },
    { immediate: true },
  );
</script>
