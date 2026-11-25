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

import { onMounted, ref, watch } from 'vue';

import { Message } from 'bkui-vue';
import { useI18n } from 'vue-i18n';
import { ApiServerService } from '~/api/modules/bkmsserver';
import useEnvManager from '~/composables/use-env-manager';

import type { ResizeLayout } from 'bkui-vue';
import type { IMonacoEditorErrorMarkerItem } from '~/common/util';

/**
 * trpc/app-config.vue 与 taf/app-config.vue 的公共逻辑。
 *
 * 提取了编辑器错误状态、环境变量侧栏、模板选择器状态、
 * YAML 卡片展开/收起等两组件完全相同的逻辑。
 */
export function useAppConfigCommon() {
  const { t } = useI18n();
  const { envList, handleGetEnvList } = useEnvManager();

  // yaml 异常
  const editorErr = ref<{
    message: string[];
    type: string;
  }>({
    type: '',
    message: [],
  });
  const errorRef = ref<InstanceType<typeof ResizeLayout> | null>(null);

  // 模板选择器状态
  const showTemplateSelect = ref(false);
  const templateLoading = ref(true);

  // 环境变量侧栏 & YAML 卡片状态
  const showEnvVar = ref(false);
  const isYamlCardExpanded = ref(true);
  const resizeLayoutRef = ref<InstanceType<typeof ResizeLayout> | null>(null);

  function hideOrShowError() {
    if (!editorErr.value?.message?.length && errorRef.value) {
      errorRef.value.asideRef.hidden = true;
    } else if (editorErr.value?.message?.length && errorRef.value) {
      errorRef.value.asideRef.hidden = false;
    }
  }

  function resetStatus() {
    editorErr.value = {
      type: '',
      message: [],
    };
    hideOrShowError();
  }

  function handleEditorErr(err: IMonacoEditorErrorMarkerItem[]) {
    editorErr.value.type = 'content';
    editorErr.value.message = err.map(item => item.message);
    hideOrShowError();
  }

  function handleYamlCardChange(expanded: boolean) {
    isYamlCardExpanded.value = expanded;
  }

  function setEnvVarAsideVisible(visible: boolean) {
    if (resizeLayoutRef.value?.asideRef) {
      resizeLayoutRef.value.asideRef.hidden = !visible;
    }
  }

  function toggleEnvVarShow() {
    showEnvVar.value = !showEnvVar.value;
    setEnvVarAsideVisible(showEnvVar.value);
  }

  function handleGetVarEnv(env: string) {
    const envID = envList.value.find(item => item.name === env)?.id;
    if (!envID) return Promise.resolve([]);
    return ApiServerService.ListEnvAvailableEnvVars({ envID });
  }

  function handleTemplateAvailability(available: boolean) {
    showTemplateSelect.value = available;
  }

  function handleTemplateLoading(loading: boolean) {
    templateLoading.value = loading;
  }

  /**
   * 模板切换/初始化未完成时阻止提交并提示。
   * 该窗口期内编辑器内容仍是上一个模板的，此时提交会把旧模板内容带入创建请求。
   */
  function ensureTemplateReady() {
    if (!templateLoading.value) return true;
    Message({ theme: 'warning', message: t('模板配置加载中，请稍后') });
    return false;
  }

  // 初始化隐藏错误侧栏
  watch(
    [editorErr.value, errorRef],
    () => {
      hideOrShowError();
    },
    { immediate: true },
  );

  onMounted(async () => {
    await handleGetEnvList();
    setEnvVarAsideVisible(showEnvVar.value);
  });

  return {
    editorErr,
    ensureTemplateReady,
    errorRef,
    envList,
    handleEditorErr,
    handleGetVarEnv,
    handleTemplateAvailability,
    handleTemplateLoading,
    handleYamlCardChange,
    isYamlCardExpanded,
    resetStatus,
    resizeLayoutRef,
    showEnvVar,
    showTemplateSelect,
    templateLoading,
    toggleEnvVarShow,
  };
}
