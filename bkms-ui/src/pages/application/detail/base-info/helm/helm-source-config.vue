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
  <Sideslider
    :before-close="handleBeforeClose"
    :is-show="isShow"
    :title="$t('编辑 Helm chart 配置')"
    width="960"
    @closed="handleClose"
  >
    <div class="py-[16px] px-[24px]">
      <Form
        ref="formRef"
        form-type="vertical"
        :label-width="120"
        :model="{}"
      >
        <HelmChartSourceForm
          ref="sourceFormRef"
          :initial-data="helmSourceConfigData"
          is-edit-mode
        />
      </Form>
    </div>
    <template #footer>
      <Button
        class="mr-[10px]"
        :loading="loading"
        theme="primary"
        @click="handleSaveConfig"
      >
        {{ $t('确定') }}
      </Button>
      <Button
        :disabled="loading"
        @click="handleClose"
        >{{ $t('取消') }}</Button
      >
    </template>
  </Sideslider>
</template>
<script setup lang="ts">
  import { ref, watch } from 'vue';

  import { Button, Form, Message, Sideslider } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { HelmSpecInput, HelmSpecOutputObj, UpdateHelmSpecRequest } from '~/@types/v1/app';
  import { AppService } from '~/api/modules/v1';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import HelmChartSourceForm from '~/pages/application/template/helm-chart/helm-chart-source-form.vue';
  import { useAppDetail } from '~/stores/app-detail';

  interface IProps {
    data?: HelmSpecOutputObj;
    isShow: boolean;
  }

  const props = defineProps<IProps>();

  const emit = defineEmits(['close', 'update']);

  const { t } = useI18n();
  const appDetailStore = useAppDetail();

  const formRef = ref();
  const sourceFormRef = ref();
  const helmSourceConfigData = ref<HelmSpecInput>({} as HelmSpecInput);
  const { confirmBox, forceCleanDirtyTag } = useLeaveConfirm(helmSourceConfigData);

  // 侧边栏关闭前确认
  const handleBeforeClose = () => {
    return confirmBox();
  };

  const handleClose = async () => {
    if (await handleBeforeClose()) {
      emit('close');
    }
  };

  watch(
    () => props.isShow,
    () => {
      if (!props.data || !props.isShow) return;
      if (props.data.helmSource?.repoType === 'HelmRepo') {
        helmSourceConfigData.value = {
          helmSource: {
            repoType: props.data.helmSource?.repoType,
            valueFiles: props.data.helmSource?.valueFiles,
            helmRepoConfig: {
              ...props.data.helmSource.helmRepoConfig,
              password: '',
            },
            bcsRepoConfig: props.data.helmSource?.bcsRepoConfig,
          } as HelmSpecInput['helmSource'],
        };
      } else if (props.data.helmSource?.repoType === 'BCSRepo') {
        // TODO: BCSRepo 处理逻辑待补充
      } else if (props.data.helmSource?.repoType === 'GitRepo') {
        helmSourceConfigData.value = props.data as HelmSpecInput;
      }
      forceCleanDirtyTag();
    },
    { immediate: true },
  );

  // 修改helm chart 源码配置
  const loading = ref(false);
  async function handleSaveConfig() {
    const formValid = await formRef.value.validate().catch(() => false);
    if (!formValid || !appDetailStore.app) return;

    const sourceData = sourceFormRef.value?.getValue();
    if (!sourceData) return;

    const helmSource = sourceData.helmSource;
    // 保持密码占位符未修改时不传 password；仅清空或修改密码输入框时显式传递 password。
    if (props?.data?.helmSource?.repoType === 'HelmRepo' && helmSource.helmRepoConfig) {
      const { password, ...helmRepoConfigWithoutPassword } = helmSource.helmRepoConfig;
      const passwordModified = Boolean(sourceFormRef.value?.getPasswordModified?.());
      helmSource.helmRepoConfig = (
        passwordModified ? { ...helmRepoConfigWithoutPassword, password } : helmRepoConfigWithoutPassword
      ) as typeof helmSource.helmRepoConfig;
    }

    const params: UpdateHelmSpecRequest = {
      appID: appDetailStore.appID,
      helmSpec: {
        helmSource,
      },
    };
    loading.value = true;
    const result = await AppService.updateHelmSpec(params)
      .then(() => true)
      .catch(() => false);
    loading.value = false;
    forceCleanDirtyTag(() => {
      if (result) {
        Message({
          theme: 'success',
          message: t('保存成功'),
        });
        emit('update');
      }
    });
  }
</script>
