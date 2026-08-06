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

import { createRequire } from 'node:module';
import { describe, expect, it } from 'vitest';

const require = createRequire(import.meta.url);

describe('OpenAPI v1 API 生成器', () => {
  it('生成包含详细注释的类型文件和 API 文件', () => {
    const { generateFromSwagger } = require('../scripts/openapi-v1/generator.cjs');

    const swagger = {
      swagger: '2.0',
      info: {
        title: 'bkms-server Gin API',
        version: '1.0',
      },
      basePath: '/bkms/v1/bkms-server',
      paths: {
        '/apps/{appID}/deploy-statuses': {
          get: {
            tags: ['app'],
            summary: '查询应用在各环境及各泳道上的部署状态',
            description: '返回默认泳道和自定义泳道的部署状态。',
            parameters: [
              {
                name: 'appID',
                in: 'path',
                required: true,
                type: 'string',
                description: '应用 ID',
              },
            ],
            responses: {
              200: {
                description: 'OK',
                schema: {
                  $ref: '#/definitions/serializer.GetAppDeployStatusesOutput',
                },
              },
            },
          },
        },
      },
      definitions: {
        'serializer.GetAppDeployStatusesOutput': {
          type: 'object',
          required: ['data'],
          properties: {
            data: {
              description: '应用部署状态列表',
              type: 'array',
              items: {
                $ref: '#/definitions/serializer.AppDeployedEnvOutputObj',
              },
            },
          },
        },
        'serializer.AppDeployedEnvOutputObj': {
          type: 'object',
          properties: {
            deployStatus: {
              description: '部署状态',
              type: 'string',
            },
          },
        },
      },
    };

    const files = generateFromSwagger(swagger);

    expect(files.typeFiles['app.d.ts']).toContain('export interface GetAppDeployStatusesOutput');
    expect(files.typeFiles['app.d.ts']).toContain('* 应用部署状态列表');
    expect(files.typeFiles['app.d.ts']).toContain('data: AppDeployedEnvOutputObj[];');
    expect(files.apiFiles['app.ts']).toContain("import type { NoInfer } from '~/api/ts-helpers';");
    expect(files.apiFiles['app.ts']).toContain("import { v1Fetch } from '~/api/clients';");
    expect(files.apiFiles['app.ts']).toContain('* 查询应用在各环境及各泳道上的部署状态');
    expect(files.apiFiles['app.ts']).toContain('* 返回默认泳道和自定义泳道的部署状态。');
    expect(files.apiFiles['app.ts']).toContain('* @path /apps/{appID}/deploy-statuses');
    expect(files.apiFiles['app.ts']).toContain('* @param appID path string required 应用 ID');
    expect(files.apiFiles['app.ts']).toContain(
      'getAppDeployStatuses: async <Request extends GetAppDeployStatusesRequest = GetAppDeployStatusesRequest',
    );
    expect(files.apiFiles['app.ts']).toContain('params?: NoInfer<Request>,');
    expect(files.apiFiles['app.ts']).toContain(
      "await v1Fetch.get<Request, ResponseData>('/apps/{appID}/deploy-statuses')",
    );
  });
});
