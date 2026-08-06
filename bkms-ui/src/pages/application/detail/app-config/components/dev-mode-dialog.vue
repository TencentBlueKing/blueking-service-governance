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
    footer-align="center"
    :title="isEnabling ? $t('确认开启开发模式？') : $t('确认关闭开发模式？')"
    :width="480"
  >
    <!-- 开启开发模式内容 -->
    <div v-if="isEnabling">
      <p class="mb-[16px]">
        {{ $t('开启后，允许通过 bkms-cli 上传二进制到应用实例中进行调试') }}
      </p>

      <Alert
        class="mb-[16px] font-bold"
        theme="warning"
        :title="$t('请注意')"
      >
        <ul class="list-disc pl-[20px] line-height-[20px] font-normal">
          <li>{{ $t('开启后需到 "部署管理" 执行一次部署才能生效') }}</li>
          <li>{{ $t('该模式会在 Pod 中注入管理脚本') }}</li>
        </ul>
      </Alert>
    </div>

    <!-- 关闭开发模式内容 -->
    <div v-else>
      <p class="mb-[16px]">
        {{ $t('关闭后，开发调试功能将停用') }}
      </p>

      <Alert
        class="mb-[16px] font-bold"
        theme="warning"
        :title="$t('重要提示')"
      >
        <ul class="list-disc pl-[20px] line-height-[20px] font-normal">
          <li>{{ $t('关闭后需到 "部署管理" 再执行一次部署才能彻底关闭') }}</li>
          <li>{{ $t('关闭前请确保已保存重要的调试数据') }}</li>
          <li>{{ $t('Pod 中的管理脚本将在重新部署后移除') }}</li>
        </ul>
      </Alert>

      <div class="bg-[#F5F7FA] p-[12px]">
        <p>
          {{ $t('关闭后，仍需') }}&nbsp;
          <Button
            text
            theme="primary"
            @click="goToDeployment"
          >
            {{ $t('执行部署') }}
            <Share class="ml-[6px]" /> </Button
          >&nbsp;
          {{ $t('方可彻底关闭开发模式') }}
        </p>
      </div>
    </div>
    <template #footer>
      <Button
        theme="primary"
        @click="handleConfirm"
      >
        {{ isEnabling ? $t('确认开启') : $t('确认关闭') }}
      </Button>
      <Button
        class="ml-[8px]"
        @click="handleCancel"
      >
        {{ $t('取消') }}
      </Button>
    </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import { Alert, Button, Dialog } from 'bkui-vue';
  import { Share } from 'bkui-vue/lib/icon';
  import { useRouter } from 'vue-router';
  import { useAppDetail } from '~/stores/app-detail';
  import { useDeployEnvStore } from '~/stores/deploy-env';

  interface Emits {
    (e: 'update:modelValue', value: boolean): void;
    (e: 'confirm'): void;
    (e: 'cancel'): void;
  }

  interface Props {
    envName?: string;
    isEnabling: boolean;
    modelValue: boolean;
  }

  const props = defineProps<Props>();
  const emit = defineEmits<Emits>();
  const router = useRouter();
  const appDetailStore = useAppDetail();
  const deployEnvStore = useDeployEnvStore();

  const isShow = computed({
    get: () => props.modelValue,
    set: value => emit('update:modelValue', value),
  });

  // 跳转到部署管理
  function goToDeployment() {
    if (props.envName) {
      deployEnvStore.updateCurrentEnv(props.envName);
    }

    router.push({
      name: 'detail',
      params: {
        name: appDetailStore.appID,
        menuName: 'deployment',
        type: 'trpc',
      },
    });
  }

  const handleConfirm = () => {
    emit('confirm');
  };

  const handleCancel = () => {
    emit('cancel');
  };
</script>

<style lang="postcss" scoped>
  :deep(.bk-dialog-title) {
    text-align: center !important;
  }
  :deep(.bk-dialog-footer) {
    padding-top: 0;
    padding-bottom: 16px;
    background-color: #fff;
    border: none;
  }
</style>
