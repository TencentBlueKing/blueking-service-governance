/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

import { setupLayouts } from 'virtual:generated-layouts';
import { createRouter, createWebHashHistory } from 'vue-router';
import CustomRouterComponent from '~/components/custom-router-component.vue';
import Forbidden from '~/pages/app/403.vue';
import NotFound from '~/pages/app/404.vue';
import Application from '~/pages/application/application.vue';
import createApplication from '~/pages/application/create.vue';
import detail from '~/pages/application/detail.vue';
import createHelmTemplateApp from '~/pages/application/template/helm-chart/index.vue';
import templateAppIndex from '~/pages/application/template/index.vue';
import createTAFTemplate from '~/pages/application/template/taf/index.vue';
import createTRPCTemplate from '~/pages/application/template/trpc/index.vue';
import SpaceLayout from '~/pages/basic/space-layout.vue';
import clusterPortPool from '~/pages/env/cluster-components/port-pool/index.vue';
import clusterHealthDiagnosis from '~/pages/env/cluster-health-diagnosis/cluster-health-diagnosis.vue';
import EnvManage from '~/pages/env/env.vue';
import HomeView from '~/pages/home/home.vue';
import SpaceListView from '~/pages/home/space-list.vue';
import Marketplace from '~/pages/marketplace/marketplace.vue';
import PlatformLayout from '~/pages/platform/platform-layout.vue';
import PlatformWorkspaceDetail from '~/pages/platform/workspace/detail.vue';
import PluginLayout from '~/pages/plugin/plugin-layout.vue';
import { useSpaceStore } from '~/stores/space';

import type { RouteLocationNormalized, RouteLocationRaw } from 'vue-router';
import type { UserModule } from '~/types';

const routes = setupLayouts([
  {
    path: '/',
    redirect: 'home',
    children: [
      {
        path: '/home',
        name: 'home',
        component: HomeView,
        meta: {
          // 指定界面布局
          layout: 'main',
        },
      },
      {
        path: '/space-list',
        name: 'spaceList',
        component: SpaceListView,
        meta: {
          layout: 'main',
        },
      },
      {
        path: ':space/app/:envName?',
        name: 'app',
        props: route => ({ ...route.query, ...route.params }),
        component: Application,
        meta: {
          layout: 'empty',
          menuId: 'APP',
        },
      },
      {
        path: ':space/app/:name',
        name: 'appNavigation',
        component: detail,
        meta: {
          layout: 'empty',
          menuId: 'APP',
        },
        children: [
          {
            path: ':type/:menuName',
            name: 'detail',
            component: CustomRouterComponent,
            meta: {
              menuId: 'APP',
            },
          },
        ],
      },
      {
        path: ':space/app/create',
        component: createApplication,
        meta: {
          menuId: 'APP',
        },
        children: [
          {
            path: '',
            name: 'createApplication',
            component: templateAppIndex,
            meta: {
              menuId: 'APP',
            },
          },
          {
            path: 'trpc',
            name: 'createTrpcTemplateApp',
            component: createTRPCTemplate,
            meta: {
              menuId: 'APP',
            },
          },
          {
            path: 'helm',
            name: 'createHelmTemplateApp',
            component: createHelmTemplateApp,
            meta: {
              menuId: 'APP',
            },
          },
          {
            path: 'taf',
            name: 'createTAFTemplateApp',
            component: createTAFTemplate,
            meta: {
              menuId: 'APP',
            },
          },
          {
            path: 'agones',
            name: 'createAgonesTemplateApp',
            component: createHelmTemplateApp,
            meta: {
              menuId: 'APP',
            },
          },
        ],
      },
      {
        path: ':space/env',
        name: 'env',
        props: true,
        component: EnvManage,
        meta: {
          layout: 'content',
          menuId: 'ENV',
        },
      },
      {
        path: ':space/env/:envId/health',
        name: 'clusterHealthDiagnosis',
        props: true,
        component: clusterHealthDiagnosis,
        meta: {
          menuId: 'ENV',
        },
      },
      {
        path: ':space/env/:envId/port-pool',
        name: 'clusterPortPool',
        props: true,
        component: clusterPortPool,
        meta: {
          menuId: 'ENV',
        },
      },
      {
        path: ':space/component',
        name: 'component',
        props: route => ({ ...route.query, ...route.params }),
        component: Marketplace,
        meta: {
          layout: 'content',
          menuId: 'COMPONENT',
        },
      },
      {
        path: ':space/basic',
        name: 'basic',
        props: true,
        component: SpaceLayout,
        redirect: to => ({
          name: 'basicItem',
          params: { ...to.params, menuName: 'info' },
        }),
        meta: {
          layout: 'empty',
          menuId: 'BASIC',
        },
        children: [
          {
            path: ':menuName',
            name: 'basicItem',
            component: CustomRouterComponent,
            meta: {
              layout: 'content',
              menuId: 'BASIC',
            },
          },
        ],
      },
      {
        path: ':space/plugin',
        name: 'plugin',
        props: true,
        component: PluginLayout,
        redirect: to => ({
          name: 'componentList',
          params: { ...to.params, menuName: 'component' },
        }),
        meta: {
          layout: 'empty',
          menuId: 'PLUGIN',
        },
        children: [
          {
            path: ':menuName',
            name: 'componentList',
            component: CustomRouterComponent,
            meta: {
              layout: 'empty',
              menuId: 'PLUGIN',
            },
          },
        ],
      },
      {
        path: '/platform',
        name: 'platform',
        props: true,
        component: PlatformLayout,
        redirect: to => ({
          name: 'platformItem',
          params: { ...to.params, menuName: 'workspace' },
        }),
        meta: {
          layout: 'empty',
          menuId: 'PLATFORM',
        },
        children: [
          {
            path: 'workspace/:workspaceID',
            name: 'platformWorkspaceDetail',
            props: true,
            component: PlatformWorkspaceDetail,
            meta: {
              layout: 'empty',
              menuId: 'PLATFORM',
            },
          },
          {
            path: ':menuName',
            name: 'platformItem',
            component: CustomRouterComponent,
            meta: {
              layout: 'content',
              menuId: 'PLATFORM',
            },
          },
        ],
      },
    ],
  },
  // 无权限页面使用空布局，避免展示依赖当前空间权限的导航内容。
  {
    path: '/403',
    name: '403',
    component: Forbidden,
    meta: {
      layout: 'empty',
    },
  },
  { path: '/:pathMatch(.*)*', name: '404', component: NotFound },
]);

// Setup router
// https://router.vuejs.org/zh/guide/
export const install: UserModule = ({ app }) => {
  const router = createRouter({
    history: createWebHashHistory(import.meta.env.BK_SITE_URL),
    routes,
  });

  // ---- 覆写 router.back()，内置智能返回逻辑 ----
  const originalBack = router.back.bind(router);

  /**
   * 从 route.matched 自动推导上级页面路由
   * 1. 取 matched 倒数第二项作为 parent
   * 2. parent 有 name → 直接用
   * 3. parent 无 name → 找默认子路由（path=''）的 name
   * 4. 都找不到 → 返回 undefined
   */
  function resolveParent(route: RouteLocationNormalized): RouteLocationRaw | undefined {
    const { matched } = route;
    if (matched.length < 2) return undefined;

    const parent = matched[matched.length - 2];

    if (parent.name) {
      return { name: parent.name, params: route.params };
    }

    const defaultChild = parent.children?.find(c => c.path === '' && c.name);
    if (defaultChild?.name) {
      return { name: defaultChild.name, params: route.params };
    }

    return undefined;
  }

  /**
   * 智能返回核心逻辑
   * - 有浏览历史时使用 originalBack()，体验更自然
   * - 无历史时优先使用 fallback > 自动推导 parent > originalBack()
   */
  function smartGoBack(fallback?: RouteLocationRaw) {
    const hasHistory = window.history.state?.back != null;
    if (hasHistory) {
      originalBack();
    } else {
      const fb = fallback ?? resolveParent(router.currentRoute.value);
      if (fb) {
        router.replace(fb);
      } else {
        originalBack();
      }
    }
  }

  // 覆写 router.back()：在实例上遮盖原型方法，支持可选 fallback 参数
  router.back = smartGoBack;

  // 挂载 router.originalBack()：保留原始 back() 能力，跳过智能逻辑
  router.originalBack = originalBack;
  // ---- 覆写结束 ----

  // 全局前置守卫
  router.beforeEach(async (to, _from, next) => {
    // 访问空间路由时
    if (to.params.space) {
      const spaceStore = useSpaceStore();

      let list = spaceStore.list;
      if (list.length === 0) {
        console.log('empty space list');
        list = await spaceStore.handleGetWorkspaceList();
      }
      const space = list.find(item => item.id === to.params.space);
      // 空间不在可访问列表中时，由 403 页面通过详情接口进一步区分无权限和不存在
      if (!space) {
        console.log('space is not in the accessible list');
        next({
          name: '403',
          query: {
            redirect: to.fullPath,
            workspaceID: String(to.params.space),
          },
        });
      } else if (space?.state !== spaceStore.spaceState.Ready) {
        console.log('space not ready');
        next({ name: '404' });
      } else {
        next();
      }
    } else {
      next();
    }
  });
  app.use(router);
};
