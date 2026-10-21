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

import { BKMS_REGEX } from '~/common/const';

/** 「挂载路径」校验提示，新建与编辑表单共用，避免多处文案漂移。 */
export const MOUNT_DIR_INVALID_MESSAGE = '挂载路径必须以“/”开头，且不能包含“.”或“..”路径段及反斜杠“\\”';

/** 「挂载路径」表单校验规则。 */
export function createMountDirRules(t: (key: string) => string) {
  return [
    {
      message: t(MOUNT_DIR_INVALID_MESSAGE),
      trigger: 'blur',
      validator: isValidMountDir,
    },
  ];
}

/** 拼接挂载预览路径，补全空值占位符并合并重复斜杠。 */
export function formatMountPathPreview(mountDir: string, name: string) {
  return `${mountDir || '/[路径]'}/${name || '[文件名]'}`.replace(/\/{2,}/g, '/');
}

/**
 * 校验挂载路径：必须是绝对路径，且不能包含反斜杠、. 或 .. 路径段。
 * 空值放行，交由表单的必填规则提示，避免"未填写"被误报为格式错误。
 */
export function isValidMountDir(value?: string) {
  if (!value) return true;
  if (value.includes('\\')) return false;
  const normalized = normalizeMountDir(value);
  return (
    (normalized === '/' || BKMS_REGEX.appConfigMountDirRegex.test(normalized)) &&
    normalized.split('/').every(segment => segment !== '.' && segment !== '..')
  );
}

/**
 * 归一化挂载路径：合并重复斜杠并去掉末尾斜杠。
 * 仅用于提交前落库，保证「表单展示 / 预览 / 后端 filepath.Join 结果」三者一致；根目录 / 保持不变。
 */
export function normalizeMountDir(value: string) {
  const merged = (value || '').replace(/\/{2,}/g, '/');
  return merged.length > 1 ? merged.replace(/\/+$/, '') : merged;
}
