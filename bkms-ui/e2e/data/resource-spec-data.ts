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
import type { Schema } from '../utils/form';

/**
 * 应用配置 - 部署配置 - 资源规格表单数据类型。
 * 注意：
 * - `实例数` 是 `<Input type="number">`，由 Page Object 单独处理（role=spinbutton）；
 * - CPU/内存 均为 bkui-vue Select，可通过 schema 的 `select` 类型驱动。
 */
export type ResourceSpecFormData = {
  CPU限制: string;
  CPU预留: string;
  内存限制: string;
  内存预留: string;
};

export const ResourceSpecSchema: Schema<ResourceSpecFormData> = {
  CPU预留: {
    selector: 'CPU 预留 (Requests)',
    type: 'select',
  },
  CPU限制: {
    selector: 'CPU 限制 (Limits)',
    type: 'select',
  },
  内存预留: {
    selector: '内存预留 (Requests)',
    type: 'select',
  },
  内存限制: {
    selector: '内存限制 (Limits)',
    type: 'select',
  },
};
