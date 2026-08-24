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
    v-model:is-show="isShow"
    :before-close="handleBeforeClose"
    width="640"
    @closed="handleClosed"
  >
    <template #header>
      <DividerHeader :title="$t('部署应用')">
        <span v-if="appDetailStore.app">
          {{ appDetailStore.app }}
        </span>
        <span v-if="targetEnv?.name">
          {{ `${$t('环境')}: ${targetEnv.displayName || targetEnv.name}` }}
        </span>
      </DividerHeader>
    </template>
    <Form
      v-if="hasTargetSelector"
      ref="targetEnvFormRef"
      class="px-[24px] pt-[18px]"
      form-type="vertical"
      :model="targetFormModel"
      :rules="targetFormRules"
    >
      <!-- 目标环境 -->
      <Form.FormItem
        :label="$t('环境')"
        property="envName"
        required
      >
        <!-- 目标环境选择器 -->
        <EnvSelectPanel
          v-model="targetFormModel.envName"
          class="w-full"
          :columns="2"
          :show-env-prefix="false"
          :show-only-deployed-filter="false"
          :sync-env-store="false"
          @update:item="handleTargetEnvItemChange"
        />
      </Form.FormItem>
    </Form>
    <!-- 部署表单 -->
    <div :class="['px-[24px]', hasTargetSelector ? 'pt-0' : 'pt-[18px]']">
      <QuicklyDeployForm
        :key="targetEnv?.name || '__pending__'"
        ref="deployFormRef"
        :effective-replicas="targetEffectiveReplicas"
        :env-name="targetEnv?.name"
        :env-type="targetEnv?.type"
        :is-prod-env="targetIsProdEnv"
        :precheck-before-submit="hasTargetSelector"
        :use-provided-env="hasTargetSelector"
      />
    </div>
    <template #footer>
      <Button
        class="mr-[10px]"
        :loading="confirmLoading"
        theme="primary"
        @click="handleSubmit"
      >
        {{ $t('部署') }}
      </Button>
      <Button
        :loading="confirmLoading"
        @click="handleClose"
      >
        {{ $t('取消') }}
      </Button>
    </template>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, nextTick, reactive, ref, watch } from 'vue';

  import { Button, Form, Message, Sideslider } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import EnvSelectPanel from '~/components/env-select-panel.vue';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import QuicklyDeployForm from './quickly-deploy-form.vue';

  import type { EnvOutput } from '~/@types/v1/env';

  type OverviewDeployTarget = {
    effectiveReplicas?: number;
    env: EnvOutput;
  };

  const isShow = defineModel<boolean>('isShow');
  const emits = defineEmits<{
    update: [envName?: string];
  }>();
  const props = defineProps<{
    effectiveReplicas?: number;
    isProdEnv?: boolean;
    /** 传入时启用总览入口的目标环境选择；不传则保持实例列表入口的原有行为。 */
    targetEnvs?: OverviewDeployTarget[];
  }>();

  const { t } = useI18n();
  const trpcDeployStore = useTrpcDeployStore();
  const appDetailStore = useAppDetail();

  const deployFormRef = ref<InstanceType<typeof QuicklyDeployForm>>();
  const targetEnvFormRef = ref();
  const confirmLoading = ref(false);
  const targetFormModel = reactive({ envName: '' });
  const targetFormRules = {
    envName: [{ required: true, message: t('请选择环境'), trigger: 'change' }],
  };
  // 用 undefined 区分入口：空数组仍代表总览模式，只是环境列表尚未返回或当前没有可部署环境。
  const hasTargetSelector = computed(() => props.targetEnvs !== undefined);
  const selectedEnvItem = ref<EnvOutput>();
  const selectedTarget = computed(() => props.targetEnvs?.find(item => item.env.name === targetFormModel.envName));
  const targetEnv = computed(() =>
    hasTargetSelector.value ? selectedEnvItem.value || selectedTarget.value?.env : trpcDeployStore.curEnvItem,
  );
  const targetEffectiveReplicas = computed(() =>
    hasTargetSelector.value ? selectedTarget.value?.effectiveReplicas : props.effectiveReplicas,
  );
  const targetIsProdEnv = computed(() =>
    hasTargetSelector.value ? targetEnv.value?.type === 'production' : props.isProdEnv,
  );

  /** 规范化目标环境的初始副本数，无有效正整数时统一使用 1。 */
  function getInitialReplicas(replicas?: number) {
    return Number.isInteger(replicas) && Number(replicas) > 0 ? Number(replicas) : 1;
  }

  /** 关闭侧栏前复用部署表单的离开确认，避免丢失用户已填写的内容。 */
  function handleBeforeClose() {
    return deployFormRef.value?.confirmLeave() ?? Promise.resolve(true);
  }

  /** 用户点击取消时先执行离开确认，确认后再关闭侧栏。 */
  async function handleClose() {
    if (await handleBeforeClose()) {
      isShow.value = false;
    }
  }

  /** 侧栏关闭动画结束后清理环境和部署表单，为下次打开恢复初始状态。 */
  function handleClosed() {
    deployFormRef.value?.reset(1);
    targetFormModel.envName = '';
    selectedEnvItem.value = undefined;
  }

  /** 校验目标环境与部署表单，提交成功后关闭侧栏并通知父组件刷新对应入口的数据。 */
  async function handleSubmit() {
    try {
      if (hasTargetSelector.value) {
        const valid = await targetEnvFormRef.value?.validate?.().catch(() => false);
        if (!valid) return;
      }
      const envName = targetEnv.value?.name;
      if (!envName) return;

      confirmLoading.value = true;
      const submitted = await deployFormRef.value?.submit(envName);
      if (!submitted) return;

      Message({
        theme: 'success',
        message: t('操作成功'),
      });
      deployFormRef.value?.reset(1);
      isShow.value = false;
      emits('update', envName);
    } catch (err) {
      console.error(err);
    } finally {
      confirmLoading.value = false;
    }
  }

  function handleTargetEnvItemChange(env?: EnvOutput) {
    selectedEnvItem.value = env;
    if (env?.name) {
      nextTick(() => targetEnvFormRef.value?.clearValidate?.());
    }
  }

  // 每次从总览打开时保持目标环境未选择，由用户明确指定部署环境。
  watch(isShow, newVal => {
    if (newVal) {
      if (hasTargetSelector.value) {
        targetFormModel.envName = '';
        selectedEnvItem.value = undefined;
      }
      nextTick(() => {
        targetEnvFormRef.value?.clearValidate?.();
        deployFormRef.value?.reset(getInitialReplicas(targetEffectiveReplicas.value));
      });
    }
  });

  // 总览与环境列表是异步请求；列表变化时清理已经失效的环境。
  watch(
    () => props.targetEnvs,
    targets => {
      if (!isShow.value || !hasTargetSelector.value) return;
      if (!targetFormModel.envName) return;
      const currentTargetExists = targets?.some(item => item.env.name === targetFormModel.envName);
      if (currentTargetExists) return;
      targetFormModel.envName = '';
      selectedEnvItem.value = undefined;
      nextTick(() => targetEnvFormRef.value?.clearValidate?.());
    },
    { deep: true },
  );

  // 切换目标环境必须清空镜像、分支等内容，避免把上一环境的配置提交到新环境。
  watch(
    () => targetFormModel.envName,
    () => {
      nextTick(() => deployFormRef.value?.reset(getInitialReplicas(targetEffectiveReplicas.value)));
    },
  );
</script>
