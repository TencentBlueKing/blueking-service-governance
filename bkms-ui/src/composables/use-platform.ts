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
import { useFrontendFeaturesStore } from '~/stores/frontend-features';
import { usePlatformConfigStore } from '~/stores/platform-config';

type PlatformConfigState = ReturnType<typeof usePlatformConfigStore>['$state'];

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
      // 版本号来自 index.html 注入的构建变量，未设置时为 '--'，兜底为空串
      version: window.BK_BKMS_APP_VERSION && window.BK_BKMS_APP_VERSION !== '--' ? window.BK_BKMS_APP_VERSION : '',
    };
    const data: PlatformConfigState = import.meta.env.BK_SHARED_RES_BASE_JS_URL
      ? await getPlatformConfig(import.meta.env.BK_SHARED_RES_BASE_JS_URL, defaults)
      : await getPlatformConfig(defaults);
    // footer 按部署模式固定，有意不用 bk-shared-res 下发的 footer。
    // 远程地址整站只有一份，无法同时表达两种文案：
    // 多租户用平台库默认「技术支持 | 社区论坛 | 产品官网」；
    // 非多租户用「联系 BK 助手 | 蓝鲸桌面」。
    // 这次 getPlatformConfig 只把 markdown 转成 HTML，不会再请求远程脚本。
    const footerSource = useFrontendFeaturesStore().enableMultiTenantMode
      ? await getPlatformConfig(defaults)
      : await getPlatformConfig({
          ...defaults,
          ...footerMarkdown([
            {
              text: '联系BK助手',
              textEn: 'Contact BK Assistant',
              href: 'https://bk.tencent.com/s-mart/community',
            },
            {
              text: '蓝鲸桌面',
              textEn: 'BlueKing Desktop',
              href: 'https://bk.tencent.com/s-mart/desktop',
            },
          ]),
        });
    data.footerInfo = footerSource.footerInfo;
    data.footerInfoEn = footerSource.footerInfoEn;
    data.footerInfoHTML = footerSource.footerInfoHTML;
    data.footerInfoHTMLEn = footerSource.footerInfoHTMLEn;
    data.i18n = {
      ...data.i18n,
      footerInfoHTML: footerSource.i18n.footerInfoHTML,
    };
    (Object.keys(platformConfig.$state) as (keyof PlatformConfigState)[]).forEach(key => {
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

function footerMarkdown(links: { href: string; text: string; textEn: string }[]) {
  const line = (label: 'text' | 'textEn') => links.map(link => `[${link[label]}](${link.href})`).join(' | ');
  return {
    footerInfo: line('text'),
    footerInfoEn: line('textEn'),
  };
}
