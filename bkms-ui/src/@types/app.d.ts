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

import {
  type AppDeployedEnvOutputObj,
  type AppDetailOutputObj as AppDetailOutputObjV1,
  type AppInfoOutputObj as AppInfoOutputObjV1,
  type AppModelSpecOutputObj,
  type BuildConfigInput,
  type BuildConfigOutputObj,
  type ComponentOutputObj,
  type CreateAppRequest as CreateAppRequestV1,
  type GetAppIDAutoSuffixOutput,
  type HelmGitRepoConfigInput,
  type HelmRepoConfigInput,
  type HelmSourceInput,
  type HelmSpecInput,
  type HelmSpecOutputObj,
  type UpdateHelmSpecRequest as UpdateHelmSpecRequestV1,
  type VariableInput,
} from './v1/app';
import { type EnvVarOutputObj } from './v1/envvars';

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type AppDeployedEnvObj = AppDeployedEnvOutputObj;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type AppDetailOutputObj = AppDetailOutputObjV1;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type AppEnvVarOutputObj = EnvVarOutputObj;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type AppInfoOutputObj = AppInfoOutputObjV1;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export interface AppModelDeployRecordOutputObj {
  demo: string;
}

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type AppModelSpec = AppModelSpecOutputObj;
/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type AppModelSpecOutput = AppModelSpecOutputObj;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type BuildConfig = BuildConfigInput;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type BuildConfigOutput = BuildConfigOutputObj;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 * 事实上 `~/@types/v1` 并未提供对应 type，这里仍保留deprecated，待后续补充
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type Component = any;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type ComponentOutput = ComponentOutputObj;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type CreateAppRequest = CreateAppRequestV1;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type GetAppIDAutoSuffixResponse = GetAppIDAutoSuffixOutput;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type HelmGitRepoConfig = HelmGitRepoConfigInput;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type HelmRepoConfig = HelmRepoConfigInput;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type HelmSource = HelmSourceInput;
/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type HelmSpec = HelmSpecInput;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type HelmSpecOutput = HelmSpecOutputObj;
/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type UpdateHelmSpecRequest = UpdateHelmSpecRequestV1;
/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type Variable = VariableInput;
