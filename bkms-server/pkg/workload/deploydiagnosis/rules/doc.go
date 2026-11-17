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

// Package rules 存放部署诊断的规则与诊断结论的领域模型
//
// 输入是采集层组装好的 Snapshot（最新部署、实例、日志、事件、框架配置），规则只读快照，不访问存储或集群
//
// 规则分两类：
//   - InstanceRule 针对单个异常实例，规则集按顺序尝试，第一条命中即停；UnknownRule 恒命中，必须放最后
//   - StageRule 只在没有异常实例时运行，处理部署 / 构建已失败但实例看不出问题的情况
//
// 命中的规则输出 Group：出错原因、处理步骤、跳转按钮，以及命中的实例与证据
//
// 面向用户的文案集中在 messages.go，调整措辞或接入国际化时只改那一处
package rules
