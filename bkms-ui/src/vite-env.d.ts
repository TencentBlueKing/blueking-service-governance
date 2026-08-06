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

/// <reference types="vite/client" />

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

interface ImportMetaEnv {
  readonly BK_API_BASE_URL: string;
  readonly BK_API_PREFIX: string;
  readonly BK_API_URL_TMPL: string;
  readonly BK_APP_HOST: string;
  readonly BK_APP_PORT: string;
  readonly BK_BSCP_URL: string;
  readonly BK_DEVOPS: string;
  readonly BK_DOC_URL: string;
  readonly BK_LOGIN_URL: string;
  readonly BK_MONITOR: string;
  readonly BK_NODE_ENV: string;
  readonly BK_POLARIS_URL: string;
  readonly BK_SITE_URL: string;
  readonly BK_STATIC_URL: string;
}
