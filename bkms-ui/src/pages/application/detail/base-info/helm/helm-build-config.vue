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
    :title="$t('编辑构建配置')"
    width="960"
    @closed="handleClose"
  >
    <div class="py-[16px] px-[24px]">
      <Form
        ref="formRef"
        form-type="vertical"
        :label-width="120"
        :model="buildConfigData"
      >
        <HelmBuildConfig
          :key="formKey"
          v-model="buildConfigData"
          v-model:password-modified="passwordModified"
          :force-disable-code-repo="shouldForceDisableCodeRepo"
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
        {{ '确定' }}
      </Button>
      <Button
        :disabled="loading"
        @click="handleClose"
        >{{ '取消' }}</Button
      >
    </template>
  </Sideslider>
</template>
<script setup lang="ts">
  import { ref, watch } from 'vue';

  import { Button, Form, Message, Sideslider } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { BuildConfigInput, BuildConfigOutputObj, RepoBuildConfigInput, TagConfigOutputObj } from '~/@types/v1/app';
  import { ImageBuildConfigInput, UpdateBuildConfigRequest } from '~/@types/v1/builds';
  import { BuildsService } from '~/api/modules/v1';
  import { useAgonesFromAppDetail } from '~/composables/use-agones';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { normalizeTagConfig } from '~/composables/use-tag-config';
  import HelmBuildConfig from '~/pages/application/template/helm-chart/helm-chart-build-form.vue';
  import { useAppDetail } from '~/stores/app-detail';

  // BuildConfigInput 扩展类型，包含 tagConfig
  interface BuildConfigWithTagConfig extends BuildConfigInput {
    tagConfig?: null | TagConfigOutputObj;
  }

  interface IProps {
    data?: BuildConfigOutputObj;
    isShow: boolean;
  }

  const props = defineProps<IProps>();

  const emit = defineEmits(['close', 'update']);

  const { t } = useI18n();

  const formRef = ref();
  const buildConfigData = ref<BuildConfigWithTagConfig>({} as BuildConfigWithTagConfig);
  const { confirmBox, forceCleanDirtyTag } = useLeaveConfirm(buildConfigData);
  const appDetailStore = useAppDetail();
  const formKey = ref(0);
  const passwordModified = ref(false);

  // 使用 Agones Hook 判断是否强制禁用代码仓库（Agones 仅支持镜像）
  const { shouldForceDisableCodeRepo } = useAgonesFromAppDetail(() => appDetailStore.appDetail);

  // 侧边栏关闭前确认
  const handleBeforeClose = () => {
    return confirmBox();
  };

  const handleClose = async () => {
    if (await handleBeforeClose()) {
      emit('close');
    }
  };

  const handleUpdate = () => {
    emit('update');
  };

  watch(
    () => props.isShow,
    () => {
      if (!props.data || !props.isShow) return;
      formKey.value++;
      passwordModified.value = false;

      if (props.data.sourceType === 'codeRepository') {
        (buildConfigData.value as Pick<BuildConfigWithTagConfig, 'repoBuildConfig' | 'sourceType' | 'tagConfig'>) = {
          sourceType: props.data.sourceType,
          repoBuildConfig: {
            ...(props.data.repoBuildConfig as RepoBuildConfigInput),
          },
          tagConfig: props.data.tagConfig || null,
        };
      } else if (props.data.sourceType === 'imageRegistry') {
        (buildConfigData.value as Pick<BuildConfigWithTagConfig, 'imageBuildConfig' | 'sourceType' | 'tagConfig'>) = {
          sourceType: props.data.sourceType,
          imageBuildConfig: {
            name: props.data?.imageBuildConfig?.name ?? '',
            username: props.data?.imageBuildConfig?.username ?? '',
            password: '',
          },
          tagConfig: props.data.tagConfig || null,
        };
      }
      forceCleanDirtyTag();
    },
    { immediate: true },
  );

  // 保存构建配置
  const loading = ref(false);
  async function handleSaveConfig() {
    const formValid = await formRef.value.validate().catch(() => false);
    if (!formValid) return;

    const params = {
      appID: appDetailStore.appID,
      sourceType: buildConfigData.value.sourceType,
      // 未开启推荐版本号时置为 null
      tagConfig: normalizeTagConfig(buildConfigData.value.tagConfig),
    } as Partial<UpdateBuildConfigRequest> & { tagConfig: null | TagConfigOutputObj };

    if (buildConfigData.value.sourceType === 'codeRepository') {
      params.codeRepo = buildConfigData.value.repoBuildConfig;
    } else if (buildConfigData.value.sourceType === 'imageRegistry') {
      const { password, ...imageConfigWithoutPassword } = buildConfigData.value.imageBuildConfig!;
      params.image = (
        passwordModified.value ? { ...imageConfigWithoutPassword, password } : imageConfigWithoutPassword
      ) as ImageBuildConfigInput;
    }

    loading.value = true;
    const result = await BuildsService.updateBuildConfig(params as UpdateBuildConfigRequest)
      .then(() => true)
      .catch(() => false);
    loading.value = false;
    if (result) {
      Message({
        theme: 'success',
        message: t('保存成功'),
      });
      forceCleanDirtyTag(() => {
        handleUpdate();
      });
    }
  }
</script>
