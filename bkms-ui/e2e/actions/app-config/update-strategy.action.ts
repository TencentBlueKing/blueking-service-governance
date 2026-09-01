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
import type AppDetailPage from '../../pages/app-detail.page';

export const UPDATE_STRATEGY_INVALID_VALUE = 'abc';
export const UPDATE_STRATEGY_SAVED_MAX_SURGE = '30%';
export const UPDATE_STRATEGY_SAVED_MAX_UNAVAILABLE = '0';

/** 编辑更新策略后取消：用于验证取消不会把草稿带回查看态 */
export async function editUpdateStrategyAndCancel({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.clickUpdateStrategyEdit();
  await appDetailPage.fillUpdateStrategyConfig({
    maxSurge: UPDATE_STRATEGY_INVALID_VALUE,
    maxUnavailable: UPDATE_STRATEGY_SAVED_MAX_UNAVAILABLE,
  });
  await appDetailPage.clickUpdateStrategyCancel();
}

/** 保存有效更新策略配置：用于验证 update-strategy PUT 保存链路与查看态回显 */
export async function saveValidUpdateStrategy({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.fillUpdateStrategyConfig({
    maxSurge: UPDATE_STRATEGY_SAVED_MAX_SURGE,
    maxUnavailable: UPDATE_STRATEGY_SAVED_MAX_UNAVAILABLE,
  });
  await appDetailPage.clickUpdateStrategySaveAndWait();
}

/** 提交无效更新策略配置：用于验证 maxSurge/maxUnavailable 格式校验 */
export async function submitInvalidUpdateStrategy({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.clickUpdateStrategyEdit();
  await appDetailPage.fillUpdateStrategyConfig({
    maxSurge: UPDATE_STRATEGY_INVALID_VALUE,
    maxUnavailable: UPDATE_STRATEGY_SAVED_MAX_UNAVAILABLE,
  });
  await appDetailPage.clickUpdateStrategySave();
}
