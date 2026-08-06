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
    :loading="pipelineLoading"
    :model-value="modelValue"
    multiple-mode="tag"
    :remote-method="handleSearchPipeline"
    :scroll-loading="pipelineScrollLoading"
    @scroll-end="handlePipelineScrollEnd"
    @update:model-value="handleUpdateValue"
  >
    <template #tag>
      <span>{{ displayValue }}</span>
    </template>
    <Select.Option
      v-for="item in pipelineList"
      :id="item.id"
      :key="item.id"
      :label="item.name"
    >
    </Select.Option>
  </Select>
</template>
<script lang="ts" setup>
  import { computed, onBeforeMount, ref, watch } from 'vue';

  import { Select } from 'bkui-vue';
  import { BkCIPipelineOutputObj } from '~/@types/bkci';
  import { ApiServerService } from '~/api/modules/bkmsserver';

  const props = defineProps({
    modelValue: {
      type: String,
      default: '',
    },
    workspace: {
      type: String,
      required: true,
    },
  });

  const emit = defineEmits<{
    'update:modelValue': [value: string];
  }>();

  const pipelineList = ref<BkCIPipelineOutputObj[]>([]);
  const pipelineLoading = ref<boolean>(false);
  const pipelineScrollLoading = ref<boolean>(false);
  const pipelineSearchKeyword = ref<string>('');
  const pipelinePagination = ref({
    page: 1,
    pageSize: 10,
    total: 0,
    hasMore: true,
  });

  // 当前选中项的名称（不在列表中时通过接口获取）
  const selectedPipelineName = ref<string>('');

  // 回显文本：优先从列表中匹配，否则使用接口获取的名称
  const displayValue = computed(() => {
    if (!props.modelValue) return '';
    const matched = pipelineList.value.find(item => item.id === props.modelValue);
    return matched?.name ?? selectedPipelineName.value ?? props.modelValue;
  });

  // 获取当前选中流水线的名称
  async function fetchSelectedPipelineName() {
    if (!props.modelValue || !props.workspace) return;
    // 列表中已有则直接取
    const matched = pipelineList.value.find(item => item.id === props.modelValue);
    if (matched) {
      selectedPipelineName.value = matched.name ?? '';
      return;
    }
    try {
      const pipelineDetail = await ApiServerService.GetBkCIPipeline({
        workspaceID: props.workspace,
        pipelineID: props.modelValue,
      });
      if (pipelineDetail?.name) {
        selectedPipelineName.value = pipelineDetail.name;
      }
    } catch {
      // 获取失败不影响主流程
      console.error('Failed to fetch pipeline name');
    }
  }

  // 获取流水线列表
  async function getPipelineList(isLoadMore = false) {
    if (!props.workspace) return;

    // 如果是加载更多但已经没有更多数据，直接返回
    if (isLoadMore && !pipelinePagination.value.hasMore) return;

    if (isLoadMore) {
      pipelineScrollLoading.value = true;
    } else {
      pipelineLoading.value = true;
      pipelinePagination.value.page = 1;
    }

    try {
      const { results = [], count = 0 } = await ApiServerService.ListBkCIPipelines({
        workspaceID: props.workspace,
        keyword: pipelineSearchKeyword.value,
        page: pipelinePagination.value.page,
        pageSize: pipelinePagination.value.pageSize,
      });

      // 更新列表数据
      pipelineList.value = isLoadMore ? [...pipelineList.value, ...results] : results;
      pipelinePagination.value.total = Number(count) || 0;
      pipelinePagination.value.hasMore = pipelineList.value.length < (Number(count) || 0);
    } catch {
      if (!isLoadMore) {
        pipelineList.value = [];
      }
    } finally {
      pipelineLoading.value = false;
      pipelineScrollLoading.value = false;
    }
  }

  // 处理滚动到底部加载更多
  function handlePipelineScrollEnd() {
    if (pipelinePagination.value.hasMore && !pipelineScrollLoading.value) {
      pipelinePagination.value.page += 1;
      getPipelineList(true);
    }
  }

  function handleSearchPipeline(keyword: string) {
    pipelineSearchKeyword.value = keyword;
    getPipelineList();
  }

  // 处理值更新
  function handleUpdateValue(value: string) {
    emit('update:modelValue', value);
  }

  // 重置流水线列表（当 workspace 变化时）
  function resetPipelineList() {
    pipelineList.value = [];
    pipelinePagination.value = {
      page: 1,
      pageSize: 10,
      total: 0,
      hasMore: true,
    };
    pipelineSearchKeyword.value = '';
  }

  // 监听 workspace 变化
  watch(
    () => props.workspace,
    async newWorkspace => {
      if (newWorkspace) {
        resetPipelineList();
        await getPipelineList();
        await fetchSelectedPipelineName();
      }
    },
    { immediate: false },
  );

  // 监听 modelValue 变化，更新回显名称
  watch(
    () => props.modelValue,
    () => {
      fetchSelectedPipelineName();
    },
  );

  onBeforeMount(async () => {
    if (props.workspace) {
      await getPipelineList();
      await fetchSelectedPipelineName();
    }
  });

  defineExpose({
    refresh: getPipelineList,
    reset: resetPipelineList,
  });
</script>
