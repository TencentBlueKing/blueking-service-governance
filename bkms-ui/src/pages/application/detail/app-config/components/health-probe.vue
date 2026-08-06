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
    :show-edit-icon="false"
  >
    <template #title>
      <div class="flex items-center">
        <div class="text-[14px]">{{ $t('健康探针') }}</div>
        <EnvScopeTag :current-env="currentEnv" />
      </div>
    </template>
    <div class="bg-[#FFF] p-[16px]">
      <div :class="['flex gap-[16px]', editingProbe === null ? 'items-stretch' : 'items-start']">
        <template
          v-for="item in probeItems"
          :key="item.key"
        >
          <ProbeEditSection
            v-if="editingProbe === item.key"
            :ref="el => setEditRef(item.key, el)"
            v-model="formModel[item.key]!"
            :disable-confirm-content="item.disableConfirmContent"
            :disable-confirm-title="getDisableConfirmTitle(item.key)"
            :is-field-modified="isFieldModified(item.key)"
            :is-probe-new="isNewProbe"
            :label="item.label"
            :reset-confirm-content="item.resetConfirmContent"
            :resetting="isResetting"
            :saving="isSaving"
            @cancel="handleCancel(item.key)"
            @disable="handleDisableProbe(item.key)"
            @reset-default="handleResetSingleProbe(item.key)"
            @save="handleSaveSingleProbe(item.key)"
          />
          <ProbeViewSection
            v-else
            :class="{ 'startup-probe-deemphasized': item.key === 'startup' && arePrimaryProbesUnconfigured }"
            :description="item.description"
            :disabled="item.viewDisabled"
            :disabled-tip="item.viewDisabledTip"
            :editing-tip="
              editingProbe !== null && editingProbe !== item.key ? $t('当前有其他探针正在编辑中，请先保存或取消') : ''
            "
            :label="item.label"
            :modified="isFieldModified(item.key)"
            :probe="formModel[item.key]!"
            :show-edit-icon="true"
            @edit="handleEdit(item.key)"
          />
        </template>
      </div>
    </div>
  </BkmsContent>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Message } from 'bkui-vue';
  import { cloneDeep } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { AppSpecProbeInput, AppSpecProbeOutput, EnvAppSpecProbeInput, ProbeOutput } from '~/@types/v1/app-spec';
  import { AppSpecService } from '~/api/modules/v1';
  import BkmsContent from '~/components/bkms-content.vue';
  import { useAppDetail } from '~/stores/app-detail';

  import EnvScopeTag from './env-scope-tag.vue';
  import ProbeEditSection from './probe-edit-section.vue';
  import ProbeViewSection from './probe-view-section.vue';
  import { ProbeType } from './types';

  import type { ExtendedEnv } from './types';

  type ProbeKey = 'liveness' | 'readiness' | 'startup';

  interface Props {
    currentEnv: ExtendedEnv | null;
  }

  const props = defineProps<Props>();

  const emit = defineEmits<{
    'env-modified-change': [];
    'loading-change': [value: boolean];
  }>();

  const { t } = useI18n();
  const appDetailStore = useAppDetail();

  /** 当前正在编辑的探针 key，null 表示全部查看态 */
  const editingProbe = ref<null | ProbeKey>(null);
  const loading = ref(true);
  const isSaving = ref(false);
  const isResetting = ref(false);

  /** 存活探针是否已配置（编辑态时以编辑前快照为准，保存后才生效） */
  const isLivenessEnabled = computed(() => {
    if (editingProbe.value === 'liveness' && probeSnapshot.value) {
      return !!probeSnapshot.value.probeHandler?.type;
    }
    return !!formModel.value.liveness?.probeHandler?.type;
  });

  /** 就绪探针是否已配置（编辑态时以编辑前快照为准，保存后才生效） */
  const isReadinessEnabled = computed(() => {
    if (editingProbe.value === 'readiness' && probeSnapshot.value) {
      return !!probeSnapshot.value.probeHandler?.type;
    }
    return !!formModel.value.readiness?.probeHandler?.type;
  });

  /** 启动探针是否已配置 */
  const isStartupEnabled = computed(() => !!formModel.value.startup?.probeHandler?.type);

  const arePrimaryProbesUnconfigured = computed(() => !isLivenessEnabled.value && !isReadinessEnabled.value);

  /** 探针名称映射 */
  const PROBE_LABEL_MAP: Record<ProbeKey, string> = {
    liveness: t('存活探针'),
    readiness: t('就绪探针'),
    startup: t('启动探针'),
  };

  /** 当前编辑的探针在编辑前是否未配置（用于禁用停用按钮） */
  const isNewProbe = computed(() => !probeSnapshot.value?.probeHandler?.type);

  /** 编辑态 ref 映射 */
  const editRefs = ref<Partial<Record<ProbeKey, InstanceType<typeof ProbeEditSection>>>>({});

  function setEditRef(key: ProbeKey, el: unknown) {
    if (el) {
      editRefs.value[key] = el as InstanceType<typeof ProbeEditSection>;
    } else {
      delete editRefs.value[key];
    }
  }

  /** 探针描述映射 */
  const PROBE_DESC_MAP: Record<ProbeKey, string> = {
    liveness: t('检测容器是否存活，探测失败时 kubelet 将杀死容器并根据重启策略处理。'),
    readiness: t('检测容器是否准备好接收流量，探测失败时 Service 将移除该 Pod 的 Endpoint。'),
    startup: t('检测容器内应用是否已完成启动，成功前其余探针均不生效，探测失败时将杀死容器并重启。'),
  };

  /** 就绪探针的默认配置是否为未配置（恢复默认 == 停用） */
  const isReadinessDefaultDisabled = computed(() => !defaultCache?.readiness?.probeHandler?.type);

  /** 三个探针的配置列表，驱动 v-for 渲染 */
  const probeItems = computed(() => [
    {
      key: 'liveness' as ProbeKey,
      label: PROBE_LABEL_MAP.liveness,
      description: PROBE_DESC_MAP.liveness,
      disableConfirmContent: '',
      resetConfirmContent: '',
      viewDisabled: false,
      viewDisabledTip: '',
    },
    {
      key: 'readiness' as ProbeKey,
      label: PROBE_LABEL_MAP.readiness,
      description: PROBE_DESC_MAP.readiness,
      disableConfirmContent: isStartupEnabled.value ? t('会同步停用 “启动探针” 请确认') : '',
      resetConfirmContent:
        isReadinessDefaultDisabled.value && isStartupEnabled.value
          ? t('恢复默认配置将停用就绪探针，会同步停用 “启动探针” 请确认')
          : '',
      viewDisabled: false,
      viewDisabledTip: '',
    },
    {
      key: 'startup' as ProbeKey,
      label: PROBE_LABEL_MAP.startup,
      description: PROBE_DESC_MAP.startup,
      disableConfirmContent: '',
      resetConfirmContent: '',
      viewDisabled: !isReadinessEnabled.value,
      viewDisabledTip: t('启动探针需依赖就绪探针开启后方可配置'),
    },
  ]);

  // ---------- 默认探针值 ----------
  const DEFAULT_PROBE: ProbeOutput = {
    probeHandler: {
      type: '',
      command: [],
      url: '',
      headers: {},
      port: '' as unknown as number,
    },
    initialDelaySeconds: 0,
    timeoutSeconds: 1,
    periodSeconds: 10,
    successThreshold: 1,
    failureThreshold: 3,
  };

  // 表单数据
  const formModel = ref<AppSpecProbeOutput>({
    liveness: cloneDeep(DEFAULT_PROBE),
    readiness: cloneDeep(DEFAULT_PROBE),
    startup: cloneDeep(DEFAULT_PROBE),
  });

  // ---------- 默认值缓存 & 环境覆盖检测 ----------
  let defaultCache: AppSpecProbeOutput | null = null;
  let fetchProbeRequestID = 0;
  const probeSnapshot = ref<null | ProbeOutput>(null);
  const overriddenFields = ref<Set<ProbeKey>>(new Set());

  /** 构建提交给接口的 probe payload：未启用的探针传 null */
  function buildProbePayload(): AppSpecProbeOutput {
    const raw = cloneDeep(formModel.value);
    const clean = (p: ProbeOutput | undefined) => (p?.probeHandler?.type ? cleanProbeHandler(p) : null);
    return {
      liveness: clean(raw.liveness),
      readiness: clean(raw.readiness),
      startup: clean(raw.startup),
    } as AppSpecProbeOutput;
  }

  /** 清理探针中与当前类型无关的字段 */
  function cleanProbeHandler(probe: ProbeOutput): ProbeOutput {
    const handler = probe.probeHandler as unknown as Record<string, unknown>;
    const removeKeys: Record<string, string[]> = {
      [ProbeType.EXEC]: ['port', 'url', 'headers'],
      [ProbeType.TCP]: ['command', 'shCommand', 'url', 'headers'],
      [ProbeType.HTTP]: ['command', 'shCommand'],
    };
    for (const key of removeKeys[probe.probeHandler!.type!] ?? []) {
      delete handler[key];
    }
    // EXEC 类型下，command 与 shCommand 互斥：有 shCommand 时删 command，反之删 shCommand
    if (probe.probeHandler?.type === ProbeType.EXEC) {
      if (handler.shCommand) {
        delete handler.command;
      } else {
        delete handler.shCommand;
      }
    }
    return probe;
  }

  function createProbeFetchContext() {
    const requestID = ++fetchProbeRequestID;
    const appID = appDetailStore.appID;
    const currentEnv = props.currentEnv;
    if (!appID || !currentEnv) return null;

    return {
      appID,
      currentEnv,
      requestID,
    };
  }

  /** 获取并缓存应用级默认配置 */
  async function fetchAndCacheDefault(appID = appDetailStore.appID): Promise<AppSpecProbeOutput | null> {
    if (defaultCache) return defaultCache;
    if (!appID) return null;
    try {
      const data = await AppSpecService.getAppDefaultAppSpecProbe({ appID });
      if (data && appDetailStore.appID === appID) defaultCache = data;
      return data;
    } catch {
      return null;
    }
  }

  /** 获取环境覆盖配置，判断哪些探针被覆盖过 */
  async function fetchEnvOverrideFields(envName: string, appID = appDetailStore.appID): Promise<Set<ProbeKey>> {
    if (!appID) return new Set();

    try {
      const envOverride = await AppSpecService.getEnvAppSpecProbe({ appID, envName }, { interceptorErr: false });
      const fields = new Set<ProbeKey>();
      if (envOverride) {
        const raw = envOverride as unknown as Record<string, unknown>;
        for (const key of ['liveness', 'readiness', 'startup'] as ProbeKey[]) {
          if (raw[key] != null) fields.add(key);
        }
      }
      return fields;
    } catch {
      return new Set();
    }
  }

  /** 获取探针配置数据 */
  async function fetchProbeData() {
    const context = createProbeFetchContext();
    if (!context) {
      loading.value = false;
      return;
    }

    loading.value = true;

    try {
      const defaultData = await fetchAndCacheDefault(context.appID);
      if (context.requestID !== fetchProbeRequestID) return;

      let data: AppSpecProbeOutput | null = null;

      if (context.currentEnv.isDefault) {
        data = defaultData;
        overriddenFields.value = new Set();
      } else {
        data = await AppSpecService.getEnvEffectiveAppSpecProbe({
          appID: context.appID,
          envName: context.currentEnv.name,
        });
        if (context.requestID !== fetchProbeRequestID) return;

        const fields = await fetchEnvOverrideFields(context.currentEnv.name, context.appID);
        if (context.requestID !== fetchProbeRequestID) return;
        overriddenFields.value = fields;
      }

      if (data) {
        fillFormFromOutput(data);
      } else {
        resetFormToDefaults();
      }
    } catch {
      if (context.requestID !== fetchProbeRequestID) return;
      resetFormToDefaults();
      overriddenFields.value = new Set();
    } finally {
      if (context.requestID === fetchProbeRequestID) {
        loading.value = false;
      }
    }
  }

  /** 从接口返回数据填充表单 */
  function fillFormFromOutput(data: AppSpecProbeOutput) {
    for (const key of ['liveness', 'readiness', 'startup'] as ProbeKey[]) {
      formModel.value[key] = data[key] ? cloneDeep(data[key]) : cloneDeep(DEFAULT_PROBE);
    }
  }

  /** 获取停用确认弹窗标题 */
  function getDisableConfirmTitle(key: ProbeKey): string {
    return t('确认停用 “{0}” ？', [PROBE_LABEL_MAP[key]]);
  }

  /** 取消编辑某个探针 */
  function handleCancel(key: ProbeKey) {
    if (probeSnapshot.value) {
      formModel.value[key] = cloneDeep(probeSnapshot.value);
    }
    editingProbe.value = null;
    probeSnapshot.value = null;
  }

  /** 停用探针：清空该探针（传 null）并保存 */
  async function handleDisableProbe(key: ProbeKey) {
    editingProbe.value = null;
    probeSnapshot.value = null;

    // 清空当前探针，停用就绪探针时同步清空启动探针
    formModel.value[key] = cloneDeep(DEFAULT_PROBE);
    if (key === 'readiness') {
      formModel.value.startup = cloneDeep(DEFAULT_PROBE);
    }

    await submitPayload();
  }

  /** 进入某个探针的编辑态 */
  function handleEdit(key: ProbeKey) {
    probeSnapshot.value = cloneDeep(formModel.value[key]) as ProbeOutput;
    if (!formModel.value[key]?.probeHandler?.type) {
      formModel.value[key]!.probeHandler!.type = ProbeType.HTTP;
    }
    editingProbe.value = key;
  }

  /** 恢复单个探针为默认配置 */
  async function handleResetSingleProbe(key: ProbeKey) {
    // 判断就绪探针恢复默认是否会导致停用，需同步处理启动探针
    const shouldResetStartup = key === 'readiness' && isReadinessDefaultDisabled.value && isStartupEnabled.value;

    if (props.currentEnv && !props.currentEnv.isDefault) {
      try {
        isResetting.value = true;
        await AppSpecService.deleteEnvAppSpecProbeByType({
          appID: appDetailStore.appID,
          envName: props.currentEnv.name,
          probeType: key,
        });
        // 同步恢复启动探针的环境覆盖
        if (shouldResetStartup) {
          await AppSpecService.deleteEnvAppSpecProbeByType({
            appID: appDetailStore.appID,
            envName: props.currentEnv.name,
            probeType: 'startup',
          });
        }
        overriddenFields.value = await fetchEnvOverrideFields(props.currentEnv.name);
        emit('env-modified-change');
      } finally {
        isResetting.value = false;
      }
    }

    // 用默认值回填表单
    const defaultProbe = defaultCache?.[key];
    formModel.value[key] = defaultProbe ? cloneDeep(defaultProbe) : cloneDeep(DEFAULT_PROBE);

    // 同步回填启动探针
    if (shouldResetStartup) {
      const defaultStartup = defaultCache?.startup;
      formModel.value.startup = defaultStartup ? cloneDeep(defaultStartup) : cloneDeep(DEFAULT_PROBE);
    }

    Message({ theme: 'success', message: t('操作成功') });
    editingProbe.value = null;
    probeSnapshot.value = null;
  }

  /** 保存单个探针 */
  async function handleSaveSingleProbe(key: ProbeKey) {
    const valid = await editRefs.value[key]?.validate();
    if (!valid) return;

    await submitPayload();
    editingProbe.value = null;
    probeSnapshot.value = null;
  }

  /** 查看态：非默认环境且该探针被环境覆盖过时显示黄色标识 */
  function isFieldModified(field: ProbeKey): boolean {
    return !!props.currentEnv && !props.currentEnv.isDefault && overriddenFields.value.has(field);
  }

  /** 重置表单到硬编码默认值 */
  function resetFormToDefaults() {
    for (const key of ['liveness', 'readiness', 'startup'] as ProbeKey[]) {
      formModel.value[key] = cloneDeep(DEFAULT_PROBE);
    }
  }

  function stopProbeLoading() {
    fetchProbeRequestID += 1;
    loading.value = false;
  }

  /** 提交 payload 到接口（保存/停用共用） */
  async function submitPayload() {
    try {
      isSaving.value = true;
      const payload = buildProbePayload();

      if (props.currentEnv?.isDefault) {
        await AppSpecService.setAppDefaultAppSpecProbe({
          appID: appDetailStore.appID,
          appSpecProbe: payload as AppSpecProbeInput,
        });
        defaultCache = null;
        await fetchAndCacheDefault();
      } else if (props.currentEnv) {
        await AppSpecService.setEnvAppSpecProbe({
          appID: appDetailStore.appID,
          envName: props.currentEnv.name,
          appSpecProbe: payload as EnvAppSpecProbeInput,
        });
        overriddenFields.value = await fetchEnvOverrideFields(props.currentEnv.name);
        emit('env-modified-change');
      }

      Message({ theme: 'success', message: t('操作成功') });
    } finally {
      isSaving.value = false;
    }
  }

  // 监听环境切换，重新拉取数据
  watch(
    () => [props.currentEnv?.name, appDetailStore.appType],
    async () => {
      if (!['trpc', 'taf'].includes(appDetailStore.appType)) {
        stopProbeLoading();
        return;
      }
      editingProbe.value = null;
      probeSnapshot.value = null;
      await fetchProbeData();
    },
    { immediate: true },
  );

  watch(
    () => appDetailStore.appID,
    async newVal => {
      if (newVal && ['trpc', 'taf'].includes(appDetailStore.appType)) {
        defaultCache = null;
        await fetchProbeData();
      } else {
        stopProbeLoading();
      }
    },
  );

  watch(loading, val => emit('loading-change', val), { immediate: true });
</script>

<style lang="postcss" scoped>
  :deep(.bkms-content-title) {
    background-color: #eaebf0;
  }
</style>
