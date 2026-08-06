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

const axios = require('axios');

/**
 * Authentication headers shared by all API-test helpers. The API-test config
 * allows these headers to directly set the authenticated user.
 */
const AUTH_HEADERS = {
  'X-Bk-Authed-User-Info': '{"userId":"api-test-user"}',
  'X-Bk-Authed-User-Credential': '{"bkTicket": "xxx"}'
};

/**
 * Sleep for the given number of milliseconds.
 *
 * @param {number} ms
 * @returns {Promise<void>}
 */
function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Create a normal app config file (Helm values, etc).
 * Idempotent when `cacheKey` is provided.
 *
 * @param {Object} options
 * @param {string} options.appID - Application ID (required).
 * @param {string} options.name - File name (required).
 * @param {string} [options.type='normal'] - File type.
 * @param {string} [options.contentSourceType='local'] - Content source type.
 * @param {string} [options.fileFormat='yaml'] - File format.
 * @param {string} [options.versionDescription='Initial version'] - Version description.
 * @param {string} [options.cacheKey] - Bruno variable name used to cache the file ID.
 * @returns {Promise<{id: string}>}
 */
async function createAppConfigFile(options = {}) {
  const serviceURL = bru.getEnvVar('serviceURL');
  if (!options.appID) {
    throw new Error('Application ID is required');
  }
  if (!options.name) {
    throw new Error('App config file name is required');
  }

  if (options.cacheKey) {
    const existingID = bru.getVar(options.cacheKey);
    if (existingID) {
      console.error(`App config file ${options.name} already created, skip creation`);
      return {id: existingID};
    }
  }

  const body = {
    name: options.name,
    type: options.type || 'normal',
    contentSourceType: options.contentSourceType || 'local',
    fileFormat: options.fileFormat || 'yaml',
    versionDescription: options.versionDescription || 'Initial version'
  };

  try {
    const response = await axios.post(
      `${serviceURL}/apps/${options.appID}/app-config-files`,
      body,
      {headers: AUTH_HEADERS}
    );

    const id = response.data?.item?.id;
    if (!id) {
      throw new Error(`App config file ID not found in response: ${JSON.stringify(response.data)}`);
    }
    if (options.cacheKey) {
      bru.setVar(options.cacheKey, id);
    }
    return {id};
  } catch (error) {
    console.error('=== App Config File Creation Error ===');
    console.error('Error:', error);
    console.error('Error response:', error.response?.data);

    throw error;
  }
}

/**
 * Update the content of an app config file.
 *
 * @param {Object} options
 * @param {string} options.appID - Application ID (required).
 * @param {string} options.fileID - App config file ID (required).
 * @param {string} options.content - File content (required).
 * @param {string} [options.versionDescription='Update content'] - Version description.
 * @returns {Promise<{compiledContent: string}>}
 */
async function updateAppConfigFileContent(options = {}) {
  const serviceURL = bru.getEnvVar('serviceURL');
  if (!options.appID) {
    throw new Error('Application ID is required');
  }
  if (!options.fileID) {
    throw new Error('App config file ID is required');
  }
  if (typeof options.content !== 'string') {
    throw new Error('App config file content is required');
  }

  try {
    const response = await axios.put(
      `${serviceURL}/apps/${options.appID}/app-config-files/${options.fileID}/content`,
      {
        content: options.content,
        versionDescription: options.versionDescription || 'Update content'
      },
      {headers: AUTH_HEADERS}
    );

    return {compiledContent: response.data?.compiledContent || ''};
  } catch (error) {
    console.error('=== App Config File Content Update Error ===');
    console.error('Error:', error);
    console.error('Error response:', error.response?.data);

    throw error;
  }
}

/**
 * Create a Helm app under a workspace.
 * This helper is idempotent per app ID.
 *
 * @param {Object} options - Options.
 * @param {string} options.id - Application ID (required).
 * @param {string} options.name - Application name (optional, defaults to id).
 * @param {string} options.workspaceID - Workspace ID (required).
 * @param {string} [options.helmSourceRepoType='HelmRepo'] - Helm source repo type, one of: 'HelmRepo', 'GitRepo', 'BCSRepo'.
 * @param {Object} [options.helmRepoConfig] - Custom HelmRepo config (used when helmSourceRepoType='HelmRepo').
 * @param {Object} [options.gitRepoConfig] - Custom GitRepo config (used when helmSourceRepoType='GitRepo').
 * @param {Object} [options.bcsRepoConfig] - Custom BCSRepo config (used when helmSourceRepoType='BCSRepo').
 * @returns {Promise<{id: string, name: string}>}
 */
async function createHelmApp(options = {}) {
  const serviceURL = bru.getEnvVar('serviceURL');
  const workspaceID = options.workspaceID;
  if (!workspaceID) {
    throw new Error('Workspace ID is required');
  }
  const url = `${serviceURL}/workspaces/${workspaceID}/apps`;

  const appID = options.id;
  if (!appID) {
    throw new Error('Application ID is required');
  }
  const appName = options.name || appID;
  const helmAppKey = `HelmAppCreated_${appID}`;

  if (bru.getVar(helmAppKey)) {
    console.error(`Helm app ${appID} already created, skip creation`);
    return { id: appID, name: appName };
  }

  const defaultBody = {
    name: appName,
    id: appID,
    type: 'helm',
    buildConfig: {
      sourceType: 'imageRegistry',
      pipelineType: '123',
      imageBuildConfig: {
        name: 'api-test-app'
      }
    },
    helmSpec: {
      helmSource: {
        repoType: 'HelmRepo',
        helmRepoConfig: {
          repoURL: 'http://www.example.com/foo',
          chartName: 'foobar'
        }
      }
    }
  };

  // 根据 helmSourceRepoType 参数设置不同的 helmSource 配置
  const helmSourceRepoType = options.helmSourceRepoType || 'HelmRepo';
  if (helmSourceRepoType === 'HelmRepo') {
    defaultBody.helmSpec.helmSource = {
      repoType: 'HelmRepo',
      helmRepoConfig: options.helmRepoConfig || {
        repoURL: 'http://www.example.com/foo',
        chartName: 'foobar'
      }
    };
  } else if (helmSourceRepoType === 'GitRepo') {
    defaultBody.helmSpec.helmSource = {
      repoType: 'GitRepo',
      gitRepoConfig: options.gitRepoConfig || {
        type: 'TGit',
        repoAlias: 'test-repo',
        repoURL: 'https://example.com/test/repo.git',
        revision: 'main',
        sourceDir: 'charts'
      }
    };
  } else if (helmSourceRepoType === 'BCSRepo') {
    defaultBody.helmSpec.helmSource = {
      repoType: 'BCSRepo',
      bcsRepoConfig: options.bcsRepoConfig || {
        projectCode: 'test-project',
        repoName: 'test-repo',
        chartName: 'test-chart'
      }
    };
  }

  const body = {...defaultBody};
  if (options.buildConfig) {
    body.buildConfig = options.buildConfig;
  }
  if (options.helmSpec) {
    body.helmSpec = options.helmSpec;
  }

  try {
    const response = await axios.post(url, body, {
      headers: AUTH_HEADERS
    });

    if (response.status === 200) {
      console.log("Helm app created, resp:", response.data);
      // Set the var to indicate the app has been created
      bru.setVar(helmAppKey, true);
      const created = response.data?.data || {};
      return {
        id: created.id || appID,
        name: created.name || appName
      };
    }
  } catch (error) {
    console.error("=== Helm App Creation Error ===");
    console.error("Error:", error);
    console.error("Error response:", error.response?.data);

    throw error;
  }
  return { id: appID, name: appName };
}


/**
 * 创建工作空间
 * 通过 API 创建一个新的工作空间，包含镜像仓库配置等信息。
 * 函数内部会检查该工作空间是否已创建，避免重复创建（key：workspaceID）。
 * 创建成功后会将工作空间 ID 保存到变量中供后续使用。
 *
 * @param {Object} options - 可选参数
 * @param {string} options.id - 工作空间 ID（必填，该值获取顺序：参数 option.workspaceID、bru 环境变量 'workspaceID' 获取）
 * @param {string} options.displayName - 工作空间显示名称（默认为 'API Test Workspace'）
 * @param {string} options.description - 工作空间描述（默认为 '这是一个API测试创建的工作空间'）
 * @param {string} options.bkCIProjectID - 蓝鲸 CI 项目 ID（默认为空字符串）
 * @param {number} options.bkCCBizID - 蓝鲸配置中心业务 ID（默认为 100001）
 */
async function createWorkspace(options = {}) {
  if (!options.id) {
    throw new Error('Workspace ID is required');
  }

  // Read environment variables
  const serviceURL = bru.getEnvVar('serviceURL');
  const url = `${serviceURL}/workspaces`;
  const workspaceID = options.id;
  const workspaceKey = `WorkspaceCreated_${workspaceID}`;
  if (bru.getVar(workspaceKey)) {
    console.error(`Workspace ${workspaceID} already created, skip creation`);
    return {"id": workspaceID};
  }

  const defaultBody = {
    id: workspaceID,
    displayName: options.displayName || 'API Test Workspace',
    description: options.description || '这是一个API测试创建的工作空间',
    bkCIProjectID: options.bkCIProjectID || '',
    bkCCBizID: options.bkCCBizID || 100001,
    imageRegistry: {
      registry: 'mirror.example.com',
      username: 'registry-user',
      password: 'registry-foo-password'
    }
  };

  const body = {...defaultBody, ...options};

  try {
    const response = await axios.post(url, body, {
      headers: AUTH_HEADERS
    });

    if (response.status === 200) {
      console.log("Workspace created, resp:", response.data);
      // Set the var to indicate the workspace has been created
      bru.setVar(workspaceKey, true);
      // Save workspace ID for later use
      if (response.data && response.data.data && response.data.data.id) {
        bru.setVar('createdWorkspaceID', response.data.data.id);
      }
    }
  } catch (error) {
    console.error("=== Workspace Creation Error ===");
    console.error("Error:", error);
    console.error("Error response:", error.response?.data);

    throw error;
  }

  return {"id": workspaceID};
}

/**
 * 创建环境
 * 在指定的工作空间下创建一个环境，需要配置集群信息（集群 ID、命名空间等）。
 * 调用此函数前必须确保工作空间已存在。
 * 函数内部会检查该环境是否已创建，避免重复创建（key：workspaceID + envName）。
 *
 * @param {Object} options - 环境配置参数
 * @param {string} options.workspaceID - 工作空间 ID（必填，该值获取顺序：参数 options.workspaceID、bru 环境变量 'workspaceID' 获取）
 * @param {string} options.name - 环境名称（必填）
 * @param {string} options.displayName - 环境显示名称（默认使用 name）
 * @param {string} options.type - 环境类型，可选值：'development'、'test'、'staging'、'production'，默认为 'test'；默认预发布环境名为 'staging'
 * @param {string} options.description - 环境描述（默认为空字符串）
 * @param {Object} options.cluster - 集群配置（必填）
 * @param {string} options.cluster.bizID - 业务 ID
 * @param {string} options.cluster.projectCode - 项目代码
 * @param {string} options.cluster.clusterID - 集群 ID（必填）
 * @param {string} options.cluster.clusterType - 集群类型，默认为 'single'
 * @param {string} options.cluster.namespace - 命名空间（必填）
 *
 * @example
 * // 完整示例
 *     const apiTestEnvName = "test-" + bru.getVar("randomSuffix");
 *     await common.createEnvironment({
 *         name: apiTestEnvName,
 *         workspaceID: apiTestWorkspaceID,
 *         type: "development",
 *         description: "test",
 *         cluster:{
 *             clusterID: "BCS-K8S-12345",
 *             clusterType: "single",
 *             namespace: "default",
 *         }
 *     });
 *     bru.setVar('apiTestEnvName',apiTestEnvName);
 */
async function createEnvironment(options = {}) {
  const serviceURL = bru.getEnvVar('serviceURL');

  // Validate required parameters
  if (!options.workspaceID) {
    throw new Error('Workspace ID is required');
  }
  if (!options.name) {
    throw new Error('Environment name is required');
  }
  if (!options.cluster || !options.cluster.clusterID || !options.cluster.namespace) {
    throw new Error('Cluster configuration (clusterID and namespace) is required');
  }

  const workspaceID = options.workspaceID;
  const url = `${serviceURL}/workspaces/${workspaceID}/envs`;
  const envName = options.name;
  const envKey = `EnvCreated_${workspaceID}_${envName}`;

  const existingEnvID = bru.getVar(envKey);
  if (existingEnvID) {
    console.error(`Environment ${envName} already created, skip creation`);
    return {id: existingEnvID, name: envName};
  }

  const defaultBody = {
    name: envName,
    displayName: options.displayName || options.name,
    type: options.type || 'test',
    description: options.description || '',
    cluster: {
      clusterID: options.cluster.clusterID,
      clusterType: options.cluster.clusterType || 'single',
      namespace: options.cluster.namespace,
    }
  };


  const body = {...defaultBody};

  try {
    const response = await axios.post(url, body, {
      headers: AUTH_HEADERS
    });

    if (response.status === 200) {
      console.log("Environment created, resp:", response.data);
      const created = response.data?.data || {};
      const envID = created.id;
      if (!envID) {
        throw new Error(`Environment ID not found in response for envName: ${envName}`);
      }
      bru.setVar(envKey, envID);
      return {id: envID, name: envName};
    }
  } catch (error) {
    console.error("=== Environment Creation Error ===");
    console.error("Error:", error);
    console.error("Error response:", error.response?.data);

    throw error;
  }

  return {name: envName};
}

/**
 * Create a TRPC app under a workspace.
 * This helper is idempotent per app ID.
 *
 * @param {Object} options - TRPC app options.
 * @param {string} options.id - Application ID (required).
 * @param {string} options.name - Application name (optional, defaults to id).
 * @param {string} options.workspaceID - Workspace ID (required).
 * @param {string} options.displayName - Application display name (optional, defaults to name).
 * @param {Object} options.appModelSpec - AppModel spec (optional).
 * @param {Object} options.buildConfig - Build config (optional).
 * @returns {Promise<{id: string, name: string}>}
 */
async function createTrpcApp(options = {}) {
  const serviceURL = bru.getEnvVar('serviceURL');
  const workspaceID = options.workspaceID;
  if (!workspaceID) {
    throw new Error('Workspace ID is required');
  }
  const url = `${serviceURL}/workspaces/${workspaceID}/apps`;
  
  const appID = options.id;
  if (!appID) {
    throw new Error('Application ID is required');
  }
  const appName = options.name || appID;
  const appKey = `AppModelAppCreated_${appID}`;

  if (bru.getVar(appKey)) {
    console.error(`AppModel app ${appID} already created, skip creation`);
    return { id: appID, name: appName };
  }


  const body = {
    id: appID,
    type: 'trpc',
    name: appName,
    displayName: options.displayName || appName,
    // The default appModelSpec and buildConfig can be overridden by options
    appModelSpec: {
      trpcSpec: {
        language: "go",
        fileName: "trpc_go.yaml",
        filePath: "/usr/local/trpc/bin/",
        fileContent: "dGVzdAo="
      },
      command: [],
      args: [],
      envVars: []
    },
    buildConfig: {
      sourceType: 'codeRepository',
      repoBuildConfig: {
        type: 'TGit',
        repoAlias: 'test',
        repoURL: 'https://example.com/example.git',
        defaultBranch: 'main',
        sourceDir: '.',
        dockerfile: 'Dockerfile',
        dockerBuildArgs: {
          ARG1: 'value1',
        }
      }
    }
  };

  if (options.appModelSpec) {
    body.appModelSpec = options.appModelSpec;
  }

  if (options.buildConfig) {
    body.buildConfig = options.buildConfig;
  }

  try {
    const response = await axios.post(url, body, {
      headers: AUTH_HEADERS
    });

    if (response.status === 200) {
      console.log("AppModel app created successfully");
      console.log("Response data:", JSON.stringify(response.data, null, 2));

      bru.setVar(appKey, true);

      const created = response.data?.data || {};
      return {
        id: created.id || appID,
        name: created.name || appName
      };
    }
  } catch (error) {
    console.error("=== AppModel App Creation Error ===");
    console.error("Error:", error);
    console.error("Error response:", error.response?.data);

    throw error;
  }

  return { id: appID, name: appName };
}

/**
 * Create a TAF app under a workspace.
 * This helper is idempotent per app ID.
 *
 * @param {Object} options - TAF app options.
 * @param {string} options.id - Application ID (required).
 * @param {string} options.name - Application name (optional, defaults to id).
 * @param {string} options.workspaceID - Workspace ID (required).
 * @param {string} options.displayName - Application display name (optional, defaults to name).
 * @param {Object} options.appModelSpec - AppModel spec (optional).
 * @param {Object} options.buildConfig - Build config (optional).
 * @returns {Promise<{id: string, name: string}>}
 */
async function createTafApp(options = {}) {
  const serviceURL = bru.getEnvVar('serviceURL');
  const workspaceID = options.workspaceID;
  if (!workspaceID) {
    throw new Error('Workspace ID is required');
  }
  const url = `${serviceURL}/workspaces/${workspaceID}/apps`;

  const appID = options.id;
  if (!appID) {
    throw new Error('Application ID is required');
  }
  const appName = options.name || appID;
  const appKey = `AppModelAppCreated_${appID}`;

  if (bru.getVar(appKey)) {
    console.error(`AppModel app ${appID} already created, skip creation`);
    return { id: appID, name: appName };
  }

  const body = {
    id: appID,
    type: 'taf',
    name: appName,
    displayName: options.displayName || appName,
    appModelSpec: {
      tafSpec: {
        fileName: 'taf_config.conf',
        filePath: '/usr/local/taf/conf/',
        fileContent: 'dGVzdAo='
      },
      command: [],
      args: [],
      envVars: []
    },
    buildConfig: {
      sourceType: 'codeRepository',
      repoBuildConfig: {
        type: 'TGit',
        repoAlias: 'test',
        repoURL: 'https://example.com/example.git',
        defaultBranch: 'main',
        sourceDir: '.',
        dockerfile: 'Dockerfile',
        dockerBuildArgs: {
          ARG1: 'value1',
        }
      }
    }
  };

  if (options.appModelSpec) {
    body.appModelSpec = options.appModelSpec;
  }

  if (options.buildConfig) {
    body.buildConfig = options.buildConfig;
  }

  try {
    const response = await axios.post(url, body, {
      headers: AUTH_HEADERS
    });

    if (response.status === 200) {
      console.log("TAF app created successfully");
      console.log("Response data:", JSON.stringify(response.data, null, 2));

      bru.setVar(appKey, true);

      const created = response.data?.data || {};
      return {
        id: created.id || appID,
        name: created.name || appName
      };
    }
  } catch (error) {
    console.error("=== TAF App Creation Error ===");
    console.error("Error:", error);
    console.error("Error response:", error.response.data);

    throw error;
  }

  return { id: appID, name: appName };
}

/**
 * Update default app spec resources for an app.
 *
 * @param {Object} options - Resource options.
 * @param {string} options.appID - Application ID (required).
 * @param {Object} options.resources - App spec resources (required).
 * @returns {Promise<Object>}
 */
async function updateDefaultAppSpecResources(options = {}) {
  const serviceURL = bru.getEnvVar('serviceURL');
  if (!options.appID) {
    throw new Error('Application ID is required');
  }
  if (!options.resources) {
    throw new Error('App spec resources are required');
  }

  try {
    const response = await axios.put(
      `${serviceURL}/apps/${options.appID}/app-spec/default-resources`,
      {appSpecResources: options.resources},
      {headers: AUTH_HEADERS}
    );

    return response.data?.data || response.data;
  } catch (error) {
    console.error('=== Default App Spec Resources Update Error ===');
    console.error('Error:', error);
    console.error('Error response:', error.response?.data);

    throw error;
  }
}

/**
 * Ensure a workspace component exists and return its name.
 * Idempotent by resultVar when provided.
 *
 * @param {Object} options - Workspace component options.
 * @param {string} options.workspaceID - Workspace ID (required).
 * @param {string} options.resultVar - Variable name to cache component name (optional).
 * @param {string} options.name - Component name (optional).
 * @param {string} options.type - Component type (required).
 * @param {Object} options.properties - Component properties (required).
 * @param {string} options.scopeType - Component scope type (required).
 * @param {string[]} options.scopeEnvNames - Component scope env names (optional).
 * @returns {Promise<{name: string}>}
 */
async function ensureWorkspaceComponent(options = {}) {
  const serviceURL = bru.getEnvVar('serviceURL');
  const workspaceID = options.workspaceID || bru.getVar('workspaceID');
  if (!workspaceID) {
    throw new Error('Workspace ID is required');
  }

  const resultVar = options.resultVar;
  if (resultVar) {
    const existingName = bru.getVar(resultVar);
    if (existingName) {
      console.error(`Workspace component ${existingName} already created, skip creation`);
      return { name: existingName };
    }
  }

  if (!options.type) {
    throw new Error('Component type is required');
  }
  if (!options.properties) {
    throw new Error('Component properties are required');
  }
  if (!options.scopeType) {
    throw new Error('Component scopeType is required');
  }

  const url = `${serviceURL}/workspaces/${workspaceID}/components`;
  const body = {
    type: options.type,
    properties: options.properties,
    scopeType: options.scopeType,
    scopeEnvNames: options.scopeEnvNames || []
  };

  if (options.name) {
    body.compName = options.name;
  }

  try {
    const response = await axios.post(url, body, {
      headers: AUTH_HEADERS
    });

    if (response.status === 200) {
      const created = response.data?.data || {};
      const compName = created.name || options.name;
      if (!compName) {
        throw new Error('Component name is missing in response');
      }
      if (resultVar) {
        bru.setVar(resultVar, compName);
      }
      return { name: compName };
    }
  } catch (error) {
    console.error("=== Workspace Component Creation Error ===");
    console.error("Error:", error);
    console.error("Error response:", error.response?.data);

    throw error;
  }

  throw new Error('Failed to create workspace component');
}

/**
 * Ensure a global-scope VolumeSecret workspace component exists.
 *
 * @param {Object} options - Helper options.
 * @param {string} options.workspaceID - Workspace ID (required).
 * @param {string} options.resultVar - Variable name to cache component name (optional).
 * @param {Object} options.properties - Override properties (optional).
 * @returns {Promise<{name: string}>}
 */
async function ensureGlobalVolumeSecretWorkspaceComponent(options = {}) {
  return ensureWorkspaceComponent({
    workspaceID: options.workspaceID,
    resultVar: options.resultVar,
    type: 'VolumeSecret',
    properties: options.properties || {
      path: '/data/secret',
      secretName: 'test-secret'
    },
    scopeType: 'global',
    scopeEnvNames: []
  });
}

/**
 * Ensure an environment-scope VolumeSecret workspace component exists.
 *
 * @param {Object} options - Helper options.
 * @param {string} options.workspaceID - Workspace ID (required).
 * @param {string} options.resultVar - Variable name to cache component name (optional).
 * @param {string} options.name - Component name (required).
 * @param {Object} options.properties - Override properties (optional).
 * @param {string[]} options.scopeEnvNames - Scope environment names (optional).
 * @returns {Promise<{name: string}>}
 */
async function ensureEnvVolumeSecretWorkspaceComponent(options = {}) {
  if (!options.name) {
    throw new Error('Component name is required for environment scope');
  }

  return ensureWorkspaceComponent({
    workspaceID: options.workspaceID,
    resultVar: options.resultVar,
    name: options.name,
    type: 'VolumeSecret',
    properties: options.properties || {
      path: '/data/env-secret',
      secretName: 'test-env-secret'
    },
    scopeType: 'environment',
    scopeEnvNames: options.scopeEnvNames || ['test', 'staging']
  });
}

/**
 * 创建应用模型部署
 * 为指定的应用在特定环境中创建一个应用模型部署。
 * 需要指定镜像地址、副本数等部署配置。
 * 函数内部会检查该部署是否已创建，避免重复创建(key: appID+envName)。
 *
 * @param {Object} options - 部署配置参数
 * @param {string} options.appID - 应用 ID（必填）
 * @param {string} options.envName - 环境名称（必填）
 * @param {string} options.image - Docker 镜像地址（必填）
 * @param {number} options.replicas - 副本数量（默认为 1）
 * @param {Object} options.envVars - 环境变量配置（可选）
 * @param {Object} options.resources - 资源限制和请求配置（可选）
 *
 * @example
 * await createAppModelDeploy({
 *   appID: 'trpc-app-123',
 *   envName: 'test-env',
 *   replicas: 3,
 *   image: 'my.registry.com/app:v1.0.0'
 * });
 */
async function createAppModelDeploy(options = {}) {
  const serviceURL = bru.getEnvVar('serviceURL');
  const deployKey = `AppModelDeployCreated_${options.appID}-${options.envName}`;
  if (bru.getVar(deployKey)) {
    console.error(`AppModel deploy already created for app: ${options.appID} in env: ${options.envName}`);
    return
  }

  // Validate required parameters
  if (!options.appID) {
    throw new Error('Application ID is required');
  }
  if (!options.envName) {
    throw new Error('Environment name is required');
  }
  if (!options.image) {
    throw new Error('Docker image is required');
  }

  const body = {
    appID: options.appID,
    envName: options.envName,
    replicas: options.replicas || 1,
    image: options.image,
    creator: 'api-test-user'
  };


  const url = `${serviceURL}/apps/${options.appID}/envs/${options.envName}/appmodel-deploy`;

  try {
    const response = await axios.post(url, body, {
      headers: AUTH_HEADERS
    });

    if (response.status === 200) {
      console.log("AppModel deploy created successfully");
      console.log("Response data:", JSON.stringify(response.data, null, 2));

      // Save deploy info for later use
      if (response.data && Array.isArray(response.data) && response.data.length > 0) {
        const release = response.data[0];
        if (release.name) {
          console.log("Created release name:", release.name);
        }
        if (release.id) {
          console.log("Created release ID:", release.id);
        }
      }
      bru.setVar(deployKey, true);

      return response.data;
    }
  } catch (error) {
    console.error("=== AppModel Deploy Creation Error ===");
    console.error("Error:", error);
    console.error("Error response:", error.response?.data);

    throw error;
  }
}

module.exports = {
  AUTH_HEADERS,
  sleep,
  createAppConfigFile,
  updateAppConfigFileContent,
  createHelmApp,
  createWorkspace,
  createEnvironment,
  createTrpcApp,
  createTafApp,
  updateDefaultAppSpecResources,
  ensureWorkspaceComponent,
  ensureGlobalVolumeSecretWorkspaceComponent,
  ensureEnvVolumeSecretWorkspaceComponent,
  createAppModelDeploy,
  createTrpcDeploy: createAppModelDeploy  // backward compatibility
};
