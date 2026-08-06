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
    draggable
    render-directive="if"
    theme="primary"
    :width="960"
    @shown="handleOpen"
  >
    <template #header>
      <div class="flex items-center">
        <div>{{ $t('回滚应用') }}</div>
        <Divider
          class="mx-[8px]"
          direction="vertical"
        />
        <div class="flex items-center gap-[24px]">
          <div class="text-[14px] text-[#979ba5]">{{ appName }}</div>
          <div class="flex items-center">
            <div class="text-[14px] text-[#979ba5]">{{ $t('环境') }}：</div>
            <div class="text-[14px] text-[#979ba5]">{{ envName }}</div>
          </div>
        </div>
      </div>
    </template>
    <div
      v-if="loading"
      class="flex justify-center items-center min-h-[180px]"
    >
      <Loading />
    </div>
    <div
      v-else-if="previewData"
      class="h-[600px]"
    >
      <!-- diff 模式 -->
      <MsEditor
        :is-diff="true"
        lang="yaml"
        :model-value="previewData.target"
        :original="previewData.current"
        :readonly="true"
        :target-title="$t('回滚版本')"
        :title="$t('当前版本')"
      />
    </div>
    <Exception
      v-else
      class="min-h-[180px]"
      :description="$t('暂无数据')"
      scene="part"
      type="empty"
    >
    </Exception>
    <template #footer>
      <Button
        class="mr-[10px]"
        :disabled="!previewData"
        :loading="rollbackLoading"
        theme="primary"
        @click="submit"
      >
        {{ $t('确定') }}
      </Button>
      <Button
        :disabled="rollbackLoading"
        @click="isShow = false"
        >{{ $t('取消') }}</Button
      >
    </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { ref } from 'vue';

  import { Button, Dialog, Divider, Exception, Loading, Message } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { PreviewHelmDeployOutput } from '~/@types/v1/deploy';
  import { DeployService } from '~/api/modules/v1';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';
  import { useAppDetail } from '~/stores/app-detail';

  const isShow = defineModel('isShow', { type: Boolean });

  const props = withDefaults(defineProps<Props>(), {
    appName: '',
    envName: '',
    trafficLaneName: '',
    deployID: '',
  });

  const emit = defineEmits<Emits>();

  const { t } = useI18n();

  interface Emits {
    (e: 'success'): void; // 回滚成功
  }

  interface Props {
    appName?: string;
    deployID?: string;
    envName?: string;
    trafficLaneName?: string;
  }

  // Store
  const appDetailStore = useAppDetail();

  const loading = ref(false);
  const previewData = ref<null | PreviewHelmDeployOutput>(null);
  const rollbackLoading = ref(false);

  // 获取部署版本回滚预览
  async function getPreviewRollbackDeploy() {
    loading.value = true;
    previewData.value = null;
    try {
      const response = await DeployService.previewRollbackHelmDeploy(
        {
          appID: appDetailStore.appID,
          envName: props.envName,
          deployID: props.deployID,
          trafficLaneName: props.trafficLaneName,
        },
        { needRes: true },
      );
      previewData.value = response as unknown as PreviewHelmDeployOutput;
    } finally {
      loading.value = false;
    }
  }

  function handleOpen() {
    getPreviewRollbackDeploy();
  }

  // 确认回滚
  async function submit() {
    rollbackLoading.value = true;
    try {
      await DeployService.rollbackHelmDeploy(
        {
          appID: appDetailStore.appID,
          envName: props.envName,
          deployID: props.deployID,
          trafficLaneName: props.trafficLaneName,
        },
        { needRes: true },
      );
      Message({
        theme: 'success',
        message: t('回滚成功'),
      });
      emit('success');
      isShow.value = false;
    } finally {
      rollbackLoading.value = false;
    }
  }
</script>
