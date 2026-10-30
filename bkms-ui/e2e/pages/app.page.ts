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

import BasePage from './base.page';

export default class DeployPage extends BasePage {
  async chooseAppType(label: string) {
    const labels: Record<string, string> = {
      TRPC: 'tRPC',
      'tRPC 应用': 'tRPC',
      'Helm 应用': 'Helm Chart',
      'Agones 应用': 'Agones',
    };
    const templateLabel = labels[label] || label;
    await this.page
      .getByRole('button', { name: new RegExp(templateLabel) })
      .first()
      .click();
  }

  async goto() {
    await this.page.goto('/#/' + this.config.space + '/app');
  }
}
