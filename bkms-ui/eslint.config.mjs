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

import bkuiLint from '@blueking/bkui-lint/eslint.mjs';

// The bkui-lint preset enforces a license header belonging to another BlueKing project, so this
// project's own header is enforced instead. Keep the file globs aligned with the preset, since
// that is where the codecc plugin is registered.
const LICENSE_HEADER = `/*
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
`;

export default [
  ...bkuiLint,
  {
    files: ['*.js', '**/*.js', '*.ts', '**/*.ts', '*.tsx', '**/*.tsx'],
    rules: {
      'codecc/license': [
        'error',
        {
          license: LICENSE_HEADER,
          pattern: '.*TencentBlueKing is pleased to support the open source community.+',
        },
      ],
    },
  },
  {
    files: ['*.vue'],
    parser: 'vue-eslint-parser',
    parserOptions: {
      parser: '@typescript-eslint/parser',
      jsx: true,
    },
  },
  {
    rules: {
      '@typescript-eslint/member-ordering': 'off',
      'vue/multi-word-component-names': 'off',
    },
  },
  {
    ignores: ['src/api/modules/**/*.ts'],
  },
];
