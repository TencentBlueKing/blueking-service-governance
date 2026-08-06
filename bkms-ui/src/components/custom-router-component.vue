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
  <component
    :is="getComponent()"
    :key="componentKey"
  />
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import { useRouter } from 'vue-router';
  import { getMenuListByMenuId, MenuIdType } from '~/composables/use-router-menu';

  import type { AppNavigationType } from '~/config/navigation/app';
  import type { BaseNavigationItem } from '~/config/navigation/types';

  const router = useRouter();
  const type = computed(() => (router.currentRoute.value.params.type || '') as AppNavigationType);
  const menuName = computed(() => (router.currentRoute.value.params.menuName || '') as string);
  const componentKey = computed(() => `${type.value}-${menuName.value}`);

  // 根据type和menuName获取组件
  function getComponent() {
    const menuId = router.currentRoute.value.meta.menuId as MenuIdType;
    const menuList = getMenuListByMenuId(menuId, type.value);
    // 打平导航菜单，方便查找组件
    const formatList = menuList.reduce((acc, cur) => {
      if ('children' in cur) {
        acc.push(...cur.children);
      } else {
        acc.push(cur);
      }
      return acc;
    }, [] as BaseNavigationItem[]);
    const menu = formatList.find(item => item.key === menuName.value);
    return menu?.component || null;
  }
</script>
