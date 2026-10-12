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

/** 与接口 revisionType 一致 */
export type RevisionType = 'branch' | 'commit' | 'tag';

/** CommitID 允许 7–40 位十六进制，与构建服务的校验一致 */
const COMMIT_ID_PATTERN = /^[0-9a-fA-F]{7,40}$/;

/** 构建记录里用来判断源材料是否要去重的来源 */
export type SourceRevision = {
  commitID: string;
  revision: string;
  revisionType?: string;
};

export function isCommitID(value: string) {
  return COMMIT_ID_PATTERN.test(value.trim());
}

/** 第二项标题：分支 / Tag / CommitID */
export function revisionFieldLabel(type: RevisionType) {
  if (type === 'tag') return '代码 Tag';
  if (type === 'commit') return 'CommitID';
  return '代码分支';
}

/**
 * 组装创建构建 / 构建并部署的代码修订字段。
 * 自定义流水线不传 revisionType，只留兼容字段 branch，避免改变原请求。
 */
export function revisionRequestFields(showSource: boolean, type: RevisionType, value: string) {
  if (!showSource) {
    return { branch: value };
  }

  return {
    revisionType: type,
    revision: value,
    ...(type === 'branch' ? { branch: value } : {}),
  };
}

/** 溢出提示与单元格一致：修订（分支名 / Tag / CommitID）加短哈希 */
export function sourceMaterialText(material: SourceRevision) {
  // commit 还没回写时不要用修订名顶短哈希，否则提示会变成 "main main"
  if (!material.commitID) {
    return material.revision;
  }

  return `${material.revision} ${material.commitID.slice(0, 8)}`.trim();
}
