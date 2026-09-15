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
    <div
      v-bkloading="{ loading: pageLoading }"
      :class="pageBodyClass"
    >
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
        v-else-if="currentEnv"
        :key="routerViewKey"
      />
    </div>
    <template #side-header>
      <Select
        ref="envSelectRef"
        class="w-full h-full"
        :clearable="false"
        :filter-option="filterEnvOption"
        filterable
        :input-search="false"
        :model-value="currentEnvId"
        :popover-min-width="244"
        :scroll-height="ENV_SELECT_LIST_MAX_HEIGHT"
        :search-placeholder="$t('请输入环境名称')"
        @change="navigateToEnv"
        @toggle="isEnvPopoverVisible = $event"
      >
        <template #trigger>
          <div
            class="flex items-center justify-between w-full h-full bg-[#F0F1F5] overflow-hidden group text-[#4D4F56] text-[12px] cursor-pointer rounded-[2px] hover:bg-[#EAEBF0]"
          >
            <div class="flex items-center min-w-0 overflow-hidden px-[8px] gap-[4px]">
              <span class="whitespace-nowrap truncate">{{ currentEnvLabel }}</span>
              <Tag
                v-if="envTypeConfig?.name"
                class="shrink-0"
                :class="envTypeTagClassMap[currentEnv?.type || '']"
                size="small"
              >
                {{ envTypeConfig.name }}
              </Tag>
            </div>
            <AngleDownFill
              :class="['mr-[5px] text-[#C4C6CC] group-hover:text-[#979BA5]', isEnvPopoverVisible ? '' : 'rotate-180']"
            />
          </div>
        </template>
        <template #extension>
          <Button
            v-if="envDetailStore.listError"
            text
            @click="envDetailStore.fetchEnvList()"
            >{{ $t('重试') }}</Button
          >
          <div class="grid grid-cols-2 w-full h-full">
            <div
              class="flex h-full items-center justify-center gap-[5px] text-[12px] text-[#4D4F56] cursor-pointer hover:bg-[#F0F1F5]"
              @click="handleCreateEnv"
            >
              <Plus
                height="16px"
                width="16px"
              />
              <span>{{ $t('新建环境') }}</span>
            </div>
            <div
              class="flex h-full items-center justify-center text-[12px] text-[#4D4F56] cursor-pointer hover:bg-[#F0F1F5]"
              @click="goEnvList"
            >
              {{ $t('返回列表') }}
            </div>
          </div>
        </template>
        <Select.Option
          v-for="env in envDetailStore.envList"
          :id="env.id"
          :key="env.id"
          :name="env.displayName || env.name"
        >
          <div class="flex items-center gap-[8px] min-w-0">
            <span class="truncate">{{ env.displayName || env.name }}</span>
            <Tag
              v-if="env.type && envTypeMap[env.type]"
              class="shrink-0"
              :class="envTypeTagClassMap[env.type]"
              size="small"
            >
              {{ envTypeMap[env.type].name }}
            </Tag>
          </div>
        </Select.Option>
      </Select>
    </template>
  </CustomNavigation>
  <CreateEnv
    v-model:is-show="isShowCreateEnv"
    @confirm="envDetailStore.fetchEnvList()"
  />
</template>

<script lang="ts" setup>
  import { computed, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue';

  import { Button, Exception, Select, Tag } from 'bkui-vue';
  import { AngleDownFill, Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import {
    DEFAULT_ENV_DETAIL_MENU,
    envTypeMap,
    envTypeTagClassMap,
    isEnvDetailMenu,
  } from '~/composables/use-env-manager';
  import { getEnvMenuList } from '~/composables/use-router-menu';
  import { useEnvDetailStore } from '~/stores/env-detail';
  import { useSpaceStore } from '~/stores/space';

  import CreateEnv from './create-env.vue';

  /** 下拉列表最多展示 7 项，超出后滚动，搜索和底部入口保持可见。 */
  const ENV_SELECT_LIST_MAX_HEIGHT = 7 * 32;

  const { t } = useI18n();
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
  const envSelectRef = ref<InstanceType<typeof Select> | null>(null);
  const isEnvPopoverVisible = ref(false);
  const isShowCreateEnv = ref(false);

  const currentEnv = computed(() => envDetailStore.currentEnv);
  const currentEnvLabel = computed(() => currentEnv.value?.displayName || currentEnv.value?.name || t('请选择环境'));
  const envTypeConfig = computed(() => {
    const type = currentEnv.value?.type;
    return type && envTypeMap[type] ? envTypeMap[type] : undefined;
  });
  const pageLoading = computed(() => envDetailStore.loading);
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

  function filterEnvOption(keyword: string, option: { id?: string }) {
    const env = envDetailStore.envList.find(item => item.id === option.id);
    const search = keyword.trim().toLowerCase();
    return !!env && [env.name, env.displayName].some(name => name?.toLowerCase().includes(search));
  }

  function goEnvList() {
    envSelectRef.value?.hidePopover();
    router.push({
      name: 'env',
      params: { space: spaceStore.currentSpace },
    });
  }

  /** 打开侧栏前先收起环境选择下拉，避免浮层压在遮罩之上 */
  function handleCreateEnv() {
    envSelectRef.value?.hidePopover();
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

<style lang="postcss" scoped>
  :deep(.bk-select .bk-select-trigger) {
    height: 100%;
  }
</style>
