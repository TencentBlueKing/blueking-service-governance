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

import { expect } from '@playwright/test';

import type { AppDetailOutputObj, BuildConfigOutputObj, ResolveAppOutputObj } from '../../src/@types/v1/app';
import type {
  BkCIOAuthGitProjectOutput,
  BkCIPipelineVariableOutput,
  BkCIRepositoryRefOutput,
  PaginatedBkCIPipelineOutput,
} from '../../src/@types/v1/bkintegrations-bkci';
import type {
  BuilderPipeline,
  BuilderRepository,
  BuilderSaveRequest,
  BuilderSnapshot,
} from '../data/build-config-data';
import type { BkmsFixtures } from '../fixtures/fixtures';
import type BuildConfigPage from '../pages/app-detail/build-config.page';
import type { APIRequestContext } from '@playwright/test';

type ApiContext = { request: APIRequestContext; testConfig: BkmsFixtures['testConfig'] };

export async function discoverBuilderPipelines(context: ApiContext): Promise<BuilderPipeline[]> {
  const workspace = encodeURIComponent(context.testConfig.space);
  const pipelines: NonNullable<PaginatedBkCIPipelineOutput['results']> = [];
  let total = 0;
  for (let page = 1; page === 1 || pipelines.length < total; page++) {
    const result = await readData<PaginatedBkCIPipelineOutput>(
      context,
      `/workspaces/${workspace}/bkci-pipelines?page=${page}&pageSize=100`,
    );
    const batch = result.results ?? [];
    pipelines.push(...batch);
    total = Number(result.count) || 0;
    if (!batch.length) break;
  }
  const available = pipelines
    .filter(pipeline => pipeline.id && pipeline.name)
    .sort((a, b) => a.id!.localeCompare(b.id!));
  return Promise.all(
    available.map(async pipeline => {
      const variables = await readData<BkCIPipelineVariableOutput[] | null>(
        context,
        `/workspaces/${workspace}/bkci-pipelines/${encodeURIComponent(pipeline.id!)}/variables`,
      );
      return {
        id: pipeline.id!,
        name: pipeline.name!,
        variables: (variables ?? []).filter(
          (variable): variable is BkCIPipelineVariableOutput & { id: string } => !!variable.id,
        ),
      };
    }),
  );
}

export async function discoverBuilderRepositories(context: ApiContext): Promise<BuilderRepository[]> {
  const workspace = encodeURIComponent(context.testConfig.space);
  const projects = await readData<BkCIOAuthGitProjectOutput[] | null>(
    context,
    `/workspaces/${workspace}/bkci-git-projects`,
  );
  const unique = new Map(
    (projects ?? []).filter(project => project.alias && project.url).map(project => [project.url!, project]),
  );
  expect(unique.size, '源码仓库前置条件不足：当前测试身份至少需要两个已授权且含别名的仓库').toBeGreaterThanOrEqual(2);
  const repositories: BuilderRepository[] = [];
  for (const repository of [...unique.values()].sort((a, b) => a.url!.localeCompare(b.url!))) {
    const query = new URLSearchParams({
      repositoryID: repository.alias!,
      repositoryType: 'NAME',
      page: '1',
      pageSize: '100',
    });
    const branches = await readData<BkCIRepositoryRefOutput[]>(
      context,
      `/workspaces/${workspace}/bkci-repositories/branches?${query}`,
    );
    const branch =
      branches?.find(item => item.name === 'main' || item.name === 'master') ?? branches?.find(item => item.name);
    if (!branch?.name) continue;
    repositories.push({ alias: repository.alias!, url: repository.url!, branch: branch.name });
    if (repositories.length === 2) break;
  }
  expect(repositories.length, '源码仓库前置条件不足：至少需要两个具备可用分支的授权仓库').toBe(2);
  return repositories;
}

export async function expectBuilderPersistence(
  context: ApiContext,
  buildConfigPage: BuildConfigPage,
  appID: string,
  expected: BuilderSaveRequest,
  pipeline?: BuilderPipeline,
) {
  await expect
    .poll(() => readBuilderConfig(context, appID))
    .toMatchObject({
      sourceType: expected.sourceType,
      ...(expected.sourceType === 'pipeline'
        ? { pipelineBuildConfig: expected.pipeline }
        : { repoBuildConfig: expected.codeRepo }),
    });
  const saved = await readBuilderConfig(context, appID);
  expect(expected.sourceType === 'pipeline' ? saved.repoBuildConfig : saved.pipelineBuildConfig).toBeFalsy();
  if (expected.sourceType === 'pipeline') {
    // Go 的空参数 map 在详情接口返回 null，侧栏回显同样按空对象处理。
    expect(saved.pipelineBuildConfig?.params ?? {}, '服务端流水线参数不能残留旧字段').toEqual(
      expected.pipeline?.params,
    );
  }
  await buildConfigPage.openBuilderConfigSideslider();
  await buildConfigPage.expectSavedBuilderValues(expected, pipeline);
  await buildConfigPage.cancelBuilderConfigEdit();
  await buildConfigPage.reloadBaseInfo();
  await buildConfigPage.openBuilderConfigSideslider();
  await buildConfigPage.expectSavedBuilderValues(expected, pipeline);
  await buildConfigPage.cancelBuilderConfigEdit();
}

export async function readBuilderConfig(context: ApiContext, appID: string): Promise<BuildConfigOutputObj> {
  const detail = await readData<AppDetailOutputObj>(context, `/apps/${encodeURIComponent(appID)}`);
  expect(detail.buildConfig, '测试应用缺少构建配置').toBeTruthy();
  return detail.buildConfig!;
}

/** Teardown 比较服务端状态；包括保存失败但服务端已发生变化的情况。 */
export async function restoreBuilderConfig(context: ApiContext, snapshot: BuilderSnapshot) {
  const original = snapshotSaveRequest(snapshot.buildConfig);
  const current = snapshotSaveRequest(await readBuilderConfig(context, snapshot.appID));
  if (JSON.stringify(current) !== JSON.stringify(original)) {
    const response = await context.request.put(apiPath(`/apps/${encodeURIComponent(snapshot.appID)}/build-configs`), {
      headers: { Authorization: `Bearer ${context.testConfig.token}` },
      data: original,
    });
    expect(response.ok(), `恢复原构建配置失败（HTTP ${response.status()}）`).toBeTruthy();
  }
  expect(snapshotSaveRequest(await readBuilderConfig(context, snapshot.appID)), '原构建配置未完整恢复').toEqual(
    original,
  );
}

export async function snapshotBuilderConfig(context: ApiContext): Promise<BuilderSnapshot> {
  expect(context.testConfig.space, '请配置 BKMS_TEST_DEFAULT_SPACE').not.toBe('');
  expect(context.testConfig.token, '请配置 BKMS_TEST_ACCESS_TOKEN').not.toBe('');
  const app = await readData<ResolveAppOutputObj>(
    context,
    `/workspaces/${encodeURIComponent(context.testConfig.space)}/apps/resolve/${encodeURIComponent(context.testConfig.app)}`,
  );
  expect(app.id, '无法解析构建配置测试应用').toBeTruthy();
  const buildConfig = await readBuilderConfig(context, app.id!);
  expect(buildConfig.sourceType, '构建配置测试应用应预置流水线来源').toBe('pipeline');
  return { appID: app.id!, buildConfig: structuredClone(buildConfig) };
}

function apiPath(pathname: string) {
  const prefix =
    process.env.BK_API_V1_PREFIX || `${(process.env.BK_API_PREFIX || '').replace(/\/$/, '')}/bkms/v1/bkms-server`;
  return `${prefix.replace(/\/$/, '')}${pathname}`;
}

async function readData<T>({ request, testConfig }: ApiContext, pathname: string): Promise<T> {
  const response = await request.get(apiPath(pathname), {
    headers: { Authorization: `Bearer ${testConfig.token}` },
  });
  expect(response.ok(), `读取测试数据失败（HTTP ${response.status()}）：${pathname}`).toBeTruthy();
  return (await response.json()).data as T;
}

function snapshotSaveRequest(config: BuildConfigOutputObj): BuilderSaveRequest {
  if (config.sourceType !== 'pipeline' && config.sourceType !== 'codeRepository') {
    throw new Error('测试应用的构建配置来源不支持恢复');
  }
  return {
    sourceType: config.sourceType,
    codeRepo: config.repoBuildConfig ?? null,
    pipeline: config.pipelineBuildConfig ?? null,
    tagConfig: config.tagConfig ?? null,
  };
}
