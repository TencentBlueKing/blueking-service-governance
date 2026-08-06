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

interface Window {
  readonly BK_API_PREFIX: string;
  readonly BK_BCS: string;
  readonly BK_DOC_URL: string;
  readonly BK_LOGIN_URL: string;
  readonly BK_SHARED_RES_BASE_JS_URL: string;
  // eslint-disable-next-line @typescript-eslint/consistent-type-imports
  i18n: import('vue-i18n').I18nGlobal;
  MonacoEnvironment?: {
    getWorker(workerId: string, label: string): Worker;
  };
}

declare const BK_BKMS_WELCOME: string;

declare const BK_BKMS_VERSION: string;

declare module '*.vue' {
  import type { DefineComponent } from 'vue';

  const component: DefineComponent<object, object, unknown>;
  export default component;
}

declare module '*.svg' {
  import type { DefineComponent } from 'vue';

  const component: DefineComponent;
  export default component;
}

declare module '@blueking/platform-config';

declare module '@blueking/notice-component';

declare module '@blueking/user-selector';

declare module '@blueking/xss-filter';

declare module 'bkui-vue';

declare module 'pluralize';

declare module '@blueking/monitor-vue3-components';

declare module 'highlight.js/lib/languages/*';
