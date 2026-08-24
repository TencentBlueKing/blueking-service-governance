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
    render-directive="if"
    width="640"
    @closed="handleClosed"
  >
    <template #header>
      <DividerHeader>
        <template #title>
          <span class="text-[16px]">{{ $t('新建环境') }}</span>
        </template>
        <span v-if="appDetailStore.app">
          {{ appDetailStore.app }}
        </span>
      </DividerHeader>
    </template>

    <!-- 新建环境和部署是两个动作，创建成功后留在侧栏内引导下一步，避免用户拿到一个空环境后不知道往哪走 -->
    <div
      v-if="createdEnv"
      class="flex h-full flex-col items-center px-[24px] pt-[60px]"
    >
      <Success
        class="mb-[36px]"
        fill="#2CAF5E"
        height="64px"
        width="64px"
      />
      <span class="mb-[16px] text-[24px] leading-[32px] text-[#313238]">
        {{ $t('环境创建成功') }}
      </span>
      <p class="mb-[28px] max-w-[480px] text-center text-[14px] leading-[22px] text-[#4D4F56]">
        {{ $t('环境尚未部署实例，应用配置为默认值。如需调整配置，请先去配置再部署。') }}
      </p>
      <div class="mb-[28px] w-full max-w-[480px] bg-[#F5F7FA] px-[16px] py-[12px] text-left text-[14px] leading-[22px]">
        <div>
          <span class="text-[#63656E]">{{ $t('环境展示名称') }}：</span>
          <span class="text-[#313238]">{{ createdEnv.displayName || createdEnv.name }}</span>
        </div>
        <div>
          <span class="text-[#63656E]">{{ $t('命名空间') }}：</span>
          <span class="text-[#313238]">{{ createdEnv.cluster?.namespace || createdEnv.name }}</span>
        </div>
      </div>
      <div>
        <Button
          class="mr-[8px]"
          theme="primary"
          @click="handleDeployNow"
        >
          {{ $t('立即部署') }}
        </Button>
        <Button
          class="mr-[8px]"
          @click="handleGoConfig"
        >
          {{ $t('去配置') }}
        </Button>
        <Button @click="isShow = false">
          {{ $t('关闭') }}
        </Button>
      </div>
    </div>

    <div
      v-else
      class="px-[24px] pt-[24px]"
    >
      <Alert
        class="mb-[16px]"
        theme="info"
        :title="
          $t('这里创建的是当前应用的特性环境：只属于本应用，其他应用无法使用，适合灰度验证、联调等场景，用完可以销毁。')
        "
      />
      <Form
        ref="formRef"
        form-type="vertical"
        :model="formModel"
        :rules="rules"
      >
        <Form.FormItem
          :label="$t('来源环境')"
          property="sourceEnvID"
          required
        >
          <EnvSelectPanel
            v-model="sourceEnvName"
            class="w-full"
            :columns="2"
            :kinds="['standard']"
            :show-env-prefix="false"
            :show-only-deployed-filter="false"
            :sync-env-store="false"
            @update:item="handleSourceEnvChange"
          />
          <div class="mt-[8px] text-[12px] leading-[20px] text-[#979BA5]">
            {{ $t('沿用来源环境的集群和环境分类。应用配置不会复制，新环境使用应用的默认配置。') }}
          </div>
        </Form.FormItem>

        <Form.FormItem
          :label="$t('环境展示名称')"
          property="displayName"
          required
        >
          <Input
            v-model.trim="formModel.displayName"
            :placeholder="$t('请输入 1-32 字符的环境展示名')"
          />
        </Form.FormItem>
      </Form>

      <div class="mt-[32px]">
        <Button
          class="mr-[10px]"
          :loading="confirmLoading"
          theme="primary"
          @click="handleSubmit"
        >
          {{ $t('创建') }}
        </Button>
        <Button
          :loading="confirmLoading"
          @click="handleClose"
        >
          {{ $t('取消') }}
        </Button>
      </div>
    </div>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, nextTick, reactive, ref, watch } from 'vue';

  import { Alert, Button, Form, Input, Message, Sideslider } from 'bkui-vue';
  import { Success } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { EnvOutput } from '~/@types/v1/env';
  import { EnvService } from '~/api/modules/v1';
  import { BKMS_REGEX } from '~/common/const';
  import DividerHeader from '~/components/divider-header.vue';
  import EnvSelectPanel from '~/components/env-select-panel.vue';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  const isShow = defineModel<boolean>('isShow');
  const emits = defineEmits<{
    (e: 'created', env: EnvOutput): void;
    (e: 'deploy', env: EnvOutput): void;
  }>();

  const { t } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const appDetailStore = useAppDetail();
  const trpcDeployStore = useTrpcDeployStore();

  // 表单引用与状态
  const formRef = ref();
  const confirmLoading = ref(false);
  const envListLoading = ref(false);
  const envList = ref<EnvOutput[]>([]);
  const sourceEnvName = ref('');
  // 有值即代表已创建成功，侧栏切换到引导态
  const createdEnv = ref<EnvOutput>();
  // 表单数据：展示名称、来源环境ID
  const formModel = reactive<{
    displayName: string;
    sourceEnvID: string;
  }>({
    displayName: '',
    sourceEnvID: '',
  });

  // 特性环境只能基于标准环境创建，未就绪的环境也没有集群可继承
  const availableEnvList = computed(() =>
    envList.value.filter(env => env.id && env.status !== 'NotReady' && (env.kind || 'standard') === 'standard'),
  );
  const {
    confirmBox: confirmFormLeave,
    forceCleanDirtyTag: markFormClean,
    withPausedWatch,
  } = useLeaveConfirm(formModel);

  // 表单验证规则
  const rules = {
    sourceEnvID: [
      {
        message: t('请选择来源环境'),
        trigger: 'change',
        validator: (value: string) => !!value,
      },
    ],
    displayName: [
      {
        validator: () => BKMS_REGEX.envDisplayNameRegex.test(formModel.displayName || ''),
        message: t('请输入 1-32 字符的环境展示名'),
        trigger: 'blur',
      },
    ],
  };

  // 关闭前确认：已创建成功则直接放行，否则检查表单是否有未保存修改
  function handleBeforeClose() {
    // 已创建成功时表单内容不再有意义，直接放行
    if (createdEnv.value) return Promise.resolve(true);
    return confirmFormLeave();
  }

  // 关闭侧栏
  async function handleClose() {
    if (await handleBeforeClose()) {
      isShow.value = false;
    }
  }

  // 侧栏关闭后重置状态
  function handleClosed() {
    formRef.value?.clearValidate?.();
    createdEnv.value = undefined;
    handleInitForm();
  }

  // 点击"立即部署"：通知父组件进入部署流程
  function handleDeployNow() {
    if (!createdEnv.value) return;
    emits('deploy', createdEnv.value);
    isShow.value = false;
  }

  // 获取应用环境列表（用于来源环境选择）
  async function handleGetEnvList() {
    if (!appDetailStore.appID) {
      envList.value = [];
      return;
    }
    envListLoading.value = true;
    try {
      envList.value = await EnvService.listAppEnvs({
        appID: appDetailStore.appID,
      });
    } catch (err) {
      console.error(err);
      envList.value = [];
    } finally {
      envListLoading.value = false;
    }
  }

  // 应用配置在另一个菜单下，新开标签页避免用户丢掉当前部署页的上下文
  function handleGoConfig() {
    if (!createdEnv.value?.name) return;
    const resolved = router.resolve({
      name: 'detail',
      params: {
        ...route.params,
        menuName: 'appConfig',
      },
      query: {
        ...route.query,
        activeTab: 'deploy-config',
        envName: createdEnv.value.name,
      },
    });
    window.open(resolved.href, '_blank');
  }

  // 初始化表单：优先选中当前环境，否则选第一个可用环境
  function handleInitForm() {
    const currentEnv = availableEnvList.value.find(env => env.name === trpcDeployStore.curEnvItem?.name);
    const defaultEnv = currentEnv || availableEnvList.value[0];
    withPausedWatch(() => {
      formModel.sourceEnvID = defaultEnv?.id || '';
      formModel.displayName = '';
    });
    sourceEnvName.value = defaultEnv?.name || '';
    markFormClean();
  }

  // 来源环境变更：更新表单值并清除校验状态
  function handleSourceEnvChange(env?: EnvOutput) {
    formModel.sourceEnvID = env?.id || '';
    if (env?.name) {
      nextTick(() => formRef.value?.clearValidate?.());
    }
  }

  // 提交创建：校验通过后调用API，成功后切换到引导态
  async function handleSubmit() {
    const valid = await formRef.value?.validate().catch(() => false);
    if (!valid || !appDetailStore.appID) return;

    confirmLoading.value = true;
    try {
      const env = await EnvService.createFeatureEnv({
        appID: appDetailStore.appID,
        displayName: formModel.displayName,
        sourceEnvID: formModel.sourceEnvID,
      });
      if (!env?.name) {
        Message({
          theme: 'error',
          message: t('无法获取环境标识'),
        });
        return;
      }
      markFormClean();
      createdEnv.value = env;
      emits('created', env);
    } catch (err) {
      console.error(err);
    } finally {
      confirmLoading.value = false;
    }
  }

  // 监听侧栏打开：重置状态、加载环境列表、初始化表单
  watch(isShow, async val => {
    if (val) {
      createdEnv.value = undefined;
      await handleGetEnvList();
      nextTick(handleInitForm);
    }
  });
</script>
