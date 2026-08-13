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
  <div class="h-full overflow-hidden">
    <Alert
      class="mb-[16px]"
      theme="info"
      :title="tipText"
    />
    <DeployConfig />
  </div>
</template>

<script setup lang="ts">
  import { Alert } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';

  import DeployConfig from './deploy-config.vue';

  const { t } = useI18n();
  const tipText = t(
    '本页配置只会影响本空间之后新建的应用，已有应用不受影响。仅适用于平台托管部署的应用（如 tRPC、TAF），Helm / Agones 等自定义编排应用不适用。',
  );
</script>
