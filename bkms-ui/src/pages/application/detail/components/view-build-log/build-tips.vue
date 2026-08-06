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
  <Alert
    class="build-tips mb-[16px]"
    :closable="needClose && alertTheme !== 'info'"
    :theme="alertTheme"
  >
    <template #icon>
      <ColorIcon
        v-if="alertTheme === 'info'"
        icon="loading"
        :size="16"
      />
      <Success
        v-else-if="alertTheme === 'success'"
        fill="#65C389"
        height="16px"
        width="16px"
      />
      <Close
        v-else-if="alertTheme === 'error'"
        fill="#EA3636"
        height="16px"
        width="16px"
      />
      <Warn
        v-else
        fill="#FF9C01"
        height="16px"
        width="16px"
      />
      <span class="text-[#4D4F56] text-[12px] font-bold ml-[8px]">{{ $t(statusText) }}</span>
    </template>
    <template #default>
      <div class="flex px-[24px] gap-[36px] text-[12px] text-[#4D4F56] h-[18px] leading-[18px]">
        <span>{{ $t('代码分支') }}: {{ buildInfo.revision || '--' }}</span>
        <span>{{ $t('镜像 Tag') }}: {{ buildInfo.imageTag || '--' }}</span>
        <span>{{ $t('操作人') }}: {{ buildInfo.operator || '--' }}</span>
        <Button
          class="inline-flex items-center ml-auto"
          text
          theme="primary"
          @click="handleDetail"
        >
          {{ $t('查看详情') }}
          <Share class="ml-[8px]" />
        </Button>
      </div>
    </template>
  </Alert>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { Alert, Button } from 'bkui-vue';
  import { Close, Share, Success, Warn } from 'bkui-vue/lib/icon';
  import ColorIcon from '~/components/color-icon.vue';

  import type { BuildAlertTheme, BuildStatus, BuildTipsProps } from './type';

  const props = withDefaults(defineProps<BuildTipsProps>(), {
    /** 是否显示 Alert 关闭按钮，默认不显示。 */
    needClose: false,
  });

  /** 不同构建状态对应的提示主题。 */
  const alertThemeMap: Record<BuildStatus, BuildAlertTheme> = {
    running: 'info',
    success: 'success',
    failed: 'error',
    pollingBroken: 'error',
    warning: 'warning',
  };
  const alertTheme = computed(() => alertThemeMap[props.buildInfo.status] ?? 'error');

  /** 不同构建状态对应的提示文字。 */
  const statusTextMap: Record<BuildStatus, string> = {
    running: '构建中...',
    success: '构建成功',
    failed: '构建失败',
    pollingBroken: '构建失败',
    warning: '构建中断',
  };
  const statusText = computed(() => statusTextMap[props.buildInfo.status] ?? '构建失败');

  const emit = defineEmits<{
    detail: [];
  }>();

  /** 通知父组件打开蓝盾流水线详情。 */
  function handleDetail() {
    emit('detail');
  }
</script>
<style lang="postcss" scoped>
  .build-tips :deep(.bk-alert-close) {
    display: flex;
  }
</style>
