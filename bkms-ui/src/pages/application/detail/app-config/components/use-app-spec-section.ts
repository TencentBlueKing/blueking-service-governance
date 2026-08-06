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

import { ref, watch } from 'vue';
import type { Ref } from 'vue';

import { Message } from 'bkui-vue';
import { useI18n } from 'vue-i18n';
import useLeaveConfirm from '~/composables/use-leave-confirm';
import { useAppDetail } from '~/stores/app-detail';

import type { ExtendedEnv } from './types';

/** AppSpec Section 配置项 */
/* eslint-disable @typescript-eslint/no-empty-object-type */
export interface AppSpecSectionOptions<
  TModel extends {},
  TOutput extends {},
  TEnvInput extends {},
  TFieldKey extends string,
> {
  /* eslint-enable @typescript-eslint/no-empty-object-type */
  allFieldKeys: TFieldKey[];
  fieldAccessors: Record<TFieldKey, FieldAccessor<TModel>>;
  /** API 无返回时的兜底默认值 */
  formDefaults: TModel;

  buildDefaultPayload: (model: TModel) => TModel;
  buildEnvPayload: (
    model: TModel,
    isFieldDiff: (key: TFieldKey) => boolean,
    isFieldReset: (key: TFieldKey) => boolean,
  ) => TEnvInput;
  deleteEnv: (appID: string, envName: string) => Promise<void>;
  fetchDefault: (appID: string) => Promise<null | TModel>;
  fetchEnvEffective: (appID: string, envName: string) => Promise<null | TOutput>;
  fetchEnvOverride: (appID: string, envName: string) => Promise<TOutput>;
  /** 填充环境数据到表单 */
  fillEnvFormData: (model: TModel, data: TOutput, defaults: null | TModel, formDefaults: TModel) => void;
  saveDefault: (appID: string, payload: TModel) => Promise<void>;
  saveEnv: (appID: string, envName: string, payload: TEnvInput) => Promise<void>;
}

/** 字段访问器：封装字段的读取、默认值获取、设置操作 */
export interface FieldAccessor<TModel, TValue = number | string> {
  get: (model: TModel) => TValue;
  getDefault: (defaults: TModel) => TValue;
  set: (model: TModel, value: TValue) => void;
}

/** AppSpec Section 通用 Composable：表单状态管理、默认值缓存、环境数据加载、字段差异判断、保存/删除操作 */
/* eslint-disable @typescript-eslint/no-empty-object-type */
export function createFieldAccessors<TModel extends {}, TFieldKey extends keyof TModel & string>(
  keys: TFieldKey[],
  formDefaults: TModel,
): Record<TFieldKey, FieldAccessor<TModel>> {
  const entries = keys.map(key => [
    key,
    {
      get: (m: TModel) => m[key],
      getDefault: (def: TModel) => def[key] ?? formDefaults[key],
      set: (m: TModel, v: unknown) => {
        (m as Record<string, unknown>)[key] = v;
      },
    },
  ]);
  return Object.fromEntries(entries) as unknown as Record<TFieldKey, FieldAccessor<TModel>>;
}

/**
 * 根据字段名列表和默认值自动生成 FIELD_ACCESSORS。
 * 适用于每个字段都是简单 `model[key]` 读写的场景。
 */
/* eslint-disable @typescript-eslint/no-empty-object-type */
export function createFillEnvFormData<TModel extends {}, TOutput extends {}, TFieldKey extends keyof TModel & string>(
  keys: TFieldKey[],
  formDefaults: TModel,
): (model: TModel, data: TOutput, defaults: null | TModel, fd: TModel) => void {
  return (model, data, defaults, _fd) => {
    const result: Partial<TModel> = {};
    for (const key of keys) {
      const value = (data as unknown as Record<string, unknown>)[key];
      const defaultValue =
        (defaults as unknown as null | Record<string, unknown>)?.[key] ??
        (formDefaults as unknown as Record<string, unknown>)[key];
      (result as Record<string, unknown>)[key] = value ?? defaultValue;
    }
    Object.assign(model, result);
  };
}

/**
 * 根据字段名列表自动生成 fillEnvFormData 函数。
 * 逻辑：遍历字段，收集非 null 的字段为 overriddenFields，并用 value ?? defaultValue 填充 model。
 */
/* eslint-disable @typescript-eslint/no-empty-object-type */
export function pickFields<TModel extends {}, TFieldKey extends keyof TModel & string>(
  model: TModel,
  keys: TFieldKey[],
): Pick<TModel, TFieldKey> {
  return Object.fromEntries(keys.map(key => [key, model[key]])) as Pick<TModel, TFieldKey>;
}

/**
 * 从对象中按字段名列表提取子集，可用于简化 buildDefaultPayload。
 */
/* eslint-disable @typescript-eslint/no-empty-object-type */
export function useAppSpecSection<
  TModel extends {},
  TOutput extends {},
  TEnvInput extends {},
  TFieldKey extends string,
>(
  /* eslint-enable @typescript-eslint/no-empty-object-type */
  options: AppSpecSectionOptions<TModel, TOutput, TEnvInput, TFieldKey>,
  emit: {
    (e: 'env-modified-change'): void;
    (e: 'loading-change', value: boolean): void;
  },
) {
  const { t } = useI18n();
  const appDetailStore = useAppDetail();

  const formRef = ref();
  const saving = ref(false);
  const resetting = ref(false);
  const loading = ref(true);
  const isEditing = ref(false);
  const formModel = ref<TModel>({ ...options.formDefaults }) as Ref<TModel>;
  const customFields = ref<TFieldKey[]>([]) as Ref<TFieldKey[]>;
  /** 本次编辑中被用户主动重置（点击 ResetIcon）的字段，保存时这些字段应传 null */
  const resetFields = ref<Set<TFieldKey>>(new Set()) as Ref<Set<TFieldKey>>;

  const { confirmBox, forceCleanDirtyTag } = useLeaveConfirm(formModel);

  let defaultCache: null | TModel = null;
  let formSnapshot: null | TModel = null;

  const getFieldValue = (field: TFieldKey) => options.fieldAccessors[field].get(formModel.value);

  const getFieldDefaultValue = (field: TFieldKey) =>
    defaultCache ? options.fieldAccessors[field].getDefault(defaultCache) : undefined;

  /** 判断字段当前值是否与默认值不同 */
  const isFieldDiffFromDefault = (field: TFieldKey) => {
    const defaultVal = getFieldDefaultValue(field);
    return defaultVal !== undefined && String(getFieldValue(field)) !== String(defaultVal);
  };

  /** 判断字段是否被环境覆盖过（只要设置过值，不论当前值是否与默认值相同） */
  const isFieldOverridden = (field: TFieldKey) => customFields.value.includes(field);

  /** 判断字段是否需要保存到环境覆盖（被覆盖过 或 当前值与默认值不同） */
  const isFieldNeedSave = (field: TFieldKey) => isFieldOverridden(field) || isFieldDiffFromDefault(field);

  /** 判断字段是否在本次编辑中被用户主动重置（应传 null 以清除覆盖） */
  const isFieldReset = (field: TFieldKey) => resetFields.value.has(field);
  const shouldShowResetIcon = (field: TFieldKey, currentEnv: ExtendedEnv | null) =>
    !currentEnv?.isDefault && isFieldOverridden(field);

  /** 重置指定字段为默认值 */
  const handleResetField = (field: TFieldKey) => {
    const index = customFields.value.indexOf(field);
    if (index > -1) customFields.value.splice(index, 1);
    resetFields.value.add(field);
    if (defaultCache) {
      options.fieldAccessors[field].set(formModel.value, options.fieldAccessors[field].getDefault(defaultCache));
    }
  };

  const clearFormValidate = () => {
    setTimeout(() => formRef.value?.clearValidate?.(), 0);
  };

  /** 获取并缓存应用级默认配置 */
  async function fetchDefault(): Promise<null | TModel> {
    if (defaultCache) return defaultCache;
    if (!appDetailStore.appID) return null;

    const result = await options.fetchDefault(appDetailStore.appID);
    if (result) defaultCache = result;
    return result ?? null;
  }

  /** 用数据填充表单，缺失字段使用默认值兜底 */
  const fillFormData = (data: TModel) => {
    const dataRecord = data as unknown as Record<string, unknown>;
    const defaultsRecord = options.formDefaults as unknown as Record<string, unknown>;
    const filled: Record<string, unknown> = {};
    for (const key of options.allFieldKeys) {
      filled[key] = dataRecord[key] ?? defaultsRecord[key];
    }
    Object.assign(formModel.value, filled);
  };

  async function fillWithDefault() {
    const defaultSpec = await fetchDefault();
    if (defaultSpec) {
      fillFormData(defaultSpec);
    } else {
      initFormData();
    }
  }

  const initFormData = () => {
    Object.assign(formModel.value, { ...options.formDefaults });
  };

  /** 根据环境加载配置：默认环境 / 未修改环境 / 已覆盖环境 */
  async function loadEnvData(env: ExtendedEnv) {
    const defaultSpec = await fetchDefault();

    // 默认环境
    if (env.isDefault) {
      if (defaultSpec) {
        fillFormData(defaultSpec);
        customFields.value = [...options.allFieldKeys];
      }
      return;
    }

    // 环境未覆盖，继承默认值
    if (!env.isModified) {
      await fillWithDefault();
      customFields.value = [];
      return;
    }

    // 环境有覆盖配置
    try {
      const envSpec = await options.fetchEnvEffective(appDetailStore.appID, env.name);

      if (!envSpec) {
        await fillWithDefault();
        customFields.value = [];
        return;
      }

      // 用 effective 数据填充表单
      options.fillEnvFormData(formModel.value, envSpec, defaultCache, options.formDefaults);

      // 用 envOverride（仅返回修改项）判断哪些字段被覆写
      try {
        const envOverride = await options.fetchEnvOverride(appDetailStore.appID, env.name);
        customFields.value = options.allFieldKeys.filter(
          key => (envOverride as unknown as Record<string, unknown>)[key] != null,
        );
      } catch {
        // GetEnv 返回 404 时，所有字段均为继承
        customFields.value = [];
      }
    } catch {
      await fillWithDefault();
      customFields.value = [];
    }
  }

  /** 取消编辑，恢复快照 */
  const handleCancelEdit = () => {
    if (formSnapshot) Object.assign(formModel.value, formSnapshot);
    isEditing.value = false;
    resetFields.value.clear();
    formRef.value?.clearValidate?.();
  };

  /** 进入编辑态，保存快照 */
  const handleEdit = () => {
    formSnapshot = { ...formModel.value };
    resetFields.value.clear();
    isEditing.value = true;
  };

  /** 环境切换：确认离开 → 退出编辑 → 加载数据 → 清除脏标记，返回是否切换成功 */
  async function handleEnvChange(env: ExtendedEnv) {
    if (!(await confirmBox())) return false;

    isEditing.value = false;
    resetFields.value.clear();
    loading.value = true;

    try {
      await loadEnvData(env);
      forceCleanDirtyTag();
    } catch {
      initFormData();
    } finally {
      loading.value = false;
      clearFormValidate();
    }

    return true;
  }

  /** 删除环境覆盖，恢复为默认配置 */
  async function handleResetToDefault(currentEnv: ExtendedEnv | null) {
    if (!currentEnv || currentEnv.isDefault) return;

    try {
      resetting.value = true;
      await options.deleteEnv(appDetailStore.appID, currentEnv.name);

      await fillWithDefault();
      customFields.value = [];

      Message({ theme: 'success', message: t('操作成功') });
      forceCleanDirtyTag();
      isEditing.value = false;
      resetFields.value.clear();
      emit('env-modified-change');
    } finally {
      resetting.value = false;
    }
  }

  /** 保存后重新获取 envOverride，刷新 customFields（已修改标识） */
  async function refreshCustomFields(envName: string) {
    try {
      const envOverride = await options.fetchEnvOverride(appDetailStore.appID, envName);
      customFields.value = options.allFieldKeys.filter(
        key => (envOverride as unknown as Record<string, unknown>)[key] != null,
      );
    } catch {
      customFields.value = [];
    }
  }

  /** 保存配置：默认环境调用 SetDefault，普通环境调用 SetEnv */
  async function handleSave(currentEnv: ExtendedEnv | null): Promise<boolean> {
    try {
      const valid = await formRef.value?.validate?.().catch(() => false);
      if (!valid || !currentEnv) return false;

      saving.value = true;

      const success = currentEnv.isDefault
        ? await options
            .saveDefault(appDetailStore.appID, options.buildDefaultPayload(formModel.value))
            .then(() => {
              defaultCache = options.buildDefaultPayload(formModel.value);
              return true;
            })
            .catch(() => false)
        : await options
            .saveEnv(
              appDetailStore.appID,
              currentEnv.name,
              options.buildEnvPayload(formModel.value, isFieldNeedSave, isFieldReset),
            )
            .then(() => true)
            .catch(() => false);

      if (!success) return false;

      if (!currentEnv.isDefault) {
        // 保存成功后重新获取 envOverride，刷新已修改标识
        await refreshCustomFields(currentEnv.name);
        emit('env-modified-change');
      }

      Message({ theme: 'success', message: t('操作成功') });
      forceCleanDirtyTag();
      isEditing.value = false;
      resetFields.value.clear();
      return true;
    } finally {
      saving.value = false;
    }
  }

  // appID 变化时重新获取默认配置
  watch(
    () => appDetailStore.appID,
    newVal => {
      if (newVal && ['trpc', 'taf'].includes(appDetailStore.appType)) {
        defaultCache = null;
        fetchDefault();
      }
    },
    { immediate: true },
  );

  watch(loading, val => emit('loading-change', val), { immediate: true });

  return {
    formRef,
    saving,
    resetting,
    loading,
    isEditing,
    formModel,
    customFields,
    isFieldDiffFromDefault,
    isFieldOverridden,
    shouldShowResetIcon,
    getFieldDefaultValue,
    handleResetField,
    handleCancelEdit,
    handleEdit,
    handleEnvChange,
    handleResetToDefault,
    handleSave,
  };
}
