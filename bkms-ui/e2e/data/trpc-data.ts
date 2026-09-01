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
import { type Schema, createValidForm } from '../utils/form';

// TRPC 应用表单数据类型
export type TrpcFormData = {
  代码库: string;
  启动命令: string[];
  命令参数: string[];
  应用名称: string;
  来源: 'git' | 'pipeline';
  语言: string;
  默认分支: string;
};

export const TrpcFormSchema: Schema<TrpcFormData> = {
  应用名称: {
    selector: '应用名称',
    type: 'input',
    default: `test-trpc-app-${parseInt(String(Math.random() * 1000000), 10)}`,
  },
  代码库: {
    selector: '代码库',
    type: 'select',
  },
  语言: {
    selector: '语言',
    type: 'select',
    default: 'go',
  },
  默认分支: {
    selector: '默认分支',
    type: 'input',
    default: '',
  },
  来源: {
    selector: '源码来源',
    type: 'select',
    default: 'git',
  },
  启动命令: {
    selector: '启动命令',
    type: 'array',
    default: [],
  },
  命令参数: {
    selector: '命令参数',
    type: 'array',
    default: [],
  },
};

export const TrpcFormCases = [
  {
    title: '名称为空',
    data: createValidForm<Partial<TrpcFormData>>({ 应用名称: '' }),
    error: '必填项',
  },
];
