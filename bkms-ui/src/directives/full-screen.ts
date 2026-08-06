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

// import { messageInfo } from '@/common/bkmagic';
// import { copyText } from '@/common/util';
import type { DirectiveBinding } from 'vue';

import useFullScreen from '~/composables/use-fullscreen';

interface IElement extends HTMLElement {
  [prop: string]: any;
}

export default {
  mounted(el: IElement, bind: DirectiveBinding) {
    const tools = bind.value?.tools || ['fullscreen'];
    if (!tools?.length) return;

    const { contentRef, switchFullScreen } = useFullScreen();
    contentRef.value = el;

    // el.handleExitFullScreen = (event: { code: string }) => {
    //   if (event.code === 'Escape' && el.fullscreen) {
    //     el.fullscreen.className = 'bkms-icon bkms-icon-close-circle-shape'; // 还原图标
    //     tools.forEach((tool: string) => {
    //       el[tool].style.position = 'absolute';
    //     });
    //     el.classList.remove('bkms-full-screen');
    //   }
    // };
    // el.addEventListener('mouseenter', () => {
    //   document.addEventListener('keyup', el.handleExitFullScreen);
    // });
    // el.addEventListener('mouseleave', () => {
    //   document.removeEventListener('keyup', el.handleExitFullScreen);
    // });

    el.defaultConfig = {
      fullscreen: {
        icon: 'bkms-icon bkms-icon-filliscreen-line',
        handler: (e: { target: IElement }) => {
          switchFullScreen();
          const { target } = e;
          if (target.className === 'bkms-icon bkms-icon-filliscreen-line') {
            target.className = 'bkms-icon bkms-icon-un-full-screen-2';
            // tools.forEach((tool: string) => {
            //   el[tool].style.position = 'fixed';
            // });
            // el.classList.add('bkms-full-screen');
          } else {
            target.className = 'bkms-icon bkms-icon-filliscreen-line';
            // tools.forEach((tool: string) => {
            //   el[tool].style.position = 'absolute';
            // });
            // el.classList.remove('bkms-full-screen');
          }
        },
      },
      copy: {
        icon: 'bcs-icon bcs-icon-copy',
        handler: () => {
          // copyText(bind.value?.content);
        },
      },
    };
    el.style.position = 'relative';

    const css = bind.value?.css || '';
    tools.forEach((tool: string, index: number) => {
      const icon = document.createElement('i');
      icon.className = el.defaultConfig[tool]?.icon;
      icon.style.cssText = `position: absolute;right: ${(index + 1) * 20}px;top: 15px;cursor: pointer;z-index: 200;margin-right: ${index * 10}px;color: #fff;${css}`;
      el[tool] = icon;
      icon.addEventListener('click', el.defaultConfig[tool]?.handler);
      el.append(icon);
    });
  },
  beforeUnmount(el: IElement, bind: DirectiveBinding) {
    const tools = bind.value?.tools || ['fullscreen', 'copy'];
    document.removeEventListener('keyup', el.handleExitFullScreen);
    tools.forEach((tool: string) => {
      el[tool]?.removeEventListener('click', el.defaultConfig[tool]?.handler);
      el[tool]?.remove();
    });
  },
};
