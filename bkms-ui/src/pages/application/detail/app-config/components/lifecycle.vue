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
  <BkmsContent
    :collapsible="true"
    :show-edit-icon="!isEditing"
    @edit="handleEdit"
  >
    <template #title>
      <div class="flex items-center">
        <div class="text-[14px]">{{ $t('生命周期') }}</div>
        <EnvScopeTag :current-env="currentEnv" />
      </div>
    </template>
    <div class="bg-[#FFF] p-[16px]">
      <!-- 查看态 -->
      <template v-if="!isEditing">
        <div class="grid grid-cols-2 gap-[12px] gap-y-2">
          <FieldItem
            class="min-h-[20px]"
            :class="{ '!items-start': preStopExecMode === 'exec' && formModel?.preStopCommand?.length }"
            container-height="auto"
            :field-width="205"
            value-color="#313238"
          >
            <template #field>
              <ModifiedFieldLabel
                :label="$t('退出前命令 (preStop)')"
                :line-height="20"
                :modified="isFieldModified('preStop')"
              />
            </template>
            <template #value>
              <span
                v-bk-tooltips="{
                  content: getDefaultValueTip('preStop'),
                  disabled: !getDefaultValueTip('preStop'),
                }"
                :class="[VIEW_VALUE_CLASS, { [VIEW_VALUE_MODIFIED_CLASS]: getDefaultValueTip('preStop') }]"
              >
                <template v-if="preStopEnabled">
                  <template v-if="preStopMode === 'wait'">
                    {{ $t('停止等待时间 {0} 秒', [formModel.preStopWait]) }}
                  </template>
                  <template v-else-if="preStopExecMode === 'shell'">
                    <OverflowTitle type="tips">{{ formModel.preStopShCommand || '--' }}</OverflowTitle>
                  </template>
                  <div
                    v-else
                    class="flex flex-col gap-[4px] w-full min-w-0"
                  >
                    <div
                      v-for="item in formModel.preStopCommand || []"
                      :key="item"
                      class="leading-[20px]"
                    >
                      <OverflowTitle type="tips">{{ item }}</OverflowTitle>
                    </div>
                    <span
                      v-if="!formModel.preStopCommand?.length"
                      class="text-[12px] text-[#313238]"
                      >--</span
                    >
                  </div>
                </template>
                <Tag v-else>
                  {{ $t('未启用') }}
                </Tag>
              </span>
            </template>
          </FieldItem>
          <FieldItem
            class="min-h-[20px]"
            container-height="auto"
            :field-width="205"
            value-color="#313238"
          >
            <template #field>
              <ModifiedFieldLabel
                :label="$t('优雅退出时间')"
                :modified="isFieldModified('terminationGracePeriodSeconds')"
              />
            </template>
            <template #value>
              <span
                v-bk-tooltips="{
                  content: getDefaultValueTip('terminationGracePeriodSeconds'),
                  disabled: !getDefaultValueTip('terminationGracePeriodSeconds'),
                }"
                :class="[
                  VIEW_VALUE_CLASS,
                  { [VIEW_VALUE_MODIFIED_CLASS]: getDefaultValueTip('terminationGracePeriodSeconds') },
                ]"
              >
                {{ formModel.terminationGracePeriodSeconds }} {{ $t('秒') }}
              </span>
            </template>
          </FieldItem>
        </div>
      </template>
      <!-- 编辑态 -->
      <div v-else>
        <div class="mb-[16px]">
          <div class="flex flex-col mb-[24px] text-[12px] text-[#4D4F56]">
            <div class="flex items-center gap-[4px]">
              <p :class="['text-[14px]', { 'field-diff-highlight': isFieldModified('preStop') }]">
                {{ $t('退出前命令 (preStop)') }}
                <span class="relative top-[2px] text-[#ea3636] ml-[4px]">*</span>
              </p>
            </div>
            <div class="flex items-center mt-[8px]">
              <Switcher
                v-model="preStopEnabled"
                theme="primary"
              />
              <span class="ml-[10px]">
                {{ $t('容器终止前执行的命令，常用于优雅退出场景，如等待北极星权重变更在主调方生效。') }}
              </span>
            </div>
          </div>
          <!-- preStop 关闭时，下方所有内容隐藏 -->
          <template v-if="preStopEnabled">
            <!-- preStop 配置区域 -->
            <Radio.Group
              v-model="preStopMode"
              class="flex flex-col gap-[16px]"
              @change="handlePreStopModeChange"
            >
              <div class="flex flex-col gap-[8px]">
                <Radio label="wait">
                  {{ $t('停止等待时间') }}
                </Radio>
                <Input
                  v-if="preStopMode === 'wait'"
                  v-model.trim="formModel.preStopWait"
                  class="!w-[120px] ml-[8px]"
                  :disabled="preStopMode !== 'wait'"
                  :min="0"
                  :precision="0"
                  suffix="秒"
                  type="number"
                />
              </div>
              <!-- 自定义命令 -->
              <div>
                <Radio label="command">
                  {{ $t('自定义命令') }}
                </Radio>
                <div
                  v-if="preStopMode === 'command'"
                  class="mt-[8px] w-[400px]"
                >
                  <Radio.Group
                    v-model="preStopExecMode"
                    class="pre-stop-exec-mode mb-[8px]"
                    type="capsule"
                    @change="handlePreStopExecModeChange"
                  >
                    <Radio.Button label="shell">shell</Radio.Button>
                    <Radio.Button label="exec">exec</Radio.Button>
                  </Radio.Group>
                  <RepeatableInput
                    v-if="preStopExecMode === 'exec'"
                    ref="preStopCommandRef"
                    v-model="formModel.preStopCommand"
                    required
                    trim-on-input
                  />
                  <!-- shell 模式 -->
                  <Form
                    v-else
                    ref="preStopShellFormRef"
                    form-type="vertical"
                    :model="formModel"
                  >
                    <Form.FormItem
                      class="!mb-0"
                      property="preStopShCommand"
                      :rules="rules.preStopShCommand"
                    >
                      <Input
                        v-model="formModel.preStopShCommand"
                        :placeholder="$t('请输入')"
                        :rows="4"
                        type="textarea"
                      />
                    </Form.FormItem>
                  </Form>
                </div>
              </div>
            </Radio.Group>

            <!-- 优雅退出时间 (terminationGracePeriodSeconds) -->
            <div class="mt-[24px] mb-[8px] relative">
              <div class="flex items-center gap-[4px]">
                <div
                  :class="[
                    'text-[14px] text-[#4D4F56]',
                    { 'field-diff-highlight': isFieldModified('terminationGracePeriodSeconds') },
                  ]"
                >
                  {{ $t('优雅退出时间 (terminationGracePeriodSeconds)') }}
                </div>
              </div>
              <Input
                v-model.trim="formModel.terminationGracePeriodSeconds"
                class="!w-[120px] mt-[8px]"
                :min="0"
                :precision="0"
                suffix="秒"
                type="number"
              />
              <p class="mt-[4px] text-[12px] text-[#979BA5]">
                {{ $t('等待 preStop 执行的最大时间，时间超过而 preStop 未执行完时，将强制杀死容器') }}
              </p>
            </div>
          </template>
        </div>

        <div class="!mb-0 mt-[16px] flex items-center">
          <Button
            :loading="isLoading"
            theme="primary"
            @click="handleSave"
          >
            {{ $t('保存') }}
          </Button>
          <Button
            class="ml-[8px]"
            @click="handleCancel"
          >
            {{ $t('取消') }}
          </Button>
          <!-- 仅非默认环境显示"恢复默认配置"按钮 -->
          <Button
            v-if="currentEnv && !currentEnv.isDefault"
            class="ml-[8px]"
            :loading="isResetting"
            @click="handleResetToDefault"
          >
            {{ $t('恢复默认配置') }}
          </Button>
        </div>
      </div>
    </div>
  </BkmsContent>
</template>

<script setup lang="ts">
  import { nextTick, ref, watch } from 'vue';

  import { Button, Form, Input, Message, Radio, Switcher } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import {
    AppSpecLifecycleInput,
    AppSpecLifecycleOutput,
    EnvAppSpecLifecycleInput,
    LifecycleHandlerInput,
  } from '~/@types/v1/app-spec';
  import { AppSpecService } from '~/api/modules/v1';
  import BkmsContent from '~/components/bkms-content.vue';
  import FieldItem from '~/components/field-item.vue';
  import RepeatableInput from '~/components/repeatable-input.vue';
  import { useAppDetail } from '~/stores/app-detail';

  import EnvScopeTag from './env-scope-tag.vue';
  import ModifiedFieldLabel from './modified-field-label.vue';

  import type { ExtendedEnv } from './types';

  // 查看态值样式常量
  const VIEW_VALUE_CLASS = 'text-[12px] text-[#313238]';
  const VIEW_VALUE_MODIFIED_CLASS = 'border-b border-dashed border-b-[#313238]';

  type FieldKey = 'preStop' | 'terminationGracePeriodSeconds';
  type PreStopExecMode = 'exec' | 'shell';
  type PreStopMode = 'command' | 'wait';

  interface PreStopSummary {
    command: string[];
    enabled: boolean;
    execMode: PreStopExecMode;
    grace: number;
    mode: PreStopMode;
    shCommand: string;
    wait: number;
  }

  interface Props {
    currentEnv: ExtendedEnv | null;
  }

  const props = defineProps<Props>();

  const emit = defineEmits<{
    'env-modified-change': [];
  }>();

  const { t } = useI18n();
  const appDetailStore = useAppDetail();

  const isEditing = ref(false);
  const isLoading = ref(false);
  const isResetting = ref(false);

  const preStopCommandRef = ref<InstanceType<typeof RepeatableInput> | null>(null);
  const preStopShellFormRef = ref<InstanceType<typeof Form> | null>(null);

  // preStop 是否启用
  const preStopEnabled = ref(false);

  // preStop 模式：wait（停止等待时间）或 command（自定义命令）
  const preStopMode = ref<PreStopMode>('wait');
  const preStopExecMode = ref<PreStopExecMode>('exec');

  // 表单数据
  const formModel = ref({
    preStopWait: 30,
    preStopCommand: [] as string[],
    preStopShCommand: '',
    terminationGracePeriodSeconds: 30,
  });

  const rules = {
    preStopShCommand: [
      {
        validator: (value: string) => !!value?.trim(),
        message: t('必填项'),
        trigger: 'blur',
      },
    ],
  };

  // ---------- 默认值缓存 & 环境覆盖检测 ----------

  /** 缓存应用级默认配置（用于对比差异） */
  let defaultCache: AppSpecLifecycleOutput | null = null;
  let defaultPreStopSummary = createPreStopSummary();

  /** 被环境覆盖过的字段集合（通过 GetEnvAppSpecLifecycle 接口判断） */
  const overriddenFields = ref<Set<FieldKey>>(new Set());

  let formSnapshot: null | typeof formModel.value = null;
  let preStopEnabledSnapshot = false;
  let preStopModeSnapshot: PreStopMode = 'wait';
  let preStopExecModeSnapshot: PreStopExecMode = 'exec';

  function applyPreStopSummary(summary: PreStopSummary) {
    preStopEnabled.value = summary.enabled;
    preStopMode.value = summary.mode;
    preStopExecMode.value = summary.execMode;
    formModel.value.preStopWait = summary.wait;
    formModel.value.preStopCommand = [...summary.command];
    formModel.value.preStopShCommand = summary.shCommand;
    formModel.value.terminationGracePeriodSeconds = summary.grace;
  }

  /** 构建默认配置 payload（全量提交） */
  function buildDefaultPayload(): AppSpecLifecycleInput {
    return {
      preStop: buildPreStopValue() as LifecycleHandlerInput,
      terminationGracePeriodSeconds: Number(formModel.value.terminationGracePeriodSeconds) || 30,
    };
  }

  /** 构建环境覆盖 payload：仅传需要覆盖的字段，其余传 null（由后端视为继承默认值） */
  function buildEnvPayload() {
    return {
      preStop: isFieldNeedSave('preStop')
        ? buildPreStopValue()
        : (null as unknown as ReturnType<typeof buildPreStopValue>),
      terminationGracePeriodSeconds: isFieldNeedSave('terminationGracePeriodSeconds')
        ? Number(formModel.value.terminationGracePeriodSeconds) || 30
        : (null as unknown as number),
    };
  }

  /** 构建 preStop 字段值 */
  function buildPreStopValue(): LifecycleHandlerInput | null {
    if (!preStopEnabled.value) {
      return null;
    }
    if (preStopMode.value === 'wait') {
      return {
        type: 'EXEC',
        exec: { command: [], sleepSeconds: Number(formModel.value.preStopWait) || 0 },
      };
    }
    // 自定义命令模式
    if (preStopExecMode.value === 'shell') {
      return {
        type: 'EXEC',
        exec: { shCommand: formModel.value.preStopShCommand.trim(), sleepSeconds: 0 },
      };
    }
    return {
      type: 'EXEC',
      exec: { command: formModel.value.preStopCommand.filter(s => s.trim()), sleepSeconds: 0 },
    };
  }

  /** 将默认配置解析并缓存到本地变量 */
  function cacheDefaults(data: AppSpecLifecycleOutput | null) {
    defaultPreStopSummary = parsePreStopSummary(data);
  }

  function createPreStopSummary(partial: Partial<PreStopSummary> = {}): PreStopSummary {
    return {
      enabled: false,
      mode: 'wait',
      execMode: 'exec',
      wait: 30,
      command: [],
      shCommand: '',
      grace: 30,
      ...partial,
    };
  }

  /** 获取并缓存应用级默认配置 */
  async function fetchAndCacheDefault(): Promise<AppSpecLifecycleOutput | null> {
    if (defaultCache) return defaultCache;
    if (!appDetailStore.appID) return null;
    try {
      const data = await AppSpecService.getAppDefaultAppSpecLifecycle({ appID: appDetailStore.appID });
      if (data) {
        defaultCache = data;
        cacheDefaults(data);
      }
      return data;
    } catch {
      return null;
    }
  }

  /** 调用 GetEnvAppSpecLifecycle 获取环境覆盖配置，判断哪些字段被覆盖过 */
  async function fetchEnvOverride(envName: string) {
    try {
      const envOverride = await AppSpecService.getEnvAppSpecLifecycle(
        { appID: appDetailStore.appID, envName },
        { interceptorErr: false },
      );
      const fields = new Set<FieldKey>();
      if (envOverride) {
        const raw = envOverride as unknown as Record<string, unknown>;
        if (raw.preStop != null) fields.add('preStop');
        if (raw.terminationGracePeriodSeconds != null) fields.add('terminationGracePeriodSeconds');
      }
      overriddenFields.value = fields;
    } catch {
      // GetEnvAppSpecLifecycle 返回 404 时，所有字段均为继承
      overriddenFields.value = new Set();
    }
  }

  /** 获取生命周期配置数据 */
  async function fetchLifecycleData() {
    if (!appDetailStore.appID || !props.currentEnv) return;
    try {
      // 始终获取默认配置（用于差异对比）
      await fetchAndCacheDefault();

      let data: AppSpecLifecycleOutput | null = null;

      if (props.currentEnv.isDefault) {
        // 默认配置
        data = defaultCache;
        overriddenFields.value = new Set();
      } else {
        // 环境配置（取实际生效的）
        data = await AppSpecService.getEnvEffectiveAppSpecLifecycle({
          appID: appDetailStore.appID,
          envName: props.currentEnv.name,
        });
        // 获取环境覆盖配置，判断哪些字段被覆盖过
        await fetchEnvOverride(props.currentEnv.name);
      }

      if (data) {
        fillFormFromOutput(data);
      }
    } catch {
      // 接口失败时使用默认值
      resetFormToDefaults();
      overriddenFields.value = new Set();
    }
  }

  /** 从接口返回数据填充表单 */
  function fillFormFromOutput(data: AppSpecLifecycleOutput) {
    applyPreStopSummary(parsePreStopSummary(data));
  }

  /** 用默认配置填充表单 */
  function fillFormWithDefault() {
    if (defaultCache) {
      fillFormFromOutput(defaultCache);
    } else {
      resetFormToDefaults();
    }
  }

  /** 查看态 tooltip：显示默认值 */
  function getDefaultValueTip(field: FieldKey): string {
    if (!isFieldModified(field)) return '';
    if (field === 'preStop') {
      if (!defaultPreStopSummary.enabled) return t('默认值：未启用');
      if (defaultPreStopSummary.mode === 'wait') {
        return t('默认值：停止等待时间 {0} 秒', [defaultPreStopSummary.wait]);
      }
      if (defaultPreStopSummary.execMode === 'shell') return t('默认值：{0}', [defaultPreStopSummary.shCommand]);
      return t('默认值：{0}', [defaultPreStopSummary.command.join(' ')]);
    }
    if (field === 'terminationGracePeriodSeconds') {
      return t('默认值：{0}', [defaultPreStopSummary.grace]);
    }
    return '';
  }

  function handleCancel() {
    // 恢复快照
    if (formSnapshot) {
      formModel.value = { ...formSnapshot };
      preStopEnabled.value = preStopEnabledSnapshot;
      preStopMode.value = preStopModeSnapshot;
      preStopExecMode.value = preStopExecModeSnapshot;
    }
    isEditing.value = false;
  }

  function handleEdit() {
    // 保存快照
    formSnapshot = { ...formModel.value, preStopCommand: [...formModel.value.preStopCommand] };
    preStopEnabledSnapshot = preStopEnabled.value;
    preStopModeSnapshot = preStopMode.value;
    preStopExecModeSnapshot = preStopExecMode.value;
    isEditing.value = true;
  }

  function handlePreStopExecModeChange(type: string) {
    if (type === 'exec' && formModel.value.preStopCommand.length === 0) {
      nextTick(() => {
        preStopCommandRef.value?.add();
      });
    }
    nextTick(() => {
      preStopShellFormRef.value?.clearValidate?.();
    });
  }

  function handlePreStopModeChange(type: string) {
    if (type === 'command' && formModel.value.preStopCommand.length === 0) {
      // 切换到自定义命令模式且列表为空时，自动添加一个空行
      nextTick(() => {
        preStopCommandRef.value?.add();
      });
    }
  }

  /** 恢复默认配置：删除环境覆盖，回填默认值 */
  async function handleResetToDefault() {
    if (!props.currentEnv || props.currentEnv.isDefault) return;

    try {
      isResetting.value = true;
      await AppSpecService.deleteEnvAppSpecLifecycle({
        appID: appDetailStore.appID,
        envName: props.currentEnv.name,
      });

      fillFormWithDefault();
      overriddenFields.value = new Set();

      Message({ theme: 'success', message: t('操作成功') });
      isEditing.value = false;
      emit('env-modified-change');
    } finally {
      isResetting.value = false;
    }
  }

  /** 保存生命周期配置 */
  async function handleSave() {
    // 自定义命令模式下，按 exec/shell 子模式校验
    if (preStopEnabled.value && preStopMode.value === 'command') {
      if (preStopExecMode.value === 'shell') {
        const valid = await preStopShellFormRef.value?.validate().catch(() => false);
        if (!valid) return;
      } else {
        const valid = await preStopCommandRef.value?.validate();
        if (!valid) return;
      }
    }

    try {
      isLoading.value = true;

      if (props.currentEnv?.isDefault) {
        await AppSpecService.setAppDefaultAppSpecLifecycle({
          appID: appDetailStore.appID,
          appSpecLifecycle: buildDefaultPayload(),
        });
        // 保存默认配置后更新缓存
        defaultCache = null;
        await fetchAndCacheDefault();
      } else if (props.currentEnv) {
        // 所有字段均无变化时，跳过 SetEnv 调用，避免后端创建空的环境覆盖记录
        if (hasAnyFieldNeedSave()) {
          await AppSpecService.setEnvAppSpecLifecycle({
            appID: appDetailStore.appID,
            envName: props.currentEnv.name,
            appSpecLifecycle: buildEnvPayload() as EnvAppSpecLifecycleInput,
          });
        }
        // 保存成功后重新获取 envOverride，刷新已修改标识
        await fetchEnvOverride(props.currentEnv.name);
        emit('env-modified-change');
      }

      normalizePreStopDraftAfterSave();
      Message({ theme: 'success', message: t('操作成功') });
      isEditing.value = false;
    } finally {
      isLoading.value = false;
    }
  }

  /** 判断是否有任何字段需要保存到环境覆盖 */
  function hasAnyFieldNeedSave(): boolean {
    return isFieldNeedSave('preStop') || isFieldNeedSave('terminationGracePeriodSeconds');
  }

  /** 查看态：非默认环境且该字段被环境覆盖过时显示黄色标识 */
  function isFieldModified(field: FieldKey): boolean {
    return !!props.currentEnv && !props.currentEnv.isDefault && overriddenFields.value.has(field);
  }

  /** 判断某个字段是否需要保存到环境覆盖（仅看当前值与默认值是否不同） */
  function isFieldNeedSave(field: FieldKey): boolean {
    return field === 'preStop' ? isPreStopDiffFromDefault() : isGracePeriodDiffFromDefault();
  }

  /** 判断 terminationGracePeriodSeconds 当前值是否与默认值不同 */
  function isGracePeriodDiffFromDefault(): boolean {
    if (!defaultCache) return false;
    return Number(formModel.value.terminationGracePeriodSeconds) !== Number(defaultPreStopSummary.grace);
  }

  /** 判断 preStop 字段当前值是否与默认值不同 */
  function isPreStopDiffFromDefault(): boolean {
    if (!defaultCache) return false;
    const currentEnabled = preStopEnabled.value;
    const currentMode = preStopMode.value;
    const currentExecMode = preStopExecMode.value;
    const currentWait = Number(formModel.value.preStopWait);
    const currentCommand = formModel.value.preStopCommand.filter(s => s.trim()).join(',');
    const currentShCommand = formModel.value.preStopShCommand.trim();
    if (currentEnabled !== defaultPreStopSummary.enabled) return true;
    if (!currentEnabled) return false;
    if (currentMode !== defaultPreStopSummary.mode) return true;
    if (currentMode === 'wait') return currentWait !== defaultPreStopSummary.wait;
    if (currentExecMode !== defaultPreStopSummary.execMode) return true;
    if (currentExecMode === 'shell') return currentShCommand !== defaultPreStopSummary.shCommand.trim();
    return currentCommand !== defaultPreStopSummary.command.filter(s => s.trim()).join(',');
  }

  function normalizePreStopDraftAfterSave() {
    if (!preStopEnabled.value || preStopMode.value === 'wait') {
      preStopExecMode.value = 'exec';
      formModel.value.preStopCommand = [];
      formModel.value.preStopShCommand = '';
      return;
    }
    if (preStopExecMode.value === 'shell') {
      formModel.value.preStopCommand = [];
      formModel.value.preStopShCommand = formModel.value.preStopShCommand.trim();
    } else {
      formModel.value.preStopCommand = formModel.value.preStopCommand.filter(s => s.trim());
      formModel.value.preStopShCommand = '';
    }
  }

  /** 从 output 数据中解析出 preStop 摘要信息 */
  function parsePreStopSummary(data: AppSpecLifecycleOutput | null) {
    // 后端可能返回字符串，统一转为数字
    const grace = Number(data?.terminationGracePeriodSeconds) || 30;
    const exec = data?.preStop?.type === 'EXEC' ? data.preStop.exec : undefined;
    if (!exec) return createPreStopSummary({ grace });

    const wait = Number(exec.sleepSeconds) || 0;
    if (exec.shCommand) {
      return createPreStopSummary({
        enabled: true,
        mode: 'command',
        execMode: 'shell',
        wait,
        shCommand: exec.shCommand,
        grace,
      });
    }
    if (exec.command?.length) {
      return createPreStopSummary({
        enabled: true,
        mode: 'command',
        wait,
        command: [...exec.command],
        grace,
      });
    }
    return createPreStopSummary({ enabled: true, wait, grace });
  }

  /** 重置表单到硬编码默认值 */
  function resetFormToDefaults() {
    preStopEnabled.value = false;
    preStopMode.value = 'wait';
    preStopExecMode.value = 'exec';
    formModel.value.preStopWait = 30;
    formModel.value.preStopCommand = [];
    formModel.value.preStopShCommand = '';
    formModel.value.terminationGracePeriodSeconds = 30;
  }

  // 监听环境切换，重新拉取数据（仅 trpc/taf 应用）
  watch(
    () => props.currentEnv?.name,
    async () => {
      if (!['trpc', 'taf'].includes(appDetailStore.appType)) return;
      isEditing.value = false;
      await fetchLifecycleData();
    },
    { immediate: true },
  );

  watch(
    () => appDetailStore.appID,
    async newVal => {
      if (newVal && ['trpc', 'taf'].includes(appDetailStore.appType)) {
        defaultCache = null;
        await fetchLifecycleData();
      }
    },
  );
</script>

<style lang="postcss" scoped>
  :deep(.bkms-content-title) {
    background-color: #eaebf0;
  }

  :deep(.bk-radio-label) {
    font-size: 12px;
  }

  :deep(.pre-stop-exec-mode.bk-radio-capsule .bk-radio-button-label) {
    border: none;
    height: 24px;
    line-height: 24px;
  }

  :deep(.pre-stop-exec-mode.bk-radio-capsule .bk-radio-button.is-checked) {
    .bk-radio-button-label {
      background: #fff !important;
    }
  }

  .field-diff-highlight {
    border-left: 3px solid #ff9c01;
    padding-left: 6px;
  }
</style>
