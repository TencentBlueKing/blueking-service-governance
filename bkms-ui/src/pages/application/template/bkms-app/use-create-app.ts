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

import { ref } from 'vue';

import { cloneDeep } from 'lodash-es';
import { AppService } from '~/api/modules/v1/app';
import { isRuntimeModSupported } from '~/composables/use-app-runtime';

import type {
  AppRuntimeDefinitionsOutputObj,
  BKMSAppModelSpecInput,
  BuildConfigInput,
  CreateBKMSAppRequest,
  FrameworkConfigFileInput,
  PipelineBuildConfigInput,
  PlatformBuildConfigInput,
  PublicView,
  RepoBuildConfigInput,
} from '~/@types/v1/app';

// 首版创建入口仅开放这三种语言，详情仍兼容已有的 blank + C++ 应用。
export const BLANK_LANGUAGES = ['go', 'python', 'nodejs'] as const;

export interface BKMSAppForm {
  appModelSpec: BKMSAppModelSpecInput & { args: string[]; command: string[]; shCommand: string };
  framework: string;
  frameworkConfigFile: FrameworkConfigFileInput & { content: string };
  id: string;
  language: string;
  name: string;
  workspaceID: string;
  buildConfig: BuildConfigInput & {
    pipelineBuildConfig: PipelineBuildConfigInput;
    repoBuildConfig: RepoBuildConfigInput & {
      dockerBuildArgs: Record<string, string>;
      imageBuildMode: 'platform' | 'repositoryDockerfile';
      platformBuildConfig: ReturnType<typeof createPlatformBuildConfig>;
    };
  };
}

/** 仅提交当前构建方式和技术栈使用的字段，清除表单中的残留配置。 */
export function buildBKMSAppPayload(form: BKMSAppForm, runtime: PublicView): CreateBKMSAppRequest {
  const buildConfig = cloneDeep(form.buildConfig) as BuildConfigInput;
  if (buildConfig.sourceType === 'pipeline') {
    delete buildConfig.repoBuildConfig;
  } else {
    delete buildConfig.pipelineBuildConfig;
    if (buildConfig.repoBuildConfig) {
      if (!isRuntimeModSupported(runtime, 'platformBuild'))
        buildConfig.repoBuildConfig.imageBuildMode = 'repositoryDockerfile';
      if (buildConfig.repoBuildConfig.imageBuildMode !== 'platform')
        delete buildConfig.repoBuildConfig.platformBuildConfig;
      else buildConfig.repoBuildConfig.dockerfile = '';
    }
  }
  const appModelSpec = cloneDeep(form.appModelSpec);
  if (form.framework === 'blank') {
    // blank 创建时只填写命令和参数，脚本、环境变量在详情页配置。
    appModelSpec.shCommand = '';
    appModelSpec.envVars = [];
    appModelSpec.command = appModelSpec.command?.map(value => value.trim()).filter(Boolean);
    appModelSpec.args = appModelSpec.args?.map(value => value.trim()).filter(Boolean);
  }
  return {
    workspaceID: form.workspaceID,
    name: form.name,
    id: form.id,
    framework: form.framework,
    language: form.language,
    buildConfig,
    appModelSpec,
    // blank 不接受框架文件字段，必须省略，不能传空对象。
    ...(isRuntimeModSupported(runtime, 'frameworkFile')
      ? { frameworkConfigFile: cloneDeep(form.frameworkConfigFile) }
      : {}),
  };
}

export function createBKMSAppForm(workspaceID: string, framework: string, language: string): BKMSAppForm {
  return {
    workspaceID,
    framework,
    language,
    name: '',
    id: '',
    buildConfig: {
      sourceType: 'codeRepository',
      repoBuildConfig: {
        type: 'TGit',
        defaultBranch: 'master',
        repoAlias: '',
        repoURL: '',
        sourceDir: '',
        dockerfile: './Dockerfile',
        dockerBuildArgs: {},
        imageBuildMode: 'repositoryDockerfile',
        platformBuildConfig: createPlatformBuildConfig(),
      },
      pipelineBuildConfig: { pipelineID: '', params: {} },
    },
    appModelSpec: { command: [], args: [], shCommand: '', envVars: [] },
    frameworkConfigFile: {
      name: framework === 'taf' ? 'config.conf' : `trpc_${language}.yaml`,
      mountDir: framework === 'taf' ? '/usr/local/taf/conf' : '/usr/local/trpc/bin/',
      content: '',
    },
  };
}

export function createPlatformBuildConfig() {
  return {
    builderImage: '',
    runnerImage: '',
    extraFiles: [] as string[],
    commands: { build: [] as string[], preBuild: [] as string[], runtimeEnv: [] as string[], start: '' },
  } satisfies PlatformBuildConfigInput;
}

export function findRuntime(definitions: AppRuntimeDefinitionsOutputObj | null, framework: string, language: string) {
  if (framework === 'blank' && !BLANK_LANGUAGES.some(item => item === language)) return undefined;
  const definition = definitions?.frameworks?.find(item => item.name === framework);
  if (!definition?.supportedLanguages?.includes(language)) return undefined;
  return definitions?.matrix?.find(item => item.framework === framework && item.language === language);
}

export function useRuntimeDefinitions() {
  const definitions = ref<AppRuntimeDefinitionsOutputObj | null>(null);
  const loading = ref(false);
  const error = ref(false);
  async function load() {
    loading.value = true;
    error.value = false;
    try {
      definitions.value = await AppService.getAppRuntimeDefinitions({}, { interceptorErr: false });
    } catch {
      definitions.value = null;
      error.value = true;
    } finally {
      loading.value = false;
    }
  }
  return { definitions, loading, error, load };
}
