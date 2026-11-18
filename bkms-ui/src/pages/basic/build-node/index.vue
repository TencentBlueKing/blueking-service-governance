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
  <Loading :loading="isLoading">
    <div class="min-h-[200px]">
      <Alert
        v-if="!isLoading && !projectCode"
        theme="warning"
        :title="$t('工作空间尚未初始化蓝盾项目，暂时无法关联构建机')"
      />
      <template v-else-if="!isLoading">
        <Alert
          class="mb-[16px]"
          theme="info"
        >
          <template #title>
            <i18n-t
              keypath="私有构建机需由 SRE 导入蓝盾项目 {0}（需预装 Docker），并在此关联到蓝鲸服务治理，关联后内置构建流水线将使用该构建机。"
            >
              <a
                class="inline-flex items-center text-[#3a84ff]"
                :href="bkciNodeListUrl"
                rel="noopener noreferrer"
                target="_blank"
              >
                {{ projectCode }}
                <i class="bkms-icon bkms-icon-jump-link ml-[2px]"></i>
              </a>
            </i18n-t>
          </template>
        </Alert>
        <div class="p-[16px] bg-[#fff] shadow-[0_2px_4px_0_#1919290d]">
          <!-- 与其他列表页一致，主操作放在表格左上方 -->
          <div
            v-if="state?.editable"
            class="mb-[16px]"
          >
            <Button
              theme="primary"
              @click="isBindDialogShow = true"
            >
              {{ binding ? $t('更换构建机') : $t('关联构建机') }}
            </Button>
          </div>
          <BindingInfo
            :binding="binding"
            :editable="state?.editable"
            @unbind="handleUnbind"
          />
        </div>
      </template>
    </div>
  </Loading>

  <BindDialog
    v-model:is-show="isBindDialogShow"
    :current-node-hash-id="binding?.nodeHashID"
    :project-code="projectCode"
    :workspace-id="spaceStore.currentSpace"
    @saved="fetchBinding"
  />
</template>

<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue';

  import { Alert, Button, InfoBox, Loading, Message } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { BkintegrationsBkciService } from '~/api/modules/v1';
  import { useSpaceStore } from '~/stores/space';

  import BindDialog from './components/bind-dialog.vue';
  import BindingInfo from './components/binding-info.vue';

  import type { BuildNodeBindingStateOutput } from '~/@types/v1/bkintegrations-bkci';

  const { t } = useI18n();
  const spaceStore = useSpaceStore();

  const state = ref<BuildNodeBindingStateOutput | null>(null);
  const isLoading = ref(false);
  const isBindDialogShow = ref(false);

  const binding = computed(() => state.value?.binding ?? null);
  const projectCode = computed(() => state.value?.projectCode || '');
  /** 蓝盾项目环境管理下的节点列表，SRE 在这里导入第三方构建机 */
  const bkciNodeListUrl = computed(
    () => `${import.meta.env.BK_DEVOPS}/console/environment/${projectCode.value}/pipeline/node/allNode`,
  );

  async function fetchBinding() {
    isLoading.value = true;
    try {
      state.value = await BkintegrationsBkciService.getBuildNodeBinding({ workspaceID: spaceStore.currentSpace });
    } finally {
      isLoading.value = false;
    }
  }

  function handleUnbind() {
    InfoBox({
      title: t('确定解除关联构建机？'),
      content: t('解除后，空间下的内置构建流水线会回到公共构建机'),
      confirmText: t('解除关联'),
      cancelText: t('取消'),
      onConfirm: async () => {
        await BkintegrationsBkciService.unbindBuildNode({ workspaceID: spaceStore.currentSpace });
        Message({ theme: 'success', message: t('已解除关联') });
        await fetchBinding();
      },
    });
  }

  onMounted(fetchBinding);
</script>
