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
    :width="sliderWidth"
    @closed="handleClose"
  >
    <template #header>
      <FlexRow class="w-full pr-[12px]">
        <template #left>
          <DividerHeader
            :show-divider="isEditMode"
            :title="isEditMode ? $t('编辑北极星') : $t('添加北极星')"
            :title-size="16"
          >
            <span>{{ formModel.polarisName }}</span>
          </DividerHeader>
        </template>
        <template
          v-if="!showRedeployTip"
          #right
        >
          <IconTextButton
            :active="!isCollaspeAside"
            class="text-[12px] p-[10px]"
            icon="bkms-icon bkms-icon-variable"
            :text="$t('环境变量')"
            @click="handleToggleisCollaspeAside"
          />
        </template>
      </FlexRow>
    </template>
    <CollapsibleAsideLayout
      v-if="!showRedeployTip"
      v-model:is-collapsed="isCollaspeAside"
      :layout-config="{
        viewportOffset: 100,
      }"
      @after-resize="handleRefreshEnvVar"
    >
      <template #main>
        <div class="p-[24px] pb-0">
          <Form
            ref="formRef"
            form-type="vertical"
            :model="formModel"
            :rules="rules"
          >
            <ToggleCard
              :name="$t('基本信息')"
              type="normal"
            >
              <!-- 实例名称 -->
              <Form.FormItem
                class="!text-[#4D4F56]"
                :label="$t('实例名称')"
                property="instanceKey"
                required
              >
                <Input
                  v-model.trim="formModel.instanceKey"
                  :placeholder="$t('请输入实例名称，需以字母开头，只能包含字母、数字、下划线')"
                />
              </Form.FormItem>

              <!-- 服务端口 -->
              <Form.FormItem
                :label="$t('服务端口')"
                property="servicePort"
                required
              >
                <Input
                  v-model.trim="formModel.servicePort"
                  :min="1"
                  :precision="0"
                  type="number"
                />
                <div class="text-[12px] leading-[20px] text-[#979BA5]">
                  {{
                    $t('服务端口会写入到应用的环境变量 {0}_serviceport 中', [formModel.instanceKey || $t('实例名称')])
                  }}
                </div>
                <Alert
                  v-if="showServicePortChangeAlert"
                  class="mt-[12px]"
                  closable
                  theme="warning"
                  :title="$t('{0}已修改，保存后需重新部署才会生效。', [$t('服务端口')])"
                />
              </Form.FormItem>
            </ToggleCard>

            <ToggleCard
              class="mb-[24px]"
              content-class="!p-0 !mt-[12px]"
              :name="$t('组件可用环境')"
              type="normal"
            >
              <EnvGroupSelector
                v-model="formModel.scopeEnvNames"
                :env-list="envList"
              />
            </ToggleCard>

            <ToggleCard
              class="mt-[10px]"
              :name="$t('北极星信息')"
              type="normal"
            >
              <!-- 类型 -->
              <Form.FormItem
                :label="$t('类型')"
                property="createNewService"
                required
              >
                <Radio.Group
                  v-model="formModel.createNewService"
                  :disabled="isEditMode"
                >
                  <Radio :label="true">
                    {{ $t('平台自动生成') }}
                  </Radio>
                  <Radio :label="false">
                    {{ $t('从现有引入') }}
                  </Radio>
                </Radio.Group>
              </Form.FormItem>

              <!-- 北极星环境类型 -->
              <Form.FormItem
                :label="$t('北极星环境类型')"
                property="polarisNamespace"
                required
              >
                <Radio.Group
                  v-model="formModel.polarisNamespace"
                  :disabled="isEditMode"
                >
                  <Radio
                    v-for="env in polarisEnvTypes"
                    :key="env"
                    :label="env"
                  />
                </Radio.Group>
              </Form.FormItem>

              <!-- 北极星服务名 -->
              <Form.FormItem
                :label="$t('北极星服务名')"
                property="polarisName"
                required
              >
                <Input
                  v-model.trim="formModel.polarisName"
                  :placeholder="$t('请输入数字、英文字母、.、-、_，长度不超过128个字符')"
                  :readonly="isEditMode"
                />
              </Form.FormItem>

              <!-- 健康检查 -->
              <Form.FormItem
                :label="$t('健康检查')"
                property="enableHealthCheck"
                required
              >
                <div class="flex items-center">
                  <Switcher
                    v-model="formModel.enableHealthCheck"
                    theme="primary"
                  />
                  <span class="ml-[10px] text-[12px] text-[#63656E]">
                    {{ $t('开启后将自动上报 tRPC 服务的健康状态到北极星') }}
                  </span>
                </div>
                <Alert
                  v-if="showHealthCheckWarning"
                  class="mt-[12px]"
                  theme="warning"
                >
                  <template #title>
                    <span class="text-[#4D4F56] text-[12px]">
                      {{ $t('需要在框架配置文件的 server.service 中包含该北极星服务名，健康检查才会生效：') }}
                    </span>
                  </template>
                  <MsHighlightjs
                    class="mt-[8px]"
                    :code="healthCheckExample.code"
                    :highlights="healthCheckExample.highlights"
                    :show-tools="false"
                    :title="currentFileSpec?.fileName"
                  >
                  </MsHighlightjs>
                </Alert>
              </Form.FormItem>

              <!-- 北极星Token (仅在从现有引入时显示) -->
              <Form.FormItem
                v-if="!formModel.createNewService"
                :label="$t('北极星Token')"
                property="polarisToken"
                required
              >
                <Input
                  v-model.trim="formModel.polarisToken"
                  :placeholder="$t('请输入北极星Token')"
                />
                <Alert
                  v-if="showPolarisTokenChangeAlert"
                  class="mt-[12px]"
                  closable
                  theme="warning"
                  :title="$t('{0}已修改，保存后需重新部署才会生效。', [$t('北极星Token')])"
                />
              </Form.FormItem>

              <Form.FormItem
                v-if="formModel.createNewService"
                :label="$t('北极星负责人')"
                property="operator"
                required
              >
                <UserSelector
                  v-model="formModel.operator"
                  class="w-[552px]"
                  :clearable="!isEditMode"
                  :disabled="isEditMode"
                  multiple
                />
                <template v-if="!isEditMode">
                  <span class="text-[#979BA5]">{{ $t('创建后不可修改，请认真填写；可快捷填入') }}: </span>
                  <Button
                    text
                    theme="primary"
                    @click="handleFillOperator('admin')"
                  >
                    <i class="bkms-icon bkms-icon-usergroup text-[14px]"></i>
                    <span class="ml-[4px]">{{ $t('所有管理员') }}</span>
                  </Button>
                  <Button
                    class="ml-[12px]"
                    text
                    theme="primary"
                    @click="handleFillOperator('sre')"
                  >
                    <i class="bkms-icon bkms-icon-usergroup text-[14px]"></i>
                    <span class="ml-[4px]">{{ $t('所有SRE') }}</span>
                  </Button>
                  <Button
                    class="ml-[12px]"
                    text
                    theme="primary"
                    @click="handleFillOperator('developer')"
                  >
                    <i class="bkms-icon bkms-icon-usergroup text-[14px]"></i>
                    <span class="ml-[4px]">{{ $t('所有开发者') }}</span>
                  </Button>
                </template>
              </Form.FormItem>

              <!-- 服务标签 -->
              <Form.FormItem
                :label="$t('服务标签')"
                property="serviceLabels"
              >
                <template #label>
                  <span>{{ $t('服务标签') }}</span>
                  <i class="bkms-icon bkms-icon-warning-circle text-[14px] text-[#979BA5] ml-[4px]"></i>
                  <span class="text-[#979BA5] text-[12px] ml-[4px]">
                    {{ $t('服务标签支持通过表达式 {0} 引用环境变量', [expressTemplate]) }}
                  </span>
                </template>
                <FlexRow
                  class="mb-[12px]"
                  rclass="flex items-center"
                >
                  <template #left>
                    <Radio.Group
                      v-model="inputMode"
                      type="capsule"
                    >
                      <Radio.Button label="keyvalue">
                        <span class="flex items-center">
                          <i class="bkms-icon bkms-icon-single-column mr-[2px]"></i>
                          <span>{{ $t('表格模式') }}</span>
                        </span>
                      </Radio.Button>
                      <Radio.Button label="text">
                        <span class="flex items-center">
                          <span class="mr-[4px] h-[12px] leading-[12px] underline underline-offset-1">A</span>
                          <span>{{ $t('文本模式') }}</span>
                        </span>
                      </Radio.Button>
                    </Radio.Group>
                  </template>
                  <template #right>
                    <Copy
                      v-show="inputMode === 'text'"
                      class="cursor-pointer !hover:text-[#3A84FF]"
                      height="16"
                      :title="$t('复制')"
                      width="16"
                      @click.stop="handleCopyLabels"
                    />
                  </template>
                </FlexRow>
                <div v-show="inputMode === 'keyvalue'">
                  <KeyValue
                    v-model="formModel.serviceLabels"
                    @init:model-value="handleInitKeyValue"
                  />
                </div>
                <Input
                  v-show="inputMode === 'text'"
                  v-model="textContent"
                  :placeholder="$t('请输入参数名和参数值，如 {0}，多个参数换行分隔', ['key=value'])"
                  :rows="10"
                  type="textarea"
                />
              </Form.FormItem>
            </ToggleCard>
          </Form>
        </div>
      </template>
      <template #aside>
        <ViewDefaultEnvVars
          ref="envVarRef"
          :border="false"
          :custom-request-fn="handleGetVarEnv"
          :env-list="envList"
        />
      </template>
    </CollapsibleAsideLayout>
    <!-- 端口，北极星Token变更，部署提示 -->
    <PolarisRedeployTip
      v-else
      :config="redeployConfig"
      :env-list="envList"
      :loading="redeployChecking"
      @close="handleClose"
      @go-deploy="envName => emit('go-deploy', envName)"
      @no-redeploy="emit('no-redeploy')"
    />
    <template
      v-if="!showRedeployTip"
      #footer
    >
      <div class="flex items-center">
        <Button
          class="min-w-[88px]"
          :loading="confirmLoading"
          theme="primary"
          @click="handleSave"
        >
          {{ $t('确定') }}
        </Button>
        <Button
          class="min-w-[88px] ml-[8px]"
          @click="handleClose"
        >
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Alert, Button, Form, Input, Message, Radio, Sideslider, Switcher } from 'bkui-vue';
  import { Copy } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { EnvOutput } from '~/@types/v1/env';
  import {
    CreateAppPolarisConfigInput,
    CreateAppPolarisConfigRequest,
    PatchAppPolarisConfigRequest,
    PolarisConfigOutputObj,
  } from '~/@types/v1/polaris-config';
  import { EnvvarsService, PolarisConfigService } from '~/api/modules/v1';
  import { WorkspaceService } from '~/api/modules/v1/workspace';
  import { BKMS_REGEX } from '~/common/const';
  import { copyText } from '~/common/util';
  import EnvGroupSelector from '~/components/env-group-selector.vue';
  import KeyValue from '~/components/key-value.vue';
  import { useFocusOnErrorField } from '~/composables/use-focus-on-error-field';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import useSpecField from '~/composables/use-spec-field';
  import { useAppDetail } from '~/stores/app-detail';
  import { useUserStore } from '~/stores/user';

  import PolarisRedeployTip from './polaris-redeploy-tip.vue';

  const expressTemplate = '${{ env.<Key> }}';

  /** 为搭配UserSelector使用，operator类型在前端应为string[]，接口需要转换为string */

  type FormModelType = Partial<Omit<CreateAppPolarisConfigRequest, 'operator'> & { operator: string[] }>;

  const isShow = defineModel<boolean>('isShow');
  const props = withDefaults(
    defineProps<{
      editData?: PolarisConfigOutputObj; // 编辑时传入的数据
      envList: EnvOutput[];
      redeployChecking?: boolean;
      redeployConfig?: PolarisConfigOutputObj;
    }>(),
    {
      redeployChecking: false,
    },
  );
  const emit = defineEmits<{
    (e: 'confirm', payload: { configName?: string; mode: 'create' | 'edit'; needsRedeployTip?: boolean }): void;
    (e: 'go-deploy', envName?: string): void;
    (e: 'no-redeploy'): void;
  }>();

  const { t } = useI18n();
  const appDetailStore = useAppDetail();
  const userStore = useUserStore();
  const { focusOnErrorField } = useFocusOnErrorField();

  // 角色成员映射表 { admin: [...], developer: [...], sre: [...], operator: [...] }
  const memberRoleMap = ref<Record<string, string[]>>({});

  // 获取工作空间下角色成员组列表
  async function fetchRoleMemberGroups() {
    const workspaceID = appDetailStore.appDetail?.workspaceID;
    if (!workspaceID) return;
    try {
      const res = await WorkspaceService.listWorkspaceRoleMemberGroups({ workspaceID });
      const map: Record<string, string[]> = {};
      res.forEach((group: { members?: string[]; roleCode?: string }) => {
        map[group.roleCode!] = group.members || [];
      });
      memberRoleMap.value = map;
    } catch (error) {
      console.error(error);
    }
  }
  function handleFillOperator(roleCode: string) {
    const members = memberRoleMap.value[roleCode] || [];
    if (members.length === 0) {
      Message({
        message: t('该角色下暂无成员'),
        theme: 'warning',
      });
      return;
    }
    formModel.value.operator = [...members];
  }
  const { specFieldName } = useSpecField();

  const currentFileSpec = computed(() => appDetailStore.appDetail?.appModelSpec?.[specFieldName.value]);

  const formRef = ref();
  const confirmLoading = ref(false);
  const isEditMode = computed(() => !!props.editData);
  const isEditSaved = ref(false);
  const needsRedeployTipAfterSave = ref(false);
  const isCollaspeAside = ref(true);
  const envVarRef = ref();
  const showRedeployTip = computed(
    () => needsRedeployTipAfterSave.value && isEditSaved.value && (props.redeployChecking || !!props.redeployConfig),
  );
  // 侧栏宽度
  const sliderWidth = computed(() => (showRedeployTip.value || isCollaspeAside.value ? 960 : 1400));

  // 北极星环境类型选项
  const polarisEnvTypes = ['Development', 'Test', 'Pre-release', 'Production'];

  const defaultFormValue = ref<FormModelType>({
    instanceKey: '',
    servicePort: undefined,
    scopeEnvNames: [],
    createNewService: false,
    enableHealthCheck: false,
    polarisToken: '',
    polarisName: '',
    polarisNamespace: '' as CreateAppPolarisConfigInput['polarisNamespace'],
    serviceLabels: {},
    operator: userStore.userInfo.user_id ? [userStore.userInfo.user_id] : [],
  });
  const formModel = ref<FormModelType>({ ...defaultFormValue.value });

  // 检测配置文件中是否包含北极星服务名
  const hasPolarisNameInConfig = computed(() => {
    const fileContent = currentFileSpec.value?.fileContent || '';
    const polarisName = formModel.value.polarisName;
    if (!polarisName || !fileContent) return false;
    return fileContent.includes(polarisName);
  });

  // 是否显示健康检查的配置文件提示（开启健康检查且配置文件中未包含北极星服务名时显示）
  const showHealthCheckWarning = computed(() => formModel.value.enableHealthCheck && !hasPolarisNameInConfig.value);

  // 编辑模式下，服务端口与原始值不一致时提示需重新部署
  const showServicePortChangeAlert = computed(() => {
    if (!isEditMode.value) return false;
    const currentPort = formModel.value.servicePort ? Number(formModel.value.servicePort) : '';
    const originalPort = props.editData?.servicePort ? Number(props.editData.servicePort) : '';
    return currentPort !== originalPort;
  });

  // 编辑模式下，北极星 Token 与原始值不一致时提示需重新部署
  const showPolarisTokenChangeAlert = computed(() => {
    if (!isEditMode.value || formModel.value.createNewService) return false;
    return String(formModel.value.polarisToken || '') !== String(props.editData?.polarisToken || '');
  });

  const hasRedeployFieldChanged = computed(() => showServicePortChangeAlert.value || showPolarisTokenChangeAlert.value);

  // 健康检查示例代码及高亮标记（服务名为空时，占位文本用醒目颜色提示）
  const healthCheckExample = computed(() => {
    const placeholder = t('北极星服务名需与此处一致');
    const name = formModel.value.polarisName || placeholder;
    return {
      code: `server:
    service:
        - name: ${name}`,
      highlights: formModel.value.polarisName ? [] : [{ text: placeholder, color: '#F59500' }],
    };
  });

  // 使用 useLeaveConfirm hook 管理表单变化检测
  const { confirmBox, forceCleanDirtyTag, withPausedWatch } = useLeaveConfirm(formModel);
  // 表单验证规则
  const rules = {
    instanceKey: [
      {
        validator: (value: string) => BKMS_REGEX.instanceKeyNoLimitRegex.test(value),
        message: t('实例名称格式不正确，需以字母开头，只能包含字母、数字、下划线'),
        trigger: 'blur',
      },
    ],
    servicePort: [
      {
        validator: (value: number | string) => {
          const num = Number(value);
          return num >= 1 && num <= 65535;
        },
        message: t('请输入1-65535之间的端口号'),
        trigger: 'blur',
      },
    ],
    polarisToken: [
      {
        validator: (value: string) => {
          if (!formModel.value.createNewService) {
            return value.length > 0;
          }
          return true;
        },
        message: t('北极星Token不能为空'),
        trigger: 'blur',
      },
    ],
    polarisName: [
      {
        validator: (value: string) => BKMS_REGEX.polarisServiceNameRegex.test(value),
        message: t('请输入数字、英文字母、.、-、_，长度不超过128个字符'),
        trigger: 'blur',
      },
    ],
    operator: [
      {
        required: true,
        message: t('北极星负责人不能为空'),
        trigger: 'blur',
      },
    ],
  };

  function handleBeforeClose() {
    return confirmBox();
  }

  async function handleClose() {
    if (await handleBeforeClose()) {
      isShow.value = false;
      isEditSaved.value = false;
      needsRedeployTipAfterSave.value = false;
      formRef.value?.clearValidate?.();
      inputMode.value = 'keyvalue';
      textContent.value = '';
    }
  }

  // 获取环境变量
  function handleGetVarEnv(env: string) {
    return EnvvarsService.listAppEnvVars({
      appID: appDetailStore.appID,
      envName: env,
    });
  }
  /**
   * @description 初始化会update两次,这里的formData.selector不好用watch + once清除DirtyTag
   * 故选择让KeyValue抛出init方法，作为初始化成功的钩子，从而清除formChange的dirtyTag
   */
  function handleInitKeyValue() {
    forceCleanDirtyTag();
  }

  let refreshed = false;
  function handleRefreshEnvVar() {
    envVarRef.value?.reRefreshTable();
  }

  async function handleToggleisCollaspeAside() {
    isCollaspeAside.value = !isCollaspeAside.value;
    if (!isCollaspeAside.value && !refreshed) {
      setTimeout(() => {
        handleRefreshEnvVar();
        refreshed = true;
      }, 300);
    }
  }

  // 初始化表单数据
  function initFormData() {
    if (props.editData) {
      // 编辑模式：填充现有数据
      formModel.value = {
        instanceKey: props.editData?.instanceKey,
        servicePort: props.editData?.servicePort,
        scopeEnvNames: props.editData?.scopeEnvNames || [],
        createNewService: Boolean(props.editData?.depSvcInstID),
        enableHealthCheck: props.editData?.enableHealthCheck ?? false,
        polarisToken: props.editData?.polarisToken,
        polarisName: props.editData?.polarisName,
        polarisNamespace: props.editData?.polarisNamespace as CreateAppPolarisConfigInput['polarisNamespace'],
        serviceLabels: props.editData?.serviceLabels,
        operator: props.editData?.operator?.split(',') || [],
      };
    } else {
      formModel.value = { ...defaultFormValue.value };
    }
  }

  // 服务标签输入模式
  const inputMode = ref<'keyvalue' | 'text'>('keyvalue');
  const textContent = ref('');

  /** 将服务端口从字符串转为数字类型并回写表单，确保提交时端口值为 Number */
  function normalizeServicePort() {
    const servicePort = Number(formModel.value.servicePort);
    formModel.value.servicePort = servicePort;
    return servicePort;
  }

  // Record<string, string> 转文本
  function recordToText(data: Record<string, string>): string {
    return Object.entries(data)
      .filter(([key]) => key) // 只保留有 key 的行
      .map(([key, value]) => `${key}=${value}`)
      .join('\n');
  }

  // 文本转 Record<string, string>
  function textToRecord(text: string): Record<string, string> {
    if (!text.trim()) {
      return {};
    }

    const lines = text.split('\n');
    const result: Record<string, string> = {};

    for (const line of lines) {
      const trimmedLine = line.trim();

      // 跳过空行
      if (!trimmedLine) continue;

      // 查找等号分隔符
      const delimiterIndex = trimmedLine.indexOf('=');

      // 如果没有等号，或者等号在开头，忽略该行
      if (delimiterIndex <= 0) continue;

      const key = trimmedLine.substring(0, delimiterIndex).trim();
      const value = trimmedLine.substring(delimiterIndex + 1).trim();

      // 只添加有 key 的行（key 不为空）
      if (key) {
        result[key] = value;
      }
    }

    return result;
  }
  // 监听模式切换，进行数据同步
  watch(inputMode, async (newType, oldType) => {
    if (oldType === 'keyvalue' && newType === 'text') {
      // 从 keyvalue 切换到文本：同步 Record 数据到文本
      textContent.value = recordToText(formModel.value.serviceLabels || {});
    } else if (oldType === 'text' && newType === 'keyvalue') {
      // 从文本切换到 keyvalue：同步文本数据到 Record
      formModel.value.serviceLabels = textToRecord(textContent.value);
    }
  });

  /**
   * @description 复制服务标签
   */
  function handleCopyLabels() {
    copyText(textContent.value);
  }

  // 创建北极星配置
  async function handleCreate() {
    try {
      confirmLoading.value = true;

      if (inputMode.value === 'text') {
        formModel.value.serviceLabels = textToRecord(textContent.value);
      }
      const servicePort = normalizeServicePort();

      // 处理 operator：如果是 createNewService，将数组转为逗号分隔的字符串
      const operator =
        formModel.value.createNewService && formModel.value.operator ? formModel.value.operator.join(',') : '';

      // 构建请求参数，确保类型正确
      const requestParams: CreateAppPolarisConfigRequest = {
        appID: appDetailStore.appID,
        ...(formModel.value as FormModelType),
        servicePort,
        operator,
      } as CreateAppPolarisConfigRequest;

      await PolarisConfigService.createAppPolarisConfig(requestParams);
      forceCleanDirtyTag(() => {
        Message({
          message: t('操作成功'),
          theme: 'success',
        });
        emit('confirm', { mode: 'create' });
        handleClose();
      });
    } catch (err) {
      console.error(err);
    } finally {
      confirmLoading.value = false;
    }
  }

  // 保存（根据模式调用对应方法）
  async function handleSave() {
    const valid = await formRef.value.validate().catch(() => false);
    if (!valid) {
      focusOnErrorField();
      return;
    }

    if (isEditMode.value) {
      await handleUpdate();
    } else {
      await handleCreate();
    }
  }

  // 更新北极星配置
  async function handleUpdate() {
    try {
      confirmLoading.value = true;

      if (inputMode.value === 'text') {
        formModel.value.serviceLabels = textToRecord(textContent.value);
      }
      const servicePort = normalizeServicePort();
      const needsRedeployTip = hasRedeployFieldChanged.value;
      // 构建更新参数，确保必需字段存在
      const params: PatchAppPolarisConfigRequest = {
        appID: appDetailStore.appID,
        configName: props.editData?.name || '',
        servicePort,
        enableHealthCheck: formModel.value.enableHealthCheck,
        serviceLabels: formModel.value.serviceLabels,
        instanceKey: formModel.value.instanceKey || '',
        polarisToken: formModel.value.polarisToken,
        scopeEnvNames: formModel.value.scopeEnvNames as string[],
      };
      await PolarisConfigService.patchAppPolarisConfig(params);
      forceCleanDirtyTag(() => {
        confirmLoading.value = false;
        needsRedeployTipAfterSave.value = needsRedeployTip;
        if (needsRedeployTip) {
          isEditSaved.value = true;
          emit('confirm', { mode: 'edit', configName: props.editData?.name || '', needsRedeployTip });
          return;
        }

        Message({
          message: t('操作成功'),
          theme: 'success',
        });
        emit('confirm', { mode: 'edit', configName: props.editData?.name || '', needsRedeployTip });
        handleClose();
      });
    } catch (err) {
      console.error(err);
      confirmLoading.value = false;
    }
  }

  // 监听 isShow 变化，当对话框打开时初始化表单
  watch(
    () => isShow.value,
    newVal => {
      if (newVal) {
        isEditSaved.value = false;
        needsRedeployTipAfterSave.value = false;
        withPausedWatch(() => {
          initFormData();
        });
        // 非编辑模式下预加载角色成员组数据
        if (!isEditMode.value) {
          fetchRoleMemberGroups();
        }
      }
    },
    { immediate: true },
  );
</script>

<style lang="postcss" scoped>
  :deep(.bk-form-label) {
    color: #4d4f56;
  }
  :deep(.bk-checkbox-group) {
    .bk-checkbox {
      margin-left: 0px;
    }
  }
  :deep(.bk-alert-content) {
    padding-right: 22px;
  }
</style>
