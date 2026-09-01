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
import { fillFormByType } from '../form.action';

import type AppDetailPage from '../../pages/app-detail.page';
import type BasePage from '../../pages/base.page';

/**
 * 编辑应用配置 - 部署配置 - 资源规格：进入编辑态 → 按 schema 填写 → 保存
 *
 * 注意：`实例数` 因为是 `<Input type="number">`（role=spinbutton），不进入 schema 驱动，
 *       如果需要修改请扩展 `appDetailPage` 新增原子方法。
 */
export async function editResourceSpec(
  { appDetailPage, basePage }: { appDetailPage: AppDetailPage; basePage: BasePage },
  data: Record<string, unknown>,
) {
  await appDetailPage.clickResourceEdit();
  await fillFormByType(basePage, 'ResourceSpec', data as Parameters<typeof fillFormByType<'ResourceSpec'>>[2]);
  await appDetailPage.clickResourceSave();
}

/** 恢复资源规格为默认配置：编辑态点击「恢复默认配置」 */
export async function resetResourceToDefault({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.clickResourceEdit();
  await appDetailPage.clickResourceResetToDefault();
}
