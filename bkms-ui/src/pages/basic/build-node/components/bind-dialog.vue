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
    :quick-close="false"
    :title="currentNodeHashId ? $t('更换构建机') : $t('关联构建机')"
    :width="640"
    @closed="handleClose"
  >
    <Alert
      class="mb-[16px]"
      theme="warning"
      :title="$t('保存后，空间下的内置构建流水线会切换到这台构建机')"
    />

    <div class="mb-[8px] text-[14px] text-[#63656e]">{{ $t('构建机') }}</div>
    <Select
      v-model="selectedNodeHashId"
      :clearable="false"
      filterable
      :loading="isLoading"
      :placeholder="$t('请选择构建机，可按名称或 IP 搜索')"
    >
      <!-- name 同时包含名称与 IP：既用于搜索，也是选中后输入框里展示的文本 -->
      <Select.Option
        v-for="node in candidates"
        :id="node.nodeHashID!"
        :key="node.nodeHashID"
        class="build-node-option"
        :disabled="!node.selectable"
        :name="`${node.displayName}（${node.ip}）`"
      >
        <div class="w-full py-[6px] leading-[20px]">
          <div class="flex items-center justify-between gap-[12px]">
            <span class="truncate text-[#313238]">{{ node.displayName }}</span>
            <StatusIcon
              class="shrink-0 text-[12px]"
              :status="node.online ? 'online' : 'offline'"
              :status-color-map="BUILD_NODE_STATE_COLOR_MAP"
              :status-text-map="onlineTextMap"
            />
          </div>
          <div class="text-[12px] text-[#979ba5]">
            {{ node.ip }} · {{ node.osName || '--' }}
            <span
              v-if="!node.selectable"
              class="ml-[8px] text-[#ea3636]"
            >
              {{ unselectableReasonText(node.unselectableReason) }}
            </span>
          </div>
        </div>
      </Select.Option>
    </Select>
    <div
      v-if="!isLoading && !candidates.length"
      class="mt-[8px] text-[12px] text-[#979ba5]"
    >
      {{ $t('蓝盾项目 {0} 下还没有第三方构建机，请联系 SRE 导入', [projectCode]) }}
    </div>

    <template #footer>
      <Button
        class="mr-[8px]"
        :disabled="!canSubmit"
        :loading="isSubmitting"
        theme="primary"
        @click="handleSubmit"
      >
        {{ $t('保存') }}
      </Button>
      <Button @click="handleClose">
        {{ $t('取消') }}
      </Button>
    </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Alert, Button, Dialog, Message, Select } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { BkintegrationsBkciService } from '~/api/modules/v1';
  import StatusIcon from '~/components/status-icon.vue';

  import { BUILD_NODE_STATE_COLOR_MAP } from '../constants';

  import type { BuildNodeCandidateOutput } from '~/@types/v1/bkintegrations-bkci';

  const props = defineProps<{
    /** 当前已关联的构建机，用于更换时回显 */
    currentNodeHashId?: string;
    projectCode: string;
    workspaceId: string;
  }>();

  const emit = defineEmits<{ saved: [] }>();

  const isShow = defineModel<boolean>('isShow', { default: false });

  const { t } = useI18n();

  const candidates = ref<BuildNodeCandidateOutput[]>([]);
  const selectedNodeHashId = ref('');
  const isLoading = ref(false);
  const isSubmitting = ref(false);

  const onlineTextMap = computed(() => ({ online: t('在线'), offline: t('离线') }));
  // 没换机器时提交没有意义，后端也会当作无变更
  const canSubmit = computed(() => !!selectedNodeHashId.value && selectedNodeHashId.value !== props.currentNodeHashId);

  watch(isShow, show => {
    if (!show) return;

    selectedNodeHashId.value = props.currentNodeHashId || '';
    fetchCandidates();
  });

  async function fetchCandidates() {
    isLoading.value = true;
    try {
      candidates.value = await BkintegrationsBkciService.listBkCIBuildNodes({ workspaceID: props.workspaceId });
    } catch {
      candidates.value = [];
    } finally {
      isLoading.value = false;
    }
  }

  function handleClose() {
    isShow.value = false;
  }

  async function handleSubmit() {
    isSubmitting.value = true;
    try {
      await BkintegrationsBkciService.bindBuildNode({
        workspaceID: props.workspaceId,
        nodeHashID: selectedNodeHashId.value,
      });
      Message({ theme: 'success', message: t('保存成功') });
      emit('saved');
      isShow.value = false;
    } finally {
      isSubmitting.value = false;
    }
  }

  function unselectableReasonText(reason?: BuildNodeCandidateOutput['unselectableReason']) {
    if (reason === 'offline') return t('Agent 离线');
    if (reason === 'osUnsupported') return t('仅支持 Linux 构建机');
    return '';
  }
</script>

<style lang="postcss" scoped>
  /* 选项是两行内容，需要撑开 bkui 默认的固定行高 */
  :global(.build-node-option.bk-select-option) {
    height: auto !important;
  }
</style>
