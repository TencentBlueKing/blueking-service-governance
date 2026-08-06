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
    :list="navigationList"
    :need-title="false"
  >
    <RouterView :key="routerViewKey"></RouterView>
  </CustomNavigation>
</template>

<script lang="ts" setup>
  import { computed, onMounted, ref } from 'vue';

  import { useRoute, useRouter } from 'vue-router';
  import { getPlatformMenuList } from '~/composables/use-router-menu';

  import type { NavigationItem } from '~/config/navigation/types';

  const route = useRoute();
  const router = useRouter();

  const menuList = ref<NavigationItem[]>([]);
  const navigationList = computed(() => {
    if (route.name !== 'platformWorkspaceDetail') {
      return menuList.value;
    }

    return menuList.value.map(item => {
      if ('children' in item || item.key !== 'workspace') {
        return item;
      }

      return {
        ...item,
        meta: {
          ...item.meta,
          layout: 'empty' as const,
        },
      };
    });
  });

  const activeKey = computed({
    get() {
      if (route.name === 'platformWorkspaceDetail') return 'workspace';

      const { menuName } = route.params;
      return (Array.isArray(menuName) ? menuName[0] : menuName) || 'workspace';
    },
    set(key: string) {
      if (key === activeKey.value && route.name === 'platformItem') return;

      router.push({
        name: 'platformItem',
        params: {
          menuName: key,
        },
      });
    },
  });

  const routerViewKey = computed(() => {
    const { menuName, name, space, workspaceID } = route.params;
    return `${String(route.name)}-${menuName}-${name}-${space}-${workspaceID}`;
  });

  onMounted(() => {
    menuList.value = getPlatformMenuList();
  });
</script>
