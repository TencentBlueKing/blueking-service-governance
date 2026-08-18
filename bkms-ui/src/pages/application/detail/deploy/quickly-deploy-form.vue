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
  <Form
    ref="formRef"
    form-type="vertical"
    :model="formModel"
  >
    <Form.FormItem
      :label="$t('实例数')"
      property="replicas"
      required
    >
      <Input
        v-model.number="formModel.replicas"
        :min="1"
        :precision="0"
        type="number"
      />
    </Form.FormItem>

    <Form.FormItem
      :label="$t('镜像来源')"
      required
    >
      <Button.ButtonGroup class="flex items-center">
        <Button
          class="flex-1"
          :selected="imageSource === 'image'"
          @click="handleChangeImageSource('image')"
        >
          {{ $t('已构建镜像') }}
        </Button>
        <Button
          v-bk-tooltips="{
            content: $t('生产类型环境只能部署已经晋级的镜像 Tag'),
            disabled: !isProdEnv,
          }"
          class="flex-1"
          :disabled="isProdEnv"
          :selected="imageSource === 'code'"
          @click="handleChangeImageSource('code')"
        >
          {{ $t('从源码构建') }}
        </Button>
      </Button.ButtonGroup>
    </Form.FormItem>
    <template v-if="imageSource === 'code'">
      <Form.FormItem
        :label="$t('代码分支')"
        property="branch"
        required
      >
        <Input v-model.trim="formModel.branch" />
      </Form.FormItem>
    </template>
    <Form.FormItem
      :label="$t('镜像 Tag')"
      property="imageTag"
      required
    >
      <Input
        v-if="imageSource === 'code'"
        v-model.trim="formModel.imageTag"
      />
      <ImageSelect
        v-else-if="envName"
        ref="imageSelectRef"
        v-model:value="formModel.imageTag"
        :env-name="envName"
        :env-type="envType"
      />
      <!-- 占位 -->
      <Select
        v-else
        v-bk-tooltips="$t('请选择环境')"
        disabled
      />
    </Form.FormItem>
  </Form>

  <!-- 环境变量预检查弹窗 -->
  <EnvVarPrecheckDialog
    v-if="precheckBeforeSubmit"
    v-model:is-show="isShowPrecheckDialog"
    :env-name="precheckEnvName"
    :undefined-vars="undefinedVars"
    @cancel="cancelDeploy"
    @go-modify="cancelDeploy"
    @still-deploy="continueDeploy"
  />
</template>

<script lang="ts" setup>
  import { computed, reactive, ref } from 'vue';

  import { Button, Form, Input, Select } from 'bkui-vue';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useRecommendTag } from '~/composables/use-recommend-tag';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import ImageSelect from '../../components/image-select.vue';
  import EnvVarPrecheckDialog from './env-var-precheck-dialog.vue';
  import { type DeployableAppType, type DeployParams, useDeployAPIs } from './use-deploy';
  import { useEnvVarPrecheck } from './use-env-var-precheck';

  // 镜像来源类型：从源码构建 / 已构建镜像
  type ImageSourceType = 'code' | 'image';

  const props = withDefaults(
    defineProps<{
      effectiveReplicas?: number; // 当前生效的实例数，用于重置表单时回填
      envName?: string;
      envType?: string;
      isProdEnv?: boolean;
      /** 是否在提交表单时执行预检查；普通部署已在打开侧栏前完成检查。 */
      precheckBeforeSubmit?: boolean;
      /** 总览入口强制使用显式传入的目标环境，防止未选中时误回退到 store 中的旧环境。 */
      useProvidedEnv?: boolean;
    }>(),
    {
      precheckBeforeSubmit: true,
    },
  );

  const trpcDeployStore = useTrpcDeployStore();
  const appDetailStore = useAppDetail();
  const { cancelDeploy, continueDeploy, isShowPrecheckDialog, precheck, precheckEnvName, undefinedVars } =
    useEnvVarPrecheck();

  // 当前选择的镜像来源
  const imageSource = ref<ImageSourceType>('image');
  const formRef = ref();
  const imageSelectRef = ref();
  // 表单数据模型
  const formModel = reactive<{
    branch: string;
    imageTag: string;
    replicas: number;
  }>({
    replicas: 1,
    imageTag: '',
    branch: '',
  });

  // 实例列表入口沿用 store 当前环境；总览入口只认侧栏选中的目标环境，即使暂时为空也不回退。
  const envName = computed(() =>
    props.useProvidedEnv ? (props.envName ?? '') : props.envName || trpcDeployStore.curEnvItem?.name || '',
  );
  const envType = computed(() =>
    props.useProvidedEnv ? (props.envType ?? '') : props.envType || trpcDeployStore.curEnvItem?.type || '',
  );
  // 是否为生产环境：props 优先，否则按 envType 判断
  const isProdEnv = computed(() => props.isProdEnv ?? envType.value === 'production');

  // 推荐镜像 Tag：源码构建模式下自动推荐
  const { getDefaultBranch, fetchRecommendTag } = useRecommendTag(() => formModel.branch, {
    onRecommend: tag => {
      if (imageSource.value === 'code') {
        formModel.imageTag = tag;
      }
    },
  });
  // 离开确认：表单有变更时提示用户
  const { confirmBox, forceCleanDirtyTag, withPausedWatch } = useLeaveConfirm(formModel);

  // 清除表单校验结果
  function clearValidate() {
    formRef.value?.clearValidate?.();
  }

  // 关闭镜像选择下拉框
  function closeDropdown() {
    imageSelectRef.value?.closeDropdown?.();
  }

  function confirmLeave() {
    return confirmBox();
  }

  // 执行部署：根据镜像来源调用直接部署或构建+部署接口
  async function deploy(targetEnvName: string) {
    const deployAPIs = useDeployAPIs(appDetailStore.appType as DeployableAppType);
    const params: DeployParams = {
      appID: appDetailStore.appID,
      envName: targetEnvName,
      imageTag: formModel.imageTag,
      replicas: Number(formModel.replicas),
    };
    if (imageSource.value === 'image') {
      await deployAPIs.createDeployDirectly!(params);
    } else {
      params.branch = formModel.branch;
      await deployAPIs.buildAndCreateDeploy!(params);
    }
  }

  /** 切换镜像来源；源码模式补充分支和推荐 Tag，镜像模式清空源码相关字段。 */
  function handleChangeImageSource(source: ImageSourceType) {
    imageSource.value = source;
    if (source === 'code') {
      const branch = getDefaultBranch();
      formModel.branch = branch;
      fetchRecommendTag(branch);
    } else {
      formModel.branch = '';
      formModel.imageTag = '';
    }
  }

  // 重置表单：回填实例数，其余字段清空，并统一清理内部 UI 状态
  function reset(replicas = props.effectiveReplicas || 1) {
    withPausedWatch(() => {
      formModel.replicas = replicas;
      formModel.imageTag = '';
      formModel.branch = '';
      imageSource.value = 'image';
    });
    forceCleanDirtyTag(() => {
      clearValidate();
      closeDropdown();
    });
  }

  // 提交部署：内部完成校验、部署和脏标记清理
  async function submit(targetEnvName: string) {
    const valid = await validate();
    if (!valid) return false;

    if (props.precheckBeforeSubmit) {
      const precheckPassed = await precheck(targetEnvName);
      if (!precheckPassed) return false;
    }

    await deploy(targetEnvName);
    forceCleanDirtyTag();
    return true;
  }

  // 校验表单
  async function validate() {
    return await formRef.value?.validate?.().catch(() => false);
  }

  // 暴露业务级方法供父组件调用
  defineExpose({
    confirmLeave,
    reset,
    submit,
    validate,
  });
</script>
