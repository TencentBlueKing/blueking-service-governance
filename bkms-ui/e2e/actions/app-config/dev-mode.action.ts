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

/** 尝试开启开发模式后取消：用于验证取消不会改变开关状态 */
export async function cancelEnableDevMode({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.clickDevModeSwitch();
  await appDetailPage.clickDevModeCancel();
}

/** 关闭开发模式：用于验证 DELETE 保存链路与关闭态回显 */
export async function disableDevMode({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.clickDevModeSwitch();
  await appDetailPage.clickDevModeConfirm('disable');
}

/** 开启开发模式：用于验证 PUT 保存链路与操作步骤展示 */
export async function enableDevMode({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.clickDevModeSwitch();
  await appDetailPage.clickDevModeConfirm('enable');
}

/** 确保开发模式关闭：用于保证用例初始状态稳定 */
export async function ensureDevModeDisabled({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.ensureDevModeDisabled();
}

/** 刷新应用配置页并切回测试环境：用于验证开发模式刷新后仍正确回显 */
export async function reloadAndSelectTestEnv({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.reloadAppConfigAndSelectFirstTestEnv();
}
