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

const fs = require('fs');
const path = require('path');

const { generateFromSwagger } = require('./openapi-v1/generator.cjs');

const swaggerPath = path.resolve(__dirname, '../../bkms-server/docs/apis/swagger.json');
const apiOutputDir = path.resolve(__dirname, '../src/api/modules/v1');
const typeOutputDir = path.resolve(__dirname, '../src/@types/v1');

function writeFiles(outputDir, files) {
  fs.mkdirSync(outputDir, { recursive: true });

  for (const [fileName, content] of Object.entries(files)) {
    fs.writeFileSync(path.join(outputDir, fileName), content);
  }
}

function main() {
  const swagger = JSON.parse(fs.readFileSync(swaggerPath, 'utf-8'));
  const files = generateFromSwagger(swagger);

  writeFiles(apiOutputDir, files.apiFiles);
  writeFiles(typeOutputDir, files.typeFiles);

  console.log(`v1 API 文件已生成：${apiOutputDir}`);
  console.log(`v1 类型文件已生成：${typeOutputDir}`);
}

main();
