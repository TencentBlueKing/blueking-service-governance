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

/**
 * 事实上 `~/@types/v1` 并未提供对应 type，这里仍保留deprecated，待后续补充
 */

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export interface Component {
  // 管理员
  administrator: string;
  // 组件可见范围，如果为空，组件属于市场组件，不为空属于空间组件
  allowedRange: string;
  // 组件创建时间
  createTime: Date;
  // 组件创建者
  creator: string;
  // 组件描述
  definition: ComponentDefinition;
  // 中文名字
  displayName: string;
  labels: Record<string, string>;
  // 组件名字，全局唯一
  name: string;
  // 组件输出，yaml直接放到这个字段就好
  output: string;
  // 是否公开
  public: boolean;
  // 引用次数
  referenceCount: number;
  // 组件操作
  restOperations: RestOperations;
  // 组件状态
  status: string;
  // 组件类型，Component/Strategy/Storage/Deploy
  type: string;
  // 组件更新者
  updatedBy: string;
  // 组件更新时间
  updatedTime: Date;
  // 版本
  version: string;
}
/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export interface ComponentDefinition {
  description: string;
  properties: Record<string, any>[];
}
