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

import type { BuildConfigOutputObj, RepoBuildConfigOutputObj } from '../../src/@types/v1/app';
import type { BkCIPipelineVariableOutput } from '../../src/@types/v1/bkintegrations-bkci';
import type { Schema } from '../utils/form';

export type RepoBuilderFormData = {
  defaultBranch: string;
  dockerfile: string;
  sourceDir: string;
};

export const RepoBuilderFormSchema: Schema<RepoBuilderFormData> = {
  defaultBranch: { selector: '默认分支', type: 'input' },
  sourceDir: { selector: '构建目录', type: 'input' },
  dockerfile: { selector: 'Dockerfile 路径', type: 'input' },
};

export type BuilderPipeline = {
  id: string;
  name: string;
  variables: (BkCIPipelineVariableOutput & { id: string })[];
};

export type BuilderRepository = {
  alias: string;
  branch: string;
  url: string;
};

export type BuilderSaveRequest = {
  codeRepo: null | RepoBuildConfigOutputObj;
  pipeline: BuildConfigOutputObj['pipelineBuildConfig'] | null;
  sourceType: 'codeRepository' | 'pipeline';
  tagConfig?: BuildConfigOutputObj['tagConfig'] | null;
};

export type BuilderSnapshot = {
  appID: string;
  buildConfig: BuildConfigOutputObj;
};

/** 已保存的流水线按缓存值回显，其余流水线使用参数默认值。 */
export function pipelineInitialValues(
  pipeline: BuilderPipeline,
  savedPipeline?: BuildConfigOutputObj['pipelineBuildConfig'],
): Record<string, string> {
  return Object.fromEntries(
    pipeline.variables.map(variable => [
      variable.id,
      (savedPipeline?.pipelineID === pipeline.id ? savedPipeline.params?.[variable.id] : undefined) ??
        variable.defaultValue ??
        '',
    ]),
  );
}

export function pipelineValues(pipeline: BuilderPipeline, suffix: string): Record<string, string> {
  return Object.fromEntries(pipeline.variables.map((variable, index) => [variable.id, `e2e-${suffix}-${index}`]));
}

export function repositorySaveRequest(repository: BuilderRepository): BuilderSaveRequest {
  return {
    sourceType: 'codeRepository',
    codeRepo: {
      type: 'TGit',
      repoURL: repository.url,
      repoAlias: repository.alias,
      defaultBranch: repository.branch,
      sourceDir: '.',
      dockerfile: 'Dockerfile',
      imageBuildMode: 'repositoryDockerfile',
      dockerBuildArgs: {},
    },
    pipeline: null,
  };
}
