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
  type AppInstanceOutputObj,
  type EventEntryOutputObj,
  type ExecuteTafAdminCmdOutput,
  type ExecuteTafAdminCmdRequest as ExecuteTafAdminCmdRequestV1,
  type ExecuteTrpcAdminCmdOutput,
  type ExecuteTrpcAdminCmdRequest as ExecuteTrpcAdminCmdRequestV1,
  type ListTrpcAdminCmdsOutput,
  type LogEntryOutputObj,
} from './v1/instance';

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type AppInstance = AppInstanceOutputObj;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type EventEntry = EventEntryOutputObj;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type ExecuteTafAdminCmdRequest = ExecuteTafAdminCmdRequestV1;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type ExecuteTafAdminCmdResponse = ExecuteTafAdminCmdOutput;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type ExecuteTrpcAdminCmdRequest = ExecuteTrpcAdminCmdRequestV1;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type ExecuteTrpcAdminCmdResponse = ExecuteTrpcAdminCmdOutput;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type InstanceExecuteTrpcAdminCmdResult = ExtractData<ExecuteTafAdminCmdResponse | ExecuteTrpcAdminCmdResponse>;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type ListTrpcAdminCmdsOutputObjs = ListTrpcAdminCmdsOutput;

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 */
export type LogEntry = LogEntryOutputObj;
