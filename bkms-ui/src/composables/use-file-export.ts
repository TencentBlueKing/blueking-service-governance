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

/** 文件导出 Composable，支持防重复点击和自动重置导出状态 */

import { ref } from 'vue';

import { downloadResponseFile } from '~/common/util';

/** 导出文件的配置选项 */
interface ExportFileOptions {
  /** 默认文件名（当响应头未指定时使用） */
  fallbackFilename: string;
  /** 发起文件下载请求的函数 */
  request: () => Promise<Response>;
}

/** 文件导出 Hook，返回导出方法和进行中状态 */
export function useFileExport() {
  const isExporting = ref(false);

  /**
   * 执行文件导出
   * @param options - 导出配置
   * @returns 是否导出成功
   */
  async function exportFile(options: ExportFileOptions) {
    if (isExporting.value) return false;
    isExporting.value = true;
    try {
      await downloadResponseFile(await options.request(), options.fallbackFilename);
      return true;
    } catch {
      return false;
    } finally {
      isExporting.value = false;
    }
  }

  return { exportFile, isExporting };
}
