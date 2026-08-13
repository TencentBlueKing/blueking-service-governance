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
  <CustomNavigation
    v-model:active-key="activeKey"
    :list="menuList"
    :need-title="false"
  >
    <!-- 与「基本信息」等 tab 共用 container-header，子页面可通过 Teleport 注入 TabHeader -->
    <template #header>
      <div
        id="space-page-header"
        class="w-full"
      >
        <span class="text-[16px] space-page-header-title">{{ headerTitle }}</span>
      </div>
    </template>
    <RouterView :key="routerViewKey"></RouterView>
  </CustomNavigation>
</template>

<script lang="ts" setup>
  import { computed, onMounted, ref, watch } from 'vue';

  import { useRouter } from 'vue-router';
  import { getBasicMenuList } from '~/composables/use-router-menu';
  import { NavigationItem } from '~/config/navigation/types';

  const router = useRouter();

  const menuList = ref<NavigationItem[]>([]);
  const activeKey = ref<string>(
    (Array.isArray(router.currentRoute.value.params.menuName)
      ? router.currentRoute.value.params.menuName[0]
      : router.currentRoute.value.params.menuName) || 'info',
  );

  const routerViewKey = computed(() => {
    const { menuName, name, space } = router.currentRoute.value.params;
    return `${menuName}-${name}-${space}`;
  });

  /** 当前菜单标题，默认展示在 container-header 中 */
  const headerTitle = computed(() => {
    const items = menuList.value.flatMap(item => ('children' in item ? item.children : [item]));
    const current = items.find(item => item.key === activeKey.value);
    return current && 'name' in current ? current.name : '';
  });

  watch(activeKey, key => {
    router.push({
      name: 'basicItem',
      params: {
        menuName: key as string,
      },
    });
  });

  onMounted(() => {
    menuList.value = getBasicMenuList();
  });
</script>

<style lang="postcss" scoped>
  /* 子页面将 TabHeader 放入 header 后，隐藏默认标题并撑开高度 */
  :deep(.container-header:has(.tab-header-container)) {
    height: auto !important;
    padding: 0 !important;
    align-items: stretch !important;
  }

  :deep(.container-header:has(.tab-header-container) .space-page-header-title) {
    display: none;
  }

  :deep(.container-header .tab-header-container) {
    box-shadow: none;
    width: 100%;
  }
</style>
