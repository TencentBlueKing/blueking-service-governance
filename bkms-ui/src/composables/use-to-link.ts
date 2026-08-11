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

export default function useToLink() {
  /**
   * 统一跳转辅助（跨域跳转统一开新标签页；同域跳转请直接使用 vue-router）
   * @param type 跳转目标类型，支持 'devops' | 'bcs' | 'monitor' | 'monitor-alert'
   * @param bKProject 业务项目 ID（monitor 类型下为 bkMonitorProjectID）
   * @param alertID 仅 'monitor-alert' 必填，告警事件 ID
   */
  function handleToLink(type: string, bKProject = '', alertID = '') {
    let url = '';
    if (!bKProject) return;
    switch (type) {
      case 'devops':
        url = `${import.meta.env.BK_DEVOPS}/console/pipeline/${bKProject}`;
        break;
      case 'bcs':
        url = `${import.meta.env.BK_BCS}/bcs/projects/${bKProject}`;
        break;
      case 'monitor':
        url = `${import.meta.env.BK_MONITOR}/?bizId=${bKProject}#/apm/home`;
        break;
      case 'monitor-alert':
        if (!alertID) return;
        url = `${import.meta.env.BK_MONITOR}/?bizId=${bKProject}/#/trace/alarm-center/detail/${alertID}`;
        break;
    }
    window.open(url, '_blank');
  }

  return {
    handleToLink,
  };
}
