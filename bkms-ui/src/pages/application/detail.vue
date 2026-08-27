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
      v-bkloading="{ loading: detailLoading }"
      class="h-full min-h-full"
    >
      <RouterView
        v-if="!detailLoading"
        :key="routerViewKey"
        :class="routerViewClass"
      >
      </RouterView>
    </div>
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
  const TrpcSpecNotHas = ['orchestrate'];
  // helm没有的子菜单
  const HelmSpecNotHas = ['module', 'observation', 'polaris', 'appConfig'];

  const activeKey = ref<string>(
    (Array.isArray(router.currentRoute.value.params.menuName)
      ? router.currentRoute.value.params.menuName[0]
      : router.currentRoute.value.params.menuName) || 'info',
  );
  const isSpacePopoverShow = ref(false);
  // 应用列表与应用详情加载完成前，不挂载子页面，避免子页面读取到空 appID/appDetail
  const detailLoading = ref(true);

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

  /**
   * 切应用 / 切菜单时的路由同步 Promise。
   * 子页会先因 detailLoading 卸载（避免 routerViewKey 变化导致先挂一次再卸），
   * 再 await 本 Promise，保证 URL 更新完成后再拉详情并重新挂载。
   */
  let pendingRouteSync: null | Promise<unknown> = null;

  watch(
    [activeKey, type, currentApplicationName],
    ([key, type, name], [oldKey]) => {
      if (name) {
        appDetailStore.updateAppName(name);
        appDetailStore.updateAppType(type);
      }
      // 如果是trpc或者helm，且key在notHas中，跳转到基本信息
      if (type === 'trpc' && TrpcSpecNotHas.includes(key as string)) {
        activeKey.value = 'info';
      } else if (isHelmLikeAppType(type) && HelmSpecNotHas.includes(key as string)) {
        activeKey.value = 'info';
      } else if (key && type && name) {
        // 同菜单内切换应用沿用 query：让新应用继承当前 Tab 等页面状态；跨菜单切换则重置为默认
        const isMenuSwitch = oldKey && oldKey !== key;
        // 快照当前 query 供新页 hook（useUrlQuerySync）接管：watch 同步阶段已固化进导航参数（快照），
        // 旧页卸载时 hook 会清理自身字段，新页挂载时按该快照写回恢复，保证状态正确继承且不残留旧页字段
        const snapshotQuery = router.currentRoute.value.query;
        pendingRouteSync = router
          .push({
            name: 'detail',
            params: {
              type,
              name,
              menuName: key as string,
            },
            query: isMenuSwitch ? undefined : snapshotQuery,
          })
          .catch(() => undefined);
      }
    },
    {
      immediate: true,
    },
  );

  // 监听应用变化，更新应用详情；加载完成前不挂载子页面，避免竞态
  watch(currentApplication, async app => {
    const currentAppId = app?.id || '';
    // 先置 loading 卸载子页：若等 push 完成后再 loading，routerViewKey 会先变并挂载一次，
    // 随后 loading 再卸/挂，网络访问等页接口会打两次。
    detailLoading.value = true;
    // 再等路由 push 落定（此时子页已卸，useUrlQuerySync 不再 replace 打断 push）
    if (pendingRouteSync) {
      await pendingRouteSync;
      pendingRouteSync = null;
    }
    appDetailStore.updateAppID(currentAppId);
    try {
      await appDetailStore.fetchAppDetail(currentAppId);
    } finally {
      // 快速切换应用 A→B 时两个 async 回调并行，仅当当前仍为本次应用时才解锁，
      // 避免 A 先返回提前解锁读到 A 数据（串台）。
      // 基于 appID（watch 内同步更新）判断：请求失败/清空选择时 appDetail 为 null，
      // 用 appID 判断可正常解锁，不会卡死在 loading 态。
      // 过期请求的覆盖由 store 层 fetchAppDetail 的 appID 校验丢弃，这里只需负责解锁。
      if (appDetailStore.appID === currentAppId) {
        detailLoading.value = false;
      }
    }
  });

  onMounted(async () => {
    await handleGetAppList();
    // 列表已加载但未匹配到应用（如空列表），无需再等待详情
    if (!currentApplication.value) {
      detailLoading.value = false;
    }
  });
</script>

<style lang="postcss" scoped>
  :deep(.bk-select .bk-select-trigger) {
    height: 100%;
  }
</style>
