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
  <div class="py-[16px] h-full w-full max-w-[1400px] overflow-auto">
    <div
      v-if="data.status === 'SUCCESS'"
      class="bg-[#fff] min-w-[600px] p-[60px] flex flex-col items-center mt-[20px]"
    >
      <Success
        class="mb-[36px]"
        fill="#2CAF5E"
        height="64px"
        width="64px"
      />
      <span class="text-[24px] leading-[32px] mb-[16px]">{{ $t('{name} 应用创建成功', data.name) }}</span>
      <span class="text-[14px] text-[#4D4F56] leading-[22px] mb-[28px]">
        <template v-if="!isHelmLikeAppType(appType)">
          {{ $t('接下来你可以直接 “查看应用” 或 继续 “创建应用”') }}
        </template>
        <template v-else>
          {{ $t('接下来可在“应用编排”中重写 values，或继续创建应用') }}
        </template>
      </span>
      <div>
        <template v-if="isHelmLikeAppType(appType)">
          <Button
            class="mr-[10px]"
            theme="primary"
            @click="handleToOrchestration"
          >
            {{ $t('应用编排') }}
          </Button>
          <Button @click="handleBackToList">{{ $t('返回列表') }}</Button>
        </template>
        <template v-else>
          <Button
            class="mr-[10px]"
            theme="primary"
            @click="handleToBasicInfo"
          >
            {{ $t('查看应用') }}
          </Button>
          <Button
            class="mr-[10px]"
            @click="emits('recreate')"
            >{{ $t('继续创建') }}</Button
          >
          <Button @click="handleBackToList">{{ $t('返回列表') }}</Button>
        </template>
      </div>
    </div>
    <div
      v-else-if="data.status === 'FAILED'"
      class="bg-[#fff] min-w-[600px] p-[60px] flex flex-col items-center mt-[20px]"
    >
      <Close
        class="mb-[36px]"
        fill="#EA3636"
        height="64px"
        width="64px"
      />
      <span class="text-[24px] leading-[32px] mb-[16px]">{{ $t('{name} 应用创建失败', data.name) }}</span>
      <span class="text-[14px] text-[#4D4F56] leading-[22px] mb-[28px]">{{
        $t('接下来你可以根据失败的原因，重新修改配置')
      }}</span>
      <div class="mb-[24px]">
        <Button
          class="mr-[10px]"
          theme="primary"
          @click="emits('recreate')"
          >{{ $t('重新创建') }}</Button
        >
        <Button @click="handleBackToList">{{ $t('返回列表') }}</Button>
      </div>
      <Alert
        v-if="data.msg"
        class="max-w-[700px]"
        theme="danger"
      >
        <template #title>{{ data.msg }}</template>
      </Alert>
    </div>
    <div
      v-else-if="data.status === 'CREATING'"
      class="bg-[#fff] min-w-[600px] p-[60px] flex flex-col items-center mt-[20px]"
    >
      <Loading
        mode="spin"
        theme="primary"
      ></Loading>
      <span class="text-[24px] leading-[32px] mt-[30px]">{{ $t('{name} 应用创建中，请稍等...') }}</span>
    </div>
  </div>
</template>
<script lang="ts" setup>
  import type { PropType } from 'vue';

  import { Alert, Button, Loading } from 'bkui-vue';
  import { Close, Success } from 'bkui-vue/lib/icon';
  import { useRouter } from 'vue-router';
  import { type IAppType, APP_TYPES, isHelmLikeAppType } from '~/composables/app-type';
  import { useAppDetail } from '~/stores/app-detail';

  export interface IResultData {
    msg: string;
    name: string;
    status: 'CREATING' | 'FAILED' | 'SUCCESS';
  }

  const props = defineProps({
    data: {
      type: Object as PropType<IResultData>,
      default: () => ({
        status: 'CREATING',
        name: '',
        msg: '',
      }),
    },
    appType: {
      type: String as PropType<IAppType>,
      default: '',
    },
  });

  const emits = defineEmits(['recreate']);
  const router = useRouter();
  const appDetailStore = useAppDetail();

  function handleBackToList() {
    router.push({ name: 'app' });
  }

  // 仅trpc应用有跳转至详情button
  function handleToBasicInfo() {
    router.push({
      name: 'detail',
      params: {
        name: props.data?.name,
        menuName: 'info',
        type: APP_TYPES.TRPC,
      },
    });
  }

  // 仅helm应用有跳转应用编排页面
  async function handleToOrchestration() {
    // 清空旧状态，防止用旧 appID 请求
    appDetailStore.reset();
    router.push({
      name: 'detail',
      params: {
        name: props.data?.name,
        menuName: 'orchestrate',
        type: props.appType,
      },
    });
  }
</script>
