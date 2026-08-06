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

import Fetch from '~/api/fetch';

const fetch = new Fetch({
  prefix: import.meta.env.BK_NODE_ENV === 'development' ? '' : `${import.meta.env.BK_BCS_API_BASE_URL}`,
});

// ===========bcs==========

// 获取bcs项目列表
export const getBcsProjects = fetch.get('/bcsapi/v4/bcsproject/v1/authorized_projects');
// 获取bcs项目详情
export const getProject = fetch.get('/bcsapi/v4/bcsproject/v1/projects/{projectId}');
// 获取bcs集群列表
export const getBcsCLusters = fetch.get('/bcsapi/v4/clustermanager/v1/projects/{projectId}/clusters');

// 获取集群命名空间
export const getNamespaceList = fetch.get(
  '/bcsapi/v4/bcsproject/v1/projects/{projectCode}/clusters/{clusterId}/namespaces',
);

// auth
export const userPerms = fetch.post('/bcsapi/v4/usermanager/v1/iam/user_perms');
export const userPermsByAction = fetch.post('/bcsapi/v4/usermanager/v1/iam/user_perms/actions/{actionId}');

// ===========devops==========

// 获取代码库列表
const repoFetch = new Fetch({
  prefix: import.meta.env.BK_NODE_ENV === 'development' ? '' : `${import.meta.env.BK_REPO_URL}`,
});
export const getGitProjects = repoFetch.get<any, any>('/ms/repository/api/user/git/getProject');
// 获取流水线列表
export const getPipelines = repoFetch.get<any, any>(
  '/ms/process/api/user/pipelines/bkms-{workspace}/hasPermissionList?permission=EXECUTE&limit=-1',
);
// 获取流水线参数
export const getPipeLineParams = repoFetch.get<any, any>(
  '/ms/process/api/user/builds/bkms-{workspace}/{pipelineId}/manualStartupInfo',
);
