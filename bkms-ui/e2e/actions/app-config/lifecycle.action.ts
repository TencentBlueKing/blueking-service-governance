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

export const LIFECYCLE_CANCEL_COMMAND = 'echo e2e-cancel-lifecycle';
export const LIFECYCLE_SAVED_COMMAND = 'echo e2e-lifecycle';
export const LIFECYCLE_SAVED_GRACE_PERIOD = '45';

/** 编辑生命周期后取消：用于验证取消不会把草稿带回查看态 */
export async function editLifecycleAndCancel({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.clickLifecycleEdit();
  await appDetailPage.fillLifecycleShellCommandConfig({
    command: LIFECYCLE_CANCEL_COMMAND,
    gracePeriod: LIFECYCLE_SAVED_GRACE_PERIOD,
  });
  await appDetailPage.clickLifecycleCancel();
}

/** 保存有效生命周期配置：用于验证 lifecycle PUT 保存链路与查看态回显 */
export async function saveValidLifecycle({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.fillLifecycleShellCommandConfig({
    command: LIFECYCLE_SAVED_COMMAND,
    gracePeriod: LIFECYCLE_SAVED_GRACE_PERIOD,
  });
  await appDetailPage.clickLifecycleSaveAndWait();
}

/** 选择生命周期自定义命令：用于显式验证 preStop 自定义命令选项与 shell 编辑器展示 */
export async function selectLifecycleCustomCommandOption({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.ensureLifecycleEditMode();
  await appDetailPage.selectLifecycleShellCommandMode();
  await appDetailPage.expectLifecycleCustomCommandEditorVisible();
}

/** 提交无效生命周期配置：用于验证 shell 命令必填校验 */
export async function submitInvalidLifecycle({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.ensureLifecycleEditMode();
  await appDetailPage.fillLifecycleShellCommandConfig({
    command: '',
    gracePeriod: LIFECYCLE_SAVED_GRACE_PERIOD,
  });
  await appDetailPage.clickLifecycleSave();
}
