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

import {
  discoverBuilderPipelines,
  discoverBuilderRepositories,
  expectBuilderPersistence,
  restoreBuilderConfig,
  snapshotBuilderConfig,
} from '../actions/build-config.action';
import { fillFormByType } from '../actions/form.action';
import { pipelineInitialValues, pipelineValues, repositorySaveRequest } from '../data/build-config-data';
import { test as base } from '../fixtures/fixtures';

import type { BuilderSaveRequest, BuilderSnapshot } from '../data/build-config-data';

const test = base.extend<{ builderSnapshot: BuilderSnapshot }>({
  testConfig: async ({ testConfig }, use) => {
    await use({
      ...testConfig,
      app: process.env.BKMS_TEST_BUILD_PIPELINE_APP?.trim() || 'e2e-build-pipeline',
      appType: 'trpc',
    });
  },
  builderSnapshot: [
    async ({ request, testConfig }, use, testInfo) => {
      expect(testInfo.config.workers, '共享应用的构建配置用例必须使用 --workers=1').toBe(1);
      const context = { request, testConfig };
      const snapshot = await snapshotBuilderConfig(context);
      try {
        await use(snapshot);
      } finally {
        await restoreBuilderConfig(context, snapshot);
      }
    },
    { auto: true },
  ],
});

test.describe('构建配置侧栏', () => {
  // 覆盖全局 fullyParallel 设置；失败不跳过下一用例。
  test.describe.configure({ mode: 'default' });

  test('源码仓库：选择切换、分支校验及保存回显', async ({ pages, request, testConfig, builderSnapshot }) => {
    const context = { request, testConfig };
    const [repositoryA, repositoryB] = await discoverBuilderRepositories(context);
    const originalPipeline = (
      await discoverBuilderPipelines(context, builderSnapshot.buildConfig.pipelineBuildConfig?.pipelineID)
    ).find(pipeline => pipeline.id === builderSnapshot.buildConfig.pipelineBuildConfig?.pipelineID);
    expect(originalPipeline, '源码仓库用例需要应用原有流水线可访问，以验证切回后清空仓库配置').toBeTruthy();
    const buildConfig = pages.appDetailPage.buildConfig;
    await buildConfig.gotoBaseInfo();
    await buildConfig.openBuilderConfigSideslider();
    await buildConfig.selectBuilderSource('codeRepository');

    await test.step('选择仓库 A，保存并验证重新打开及刷新回显', async () => {
      await buildConfig.selectBuilderRepository(repositoryA);
      await buildConfig.expectBuilderBranch('');
      await fillFormByType(buildConfig, 'RepoBuilder', {
        defaultBranch: repositoryA.branch,
        sourceDir: '.',
        dockerfile: 'Dockerfile',
      });
      const expected = repositorySaveRequest(repositoryA);
      await buildConfig.saveBuilderConfigAndExpectRequest(builderSnapshot.appID, expected);
      await expectBuilderPersistence(context, buildConfig, builderSnapshot.appID, expected);
    });

    await test.step('切换仓库 B，阻止空分支保存，再保存并验证回显', async () => {
      await buildConfig.openBuilderConfigSideslider();
      await buildConfig.selectBuilderRepository(repositoryB);
      await buildConfig.expectBuilderBranch('');
      await buildConfig.expectBuilderValidationWithoutSave(builderSnapshot.appID, '默认分支');
      await fillFormByType(buildConfig, 'RepoBuilder', {
        defaultBranch: repositoryB.branch,
        sourceDir: '.',
        dockerfile: 'Dockerfile',
      });
      const expected = repositorySaveRequest(repositoryB);
      await buildConfig.saveBuilderConfigAndExpectRequest(builderSnapshot.appID, expected);
      await expectBuilderPersistence(context, buildConfig, builderSnapshot.appID, expected);
    });

    await test.step('从已保存的仓库 B 切回流水线，清空仓库配置并验证回显', async () => {
      await buildConfig.openBuilderConfigSideslider();
      await buildConfig.selectBuilderSource('pipeline');
      await buildConfig.selectBuilderPipeline(originalPipeline!);
      const params = pipelineValues(originalPipeline!, 'from-repository');
      await buildConfig.fillPipelineValues(params);
      const expected: BuilderSaveRequest = {
        sourceType: 'pipeline',
        codeRepo: null,
        pipeline: { pipelineID: originalPipeline!.id, params },
      };
      await buildConfig.saveBuilderConfigAndExpectRequest(builderSnapshot.appID, expected);
      await expectBuilderPersistence(context, buildConfig, builderSnapshot.appID, expected, originalPipeline!);
    });
  });

  test('流水线：来源联动、必填校验、参数切换及保存回显', async ({ pages, request, testConfig, builderSnapshot }) => {
    const context = { request, testConfig };
    const pipelines = await discoverBuilderPipelines(context);
    const primary = pipelines.find(pipeline => pipeline.variables.some(variable => variable.required));
    expect(primary, '流水线前置条件不足：需要含必填参数的流水线').toBeTruthy();
    const primaryKeys = new Set(primary!.variables.map(variable => variable.id));
    const secondary = pipelines.find(
      pipeline =>
        pipeline.id !== primary!.id &&
        pipeline.variables.some(variable => !primaryKeys.has(variable.id)) &&
        primary!.variables.some(variable => !pipeline.variables.some(item => item.id === variable.id)),
    );
    expect(secondary, '流水线前置条件不足：需要另一条具有不同参数集合的流水线').toBeTruthy();
    const empty = pipelines.find(pipeline => !pipeline.variables.length);
    expect(empty, '流水线前置条件不足：需要无参数流水线').toBeTruthy();

    const buildConfig = pages.appDetailPage.buildConfig;
    await buildConfig.gotoBaseInfo();
    await buildConfig.openBuilderConfigSideslider();
    await test.step('源码仓库与流水线互切，当前来源必填校验生效', async () => {
      await buildConfig.selectBuilderSource('codeRepository');
      await buildConfig.selectBuilderSource('pipeline');
      await buildConfig.clearBuilderPipeline();
      await buildConfig.expectBuilderValidationWithoutSave(builderSnapshot.appID, '流水线');
      await buildConfig.selectBuilderPipeline(primary!);
      await buildConfig.expectPipelineValues(
        primary!,
        pipelineInitialValues(primary!, builderSnapshot.buildConfig.pipelineBuildConfig),
      );
      const required = primary!.variables.find(variable => variable.required)!;
      await buildConfig.fillPipelineValue(required.id, '');
      await buildConfig.expectBuilderValidationWithoutSave(builderSnapshot.appID, required.id, true);
    });

    const savePipeline = async (pipeline: NonNullable<typeof primary>, suffix: string) => {
      const params = pipelineValues(pipeline, suffix);
      await buildConfig.fillPipelineValues(params);
      const expected: BuilderSaveRequest = {
        sourceType: 'pipeline',
        codeRepo: null,
        pipeline: { pipelineID: pipeline.id, params },
      };
      await buildConfig.saveBuilderConfigAndExpectRequest(builderSnapshot.appID, expected);
      await expectBuilderPersistence(context, buildConfig, builderSnapshot.appID, expected, pipeline);
    };

    await test.step('有必填参数流水线保存并回显', async () => {
      await savePipeline(primary!, 'primary');
    });
    await test.step('切换参数集合，移除旧字段并保存回显', async () => {
      await buildConfig.openBuilderConfigSideslider();
      await buildConfig.selectBuilderPipeline(secondary!);
      await buildConfig.expectPipelineFieldsHidden(
        primary!.variables
          .filter(variable => !secondary!.variables.some(item => item.id === variable.id))
          .map(variable => variable.id),
      );
      // 保存回显检查已经刷新页面，缓存仅包含 primary；新选的 secondary 使用默认值。
      await buildConfig.expectPipelineValues(secondary!, pipelineInitialValues(secondary!));
      await savePipeline(secondary!, 'secondary');
    });
    await test.step('无参数流水线清空参数并保存回显', async () => {
      await buildConfig.openBuilderConfigSideslider();
      await buildConfig.selectBuilderPipeline(empty!);
      await buildConfig.expectPipelineFieldsHidden(secondary!.variables.map(variable => variable.id));
      await buildConfig.expectPipelineValues(empty!, {});
      await savePipeline(empty!, 'empty');
    });
  });
});
