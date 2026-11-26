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
      <i18n-t
        v-if="createdContext?.hasComponentRisk"
        class="mb-[28px] max-w-[480px] text-center text-[14px] leading-[22px] text-[#4D4F56]"
        :keypath="
          createdContext.isFederation
            ? '应用使用了组件，沿用来源环境的特性环境将无法部署。可前往「{0}」确认是否都需要；来源环境是联邦集群，暂不能「独立创建」。'
            : '应用使用了组件，沿用来源环境的特性环境将无法部署。可前往「{0}」确认是否都需要，或新建时选择「独立创建」。'
        "
        tag="p"
      >
        <Button
          class="!h-auto align-baseline"
          text
          theme="primary"
          @click="handleGoComponentConfig"
        >
          {{ $t('组件配置') }}
        </Button>
      </i18n-t>
      <p
        v-else
        class="mb-[28px] max-w-[480px] text-center text-[14px] leading-[22px] text-[#4D4F56]"
      >
        {{ $t('特性环境使用独立的配置文件、环境变量，不继承来源环境的配置。') }}
      </p>
      <div class="mb-[28px] w-full max-w-[480px] bg-[#F5F7FA] px-[16px] py-[12px] text-left text-[14px] leading-[22px]">
        <div>
          <span class="text-[#63656E]">{{ $t('环境展示名称') }}：</span>
          <span class="text-[#313238]">{{ createdEnv.displayName || createdEnv.name }}</span>
        </div>
        <div>
          <span class="text-[#63656E]">{{ $t('命名空间') }}：</span>
          <span class="text-[#313238]">{{ createdEnv.cluster?.namespace || '--' }}</span>
          <span
            v-if="createdContext?.nsType === 'shared'"
            class="ml-[8px] text-[#979BA5]"
          >
            {{ $t('（与来源环境共用）') }}
          </span>
        </div>
        <div>
          <span class="text-[#63656E]">{{ $t('来源环境') }}：</span>
          <span class="text-[#313238]">{{ createdContext?.sourceEnvDisplayName || '--' }}</span>
        </div>
      </div>
      <div>
        <Button
          v-if="!createdContext?.hasComponentRisk"
          class="mr-[8px]"
          theme="primary"
          @click="handleDeployNow"
        >
          {{ $t('立即部署') }}
        </Button>
        <Dropdown
          class="mr-[8px]"
          placement="bottom-start"
          :popover-options="{ clickContentAutoHide: true, width: 104 }"
          trigger="click"
        >
          <Button>
            {{ $t('调整配置') }}
            <AngleDownLine class="ml-[6px] text-[12px]" />
          </Button>
          <template #content>
            <Dropdown.DropdownMenu>
              <Dropdown.DropdownItem @click="handleGoConfig">
                {{ $t('应用配置') }}
              </Dropdown.DropdownItem>
              <Dropdown.DropdownItem @click="handleGoEnvVars">
                {{ $t('环境变量') }}
              </Dropdown.DropdownItem>
              <Dropdown.DropdownItem
                v-if="createdContext?.hasComponentRisk"
                @click="handleGoComponentConfig"
              >
                {{ $t('组件配置') }}
              </Dropdown.DropdownItem>
            </Dropdown.DropdownMenu>
          </template>
        </Dropdown>
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
        theme="warning"
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
            @update:env-list="handleSourceEnvListChange"
            @update:item="handleSourceEnvChange"
          />
          <div class="mt-[8px] text-[12px] leading-[20px] text-[#979BA5]">
            {{ $t('沿用来源环境的集群和环境分类。应用配置不会复制，新环境使用应用的默认配置。') }}
          </div>
        </Form.FormItem>

        <div
          v-if="componentLoadState === 'loading'"
          class="mb-[12px] text-[12px] leading-[20px] text-[#979BA5]"
          role="status"
        >
          {{ $t('正在获取应用组件信息...') }}
        </div>
        <Alert
          v-else-if="componentLoadState === 'error'"
          class="mb-[12px]"
          theme="error"
        >
          <span>{{ $t('获取应用组件信息失败，请重试。') }}</span>
          <Button
            class="ml-[8px]"
            text
            theme="primary"
            @click="loadAppComponents"
          >
            {{ $t('重试') }}
          </Button>
        </Alert>

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

        <div class="flex items-center">
          <Button
            :aria-expanded="showAdvancedSettings"
            text
            theme="primary"
            @click="showAdvancedSettings = !showAdvancedSettings"
          >
            {{ $t('高级设置') }}
            <AngleDownLine
              class="ml-[4px] transition-transform duration-200"
              :class="{ 'rotate-180': !showAdvancedSettings }"
              fill="#3A84FF"
              height="12px"
              width="12px"
            />
          </Button>
          <span
            v-if="!showAdvancedSettings"
            class="ml-[12px] text-[12px] text-[#979BA5]"
          >
            {{ $t('命名空间') }}：{{ namespaceTypeText }}
          </span>
        </div>
        <div
          v-if="showAdvancedSettings"
          class="mt-[12px] rounded-[2px] bg-[#F5F7FA] px-[16px] py-[12px]"
        >
          <div class="mb-[10px] text-[12px] text-[#313238]">{{ $t('命名空间') }}</div>
          <Radio.Group
            v-model="formModel.nsType"
            class="flex w-full flex-col gap-[8px]"
          >
            <div
              v-bk-tooltips="{
                content: independentNamespaceDisabledReason,
                disabled: !independentNamespaceDisabledReason,
              }"
              class="block w-full"
            >
              <Radio
                class="namespace-option !m-0 !h-auto !flex w-full items-start rounded-[2px] border border-solid px-[16px] py-[8px]"
                :class="[
                  formModel.nsType === 'independent' ? 'border-[#3A84FF] bg-[#F0F5FF]' : 'border-[#DCDEE5] bg-white',
                  { 'cursor-not-allowed opacity-60': isSourceFederationEnv },
                ]"
                :disabled="isSourceFederationEnv"
                label="independent"
              >
                <span class="flex min-w-0 flex-1 flex-wrap items-baseline gap-x-[8px] leading-[20px]">
                  <span class="shrink-0 text-[#4D4F56]">{{ $t('独立创建') }}</span>
                  <span class="text-[12px] leading-[20px] text-[#979BA5]">
                    {{ $t('独占命名空间，资源完全隔离，适合使用组件等场景。') }}
                  </span>
                </span>
              </Radio>
            </div>
            <Radio
              class="namespace-option !m-0 !h-auto !flex w-full items-start rounded-[2px] border border-solid px-[16px] py-[8px]"
              :class="formModel.nsType === 'shared' ? 'border-[#3A84FF] bg-[#F0F5FF]' : 'border-[#DCDEE5] bg-white'"
              label="shared"
            >
              <span class="flex min-w-0 flex-1 flex-wrap items-center gap-x-[8px] leading-[20px]">
                <span class="shrink-0 text-[#4D4F56]">
                  {{ $t('沿用来源环境') }}
                </span>
                <span class="text-[12px] leading-[20px] text-[#979BA5]">
                  {{ $t('不新建命名空间，与来源环境共用，不支持使用组件。') }}
                </span>
              </span>
            </Radio>
            <div
              v-if="requiresComponentRiskConfirmation"
              class="flex items-start gap-[6px] rounded-[2px] border border-solid border-[#FFD2D2] bg-[#FFF2F2] px-[12px] py-[10px]"
              role="alert"
            >
              <Error
                class="mt-[2px] h-[16px] w-[16px] shrink-0 rounded-full bg-[#EA3636]"
                fill="#FFFFFF"
                height="14px"
                width="14px"
              />
              <div class="min-w-0 flex-1 text-[12px] leading-[20px]">
                <p class="break-words text-[#EA3636]">{{ componentRiskMessage }}</p>
                <i18n-t
                  class="mt-[4px] text-[#63656E]"
                  :keypath="
                    isSourceFederationEnv
                      ? '组件需要部署在独立的命名空间，共用会与来源环境冲突。可前往「{0}」确认是否都需要；来源环境是联邦集群，暂不能「独立创建」。'
                      : '组件需要部署在独立的命名空间，共用会与来源环境冲突。可前往「{0}」确认是否都需要，或改选「{1}」。'
                  "
                  tag="p"
                >
                  <Button
                    class="!h-auto align-baseline"
                    text
                    theme="primary"
                    @click="handleGoComponentConfig"
                  >
                    {{ $t('组件配置') }}
                  </Button>
                  <Button
                    v-if="!isSourceFederationEnv"
                    class="!h-auto align-baseline"
                    text
                    theme="primary"
                    @click="formModel.nsType = 'independent'"
                  >
                    {{ $t('独立创建') }}
                  </Button>
                </i18n-t>
                <div class="mt-[10px] border-t border-dashed border-[#FFD2D2] pt-[8px]">
                  <Checkbox v-model="formModel.riskAcknowledged">
                    <span class="whitespace-normal text-[12px] text-[#313238]">
                      {{ $t('我已知晓风险，仍要沿用来源环境创建') }}
                    </span>
                  </Checkbox>
                </div>
              </div>
            </div>
          </Radio.Group>
        </div>
      </Form>

      <div class="mt-[32px]">
        <Button
          class="mr-[10px]"
          :disabled="cannotCreate"
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
  import { computed, nextTick, onScopeDispose, reactive, ref, watch } from 'vue';

  import { useEventListener, useMounted } from '@vueuse/core';
  import { Alert, Button, Checkbox, Dropdown, Form, Input, Message, Radio, Sideslider } from 'bkui-vue';
  import { AngleDownLine, Error, Success } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { ComponentOutputObj } from '~/@types/v1/app';
  import { EnvOutput, NsType } from '~/@types/v1/env';
  import { AppService, EnvService } from '~/api/modules/v1';
  import { BKMS_REGEX } from '~/common/const';
  import DividerHeader from '~/components/divider-header.vue';
  import EnvSelectPanel from '~/components/env-select-panel/env-select-panel.vue';
  import { isFederationEnv } from '~/composables/use-is-federation-env';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  const isShow = defineModel<boolean>('isShow');
  const emits = defineEmits<{
    (e: 'created', env: EnvOutput): void;
    (e: 'deploy', env: EnvOutput): void;
  }>();

  const { locale, t } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const isMounted = useMounted();
  const appDetailStore = useAppDetail();
  const trpcDeployStore = useTrpcDeployStore();

  // 表单引用与状态
  const formRef = ref();
  const confirmLoading = ref(false);
  const envList = ref<EnvOutput[]>([]);
  const sourceEnvName = ref('');
  const showAdvancedSettings = ref(false);
  const appComponents = ref<ComponentOutputObj[]>([]);
  const componentLoadState = ref<'error' | 'loading' | 'ready'>('loading');
  let componentRequestSequence = 0;
  let sliderSession = 0;
  let componentSignature = '';
  let refreshComponentsOnReturn = false;
  // 有值即代表已创建成功，侧栏切换到引导态
  const createdEnv = ref<EnvOutput>();
  const createdContext = ref<{
    hasComponentRisk: boolean;
    isFederation: boolean;
    nsType: NsType;
    sourceEnvDisplayName: string;
  }>();
  // 表单数据：展示名称、来源环境和命名空间选择，统一参与未保存修改检测。
  const formModel = reactive<{
    displayName: string;
    nsType: NsType;
    riskAcknowledged: boolean;
    sourceEnvID: string;
  }>({
    displayName: '',
    sourceEnvID: '',
    nsType: 'independent',
    riskAcknowledged: false,
  });
  // 非联邦来源默认独立创建。用户改选沿用后记住该选择；联邦来源强制沿用时不覆盖它。
  let preferredNsType: NsType = 'independent';

  // 特性环境只能基于标准环境创建，未就绪的环境也没有集群可继承
  const availableEnvList = computed(() =>
    envList.value.filter(env => env.id && env.status !== 'NotReady' && (env.kind || 'standard') === 'standard'),
  );
  const sourceEnv = computed(() => availableEnvList.value.find(env => env.id === formModel.sourceEnvID));
  const isSourceFederationEnv = computed(() => isFederationEnv(sourceEnv.value));
  const hasComponents = computed(() => appComponents.value.length > 0);
  const requiresComponentRiskConfirmation = computed(() => hasComponents.value && formModel.nsType === 'shared');
  // 只有沿用来源环境依赖组件清单。加载或失败时不能放行沿用；独立创建与清单无关。
  const cannotCreate = computed(
    () =>
      formModel.nsType === 'shared' &&
      (componentLoadState.value !== 'ready' || (hasComponents.value && !formModel.riskAcknowledged)),
  );
  const componentRiskMessage = computed(() => {
    const names = [
      ...new Set(appComponents.value.map(comp => comp.name || comp.refWorkspaceCompName || comp.type).filter(Boolean)),
    ];
    return names.length
      ? t('应用使用了组件（{0}），沿用来源环境创建的特性环境将无法部署。', [
          names.join(locale.value.startsWith('en') ? ', ' : '、'),
        ])
      : t('应用使用了组件，沿用来源环境创建的特性环境将无法部署。');
  });
  const namespaceTypeText = computed(() => (formModel.nsType === 'shared' ? t('沿用来源环境') : t('独立创建')));
  const independentNamespaceDisabledReason = computed(() =>
    isSourceFederationEnv.value
      ? t(
          hasComponents.value
            ? '来源环境（{0}）绑定了联邦集群，暂不能「独立创建」。可前往「组件配置」确认应用使用的组件是否都需要。'
            : '来源环境（{0}）绑定了联邦集群，暂不支持创建独立的特性环境。',
          [sourceEnv.value?.displayName || sourceEnv.value?.name || ''],
        )
      : '',
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
        validator: (value: string) => availableEnvList.value.some(env => env.id === value),
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

  // 联邦来源只能沿用。其他来源回到用户上一次的选择，初始为独立创建。
  function applyNamespaceForSource(env?: EnvOutput) {
    const federation = isFederationEnv(env);
    formModel.nsType = federation ? 'shared' : preferredNsType;
    if (federation && hasComponents.value) showAdvancedSettings.value = true;
  }

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
    if (isShow.value) return;
    formRef.value?.clearValidate?.();
    createdEnv.value = undefined;
    createdContext.value = undefined;
  }

  // 新标签页调整组件后，回到未提交的创建表单时刷新组件信息。
  function handleComponentConfigReturn() {
    if (!refreshComponentsOnReturn || !isShow.value || createdEnv.value || confirmLoading.value) return;
    void loadAppComponents();
  }

  // 点击"立即部署"：通知父组件进入部署流程
  function handleDeployNow() {
    if (!createdEnv.value || createdContext.value?.hasComponentRisk) return;
    emits('deploy', createdEnv.value);
    isShow.value = false;
  }

  function handleGoComponentConfig() {
    refreshComponentsOnReturn = !createdEnv.value;
    const resolved = router.resolve({
      name: 'detail',
      params: { ...route.params, menuName: 'module' },
    });
    window.open(resolved.href, '_blank');
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

  // 特性环境有独立的环境变量配置视图，使用环境 ID 打开新标签页并保留当前部署页。
  function handleGoEnvVars() {
    if (!createdEnv.value?.id) return;
    const resolved = router.resolve({
      name: 'detail',
      params: {
        ...route.params,
      },
      query: {
        ...route.query,
        view: 'feature-envs',
        envVars: createdEnv.value.id,
      },
    });
    window.open(resolved.href, '_blank');
  }

  // 重置整张表单，来源选择等待选择器返回列表后初始化。
  function handleInitForm() {
    preferredNsType = 'independent';
    withPausedWatch(() => {
      formModel.sourceEnvID = '';
      formModel.displayName = '';
      formModel.nsType = 'independent';
      formModel.riskAcknowledged = false;
    });
    showAdvancedSettings.value = false;
    sourceEnvName.value = '';
    markFormClean();
  }

  // 来源环境变更：更新表单值并清除校验状态
  function handleSourceEnvChange(env?: EnvOutput) {
    formModel.sourceEnvID = env?.id || '';
    applyNamespaceForSource(env);
    if (env?.name) {
      nextTick(() => formRef.value?.clearValidate?.());
    }
  }

  // 共用选择器返回的列表，避免默认值、校验和联邦判断读取不同快照。
  function handleSourceEnvListChange(list: EnvOutput[]) {
    envList.value = list;
    if (!isShow.value || createdEnv.value) return;
    const nextEnv =
      sourceEnv.value ||
      availableEnvList.value.find(env => env.name === trpcDeployStore.curEnvItem?.name) ||
      availableEnvList.value[0];
    // 列表加载只初始化来源，不覆盖用户已填写的名称、选择和修改标记。
    withPausedWatch(() => {
      formModel.sourceEnvID = nextEnv?.id || '';
      applyNamespaceForSource(nextEnv);
    });
    sourceEnvName.value = nextEnv?.name || '';
  }

  // 提交创建：先占住提交锁再校验，避免校验返回前连点发出两次创建。
  async function handleSubmit() {
    if (confirmLoading.value || cannotCreate.value || !isShow.value) return;
    const session = sliderSession;
    const appID = appDetailStore.appID;
    if (!appID || !sourceEnv.value) return;

    confirmLoading.value = true;
    try {
      const valid = await formRef.value?.validate().catch(() => false);
      if (!valid || appID !== appDetailStore.appID || session !== sliderSession || !isShow.value || !sourceEnv.value) {
        return;
      }
      // 校验期间来源或组件清单可能变化，按最新状态决定能否提交。
      if (isSourceFederationEnv.value) formModel.nsType = 'shared';
      if (cannotCreate.value) return;

      const context = {
        hasComponentRisk: requiresComponentRiskConfirmation.value,
        isFederation: isSourceFederationEnv.value,
        nsType: formModel.nsType,
        sourceEnvDisplayName: sourceEnv.value.displayName || sourceEnv.value.name || '',
      };
      const env = await EnvService.createFeatureEnv({
        appID,
        displayName: formModel.displayName,
        sourceEnvID: formModel.sourceEnvID,
        nsType: formModel.nsType,
      });
      if (!isMounted.value || appID !== appDetailStore.appID) return;
      if (!env?.name) {
        Message({
          theme: 'error',
          message: t('无法获取环境标识'),
        });
        return;
      }
      // 关闭或重开只阻止旧成功页写回，同一应用仍需收到创建成功通知。
      if (session === sliderSession && isShow.value) {
        markFormClean();
        createdContext.value = context;
        createdEnv.value = env;
      }
      emits('created', env);
    } catch (err) {
      console.error(err);
    } finally {
      if (session === sliderSession) confirmLoading.value = false;
    }
  }

  // 组件配置列表与配置页保持一致；请求失败不能当作没有组件放行。
  async function loadAppComponents() {
    const appID = appDetailStore.appID;
    const sequence = ++componentRequestSequence;
    componentLoadState.value = 'loading';
    if (!appID) {
      componentLoadState.value = 'error';
      return;
    }
    const isCurrent = () => sequence === componentRequestSequence;
    try {
      // 辅助查询只使用局部结果，失败不能清空全局应用详情及可见环境名单。
      const detail = await AppService.getApp({ appID });
      if (!isCurrent()) return;
      if (detail?.id !== appID) {
        componentLoadState.value = 'error';
        return;
      }
      const components = detail.appModelSpec?.components || [];
      const signature = JSON.stringify(components);
      const signatureChanged = signature !== componentSignature;
      // 沿用时清单变化会清掉确认。已折叠则重新展开，避免创建按钮禁用却看不到原因。
      const shouldExpand =
        components.length > 0 &&
        formModel.nsType === 'shared' &&
        (componentSignature === '' || !hasComponents.value || signatureChanged);
      if (signatureChanged) formModel.riskAcknowledged = false;
      appComponents.value = components;
      componentSignature = signature;
      if (shouldExpand) showAdvancedSettings.value = true;
      componentLoadState.value = 'ready';
    } catch {
      if (isCurrent()) componentLoadState.value = 'error';
    }
  }

  watch(
    () => formModel.nsType,
    nsType => {
      if (!isSourceFederationEnv.value) preferredNsType = nsType;
    },
  );

  useEventListener(window, 'focus', handleComponentConfigReturn);
  onScopeDispose(() => {
    sliderSession++;
    componentRequestSequence++;
  });

  // 侧栏与应用切换均使旧请求失效，选择器加载完成后再初始化来源环境。
  watch(
    [isShow, () => appDetailStore.appID],
    ([visible]) => {
      sliderSession++;
      componentRequestSequence++;
      refreshComponentsOnReturn = false;
      confirmLoading.value = false;
      if (!visible) return;
      createdEnv.value = undefined;
      createdContext.value = undefined;
      appComponents.value = [];
      componentSignature = '';
      envList.value = [];
      handleInitForm();
      void loadAppComponents();
    },
    { flush: 'sync', immediate: true },
  );
</script>

<style scoped>
  .namespace-option:not(.is-disabled):hover {
    border-color: #3a84ff;
  }

  .namespace-option :deep(.bk-radio-input) {
    flex-shrink: 0;
    margin-top: 3px;
  }

  .namespace-option :deep(.bk-radio-label) {
    flex: 1;
    min-width: 0;
    white-space: normal;
  }
</style>
