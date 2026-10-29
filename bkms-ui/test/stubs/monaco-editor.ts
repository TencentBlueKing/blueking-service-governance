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
 * monaco-editor 的 vitest 专用 stub：
 * monaco-editor 包体积大、依赖 worker 环境，且其入口在 vitest（SSR 解析）下无法定位；
 * 仅提供组件测试所需的语言元数据入口与错误级别，编辑器实例由专项用例 mock。
 * 由 vite.config.mts test.alias 指向本文件，仅影响 vitest，不影响构建。
 */
const editorStub = {};

export default editorStub;
export const editor = editorStub;
export const languages = { getLanguages: (): { extensions?: string[]; id: string }[] => [] };
export const MarkerSeverity = { Error: 8 };
export const Uri = { parse: (v: string) => ({ toString: () => v }) };
