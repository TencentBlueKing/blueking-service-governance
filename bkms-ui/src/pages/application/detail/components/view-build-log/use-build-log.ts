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

import { onBeforeUnmount, ref, shallowRef } from 'vue';

import { Message } from 'bkui-vue';
import { useI18n } from 'vue-i18n';
import { BuildsService } from '~/api/modules/v1';

import type { BuildLogError, BuildLogMessage, BuildLogRequest } from './type';
import type { LogEntryOutputObj } from '~/@types/v1/instance';

interface BuildLogDownloadError {
  error?: {
    message?: string;
  };
}

/** 构建日志已过期或已被清理时，后端返回的业务细分码。 */
const BUILD_LOG_UNAVAILABLE = 'BUILD_LOG_UNAVAILABLE';

export function useBuildLog(getRequest: () => BuildLogRequest) {
  const { t } = useI18n();
  const loading = ref(false);
  const logs = ref<LogEntryOutputObj[]>([]);
  const streamControllerRef = shallowRef<AbortController>();

  /** 中止当前日志流，并结束加载状态。 */
  function closeStream() {
    streamControllerRef.value?.abort();
    streamControllerRef.value = undefined;
    loading.value = false;
  }

  /** 清空当前日志并关闭已有连接。 */
  function clearLogs() {
    closeStream();
    logs.value = [];
  }

  /** 将日志面板恢复到初始状态。 */
  function resetLogs() {
    clearLogs();
  }

  /** 将后端日志行转换为 InstanceLog 使用的数据格式。 */
  function appendLogs(data: BuildLogMessage) {
    // 实际接口使用 PascalCase，camelCase 作为接口文档格式的兼容分支。
    const buildLogs = data.Logs ?? data.logs ?? [];
    logs.value.push(
      ...buildLogs.map(item => {
        const timestamp = item.Timestamp ?? item.timestamp;
        return {
          content: item.Message ?? item.message ?? '',
          timestamp: timestamp ? new Date(timestamp).toLocaleString() : '',
        };
      }),
    );
  }

  /** 处理一个完整的 SSE 事件。 */
  function handleSseEvent(eventBlock: string) {
    let eventName = 'message';
    const dataLines: string[] = [];

    for (const line of eventBlock.split(/\r?\n/)) {
      if (line.startsWith('event:')) {
        eventName = line.slice('event:'.length).trim();
      } else if (line.startsWith('data:')) {
        dataLines.push(line.slice('data:'.length).trimStart());
      }
    }

    const eventData = dataLines.join('\n');
    if (eventName === 'done') return;

    if (eventName === 'error') {
      const data = JSON.parse(eventData) as BuildLogError;
      const errorDetails = typeof data.error === 'object' ? (data.error.details ?? []) : [];
      const isBuildLogUnavailable = errorDetails.some(detail => detail.code === BUILD_LOG_UNAVAILABLE);

      if (isBuildLogUnavailable) {
        Message({
          theme: 'error',
          message: t('构建日志已过期或已清理'),
        });
        return;
      }

      const errorMessage = typeof data.error === 'string' ? data.error : data.error?.message;
      logs.value.push({
        content: errorMessage || t('构建日志连接异常，请刷新重试'),
        timestamp: new Date().toLocaleString(),
      });
      return;
    }

    if (eventName === 'message' && eventData) {
      const data = JSON.parse(eventData) as BuildLogMessage;
      if (Array.isArray(data.Logs) || Array.isArray(data.logs)) appendLogs(data);
    }
  }

  /** 调用 BuildsService.streamBuildLogs，并持续解析返回的 SSE 日志流。 */
  async function fetchLogs() {
    clearLogs();
    const request = getRequest();
    const { appID, buildID } = request;
    if (!appID || !buildID) return;

    const controller = new AbortController();
    streamControllerRef.value = controller;
    loading.value = true;
    let reader: ReadableStreamDefaultReader<Uint8Array> | undefined;
    let streamCompleted = false;
    try {
      const response = await BuildsService.streamBuildLogs<BuildLogRequest, Response>(request, {
        interceptorErr: false,
        originalResponse: true,
        signal: controller.signal,
      });
      if (!response.body) throw new Error(t('构建日志响应为空'));

      loading.value = false;
      reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';

      while (!controller.signal.aborted) {
        const { done, value } = await reader.read();
        buffer += decoder.decode(value, { stream: !done });

        let boundaryIndex = buffer.search(/\r?\n\r?\n/);
        while (boundaryIndex >= 0) {
          const eventBlock = buffer.slice(0, boundaryIndex);
          const separator = buffer.slice(boundaryIndex).match(/^\r?\n\r?\n/)?.[0] || '';
          buffer = buffer.slice(boundaryIndex + separator.length);
          handleSseEvent(eventBlock);
          boundaryIndex = buffer.search(/\r?\n\r?\n/);
        }

        if (done) {
          streamCompleted = true;
          if (buffer.trim()) handleSseEvent(buffer);
          return;
        }
      }
    } catch (error) {
      if (!(error instanceof DOMException && error.name === 'AbortError')) {
        console.warn('拉取构建日志失败', error);
        logs.value.push({
          content: error instanceof Error ? error.message : t('构建日志连接异常，请刷新重试'),
          timestamp: new Date().toLocaleString(),
        });
      }
    } finally {
      if (reader) {
        // 解析异常等提前退出场景需要主动取消；正常 EOF 或 AbortController 中止无需重复取消。
        if (!streamCompleted && !controller.signal.aborted) {
          await reader.cancel().catch(() => undefined);
        }
        reader.releaseLock();
      }
      if (streamControllerRef.value === controller) {
        streamControllerRef.value = undefined;
        loading.value = false;
      }
    }
  }

  /** 通过 API 获取日志，失败时展示错误，成功后下载 Blob。 */
  async function downloadLogs() {
    const request = getRequest();
    const { appID, buildID } = request;
    if (!appID || !buildID) return;

    try {
      const response = await BuildsService.downloadBuildLogs<BuildLogRequest, Response>(request, {
        interceptorErr: false,
        originalResponse: true,
      });
      if (!response.ok) throw new Error(t('下载构建日志失败'));

      const blob = await response.blob();
      const objectURL = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = objectURL;
      link.download = `build-log_${appID}_${buildID}.log`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(objectURL);
    } catch (error) {
      const apiError = error as BuildLogDownloadError;
      Message({
        theme: 'error',
        message: apiError.error?.message || (error instanceof Error ? error.message : t('下载构建日志失败')),
      });
    }
  }

  onBeforeUnmount(closeStream);

  return {
    downloadLogs,
    fetchLogs,
    loading,
    logs,
    resetLogs,
  };
}
