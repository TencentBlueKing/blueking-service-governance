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

import { ref } from 'vue';

import { Message } from 'bkui-vue';
import { useI18n } from 'vue-i18n';
import { type AppInstanceOutputObj } from '~/@types/v1/instance';
import { InstanceService } from '~/api/modules/v1';
import { useAppDetail } from '~/stores/app-detail';

/**
 * 后端在部署未配置 WeTerm 环境 ID 时返回 NOT_IMPLEMENTED；
 * 该状态是部署级的、与实例无关，命中后所有实例表格都应隐藏入口，故用模块级状态共享。
 */
export const isWeTermUnsupported = ref(false);

type WeTermErrorResponse = {
  error?: {
    code?: string;
    message?: string;
  };
};

/** WeTerm 登录：接口返回 weterm:// 链接，由系统唤起本地 WeTerm 客户端，不离开当前页面。 */
export function useWeTermLogin() {
  const { t } = useI18n();
  const appDetailStore = useAppDetail();

  async function loginWithWeTerm(instance: AppInstanceOutputObj, envName: string) {
    if (!instance.id || !envName) return;

    try {
      const res = await InstanceService.createAppInstanceWeTerm(
        {
          appID: appDetailStore.appID,
          envName,
          instanceID: instance.id,
        },
        // NOT_IMPLEMENTED 需要静默隐藏入口，不走统一的错误弹窗。
        { interceptorErr: false },
      );
      if (!res?.url) return;
      window.location.href = res.url;
    } catch (error) {
      if (isNotImplementedError(error)) {
        isWeTermUnsupported.value = true;
        Message({
          theme: 'warning',
          message: t('当前部署未配置 WeTerm 环境，暂不支持 WeTerm 登录'),
        });
        return;
      }
      Message({
        theme: 'error',
        message: (error as null | WeTermErrorResponse)?.error?.message || t('请求异常'),
      });
    }
  }

  return { loginWithWeTerm };
}

/** 命中 NOT_IMPLEMENTED 说明当前部署整体不支持 WeTerm。 */
function isNotImplementedError(error: unknown) {
  return (error as null | WeTermErrorResponse)?.error?.code === 'NOT_IMPLEMENTED';
}
