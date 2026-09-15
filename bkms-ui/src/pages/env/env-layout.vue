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
  >
    <div :class="pageBodyClass">
      <Exception
        v-if="envDetailStore.error"
        class="mt-[120px] large-exception"
        scene="part"
        :title="envDetailStore.error === 'notFound' ? $t('环境不存在或已被删除') : $t('环境详情加载失败')"
        :type="envDetailStore.error === 'notFound' ? '404' : '500'"
      >
        <Button
          v-if="envDetailStore.error === 'request'"
          class="mr-[8px] bg-[#fff]"
          @click="loadCurrentEnv(currentEnvId)"
          >{{ $t('重试') }}</Button
        >
        <Button
          theme="primary"
          @click="goEnvList"
        >
          {{ $t('返回列表') }}
        </Button>
      </Exception>
      <RouterView
        v-else-if="currentEnv || envDetailStore.loading"
        :key="routerViewKey"
      />
    </div>
    <template #side-header>
      <EnvDetailSelector
        :current-env-name="currentEnv?.displayName || currentEnv?.name"
        :current-env-type="currentEnv?.type"
        :env-list="envDetailStore.envList"
        :list-error="envDetailStore.listError"
        :model-value="currentEnvId"
        @back="goEnvList"
        @create="handleCreateEnv"
        @retry="envDetailStore.fetchEnvList()"
        @select="navigateToEnv"
      />
    </template>
  </CustomNavigation>
  <CreateEnv
    v-model:is-show="isShowCreateEnv"
    @confirm="envDetailStore.fetchEnvList()"
  />
</template>

<script lang="ts" setup>
  import { computed, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue';

  import { Button, Exception } from 'bkui-vue';
  import { useRoute, useRouter } from 'vue-router';
  import { DEFAULT_ENV_DETAIL_MENU, isEnvDetailMenu } from '~/composables/use-env-manager';
  import { getEnvMenuList } from '~/composables/use-router-menu';
  import { useEnvDetailStore } from '~/stores/env-detail';
  import { useSpaceStore } from '~/stores/space';

  import EnvDetailSelector from './components/env-detail-selector.vue';
  import CreateEnv from './create-env.vue';

  const route = useRoute();
  const router = useRouter();
  const spaceStore = useSpaceStore();
  const envDetailStore = useEnvDetailStore();

  const menuList = computed(() => getEnvMenuList());
  const activeKey = computed({
    get: () => (isEnvDetailMenu(route.params.menuName) ? route.params.menuName : DEFAULT_ENV_DETAIL_MENU),
    set: menuName => {
      router.push({ name: 'envDetailItem', params: { ...route.params, menuName }, query: route.query });
    },
  });
  const isShowCreateEnv = ref(false);

  const currentEnv = computed(() => envDetailStore.currentEnv);
  const routerViewKey = computed(() => {
    const { envId, menuName, space } = route.params;
    return `${menuName}-${envId}-${space}`;
  });
  const currentMenuItem = computed(() =>
    menuList.value
      .flatMap(item => ('children' in item ? item.children : [item]))
      .find(item => item.key === activeKey.value),
  );
  /** 边距打在真实包裹层上，避免子页 Skeleton 等 fragment 根节点吃掉 RouterView class */
  const pageBodyClass = computed(() => {
    if (currentMenuItem.value?.meta?.layout === 'empty') {
      return 'h-full min-h-full';
    }
    const extra = currentMenuItem.value?.meta?.class || '';
    return ['min-h-full px-[24px] py-[20px]', extra].filter(Boolean).join(' ');
  });

  const currentEnvId = computed(
    () => (Array.isArray(route.params.envId) ? route.params.envId[0] : route.params.envId) || '',
  );

  provide(
    'envName',
    computed(() => currentEnv.value?.name || ''),
  );

  function goEnvList() {
    router.push({
      name: 'env',
      params: { space: spaceStore.currentSpace },
    });
  }

  /** 打开侧栏前先收起环境选择下拉，避免浮层压在遮罩之上 */
  function handleCreateEnv() {
    isShowCreateEnv.value = true;
  }

  async function loadCurrentEnv(envId: string) {
    if (envId) await envDetailStore.fetchCurrentEnv(envId);
  }

  function navigateToEnv(envId: string) {
    if (!envId || envId === currentEnvId.value) return;
    router.push({
      name: 'envDetailItem',
      params: {
        ...route.params,
        envId,
        menuName: activeKey.value,
      },
    });
  }

  watch(
    () => route.params.menuName,
    menuName => {
      if (route.name === 'envDetailItem' && !isEnvDetailMenu(menuName)) {
        router.replace({
          name: 'envDetailItem',
          params: { ...route.params, menuName: DEFAULT_ENV_DETAIL_MENU },
          query: route.query,
        });
      }
    },
    { immediate: true },
  );

  watch(currentEnvId, loadCurrentEnv);

  onMounted(() => {
    envDetailStore.fetchEnvList(String(route.params.space));
    loadCurrentEnv(currentEnvId.value);
  });
  onBeforeUnmount(() => envDetailStore.reset());
</script>
