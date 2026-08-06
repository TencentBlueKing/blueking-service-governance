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
    :width="700"
  >
    <template #header>
      <i18n-t
        class="text-[16px] font-bold text-[#313238]"
        keypath="部署预检发现{0}类问题"
      >
        <span class="font-bold px-[4px]">{{ problemCount }}</span>
      </i18n-t>
    </template>

    <MissingVarPanel
      v-for="panel in panels"
      :key="panel.type"
      class="mb-[12px] last:mb-0"
      :data="panel.data"
      :title="$t(panel.titleKey)"
    >
      <template #default>
        <i18n-t
          class="text-[12px] leading-[20px] text-[#4D4F56]"
          :keypath="panel.descriptionKeypath"
          tag="p"
        >
          <span class="font-bold">{{ $t('可能导致服务异常！') }}</span>
        </i18n-t>

        <i18n-t
          class="mb-[8px] text-[12px] leading-[20px] text-[#4D4F56]"
          :keypath="panel.linkDescriptionKeypath"
          tag="p"
        >
          <span
            class="cursor-pointer text-[#3A84FF]"
            @click="panel.onLink"
          >
            「{{ $t(panel.linkTextKey) }}」
          </span>
        </i18n-t>
      </template>
    </MissingVarPanel>

    <template #footer>
      <div class="flex justify-end">
        <Button
          v-if="showGoModifyBtn"
          class="mr-[8px]"
          theme="primary"
          @click="handleGoModify"
        >
          {{ $t('前往修改') }}
        </Button>

        <Button
          class="mr-[8px] bg-[#fff] text-[#4D4F56]"
          @click="handleStillDeploy"
        >
          {{ $t('仍然部署') }}
        </Button>

        <Button
          class="!min-w-[60px] bg-[#fff] text-[#4D4F56]"
          @click="handleCancel"
        >
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Dialog>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { Button, Dialog } from 'bkui-vue';

  import MissingVarPanel from './components/missing-var-panel.vue';

  /** 表格行数据：仅需变量名 */
  interface MissingVarItem {
    key: string;
  }

  interface PanelConfig {
    data: MissingVarItem[];
    descriptionKeypath: string;
    linkDescriptionKeypath: string;
    linkTextKey: string;
    titleKey: string;
    type: PanelType;
    onLink: () => void;
  }

  /** 面板类型：环境变量 / 应用编排未定义变量 */
  type PanelType = 'env' | 'orchestrate';

  interface Props {
    /** env 命名空间未定义变量（对应环境变量表格） */
    missingEnvVars?: string[];
    /** 非 env 命名空间未定义变量（对应应用编排表格） */
    missingVars?: string[];
  }

  const props = withDefaults(defineProps<Props>(), {
    missingEnvVars: () => [],
    missingVars: () => [],
  });

  const emit = defineEmits<{
    cancel: [];
    goModify: [source: PanelType];
    stillDeploy: [];
  }>();

  const isShow = defineModel<boolean>('isShow', {
    default: false,
  });

  /** 将变量名列表转为表格行结构 */
  function toTableRows(keys: string[]): MissingVarItem[] {
    return keys.map(key => ({ key }));
  }

  const envVarRows = computed(() => toTableRows(props.missingEnvVars));

  const orchestrateVarRows = computed(() => toTableRows(props.missingVars));

  const hasEnvVars = computed(() => envVarRows.value.length > 0);

  const hasOrchestrateVars = computed(() => orchestrateVarRows.value.length > 0);

  /** 问题类别数量（有数据的面板数） */
  const problemCount = computed(() => Number(hasEnvVars.value) + Number(hasOrchestrateVars.value));

  /** 仅一类问题时展示「前往修改」 */
  const showGoModifyBtn = computed(() => problemCount.value === 1);

  const handleGoEnvLink = () => {
    emit('goModify', 'env');
  };

  const handleGoOrchestrateLink = () => {
    emit('goModify', 'orchestrate');
  };

  const panelConfigs = {
    env: {
      titleKey: '环境变量未定义',
      descriptionKeypath: '以下环境变量在当前配置中被引用，但在目标部署环境中未定义，部署后将被渲染为空值，{0}',
      linkDescriptionKeypath: '建议前往 {0} 补充配置后再部署，避免服务注册异常或配置错误。',
      linkTextKey: '环境管理 / 环境配置',
    },

    orchestrate: {
      titleKey: '未定义变量',
      descriptionKeypath: '以下变量在当前配置中被引用，但未定义，部署后将被渲染为空值，{0}',
      linkDescriptionKeypath: '建议前往 {0} 修改 Values 文件，避免服务注册异常或配置错误。',
      linkTextKey: '应用编排',
    },
  };

  const panels = computed<PanelConfig[]>(() => {
    const result: PanelConfig[] = [];

    if (hasEnvVars.value) {
      result.push({
        type: 'env',
        data: envVarRows.value,
        onLink: handleGoEnvLink,
        ...panelConfigs.env,
      });
    }

    if (hasOrchestrateVars.value) {
      result.push({
        type: 'orchestrate',
        data: orchestrateVarRows.value,
        onLink: handleGoOrchestrateLink,
        ...panelConfigs.orchestrate,
      });
    }

    return result;
  });

  function handleCancel() {
    isShow.value = false;
    emit('cancel');
  }

  function handleGoModify() {
    isShow.value = false;
    emit('goModify', hasEnvVars.value ? 'env' : 'orchestrate');
  }

  function handleStillDeploy() {
    isShow.value = false;
    emit('stillDeploy');
  }
</script>

<style lang="postcss" scoped>
  /* 弹窗最大高度为 80vh */
  :deep(.bk-dialog-content) {
    max-height: calc(80vh - 135px);
    overflow-y: auto;
  }

  :deep(.bk-dialog-footer) {
    border-top-color: #eaebf0;
  }
</style>
