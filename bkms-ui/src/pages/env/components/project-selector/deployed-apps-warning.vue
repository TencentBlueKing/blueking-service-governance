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
  <Dialog
    v-model:is-show="isShow"
    :width="480"
    @closed="handleClose"
  >
    <template #header>
      <div class="flex flex-col items-center pt-[10px]">
        <SvgIcon
          :height="42"
          icon="bkms-icon-tishi"
          :width="42"
        />
        <span class="text-[#313238] text-[20px] leading-[32px] text-center mt-[18px]">
          {{ dialogTitle }}
        </span>
      </div>
    </template>
    <div class="flex flex-col items-center text-[12px] text-[#4D4F56]">
      <p class="h-[34px] w-full !text-start">
        {{ tipsText }}
      </p>
      <DeployedAppsList
        class="mb-[48px] w-full"
        :deployed-apps="deployedApps"
      />
      <Button
        class="w-[88px]"
        @click="handleClose"
      >
        {{ $t('关闭') }}
      </Button>
    </div>
    <template #footer> </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { Button, Dialog } from 'bkui-vue';

  import DeployedAppsList from './deployed-apps-list.vue';

  import type { EnvAppDeployStatusOutput } from '~/@types/v1/env';

  const isShow = defineModel<boolean>('isShow');

  defineProps<{
    deployedApps: EnvAppDeployStatusOutput[];
    dialogTitle: string;
    tipsText: string;
  }>();

  function handleClose() {
    isShow.value = false;
  }
</script>

<style lang="less" scoped>
  :deep(.bk-modal-body) {
    .bk-dialog-content {
      margin-top: 30px;
      margin-bottom: 18px;
    }
    .bk-modal-footer {
      display: none;
    }
  }
</style>
