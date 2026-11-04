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
      <Alert
        v-if="isAppModelAppType(type) && appDetailStore.runtimeError"
        class="mx-[24px] mt-[16px]"
        theme="warning"
      >
        <template #title>
          <div class="flex items-center gap-[12px]">
            <span>{{ $t('应用能力加载失败，部分功能暂不可用') }}</span>
            <Button
              :loading="appDetailStore.runtimeLoading"
              text
              theme="primary"
              @click="appDetailStore.fetchAppRuntimeMods()"
              >{{ $t('重试') }}</Button
            >
          </div>
        </template>
      </Alert>
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
        :custom-content="applicationListLoading"
        filterable
        :min-height="204"
        :placeholder="$t('请选择应用')"
        :popover-min-width="244"
        :popover-options="{ extCls: 'application-selector-popover' }"
        :scroll-height="7 * 36 + 8"
        :search-placeholder="$t('搜索')"
        @toggle="isSpacePopoverShow = $event"
      >
        <template #trigger>
          <div
            class="flex items-center justify-between bg-[#F0F1F5] overflow-hidden group text-[#4D4F56] h-full text-[14px] cursor-pointer rounded-[2px] hover:bg-[#EAEBF0]"
          >
            <div class="flex items-center min-w-[43px] overflow-hidden">
              <TypeIcon
                classes="min-w-[43px] inline-block"
                :framework="iconApplication?.framework"
                :language="iconApplication?.language"
                :show-label="false"
                :type="type"
              />
              <span class="truncate">{{ currentApplicationName }}</span>
            </div>
            <AngleDownFill
              :class="[
                'mr-[5px] shrink-0 text-[#C4C6CC] group-hover:text-[#979BA5]',
                { 'rotate-180': isSpacePopoverShow },
              ]"
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
        <Loading
          v-if="applicationListLoading"
          class="block"
          :loading="true"
        >
          <div class="h-[196px]"></div>
        </Loading>
        <Select.Option
          v-for="item in applicationList"
          v-else
          :id="item.name"
          :key="item.name"
          class="application-selector-option"
          :name="item.name"
        >
          <div class="flex items-center gap-[8px] min-w-0">
            <TypeIcon
              classes="min-w-[20px] inline-block"
              :framework="item.framework"
              :language="item.language"
              :show-label="false"
              :type="item.type"
            />
            <span class="truncate">{{ item.name }}</span>
          </div>
        </Select.Option>
      </Select>
    </template>
  </CustomNavigation>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref, watch } from 'vue';

  import { Alert, Button, Loading, Select } from 'bkui-vue';
  import { AngleDownFill, Plus } from 'bkui-vue/lib/icon';
  import { onBeforeRouteUpdate, useRoute, useRouter } from 'vue-router';
  import { AppService } from '~/api/modules/v1/app';
  import { isAppModelAppType } from '~/composables/app-type';
  import { getMenuList } from '~/composables/use-router-menu';
  import { useAppDetail } from '~/stores/app-detail';
  import { useSpaceStore } from '~/stores/space';

  import TypeIcon from './components/type-icon.vue';

  import type { AppInfoOutputObj } from '~/@types/v1/app';
  import type { AppNavigationType } from '~/config/navigation/app';

  const appDetailStore = useAppDetail();
  const route = useRoute();
  const router = useRouter();
  const spaceStore = useSpaceStore();

  /** 将路由 params 单值/数组统一为 string，缺省回退 fallback */
  function getRouteParam(value: string | string[] | undefined, fallback = ''): string {
    return (Array.isArray(value) ? value[0] : value) || fallback;
  }

  // 切换应用时，在新路由生效前离开特性环境子页面，并清除旧应用的所有查询参数。
  onBeforeRouteUpdate((to, from) => {
    const isSwitchingFeatureEnvApp =
      from.name === 'detail' &&
      to.name === 'detail' &&
      from.query.view === 'feature-envs' &&
      getRouteParam(from.params.menuName) === 'deployment' &&
      getRouteParam(to.params.menuName) === 'deployment' &&
      getRouteParam(from.params.name) !== getRouteParam(to.params.name);
    const isCleanInstanceList = to.query.activeTab === 'instance' && Object.keys(to.query).length === 1;
    if (!isSwitchingFeatureEnvApp || isCleanInstanceList) return;

    return {
      name: 'detail',
      params: to.params,
      query: { activeTab: 'instance' },
      replace: true,
    };
  });

  const currentApplicationName = ref(getRouteParam(route.params.name));
  const applicationList = ref<AppInfoOutputObj[]>([]);
  const applicationListLoading = ref(false);
  const activeKey = ref<string>(getRouteParam(router.currentRoute.value.params.menuName, 'info'));
  const isSpacePopoverShow = ref(false);
  // 应用列表与应用详情加载完成前，不挂载子页面，避免子页面读取到空 appID/appDetail
  const detailLoading = ref(true);

  const currentApplication = computed(() =>
    applicationList.value.find(item => item.name === currentApplicationName.value),
  );
  // 首屏图标复用站内跳转前已知的信息，列表返回后仍以接口数据为准。
  const iconApplication = computed(() => {
    if (currentApplication.value) return currentApplication.value;
    const app = appDetailStore.navigationApp;
    const workspaceID = getRouteParam(router.currentRoute.value.params.space, spaceStore.currentSpace);
    if (
      app?.workspaceID === workspaceID &&
      app?.name === currentApplicationName.value &&
      app?.id === appDetailStore.appID
    ) {
      return app;
    }
    return undefined;
  });
  // 当前应用详情是真实类型来源；切换过程先使用列表中的类型。
  const type = computed(() => {
    const detail = appDetailStore.appDetail;
    const appType =
      detail?.id && detail.id === currentApplication.value?.id ? detail.type : currentApplication.value?.type;
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
    applicationListLoading.value = true;
    try {
      applicationList.value = await AppService.listApps({
        workspaceID: spaceStore.currentSpace,
      }).catch(() => []);
      if (!applicationList.value.some(item => item.name === currentApplicationName.value)) {
        currentApplicationName.value = applicationList.value[0]?.name || '';
      }
    } finally {
      applicationListLoading.value = false;
    }
  }

  watch(
    () => router.currentRoute.value.params.menuName,
    newValue => {
      activeKey.value = getRouteParam(newValue, 'info');
    },
  );

  // 浏览器后退/前进或外链直达时，将 URL 中的应用名同步回 Select，避免 UI 与地址栏不一致
  watch(
    () => router.currentRoute.value.params.name,
    newValue => {
      const name = getRouteParam(newValue);
      if (name && name !== currentApplicationName.value) {
        currentApplicationName.value = name;
      }
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
  /**
   * 应用切换世代号：每次 currentApplication 变化递增。
   * 快速连切时旧 async 回调在 await 后对照本值，过期则直接退出，避免抢写 appID。
   */
  let appSwitchGeneration = 0;

  watch(
    [activeKey, type, currentApplicationName, () => appDetailStore.runtime, currentApplication],
    ([key, type, name], [oldKey, oldType, oldName]) => {
      if (name) {
        appDetailStore.updateAppName(name);
        appDetailStore.updateAppType(type);
      }
      if (!key || !type || !name) return;
      const current = router.currentRoute.value;
      const isSwitchingApp = getRouteParam(current.params.name) !== name;
      // 观测入口依赖异步能力，目标应用的能力未确认前保留当前菜单。
      const waitingForObservationRuntime =
        key === 'observation' &&
        isAppModelAppType(type) &&
        (appDetailStore.appID !== currentApplication.value?.id || !appDetailStore.runtime);
      const hasMenu = menuList.value.some(item =>
        'children' in item ? item.children.some(child => child.key === key) : item.key === key,
      );
      const shouldFallback = !!currentApplication.value && !waitingForObservationRuntime && !hasMenu;
      const menuName = shouldFallback ? 'info' : key;

      // 仅能力变化时不重复导航，也不取消尚未完成的应用切换。
      if (key === oldKey && type === oldType && name === oldName && (isSwitchingApp || !shouldFallback)) return;
      const alreadyOnTarget =
        current.name === 'detail' &&
        !isSwitchingApp &&
        getRouteParam(current.params.type) === type &&
        getRouteParam(current.params.menuName) === menuName;
      if (alreadyOnTarget) return;

      const isMenuSwitch = oldKey && oldKey !== key;
      const target = {
        name: 'detail',
        params: { type, name, menuName },
        query: shouldFallback ? {} : isMenuSwitch ? undefined : current.query,
      };
      // 切应用直接进入目标有效菜单；纠正当前应用的无效链接时替换历史项。
      pendingRouteSync = (shouldFallback && !isSwitchingApp ? router.replace(target) : router.push(target)).catch(
        () => undefined,
      );
    },
    {
      immediate: true,
    },
  );

  // 监听应用变化，更新应用详情；加载完成前不挂载子页面，避免竞态
  watch(currentApplication, async app => {
    const currentAppId = app?.id || '';
    // 本轮世代号：后续 await 后若已被更新的切换取代，则不再写 store / 解锁
    const generation = ++appSwitchGeneration;
    // 先置 loading 卸载子页：若等 push 完成后再 loading，routerViewKey 会先变并挂载一次，
    // 随后 loading 再卸/挂，网络访问等页接口会打两次。
    detailLoading.value = true;
    // 路由同步也可能等待；先切换应用作用域，立即失效旧详情请求与能力。
    appDetailStore.updateAppID(currentAppId);
    // 捕获本轮看到的 sync Promise；不清空他人的最新 pending，避免快速连切时抢清空
    const routeSync = pendingRouteSync;
    if (routeSync) {
      await routeSync;
      // 仅当全局 pending 仍是我们 await 的那一个时才清空
      if (pendingRouteSync === routeSync) {
        pendingRouteSync = null;
      }
    }
    // 快速连切 A→B→C：过期回调在此退出，避免为旧应用再次发起请求。
    if (generation !== appSwitchGeneration) {
      return;
    }
    try {
      if (isAppModelAppType(app?.type)) void appDetailStore.fetchAppRuntimeMods(currentAppId);
      await appDetailStore.fetchAppDetail(currentAppId);
    } finally {
      // 仅最新一轮且 appID 仍匹配时解锁；过期请求的覆盖由 store 层 appID 校验丢弃
      if (generation === appSwitchGeneration && appDetailStore.appID === currentAppId) {
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

  :global(.application-selector-popover.bk-popover.bk-pop2-content.bk-select-popover) {
    overflow: hidden;
    border-radius: 6px;
  }

  .application-selector-option {
    height: 36px;
    font-size: 14px;
  }
</style>
