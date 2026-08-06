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

// import alertLog from '~/pages/application/detail/alert-log.vue';
import buildManagement from '~/pages/application/detail/app-build/build-management.vue';
import appConfig from '~/pages/application/detail/app-config/index.vue';
import artifact from '~/pages/application/detail/artifact/index.vue';
import orchestrate from '~/pages/application/detail/base-info/helm/app-orchestrate.vue';
import helmInfo from '~/pages/application/detail/base-info/helm/helm-info.vue';
import trpcInfo from '~/pages/application/detail/base-info/trpc/index.vue';
import trpcDeploy from '~/pages/application/detail/deploy/deploy.vue';
import helmDeploy from '~/pages/application/detail/helm-deploy/index.vue';
import modulesConfig from '~/pages/application/detail/modules-config.vue';
import networkAccess from '~/pages/application/detail/network-access/index.vue';
import observation from '~/pages/application/detail/observation.vue';
import operationHistory from '~/pages/application/detail/operation-history/index.vue';
// import overview from '~/pages/application/detail/overview.vue';
import polaris from '~/pages/application/detail/polaris/polaris.vue';

import { i18n } from '../../modules/i18n';

import type { NavigationItem } from './types';
import type { AppType } from '~/composables/app-type';

/**
 * 应用导航类型
 */
export type AppNavigationType = AppType;

/**
 * TRPC 应用导航配置
 */
export const TRPC_NAVIGATION: NavigationItem[] = [
  // HIDE，等待后续页面开发后再展示
  // {
  //   key: 'overview',
  //   name: i18n.global.t('概览'),
  //   icon: 'cc-home',
  //   component: overview,
  // },
  {
    key: 'buildGroup',
    name: i18n.global.t('构建'),
    foldName: i18n.global.t('构建'),
    children: [
      {
        key: 'build',
        name: i18n.global.t('构建管理'),
        icon: 'chuangjianyingyong',
        component: buildManagement,
        meta: {
          // 无需默认 Header
          layout: 'empty',
        },
      },
      {
        key: 'artifact',
        name: i18n.global.t('制品管理'),
        icon: 'artifact-management',
        component: artifact,
        meta: {
          layout: 'empty',
        },
      },
    ],
  },
  {
    key: 'deploymentGroup',
    name: i18n.global.t('部署'),
    foldName: i18n.global.t('部署'),
    children: [
      {
        key: 'deployment',
        name: i18n.global.t('部署管理'),
        icon: 'yingyongbushu',
        component: trpcDeploy,
        meta: {
          layout: 'empty',
        },
      },
    ],
  },
  {
    key: 'observationGroup',
    name: i18n.global.t('可观测'),
    foldName: i18n.global.t('观测'),
    children: [
      {
        key: 'observation',
        name: i18n.global.t('观测数据'),
        icon: 'monitor',
        component: observation,
      },
      // {
      //   key: 'alert',
      //   name: i18n.global.t('告警记录'),
      //   icon: 'alert-records',
      //   component: alertLog,
      // },
    ],
  },
  {
    key: 'configGroup',
    name: i18n.global.t('配置'),
    foldName: i18n.global.t('配置'),
    children: [
      {
        key: 'polaris',
        name: i18n.global.t('北极星'),
        icon: 'polaris',
        component: polaris,
      },
      {
        key: 'module',
        name: i18n.global.t('组件配置'),
        icon: 'component-configuration',
        component: modulesConfig,
      },
      {
        key: 'appConfig',
        name: i18n.global.t('应用配置'),
        icon: 'setting-fill',
        component: appConfig,
        meta: {
          layout: 'empty',
        },
      },
      {
        key: 'info',
        name: i18n.global.t('基本信息'),
        icon: 'basic-info',
        component: trpcInfo,
      },
    ],
  },
  {
    key: 'other',
    name: i18n.global.t('其他'),
    foldName: i18n.global.t('其他'),
    children: [
      {
        key: 'history',
        name: i18n.global.t('操作记录'),
        icon: 'historical-tasks',
        component: operationHistory,
      },
    ],
  },
];

/**
 * Helm 应用导航配置
 */
export const HELM_NAVIGATION: NavigationItem[] = [
  // HIDE，等待后续页面开发后再展示
  // {
  //   key: 'overview',
  //   name: i18n.global.t('概览'),
  //   icon: 'cc-home',
  //   component: overview,
  // },
  {
    key: 'buildGroup',
    name: i18n.global.t('构建'),
    foldName: i18n.global.t('构建'),
    children: [
      {
        key: 'build',
        name: i18n.global.t('构建管理'),
        icon: 'chuangjianyingyong',
        component: buildManagement,
        meta: {
          layout: 'empty',
        },
      },
      {
        key: 'artifact',
        name: i18n.global.t('制品管理'),
        icon: 'artifact-management',
        component: artifact,
        meta: {
          layout: 'empty',
        },
      },
    ],
  },
  {
    key: 'deploymentGroup',
    name: i18n.global.t('部署'),
    foldName: i18n.global.t('部署'),
    children: [
      {
        key: 'deployment',
        name: i18n.global.t('部署管理'),
        icon: 'yingyongbushu',
        component: helmDeploy,
      },
    ],
  },
  {
    key: 'configGroup',
    name: i18n.global.t('配置'),
    foldName: i18n.global.t('配置'),
    children: [
      {
        key: 'orchestrate',
        name: i18n.global.t('应用编排'),
        icon: 'orchestration',
        component: orchestrate,
        meta: {
          class: 'h-full',
        },
      },
      {
        key: 'network',
        name: i18n.global.t('网络访问'),
        icon: 'wangye',
        component: networkAccess,
        meta: {
          // 详情页无需默认 Header
          layout: 'empty',
        },
      },
      {
        key: 'info',
        name: i18n.global.t('基本信息'),
        icon: 'basic-info',
        component: helmInfo,
      },
    ],
  },
  {
    key: 'other',
    name: i18n.global.t('其他'),
    foldName: i18n.global.t('其他'),
    children: [
      {
        key: 'history',
        name: i18n.global.t('操作记录'),
        icon: 'historical-tasks',
        component: operationHistory,
      },
    ],
  },
];

/**
 * 应用导航配置映射
 */
export const appNavigationConfig: Record<AppNavigationType, NavigationItem[]> = {
  taf: TRPC_NAVIGATION,
  trpc: TRPC_NAVIGATION,
  helm: HELM_NAVIGATION,
  agones: HELM_NAVIGATION,
};
