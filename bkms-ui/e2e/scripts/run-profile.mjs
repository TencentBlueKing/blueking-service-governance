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

import { spawnSync } from 'node:child_process';

const profile = process.argv[2] || 'all';
const separatorIndex = process.argv.indexOf('--');
const extraArgs = separatorIndex === -1 ? process.argv.slice(3) : process.argv.slice(separatorIndex + 1);

const profileTags = {
  all: '',
  smoke: '@smoke',
  deploy: '@deploy-flow',
  config: '@config-flow',
  readonly: '@readonly',
};

const pnpm = 'pnpm';
const env = { ...process.env };

function run(args) {
  const result = spawnSync(pnpm, args, {
    env,
    stdio: 'inherit',
    shell: process.platform === 'win32',
  });

  if (result.error) {
    console.error(result.error.message);
    process.exit(1);
  }

  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}

if (profile === 'spec') {
  run(['playwright', 'test', '--config=playwright.spec.config.ts', ...extraArgs]);
  process.exit(0);
}

if (!(profile in profileTags)) {
  console.error(`Unknown e2e profile: ${profile}`);
  console.error(`Available profiles: ${Object.keys(profileTags).concat('spec').join(', ')}`);
  process.exit(1);
}

if (profile === 'readonly') {
  env.E2E_PARALLEL = env.E2E_PARALLEL || 'true';
  env.E2E_WORKERS = env.E2E_WORKERS || '2';
}

const tags = profileTags[profile];
run(tags ? ['bddgen', 'test', '--tags', tags] : ['bddgen', 'test']);
run(['playwright', 'test', ...extraArgs]);
