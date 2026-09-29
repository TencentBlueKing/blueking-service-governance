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

import { computed, ref, toValue } from 'vue';
import type { InjectionKey, MaybeRefOrGetter, Ref } from 'vue';

import { AppConfigFileDefsService } from '~/api/modules/v1';

import type { DefDetailObj, EnvInstanceObj, UpdateDefInput } from '~/@types/v1/app-config-file-defs';

export type ConfigFileContentSourceType = 'bscp' | 'local';
/** 配置文件定义视图模型，字段均已归一化，保证类型与默认值稳定 */
export interface ConfigFileDefView {
  /** overlay 文件所引用的基础文件 ID */
  baseAppConfigFileId: string;
  /** 配置种类：framework 框架自带 / plain 用户自定义 */
  configKind: ConfigFileKind;
  content: string;
  /** 内容来源：bscp 平台配置 / local 本地维护 */
  contentSourceType: ConfigFileContentSourceType;
  currentVersion: number;
  /** 当前可编辑的内容字段 */
  editableContentField: EditableContentField;
  /** 是否开启环境变量渲染 */
  enableEnvVarRender: boolean;
  fileFormat: ConfigFileFormat;
  fileId: string;
  /** 文件类型：normal 普通文件 / overlay 覆盖文件 */
  fileType: ConfigFileType;
  /** 指定环境下是否存在独立实例（仅按环境查询时有意义） */
  hasEnvInstance: boolean;
  id: string;
  /** 是否为统一配置 */
  isUnifiedConfig: boolean;
  mountDir: string;
  mountedEnvNames: string[];
  /** 挂载范围：all 全部环境 / envs 指定环境 */
  mountScope: ConfigFileMountScope;
  name: string;
  /** overlay 文件的覆盖内容 */
  overlayContent: string;
}
export type ConfigFileFormat = 'taf' | 'yaml';
export type ConfigFileKind = 'framework' | 'plain';
/** 列表展示项：在视图模型上补充展示用名称与路径 */
export interface ConfigFileListItem extends ConfigFileDefView {
  displayName: string;
  displayPath: string;
}
export type ConfigFileMountScope = 'all' | 'envs';

/** 文件类型：normal 普通 / overlay 覆盖 */
export type ConfigFileType = 'normal' | 'overlay';

/** 可编辑的内容字段：content 默认文件 / overlayContent 覆盖内容 / none 不可编辑 */
export type EditableContentField = 'content' | 'none' | 'overlayContent';

/** 创建 plain 文件的入参 */
export interface PlainFileCreateInput {
  content?: string;
  enableEnvVarRender: boolean;
  mountDir: string;
  mountedEnvNames?: string[];
  name: string;
}

/** 挂载预览侧滑栏显隐状态的注入 key，供子孙组件共享 */
export const MOUNT_PREVIEW_VISIBLE_KEY: InjectionKey<Ref<boolean>> = Symbol('mountPreviewVisible');

/**
 * 管理应用下配置文件定义的列表 / 详情 / 环境实例状态，并提供增删改查方法。
 * @param appID 应用 ID，支持 ref / getter
 */
export function useConfigFileDefs(appID: MaybeRefOrGetter<string>) {
  /** 当前选中的定义 ID */
  const activeDefID = ref('');
  const defs = ref<ConfigFileDefView[]>([]);
  const detail = ref<ConfigFileDefView | null>(null);
  const envInstances = ref<EnvInstanceObj[]>([]);
  const listLoading = ref(false);
  const detailLoading = ref(false);
  const defsGuard = createLatestRequestGuard();
  const detailGuard = createLatestRequestGuard();
  const envInstancesGuard = createLatestRequestGuard();

  const activeDef = computed(() => defs.value.find(item => item.id === activeDefID.value) || null);
  /** 已存在独立实例的环境名列表 */
  const modifiedEnvNames = computed(() => envInstances.value.map(item => item.envName || '').filter(Boolean));

  /** 拉取定义列表：优先选中 preferredID，否则回退到 framework 或首项 */
  async function fetchDefs(preferredID = activeDefID.value) {
    const id = toValue(appID);
    if (!id) {
      reset();
      return;
    }
    const requestID = defsGuard.begin();
    listLoading.value = true;
    try {
      const result = await AppConfigFileDefsService.listDefaultFilesWithDef({ appID: id }, { needRes: true });
      if (!defsGuard.isLatest(requestID) || id !== toValue(appID)) return;
      defs.value = (result.items || []).map(normalizeDef).filter(item => item.id);
      const preferred = defs.value.find(item => item.id === preferredID);
      const fallback = defs.value.find(item => item.configKind === 'framework') || defs.value[0];
      const nextActiveDefID = preferred?.id || fallback?.id || '';
      if (nextActiveDefID !== activeDefID.value) {
        detail.value = null;
        envInstances.value = [];
      }
      activeDefID.value = nextActiveDefID;
    } finally {
      if (defsGuard.isLatest(requestID)) listLoading.value = false;
    }
  }

  /** 拉取当前定义详情，envName 为空即默认文件 */
  async function fetchDetail(envName = '') {
    const id = toValue(appID);
    if (!id || !activeDefID.value) {
      detail.value = null;
      return null;
    }
    const requestedDefID = activeDefID.value;
    const requestID = detailGuard.begin();
    detail.value = null;
    detailLoading.value = true;
    try {
      const result = await AppConfigFileDefsService.getAppConfigFileDefDetail(
        { appID: id, id: requestedDefID, envName },
        { needRes: true },
      );
      if (!detailGuard.isLatest(requestID) || id !== toValue(appID) || requestedDefID !== activeDefID.value) {
        return null;
      }
      detail.value = normalizeDef(result.item);
      return detail.value;
    } finally {
      if (detailGuard.isLatest(requestID)) detailLoading.value = false;
    }
  }

  /** 拉取当前定义的环境实例列表 */
  async function fetchEnvInstances() {
    const id = toValue(appID);
    if (!id || !activeDefID.value) {
      envInstances.value = [];
      return;
    }
    const requestedDefID = activeDefID.value;
    const requestID = envInstancesGuard.begin();
    const result = await AppConfigFileDefsService.listAppConfigFileDefEnvInstances(
      { appID: id, id: requestedDefID },
      { needRes: true },
    );
    if (!envInstancesGuard.isLatest(requestID) || id !== toValue(appID) || requestedDefID !== activeDefID.value) {
      return;
    }
    envInstances.value = result.items || [];
  }

  /** 并行刷新详情与环境实例 */
  async function refreshActive(envName = '') {
    await Promise.all([fetchDetail(envName), fetchEnvInstances()]);
  }

  /**
   * 创建 plain 文件。
   * 创建接口不接收渲染开关与挂载环境，故创建成功后再补一次更新。
   */
  async function createPlain(input: PlainFileCreateInput) {
    const id = toValue(appID);
    const result = await AppConfigFileDefsService.createAppConfigFileDef(
      {
        appID: id,
        configKind: 'plain',
        content: input.content || '',
        contentSourceType: 'local',
        fileFormat: 'yaml',
        fileType: 'normal',
        mountDir: input.mountDir,
        name: input.name,
      },
      { needRes: true },
    );
    const createdID = result.item?.id || '';
    try {
      if (createdID && (input.enableEnvVarRender || input.mountedEnvNames !== undefined)) {
        await AppConfigFileDefsService.updateAppConfigFileDef({
          appID: id,
          id: createdID,
          ...(input.enableEnvVarRender ? { enableEnvVarRender: true } : {}),
          ...(input.mountedEnvNames !== undefined ? { mountedEnvNames: input.mountedEnvNames } : {}),
        });
      }
    } finally {
      // 无论补更新成功与否都刷新列表，保证新建项出现在列表中
      await fetchDefs(createdID);
    }
    return createdID;
  }

  /** 更新当前定义的基础信息（名称、挂载目录、环境变量渲染等） */
  async function updateDef(changes: UpdateDefInput) {
    await AppConfigFileDefsService.updateAppConfigFileDef({
      appID: toValue(appID),
      id: activeDefID.value,
      ...changes,
    });
  }

  /** 保存内容；envName 为空保存默认文件，关闭错误拦截由调用方处理版本冲突 */
  async function saveContent(envName: string, content: string, description = '') {
    return await AppConfigFileDefsService.updateAppConfigFileDefContent(
      {
        appID: toValue(appID),
        id: activeDefID.value,
        content,
        currentVersion: detail.value?.currentVersion,
        description,
      },
      {
        interceptorErr: false,
        needRes: true,
        queryParams: envName ? { envName } : undefined,
      },
    );
  }

  /** 将指定环境的配置恢复为默认值 */
  async function resetEnv(envName: string) {
    await AppConfigFileDefsService.resetAppConfigFileDefEnvToDefault({
      appID: toValue(appID),
      id: activeDefID.value,
      envName,
    });
  }

  /** 删除定义并刷新列表；删除的是当前选中项时清空选中 */
  async function deleteDef(id: string) {
    await AppConfigFileDefsService.deleteAppConfigFileDef({ appID: toValue(appID), id });
    await fetchDefs(activeDefID.value === id ? '' : activeDefID.value);
  }

  /** 重置全部状态，并作废进行中的请求 */
  function reset() {
    defsGuard.invalidate();
    detailGuard.invalidate();
    envInstancesGuard.invalidate();
    activeDefID.value = '';
    defs.value = [];
    detail.value = null;
    envInstances.value = [];
    listLoading.value = false;
    detailLoading.value = false;
  }

  return {
    activeDef,
    activeDefID,
    createPlain,
    defs,
    deleteDef,
    detail,
    detailLoading,
    fetchDefs,
    listLoading,
    modifiedEnvNames,
    refreshActive,
    reset,
    resetEnv,
    saveContent,
    updateDef,
  };
}

/** 生成「最新请求」守卫，用于丢弃过期响应，避免并发请求导致状态错乱 */
function createLatestRequestGuard() {
  let latestID = 0;
  return {
    /** 开始一次新请求并返回其序号 */
    begin: () => ++latestID,
    /** 该请求是否仍是最新的一次 */
    isLatest: (requestID: number) => requestID === latestID,
    /** 作废所有进行中的请求 */
    invalidate: () => {
      latestID += 1;
    },
  };
}

/** 将接口返回的原始对象归一化为视图模型，补齐默认值并收敛联合类型 */
function normalizeDef(source?: DefDetailObj): ConfigFileDefView {
  const field = source?.editableContentField;
  const mountedEnvNames = source?.mountedEnvNames;
  return {
    baseAppConfigFileId: source?.baseAppConfigFileId || '',
    configKind: source?.configKind === 'plain' ? 'plain' : 'framework',
    content: source?.content || '',
    contentSourceType: source?.contentSourceType === 'bscp' ? 'bscp' : 'local',
    currentVersion: Number(source?.currentVersion || 0),
    editableContentField: field === 'content' || field === 'overlayContent' ? field : 'none',
    // framework 文件默认开启环境变量渲染
    enableEnvVarRender: source?.enableEnvVarRender ?? source?.configKind === 'framework',
    fileFormat: source?.fileFormat === 'taf' ? 'taf' : 'yaml',
    fileId: source?.fileId || '',
    fileType: source?.fileType === 'overlay' ? 'overlay' : 'normal',
    hasEnvInstance: source?.hasEnvInstance ?? false,
    id: source?.id || '',
    isUnifiedConfig: source?.isUnifiedConfig ?? true,
    mountDir: source?.mountDir || '',
    // 未下发字段（及空数组）视为挂载全部环境
    mountScope: mountedEnvNames?.length ? 'envs' : 'all',
    mountedEnvNames: mountedEnvNames?.filter(Boolean) || [],
    name: source?.name || '',
    overlayContent: source?.overlayContent || '',
  };
}
