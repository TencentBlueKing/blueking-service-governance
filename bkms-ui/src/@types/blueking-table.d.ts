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

// @blueking/table 的类型入口 typings/vue3.d.ts 未导出 ISettings（列设置面板的 settings / change 载荷），
// 且 package.json 的 exports 未开放 ./typings/* 子路径，无法从包内引用。
// 这里用模块增强补上该导出，而不覆盖整个模块，以保留 Table / TableColumn 的原有组件类型。
// 结构对齐 @blueking/table/typings/components/setting-column/Index.vue.d.ts 的 ISettings。
export {};

declare module '@blueking/table' {
  export interface ISettings {
    checked?: string[];
    disabled?: string[];
    size?: 'medium' | 'mini' | 'small';
    fields?: {
      disabled: boolean;
      field: string;
      /**
       * 即将废弃，不建议使用，被 title 替换
       * @deprecated
       */
      label: string;
      title: string;
    }[];
  }
}
