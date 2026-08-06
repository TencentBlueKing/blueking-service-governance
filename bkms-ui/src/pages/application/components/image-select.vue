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
    v-model="value"
    :disabled="disabled"
    display-key="tag"
    filterable
    id-key="tag"
    :list="displayList"
    :no-data-text="$t('暂无可用镜像')"
    :remote-method="handleImageSearch"
    :scroll-loading="loading"
    @scroll-end="handleScrollToBottom"
  >
    <template #optionRender="{ item }">
      <span>
        {{ item.tag }}
      </span>
    </template>
    <template #extension>
      <div class="w-full flex items-center justify-center">
        <template v-if="isProductionEnv">
          <span class="text-[#63656e]">{{ $t('生产类型环境仅支持部署已晋级的 Tag') }}</span>
          <Button
            class="ml-[8px]"
            text
            theme="primary"
            @click="handleGotoArtifact"
          >
            <Share class="mr-[4px]" />
            {{ $t('去晋级') }}
          </Button>
        </template>
        <Button
          v-else
          text
          theme="primary"
          @click="handleGotoBuild"
        >
          <Share class="mr-[4px]" />
          {{ $t('去构建') }}
        </Button>
      </div>
    </template>
  </Select>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Button, Select } from 'bkui-vue';
  import { Share } from 'bkui-vue/lib/icon';
  import { debounce } from 'lodash-es';
  import { useRoute, useRouter } from 'vue-router';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  interface Props {
    disabled?: boolean;
    envName?: string;
    envType?: string;
  }

  const props = withDefaults(defineProps<Props>(), {
    disabled: false,
  });

  const route = useRoute();
  const router = useRouter();
  const value = defineModel<string>('value');
  const trpcDeployStore = useTrpcDeployStore();
  const appDetailStore = useAppDetail();
  let maxImageListCount: number | undefined = undefined;

  // 部署版本
  const loading = ref(false);
  const imageKeyword = ref('');
  let page = 1;
  let pageSize = 10;

  const selectRef = ref();
  const displayList = computed(() => {
    if (!loading.value && trpcDeployStore.imageList.length === 0) {
      return [];
    }
    return trpcDeployStore.imageList;
  });
  const queryEnvName = computed(() => props.envName || trpcDeployStore.curEnvItem?.name || '');
  const isProductionEnv = computed(() => (props.envType || trpcDeployStore.curEnvItem?.type) === 'production');

  async function handleGetImageList() {
    const envName = queryEnvName.value;
    if (!appDetailStore.appID || !envName) return;
    // 允许获取更多镜像Tag需要满足：
    // 1. 至少获取过一次镜像Tag(maxImageListCount初始化)
    // 2. 当前的imageList不超过count
    if (maxImageListCount !== undefined && trpcDeployStore.imageList.length >= maxImageListCount) return;
    loading.value = true;
    try {
      const res = await ApiServerService.ListDeployableImageTags({
        appID: appDetailStore.appID,
        envName,
        page,
        pageSize,
        keyword: imageKeyword.value,
      });
      maxImageListCount = Number(res.count || 0);
      trpcDeployStore.imageList = [...trpcDeployStore.imageList, ...(res.results || [])];
    } catch (err) {
      console.error(err);
    } finally {
      loading.value = false;
    }
  }

  // 跳转到制品管理页面
  function handleGotoArtifact() {
    selectRef.value?.hidePopover?.();
    router.push({
      name: 'detail',
      params: {
        ...route.params,
        menuName: 'artifact',
      },
    });
  }

  // 跳转到构建管理页面
  function handleGotoBuild() {
    selectRef.value?.hidePopover?.();
    router.push({
      name: 'detail',
      params: {
        ...route.params,
        menuName: 'build',
      },
    });
  }

  // 镜像搜索
  const handleImageSearch = debounce((keyword: string) => {
    imageKeyword.value = keyword;
    page = 1;
    maxImageListCount = undefined;
    loading.value = true;
    trpcDeployStore.imageList = [];
    handleGetImageList();
  }, 300);

  function handleScrollToBottom() {
    // 若数据正在加载中，应避免多次请求
    if (loading.value) return;
    page += 1;
    handleGetImageList();
  }

  function resetAndFetchImageList() {
    page = 1;
    maxImageListCount = undefined;
    trpcDeployStore.imageList = [];
    handleGetImageList();
  }

  watch([() => appDetailStore.appID, queryEnvName], () => resetAndFetchImageList(), { immediate: true });

  defineExpose({
    closeDropdown() {
      selectRef.value?.hidePopover?.();
    },
  });
</script>
