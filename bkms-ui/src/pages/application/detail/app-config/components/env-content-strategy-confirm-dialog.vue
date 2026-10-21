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
    :cancel-text="$t('取消')"
    class="env-content-strategy-confirm-dialog"
    :confirm-text="$t('确认切换')"
    footer-align="center"
    header-align="center"
    :quick-close="false"
    :title="strategy === 'overlay' ? $t('切换为“与默认配置合并”？') : $t('切换为“独立完整配置”？')"
    :width="480"
    @closed="finish(false)"
    @confirm="finish(true)"
  >
    <div class="text-[14px] leading-[22px] text-[#4D4F56]">
      <i18n-t
        class="mb-[16px] bg-[#F5F7FA] px-[16px] py-[12px]"
        keypath="切换后，这些环境的文件内容{0}。请在切换后{1}。"
        tag="div"
      >
        <strong class="font-bold text-[#313238]">
          {{ strategy === 'overlay' ? $t('保持原样，不会自动按环境拆分差异') : $t('保持原样，不会自动合并') }}
        </strong>
        <strong class="font-bold text-[#313238]">{{ $t('手动修改下列环境的文件内容') }}</strong>
      </i18n-t>
      <div class="overflow-hidden rounded-[2px] border border-solid border-[#EAEBF0]">
        <div class="bg-[#EAEBF0] px-[16px] py-[6px] text-[#313238]">
          {{ $t('涉及 {0} 个环境', [affectedEnvs.length]) }}
        </div>
        <div class="max-h-[180px] overflow-y-auto">
          <div
            v-for="env in affectedEnvs"
            :key="env.name"
            class="flex h-[36px] items-center px-[16px] text-[12px] even:bg-[#FAFBFD]"
          >
            <span
              class="truncate text-[#313238]"
              :title="env.name"
              >{{ env.displayName }}</span
            >
            <Tag
              v-if="envTypeMap[env.type]"
              class="ml-[8px] shrink-0"
              :class="envTypeTagClassMap[env.type]"
              size="small"
            >
              {{ envTypeMap[env.type].name }}
            </Tag>
          </div>
        </div>
      </div>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, ref } from 'vue';

  import { Dialog, Tag } from 'bkui-vue';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';

  import type { EnvOutput } from '~/@types/v1/env';
  import type { ConfigFileEnvContentStrategy } from '~/pages/application/detail/app-config/use-config-file-defs';

  const props = defineProps<{ envList: EnvOutput[] }>();
  const isShow = ref(false);
  const envNames = ref<string[]>([]);
  const strategy = ref<ConfigFileEnvContentStrategy>('overlay');
  let resolveConfirmation: ((confirmed: boolean) => void) | undefined;

  const affectedEnvs = computed(() =>
    envNames.value.map(name => {
      const env = props.envList.find(item => item.name === name);
      return { name, displayName: env?.displayName || name, type: env?.type || '' };
    }),
  );

  // 保存流程等待确认结果；取消、关闭和组件卸载均不提交策略。
  function confirm(names: string[], targetStrategy: ConfigFileEnvContentStrategy) {
    envNames.value = names;
    strategy.value = targetStrategy;
    isShow.value = true;
    return new Promise<boolean>(resolve => {
      resolveConfirmation = resolve;
    });
  }

  // 确认或取消
  function finish(confirmed: boolean) {
    isShow.value = false;
    resolveConfirmation?.(confirmed);
    resolveConfirmation = undefined;
  }

  onBeforeUnmount(() => finish(false));
  defineExpose({ confirm });
</script>

<style scoped>
  /* Dialog 传送到 body 后不携带本组件的 scope，使用专属类限定样式。 */
  :global(.env-content-strategy-confirm-dialog .bk-dialog-header) {
    padding-top: 48px;
    line-height: 32px;
  }

  :global(.env-content-strategy-confirm-dialog .bk-dialog-footer) {
    padding: 0 24px 24px;
    background: transparent;
    border: none;
  }

  :global(.env-content-strategy-confirm-dialog .bk-dialog-footer .bk-button) {
    min-width: 88px;
  }
</style>
