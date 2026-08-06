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

import { getPlatformConfig, setDocumentTitle, setShortcutIcon } from '@blueking/platform-config';
import { useI18n } from 'vue-i18n';
import { usePlatformConfigStore } from '~/stores/platform-config';
export default function usePlatform() {
  const platformConfig = usePlatformConfigStore();
  async function getPlatformInfo() {
    const { t } = useI18n();
    const defaults = {
      name: '服务治理',
      nameEn: 'BKMS-GOVERN',
      appLogo: '',
      brandName: '蓝鲸智云',
      brandNameEn: 'Tencent BlueKing',
      productName: '蓝鲸服务治理',
      productNameEn: 'BKMS-GOVERN',
      favicon: '/favicon.svg',
      helperLink: 'wxwork://message?uin=8444252571319680',
      helperText: t('技术支持'),
      footerInfoHTML: '',
      version: '',
      i18n: {
        footerInfoHTML: '',
      },
    };
    let data: { [key: string]: any } = {};
    if (import.meta.env.BK_SHARED_RES_BASE_JS_URL) {
      data = await getPlatformConfig(import.meta.env.BK_SHARED_RES_BASE_JS_URL, defaults);
    } else {
      data = await getPlatformConfig(defaults);
    }
    Object.keys(platformConfig.$state).forEach(key => {
      platformConfig.$patch({
        [key]: data[key],
      });
    });
    return data;
  }
  return {
    platformConfig,
    getPlatformInfo,
    setDocumentTitle,
    setShortcutIcon,
  };
}
