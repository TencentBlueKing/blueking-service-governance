/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

import { computed, onScopeDispose, ref, watch } from 'vue';
import type { Ref } from 'vue';

import { isEqual } from 'lodash-es';
import { PolarisConfigService } from '~/api/modules/v1';
import { PLATFORM_WEIGHT_FACTOR_CONFIG } from '~/common/const';

import type { GetImportedPolarisServiceRequest, ImportedPolarisServiceOutput } from '~/@types/v1/polaris-config';

/** save 结果：saved 成功；busy 重复提交；cancelled 取消或失效；failed 失败；其余表示需先补连通测试、处理差异或重新确认。 */
export type SaveResult =
  'busy' | 'cancelled' | 'connection-required' | 'failed' | 'remote-changed' | 'resolution-required' | 'saved';

/** 公式可展示字段：func 为函数名，其余为线性函数的数值参数。 */
export type WeightFactorKey = 'a' | 'b' | 'func' | 'max' | 'min';

/** keep 保留线上配置；overwrite 保存时采用表单开关，开启时覆盖为平台默认公式。 */
export type WeightFactorResolution = 'keep' | 'overwrite';

export interface WeightFactorState {
  // 解析后的完整 JSON；解析失败保留原文，缺失为 undefined。
  // 未知字段参与比较，避免把自定义配置误判为平台默认公式。
  config?: unknown;
  // 开关以 metadata 为准，不从公式推断。
  enabled: boolean;
}

interface ImportedPolarisOptions {
  /** 表单开关；新建默认关闭，编辑连通测试可按需回填线上值。 */
  enabled: Ref<boolean | undefined>;
  /** 侧栏是否打开；关闭后的异步结果不得更新状态。 */
  active: () => boolean;
  /** 区分新建默认关闭与编辑时主动关闭线上开关。 */
  editMode: () => boolean;
  /** 回填线上开关；调用方需避免把回填计入用户修改。 */
  hydrate: (enabled: boolean) => void;
  /** 仅引入现有服务时走连通测试与远端更新流程。 */
  imported: () => boolean;
  /** 最新连接字段；任一字段变化都使测试结果与处理选择失效。 */
  target: () => GetImportedPolarisServiceRequest;
}

/** 关闭时只比对开关；开启时完整比对公式，忽略 JSON 字段顺序。 */
export function isWeightFactorAligned(enabled: boolean, remote: WeightFactorState) {
  return enabled === remote.enabled && (!enabled || isEqual(remote.config, PLATFORM_WEIGHT_FACTOR_CONFIG));
}

/** 读取 metadata 中的开关和公式（仅 "true" 视为开启）；缺失或损坏不补默认值，保留原文用于比较与提示。 */
export function readWeightFactor(metadata: ImportedPolarisServiceOutput['metadata']): WeightFactorState {
  return {
    enabled: metadata?.['internal-enable-dynamic-weight'] === 'true',
    config: parseWeightFactorConfig(metadata?.['internal-dynamic-weight-config']),
  };
}

/**
 * 管理侧栏会话的连通状态、线上快照与处理选择；弹窗确认只改表单和选择，侧栏保存时才按需写北极星。
 * 请求顺序、确认失效与失败后的状态保留在此处理；表单校验和弹窗展示由调用方负责。
 */
export function useImportedPolaris(options: ImportedPolarisOptions) {
  // idle 未测试；loading 测试中；success 已通过；error 测试或保存前复查失败。
  const status = ref<'error' | 'idle' | 'loading' | 'success'>('idle');
  // 最近一次查询或成功写入后的线上快照，也是用户确认的依据。
  const remote = ref<null | WeightFactorState>(null);
  // 连接字段、开关或测试结果变化后，已有选择必须重新确认。
  const resolution = ref<null | WeightFactorResolution>(null);
  // 连通失败，保存错误由侧栏统一展示。
  const connectionError = ref('');
  const saveError = ref('');
  // 覆盖复查、确认、写远端和平台保存全过程，防止重复提交。
  const submitting = ref(false);
  // 远端写入后置 true，平台保存成功或会话重置后清除，用于提示部分成功。
  const remoteSaved = ref(false);
  // 会话、连接字段或测试批次变化后使旧响应失效。
  let version = 0;

  const connected = computed(() => status.value === 'success');
  // 表单开关统一收敛为布尔值，避免各处兜底 undefined。
  const formEnabled = computed(() => !!options.enabled.value);
  // 按表单开关比对线上配置：关闭时只比开关，开启时同时比对默认公式；已处理的不再提示。
  const mismatch = computed(
    () =>
      options.imported() &&
      connected.value &&
      !!remote.value &&
      !isWeightFactorAligned(formEnabled.value, remote.value) &&
      !resolution.value,
  );
  // 编辑主动关闭线上开关时由修改确认放行，不强制先选择处理方式。
  const requiresResolution = computed(
    () => mismatch.value && !(options.editMode() && !!remote.value?.enabled && !formEnabled.value),
  );
  // 保留线上配置或已一致时不写远端；覆盖需比对开关及开启后的公式。
  const needsWrite = computed(() => {
    if (!options.imported() || !remote.value || resolution.value === 'keep') return false;
    // 已确认覆盖时比对完整配置；未选择时仅编辑模式的开关变更需要写入。
    return resolution.value === 'overwrite'
      ? !isWeightFactorAligned(formEnabled.value, remote.value)
      : options.editMode() && formEnabled.value !== remote.value.enabled;
  });

  /** 清理会话状态并废弃旧响应；提交锁由当前 save 的 finally 释放。 */
  function reset() {
    version += 1;
    status.value = 'idle';
    remote.value = null;
    resolution.value = null;
    connectionError.value = '';
    saveError.value = '';
    remoteSaved.value = false;
  }

  // 同步失效，避免字段变化后 Vue 下一次更新前仍可用旧结果提交。
  watch(
    () => {
      const target = options.target();
      return [
        options.active(),
        options.imported(),
        target.appID,
        target.polarisNamespace,
        target.polarisName,
        target.polarisToken,
      ];
    },
    reset,
    { flush: 'sync' },
  );
  // 处理选择对应当时的开关，再次切换后不能沿用。
  watch(
    options.enabled,
    () => {
      resolution.value = null;
    },
    { flush: 'sync' },
  );
  onScopeDispose(reset);

  function isCurrent(requestVersion: number) {
    return options.active() && requestVersion === version;
  }

  /** 连通测试与 metadata 读取共用；错误交由页面展示，不触发公共错误弹窗。 */
  async function readRemote(target: GetImportedPolarisServiceRequest) {
    // 此接口返回顶层 service，默认拦截器只取 data，需读取完整响应。
    const result = await PolarisConfigService.getImportedPolarisService(target, {
      interceptorErr: false,
      needRes: true,
    });
    if (!result?.service) throw new Error(window.i18n.t('未能读取北极星服务信息，请重试'));
    return readWeightFactor(result.service.metadata);
  }

  /** 一次查询同时给出连通结果与权重因子快照；用户已修改开关时可保留表单值。 */
  async function testConnection({ preserveEnabled = false }: { preserveEnabled?: boolean } = {}) {
    if (!options.active() || !options.imported() || status.value === 'loading' || submitting.value) return;
    const requestVersion = ++version;
    // 固定本次请求参数；表单后续变化由 version 判定响应是否有效。
    const target = { ...options.target() };
    status.value = 'loading';
    connectionError.value = '';
    // 重新测试不抹去“远端已修改、平台未保存”的提示。
    if (!remoteSaved.value) saveError.value = '';
    remote.value = null;
    resolution.value = null;
    try {
      const result = await readRemote(target);
      if (!isCurrent(requestVersion)) return;
      remote.value = result;
      // 未修改开关时回填线上状态；已修改则保留用户选择。
      if (options.editMode() && !preserveEnabled) options.hydrate(result.enabled);
      status.value = 'success';
    } catch {
      if (!isCurrent(requestVersion)) return;
      connectionError.value = connectionFailureMessage();
      status.value = 'error';
    }
  }

  /** 表单同步为弹窗所选列的开关并记录处理意图；两种选择都不立即写远端。 */
  function resolve(action: WeightFactorResolution) {
    if (!connected.value || !remote.value || submitting.value) return;
    if (action === 'keep') options.enabled.value = remote.value.enabled;
    // 开关变更会清除旧选择，因此在回填开关后记录本次确认。
    resolution.value = action;
  }

  /**
   * 保存顺序：检查连通和差异 → 按需复查线上快照并确认 → 写远端 → 保存平台配置；无需改远端时直接保存平台。
   * @param saveConfig 调用方已完成表单校验；提交引入配置时应省略 enableWeightFactor。
   * @param confirmChange 等待修改确认，取消返回 false；已确认覆盖时不重复确认。
   */
  async function save(saveConfig: () => Promise<void>, confirmChange: () => Promise<boolean>): Promise<SaveResult> {
    if (submitting.value) return 'busy';
    if (!options.active()) return 'cancelled';
    if (options.imported() && !connected.value) return 'connection-required';
    if (requiresResolution.value) return 'resolution-required';
    submitting.value = true;
    saveError.value = '';
    const requestVersion = version;
    // 固定本次请求参数；表单后续变化由 version 判定响应是否有效。
    const target = { ...options.target() };
    const enabled = formEnabled.value;
    // 复查阶段失败意味着旧连通结果不可靠，须重新测试；写入或平台保存失败不清空快照。
    let verifyingRemote = false;
    try {
      if (needsWrite.value) {
        verifyingRemote = true;
        const latest = await readRemote(target);
        if (!isCurrent(requestVersion)) return 'cancelled';
        // 确认依据已变化，撤销选择并要求重新确认。
        if (!isEqual(latest, remote.value)) {
          remote.value = latest;
          resolution.value = null;
          saveError.value = window.i18n.t('北极星线上配置已变化，请重新确认后保存');
          return 'remote-changed';
        }
        // 已确认覆盖无需再次确认；编辑关闭需弹窗确认。
        if (resolution.value !== 'overwrite' && !(await confirmChange())) return 'cancelled';
        if (!isCurrent(requestVersion)) return 'cancelled';
        verifyingRemote = false;
        await PolarisConfigService.updateImportedPolaris(
          { ...target, enableWeightFactor: enabled },
          { interceptorErr: false },
        );
        if (!isCurrent(requestVersion)) return 'cancelled';
        // 开启时写平台公式，关闭时移除公式；同步快照使重试跳过已完成的远端写入。
        remote.value = { enabled, config: enabled ? PLATFORM_WEIGHT_FACTOR_CONFIG : undefined };
        remoteSaved.value = true;
        // 关闭后采用新快照，保留处理选择以便平台保存失败时直接重试。
        resolution.value = enabled ? null : 'keep';
      }
      await saveConfig();
      if (!isCurrent(requestVersion)) return 'cancelled';
      remoteSaved.value = false;
      return 'saved';
    } catch (error) {
      if (!isCurrent(requestVersion)) return 'cancelled';
      if (verifyingRemote) {
        status.value = 'error';
        connectionError.value = connectionFailureMessage();
        resolution.value = null;
      }
      saveError.value = remoteSaved.value
        ? `${window.i18n.t('北极星权重因子已修改，但平台配置保存失败，请重试保存')}：${errorMessage(error)}`
        : errorMessage(error);
      return 'failed';
    } finally {
      submitting.value = false;
    }
  }

  return {
    connected,
    connectionError,
    mismatch,
    remote,
    remoteSaved,
    requiresResolution,
    resolution,
    resolve,
    save,
    saveError,
    status,
    submitting,
    testConnection,
  };
}

/** 关闭时显示占位；开启时展示配置中实际存在的参数，未知字段不补默认值。 */
export function weightFactorValue(state: WeightFactorState, key: WeightFactorKey) {
  if (!state.enabled) return '--';
  const { config } = state;
  if (!isRecord(config)) return '--';
  if (key === 'func') return typeof config.func === 'string' ? config.func : '--';
  const value = isRecord(config.params) ? config.params[key] : undefined;
  return typeof value === 'number' && Number.isFinite(value) ? String(value) : '--';
}

/** 连通失败的托底文案 */
function connectionFailureMessage() {
  return window.i18n.t('无法连通北极星，请检查服务名、环境类型和 Token 后重试');
}

function errorMessage(error: unknown) {
  if (isRecord(error) && isRecord(error.error) && typeof error.error.message === 'string') {
    return error.error.message;
  }
  if (error instanceof Error) return error.message;
  return window.i18n.t('请求异常');
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/** 未配置时返回 undefined；损坏的原文保留，只有明确覆盖才修正线上配置。 */
function parseWeightFactorConfig(raw: string | undefined): unknown {
  if (raw === undefined) return undefined;
  try {
    return JSON.parse(raw) as unknown;
  } catch {
    return raw;
  }
}
