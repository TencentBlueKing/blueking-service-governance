<!--
 - TencentBlueKing is pleased to support the open source community by making
 - 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 - Copyright (C) Tencent. All rights reserved.
 - Licensed under the MIT License (the "License"); you may not use this file except
 - in compliance with the License. You may obtain a copy of the License at
 -
 -  http://opensource.org/licenses/MIT
 -
 - Unless required by applicable law or agreed to in writing, software distributed under
 - the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 - either express or implied. See the License for the specific language governing permissions and
 - limitations under the License.
 -
 - We undertake not to change the open source license (MIT license) applicable
 - to the current version of the project delivered to anyone in the future.
-->

<template>
  <DeployedAppsWarning
    v-model:is-show="showDeployedWarning"
    :deployed-apps="deployedApps"
    :dialog-title="$t('无法删除环境')"
    :tips-text="$t('该环境已部署应用，请先卸载后再删除环境')"
  />
  <DeleteEnvDialog
    v-model:is-show="showConfirm"
    :env-display-name="selectedEnv?.displayName || ''"
    :env-name="selectedEnv?.name || ''"
    @confirm="confirmDelete"
  />
</template>

<script setup lang="ts">
  import { onBeforeUnmount, ref } from 'vue';

  import { Message } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { EnvService } from '~/api/modules/v1';

  import DeleteEnvDialog from '../delete-env-dialog.vue';
  import DeployedAppsWarning from './project-selector/deployed-apps-warning.vue';

  import type { EnvAppDeployStatusOutput, EnvOutput } from '~/@types/v1/env';

  const emit = defineEmits<{ deleted: [] }>();
  const { t } = useI18n();
  const selectedEnv = ref<EnvOutput>();
  const deployedApps = ref<EnvAppDeployStatusOutput[]>([]);
  const showDeployedWarning = ref(false);
  const showConfirm = ref(false);
  let sequence = 0;
  let deleting = false;

  async function confirmDelete() {
    if (!selectedEnv.value?.id || deleting) return;
    const current = sequence;
    deleting = true;
    try {
      await EnvService.deleteEnv({ envID: selectedEnv.value.id });
      if (current !== sequence) return;
      showConfirm.value = false;
      Message({ message: t('删除成功'), theme: 'success' });
      emit('deleted');
    } catch {
      // 默认请求拦截器展示错误，保留确认框供重试。
    } finally {
      deleting = false;
    }
  }

  async function show(env: EnvOutput) {
    if (!env.id || deleting) return;
    const current = ++sequence;
    selectedEnv.value = env;
    showConfirm.value = false;
    showDeployedWarning.value = false;
    try {
      const detail = await EnvService.getEnv({ envID: env.id });
      if (current !== sequence) return;
      if (detail.appDeployStatuses?.length) {
        deployedApps.value = detail.appDeployStatuses;
        showDeployedWarning.value = true;
      } else {
        showConfirm.value = true;
      }
    } catch {
      if (current === sequence) Message({ message: t('获取环境详情失败，无法验证部署状态'), theme: 'error' });
    }
  }

  onBeforeUnmount(() => {
    ++sequence;
  });
  defineExpose({ show });
</script>
