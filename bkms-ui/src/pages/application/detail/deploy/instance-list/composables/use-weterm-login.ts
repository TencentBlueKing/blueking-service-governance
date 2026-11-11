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

import { Message } from 'bkui-vue';
import { useI18n } from 'vue-i18n';
import { type AppInstanceOutputObj } from '~/@types/v1/instance';
import { InstanceService } from '~/api/modules/v1';
import { showApiErrorMessage } from '~/common/util';
import { useAppDetail } from '~/stores/app-detail';

/** 后端在部署未配置 WeTerm 环境 ID 时返回的错误码。 */
const NOT_IMPLEMENTED = 'NOT_IMPLEMENTED';

export interface UseWeTermLoginOptions {
  /** 跳转实现；默认写入 window.location.href，单测可注入以断言目标链接（jsdom 不支持自定义协议导航）。 */
  navigate?: (url: string) => void;
}

type WeTermErrorResponse = {
  error?: {
    code?: string;
    message?: string;
  };
};

/** WeTerm 登录：接口返回 weterm:// 链接，由系统唤起本地 WeTerm 客户端，不离开当前页面。 */
export function useWeTermLogin(options: UseWeTermLoginOptions = {}) {
  const { t } = useI18n();
  const appDetailStore = useAppDetail();
  const navigate = options.navigate ?? redirectToUrl;
  // 连点会重复申请链接并多次唤起客户端，登录期间忽略后续点击。
  let isLoggingIn = false;

  async function loginWithWeTerm(instance: AppInstanceOutputObj, envName: string) {
    if (isLoggingIn || !instance.id || !envName) return;

    isLoggingIn = true;
    try {
      const res = await InstanceService.createAppInstanceWeTerm(
        {
          appID: appDetailStore.appID,
          envName,
          instanceID: instance.id,
        },
        // NOT_IMPLEMENTED 走下方专用提示，关闭统一弹窗避免同一次失败提示两遍。
        { interceptorErr: false },
      );
      if (!res?.url) return;
      navigate(res.url);
      // 本地未安装客户端时浏览器对 weterm:// 无任何反馈，补一句操作提示。
      Message({
        message: t('正在唤起 WeTerm 客户端，如未唤起请先安装 WeTerm 客户端'),
      });
    } catch (error) {
      if (isNotImplementedError(error)) {
        Message({
          theme: 'warning',
          message: t('当前部署未配置 WeTerm 环境，暂不支持 WeTerm 登录'),
        });
        return;
      }
      // 关闭了统一拦截，这里复用同样的提示结构，保留后端错误详情与 traceId。
      showApiErrorMessage(error);
    } finally {
      isLoggingIn = false;
    }
  }

  return { loginWithWeTerm };
}

/** 命中 NOT_IMPLEMENTED 说明当前部署整体不支持 WeTerm。 */
function isNotImplementedError(error: unknown) {
  return (error as null | WeTermErrorResponse)?.error?.code === NOT_IMPLEMENTED;
}

function redirectToUrl(url: string) {
  window.location.href = url;
}
