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
  <div class="bg-[#F5F7FA] p-[16px] rounded-[2px]">
    <p class="mb-[12px]">{{ $t('开启后，仍需执行以下流程，才能使用开发模式') }}：</p>
    <div
      v-for="(step, index) in steps"
      :key="index"
      class="flex items-center gap-x-[8px] line-height-[20px]"
      :class="{ 'mb-[16px]': index < steps.length - 1 }"
    >
      <div class="w-[20px] h-[20px] leading-[20px] text-center bg-[#EAEBF0] rounded-[50%] text-[14px]">
        {{ index + 1 }}
      </div>
      <Button
        text
        theme="primary"
        @click="step.onClick"
      >
        <span>{{ step.label }}</span>
        <Share
          v-if="step.showIcon"
          class="ml-[4px]"
        />
      </Button>
    </div>
  </div>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import { Button } from 'bkui-vue';
  import { Share } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { DOC_LINKS } from '~/common/const';
  import { useAppDetail } from '~/stores/app-detail';
  import { useDeployEnvStore } from '~/stores/deploy-env';

  interface Props {
    envName?: string;
  }

  const props = defineProps<Props>();

  const { t } = useI18n();
  const router = useRouter();
  const appDetailStore = useAppDetail();
  const deployEnvStore = useDeployEnvStore();

  // 跳转到 bkms-cli 文档
  function goToBkmsCliDoc() {
    const docUrl = `${import.meta.env.BK_DOC_URL}${DOC_LINKS.BKMS_CLI}`;
    window.open(docUrl, '_blank');
  }

  // 跳转到部署管理
  function goToDeployment() {
    // 跳转前更新到 store 中，部署管理需要选择对应环境
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

  // 开发模式操作步骤
  const steps = computed(() => [
    { label: t('执行部署'), onClick: goToDeployment, showIcon: false },
    { label: t('使用 bkms-cli'), onClick: goToBkmsCliDoc, showIcon: true },
  ]);
</script>
