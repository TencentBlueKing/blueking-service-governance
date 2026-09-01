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

import { ResourceSpecSchema } from '../data/resource-spec-data';
import { TrpcFormSchema } from '../data/trpc-data';

import type { ResourceSpecFormData } from '../data/resource-spec-data';
import type { TrpcFormData } from '../data/trpc-data';
import type BasePage from '../pages/base.page';
import type { Schema } from '../utils/form';

export type FormType = 'ResourceSpec' | 'TRPC';
type FormDataMap = {
  ResourceSpec: ResourceSpecFormData;
  TRPC: TrpcFormData;
};

const schemaMap: { [K in FormType]: Schema<FormDataMap[K]> } = {
  TRPC: TrpcFormSchema,
  ResourceSpec: ResourceSpecSchema,
};

export async function fillFormByType<K extends FormType>(
  basePage: BasePage,
  formType: K,
  data: Partial<FormDataMap[K]>,
) {
  const schema = schemaMap[formType];

  if (!schema) {
    throw new Error(`未知表单类型: ${formType}`);
  }

  await basePage.fillForm(schema, data);
}
