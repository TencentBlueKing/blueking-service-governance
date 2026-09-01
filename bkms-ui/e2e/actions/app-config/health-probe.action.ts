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

const LIVENESS_PROBE_LABEL = '存活探针';

export const HEALTH_PROBE_LABELS = ['存活探针', '就绪探针', '启动探针'] as const;
export type HealthProbeLabel = (typeof HEALTH_PROBE_LABELS)[number];

type HealthProbeConfig = {
  cancelPath: string;
  savedPath: string;
  savedPort: string;
};

export const HEALTH_PROBE_CANCEL_PATH = '/e2e-cancel-healthz';
export const HEALTH_PROBE_CLAMPED_PORT = '65535';
export const HEALTH_PROBE_INVALID_PORT = '70000';
export const HEALTH_PROBE_SAVED_PATH = '/e2e-healthz';
export const HEALTH_PROBE_SAVED_PORT = '18080';

const HEALTH_PROBE_CONFIG_MAP: Record<HealthProbeLabel, HealthProbeConfig> = {
  存活探针: {
    cancelPath: HEALTH_PROBE_CANCEL_PATH,
    savedPath: HEALTH_PROBE_SAVED_PATH,
    savedPort: HEALTH_PROBE_SAVED_PORT,
  },
  就绪探针: {
    cancelPath: '/e2e-cancel-readyz',
    savedPath: '/e2e-readyz',
    savedPort: '18081',
  },
  启动探针: {
    cancelPath: '/e2e-cancel-startupz',
    savedPath: '/e2e-startupz',
    savedPort: '18082',
  },
};

/** 编辑健康探针后取消：用于验证取消不会把编辑内容带回查看态 */
export async function editHealthProbeAndCancel({
  appDetailPage,
  label,
}: {
  appDetailPage: AppDetailPage;
  label: string;
}) {
  const probeLabel = resolveHealthProbeLabel(label);
  const config = getHealthProbeConfig(probeLabel);
  await appDetailPage.clickHealthProbeEdit(probeLabel);
  await appDetailPage.fillHealthProbeHttpConfig(probeLabel, {
    port: config.savedPort,
    url: config.cancelPath,
  });
  await appDetailPage.clickHealthProbeCancel(probeLabel);
}

/** 编辑存活探针后取消：用于验证取消不会把编辑内容带回查看态 */
export async function editLivenessProbeAndCancel({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await editHealthProbeAndCancel({ appDetailPage, label: LIVENESS_PROBE_LABEL });
}

export function getHealthProbeConfig(label: string) {
  return HEALTH_PROBE_CONFIG_MAP[resolveHealthProbeLabel(label)];
}

/** 保存有效健康探针配置：用于验证 PUT 保存链路与查看态回显 */
export async function saveValidHealthProbe({ appDetailPage, label }: { appDetailPage: AppDetailPage; label: string }) {
  const probeLabel = resolveHealthProbeLabel(label);
  const config = getHealthProbeConfig(probeLabel);
  await appDetailPage.fillHealthProbeHttpConfig(probeLabel, {
    port: config.savedPort,
    url: config.savedPath,
  });
  await appDetailPage.clickHealthProbeSaveAndWait(probeLabel);
}

/** 保存有效存活探针配置：用于验证 PUT 保存链路与查看态回显 */
export async function saveValidLivenessProbe({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await saveValidHealthProbe({ appDetailPage, label: LIVENESS_PROBE_LABEL });
}

/** 提交无效健康探针配置：用于验证路径为空与端口超限校验 */
export async function submitInvalidHealthProbe({
  appDetailPage,
  label,
}: {
  appDetailPage: AppDetailPage;
  label: string;
}) {
  const probeLabel = resolveHealthProbeLabel(label);
  await appDetailPage.clickHealthProbeEdit(probeLabel);
  await appDetailPage.fillHealthProbeHttpConfig(probeLabel, {
    port: HEALTH_PROBE_INVALID_PORT,
    url: '',
  });
  await appDetailPage.clickHealthProbeSave(probeLabel);
}

/** 提交无效存活探针配置：用于验证路径为空与端口超限校验 */
export async function submitInvalidLivenessProbe({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await submitInvalidHealthProbe({ appDetailPage, label: LIVENESS_PROBE_LABEL });
}

function resolveHealthProbeLabel(label: string): HealthProbeLabel {
  if (HEALTH_PROBE_LABELS.includes(label as HealthProbeLabel)) {
    return label as HealthProbeLabel;
  }
  throw new Error(`不支持的健康探针：${label}`);
}
