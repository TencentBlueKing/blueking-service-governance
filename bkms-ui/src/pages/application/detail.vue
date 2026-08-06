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
    <RouterView
      :key="routerViewKey"
      :class="routerViewClass"
    >
    </RouterView>
    <template #side-header>
      <Select
        v-model="currentApplicationName"
        class="w-full h-full"
        :clearable="false"
        filterable
        placeholder="请选择应用"
        search-placeholder="请输入应用名称"
        @toggle="isSpacePopoverShow = !isSpacePopoverShow"
      >
        <template #trigger>
          <div
            class="flex items-center justify-between bg-[#F0F1F5] overflow-hidden group text-[#4D4F56] h-full text-[14px] cursor-pointer rounded-[2px] hover:bg-[#EAEBF0]"
          >
            <div class="flex items-center min-w-[43px] overflow-hidden">
              <TypeIcon
                classes="min-w-[43px] inline-block"
                :show-label="false"
                :type="type"
              />
              <span class="whitespace-nowrap">{{ currentApplicationName }}</span>
            </div>
            <AngleDownFill
              :class="['mr-[5px] text-[#C4C6CC] group-hover:text-[#979BA5]', isSpacePopoverShow ? '' : 'rotate-180']"
            />
          </div>
        </template>
        <template #extension>
          <Button
            class="w-full flex items-center justify-center gap-[5px] text-[14px]"
            text
            @click="goCreateApplication"
          >
            <Plus
              height="20px"
              width="20px"
            />
            <span>{{ $t('创建应用') }}</span>
          </Button>
        </template>
        <Select.Option
          v-for="item in applicationList"
          :id="item.name"
          :key="item.name"
          :name="item.name"
        >
          <div class="flex items-center gap-[5px]">
            <TypeIcon
              :classes="`min-w-[20px] inline-block ${item.type === 'trpc' ? 'text-[6px]' : 'text-[19px]'}`"
              :show-label="false"
              :type="item.type"
            />
            <span>{{ item.name }}</span>
          </div>
        </Select.Option>
      </Select>
    </template>
  </CustomNavigation>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref, watch } from 'vue';

  import { Select } from 'bkui-vue';
  import { Button } from 'bkui-vue';
  import { AngleDownFill, Plus } from 'bkui-vue/lib/icon';
  import { useRoute, useRouter } from 'vue-router';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import { isHelmLikeAppType } from '~/composables/app-type';
  import { getMenuList } from '~/composables/use-router-menu';
  import { useAppDetail } from '~/stores/app-detail';
  import { useSpaceStore } from '~/stores/space';

  import TypeIcon from './components/type-icon.vue';

  import type { AppInfoOutputObj } from '~/@types/app';
  import type { AppNavigationType } from '~/config/navigation/app';

  const appDetailStore = useAppDetail();
  const route = useRoute();
  const router = useRouter();
  const spaceStore = useSpaceStore();

  const currentApplicationName = ref(
    (Array.isArray(route.params.name) ? route.params.name[0] : route.params.name) || '',
  );
  const applicationList = ref<AppInfoOutputObj[]>([]);
  // 'overview' | 'build' | 'repo' | 'deploy' | 'info' | 'orchestrate' | 'history' | 'module';
  // trpc没有的子菜单
  const TrpcSpecNotHas = ['orchestrate', 'network'];
  // helm没有的子菜单
  const HelmSpecNotHas = ['module', 'observation', 'polaris', 'appConfig'];

  const activeKey = ref<string>(
    (Array.isArray(router.currentRoute.value.params.menuName)
      ? router.currentRoute.value.params.menuName[0]
      : router.currentRoute.value.params.menuName) || 'info',
  );
  const isSpacePopoverShow = ref(false);

  const currentApplication = computed(() =>
    applicationList.value.find(item => item.name === currentApplicationName.value),
  );
  // 优先使用当前应用的 type，确保切换应用时能立即响应
  // 只有在应用信息还未加载时才使用路由参数中的 type
  const type = computed(() => {
    const appType = currentApplication.value?.type;
    const routeType = router.currentRoute.value.params.type as string;
    return (appType || routeType || '') as AppNavigationType;
  });
  const menuList = computed(() => getMenuList(type.value));
  const routerViewKey = computed(() => {
    const { menuName, name, space, type } = router.currentRoute.value.params;
    return `${menuName}-${name}-${space}-${type}`;
  });

  // 根据当前菜单项的 meta 配置动态设置样式
  const routerViewClass = computed(() => {
    const defaultClass = 'min-h-full px-[24px] py-[20px]';
    const { menuName, type: routeType } = router.currentRoute.value.params;

    if (!menuName || !routeType) {
      return defaultClass;
    }
    const currentMenu = menuList.value
      .flatMap(item => ('children' in item ? item.children : [item]))
      .find(item => item.key === menuName);

    // 无默认 header 页面，返回全屏页面
    if (currentMenu?.meta?.layout === 'empty') {
      return 'min-h-full h-full';
    }

    return currentMenu?.meta?.class || defaultClass;
  });

  // 创建应用
  function goCreateApplication() {
    router.push({
      name: 'createApplication',
      params: {
        space: spaceStore.currentSpace,
      },
    });
  }

  // 获取应用列表
  async function handleGetAppList() {
    if (!spaceStore.currentSpace) return;
    applicationList.value = await ApiServerService.ListApps({
      workspaceID: spaceStore.currentSpace,
    }).catch(() => []);
    if (!applicationList.value.some(item => item.name === currentApplicationName.value)) {
      currentApplicationName.value = applicationList.value[0]?.name || '';
    }
  }

  watch(
    () => router.currentRoute.value.params.menuName,
    newValue => {
      activeKey.value = (Array.isArray(newValue) ? newValue[0] : newValue) || 'info';
    },
  );

  // 切换命名空间后，返回应用管理页
  watch(
    () => spaceStore.currentSpace,
    newValue => {
      router.push({
        name: 'app',
        params: {
          space: newValue,
        },
      });
    },
  );

  watch(
    [activeKey, type, currentApplicationName],
    (newValue, oldValue) => {
      const [key, type, name] = newValue;
      const [oldKey] = oldValue || [];
      if (name) {
        appDetailStore.updateAppName(name);
        appDetailStore.updateAppType(type);
      }
      // 如果是trpc或者helm，且key在notHas中，跳转到基本信息
      if (type === 'trpc' && TrpcSpecNotHas.includes(key as string)) {
        activeKey.value = 'info';
      } else if (isHelmLikeAppType(type) && HelmSpecNotHas.includes(key as string)) {
        activeKey.value = 'info';
      } else {
        if (key && type && name) {
          // 只在真正切换菜单时清理 activeTab 参数，初始化时保留
          const shouldClearActiveTab = oldKey && oldKey !== key;
          const { activeTab, ...query } = router.currentRoute.value.query;

          router.push({
            name: 'detail',
            params: {
              type,
              name,
              menuName: key as string,
            },
            query: shouldClearActiveTab ? query : router.currentRoute.value.query,
          });
        }
      }
    },
    {
      immediate: true,
    },
  );

  // 监听应用变化，更新应用详情
  watch(currentApplication, async app => {
    appDetailStore.updateAppID(app?.id || '');
    await appDetailStore.fetchAppDetail(app?.id);
  });

  onMounted(async () => {
    await handleGetAppList();
  });
</script>

<style lang="postcss" scoped>
  :deep(.bk-select .bk-select-trigger) {
    height: 100%;
  }
</style>
