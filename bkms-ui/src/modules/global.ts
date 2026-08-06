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

import { BkXssFilterDirective } from '@blueking/xss-filter';
import { bkTooltips } from 'bkui-vue';
import { clickoutside } from 'bkui-vue/lib/directives';
import authority from '~/directives/authority';
import test from '~/directives/test';

import type { UserModule } from '~/types';

// 注册全局组件或者指令
export const install: UserModule = ({ app }) => {
  app.directive('bk-tooltips', bkTooltips);
  app.directive('bk-authority', authority);
  app.directive('test', test);
  app.directive('clickoutside', clickoutside);
  app.use(BkXssFilterDirective);
};
