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

import { type CustomTagOptsOutputObj, type RepoBuildConfigInput, type TagConfigOutputObj } from './v1/app';
import {
  type CreateBuildOutput,
  type CreateTafBuildDeployRequest,
  type CreateTrpcBuildDeployRequest,
} from './v1/build-autodeploy';
import {
  type BuildRecordOutputObj as BuildRecordOutputObjV1,
  type ImageBuildConfigInput,
  type UpdateBuildConfigRequest as UpdateBuildConfigRequestV1,
} from './v1/builds';
/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type BuildRecordOutputObj = BuildRecordOutputObjV1;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type CreateAppModelBuildDeployRequest = CreateTafBuildDeployRequest | CreateTrpcBuildDeployRequest;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type CreateBuildResponse = CreateBuildOutput;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type CustomTagOpts = CustomTagOptsOutputObj;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type ImageBuildConfig = ImageBuildConfigInput;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type RepositoryBuildConfig = RepoBuildConfigInput;

/** 构建相关 (占位) */
/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type TagConfig = TagConfigOutputObj;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type UpdateBuildConfigRequest = UpdateBuildConfigRequestV1;
