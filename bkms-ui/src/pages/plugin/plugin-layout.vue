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
    :default-open="{
      value: false,
      force: true,
    }"
    :list="menuList"
    :need-title="false"
  >
    <RouterView :key="routerViewKey"></RouterView>
  </CustomNavigation>
</template>

<script lang="ts" setup>
  import { computed, onMounted, ref } from 'vue';

  import { useRouter } from 'vue-router';
  import { getPluginMenuList } from '~/composables/use-router-menu';
  import { NavigationItem } from '~/config/navigation/types';

  const router = useRouter();

  const menuList = ref<NavigationItem[]>([]);
  const activeKey = ref<string>(
    (Array.isArray(router.currentRoute.value.params.menuName)
      ? router.currentRoute.value.params.menuName[0]
      : router.currentRoute.value.params.menuName) || 'component',
  );

  const routerViewKey = computed(() => {
    const { menuName, name, space } = router.currentRoute.value.params;
    return `${menuName}-${name}-${space}`;
  });

  onMounted(() => {
    menuList.value = getPluginMenuList();
  });
</script>
