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

export type IRole = 'admin' | 'developer' | 'operator' | 'sre';
/** 外部调用时记得国际化处理 */
export const PERMISSION_LIST = [
  {
    resource: '空间',
    operation: '查看空间',
    admin: true,
    developer: true,
    sre: true,
    operator: true,
  },
  {
    resource: '空间',
    operation: '创建空间',
    admin: true,
    developer: false,
    sre: false,
    operator: false,
  },
  {
    resource: '空间',
    operation: '编辑空间',
    admin: true,
    developer: false,
    sre: false,
    operator: false,
  },
  {
    resource: '空间',
    operation: '删除空间',
    admin: true,
    developer: false,
    sre: false,
    operator: false,
  },
  {
    resource: '应用',
    operation: '创建应用',
    admin: true,
    developer: true,
    sre: true,
    operator: false,
  },
  {
    resource: '应用',
    operation: '查看应用',
    admin: true,
    developer: true,
    sre: true,
    operator: true,
  },
  {
    resource: '应用',
    operation: '编辑应用',
    admin: true,
    developer: true,
    sre: true,
    operator: false,
  },
  {
    resource: '应用',
    operation: '删除应用',
    admin: true,
    developer: true,
    sre: false,
    operator: false,
  },
  {
    resource: '环境',
    operation: '查看环境',
    admin: true,
    developer: true,
    sre: true,
    operator: true,
  },
  {
    resource: '环境',
    operation: '部署到环境',
    admin: true,
    developer: true,
    sre: true,
    operator: false,
  },
  {
    resource: '环境',
    operation: '创建环境',
    admin: true,
    developer: false,
    sre: true,
    operator: false,
  },
  {
    resource: '环境',
    operation: '编辑环境',
    admin: true,
    developer: false,
    sre: true,
    operator: false,
  },
  {
    resource: '环境',
    operation: '删除环境',
    admin: true,
    developer: false,
    sre: true,
    operator: false,
  },
];
