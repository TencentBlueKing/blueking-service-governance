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

// Package txcmdb provides api client to tx-cmdb
package txcmdb

import "context"

// noopClient 是 Tx CMDB Client 的默认空实现
//
// 在未配置可用 CMDB 服务时使用，单查返回 nil、批查返回空切片。上层 cmdb 聚合层
// 对空的二级业务明细做了容忍处理，不会中断创建 Workspace 等主流程。
type noopClient struct{}

// 编译期确认 noopClient 实现 Client 接口
var _ Client = noopClient{}

// newNoopClient 创建 noopClient
func newNoopClient() Client {
	return noopClient{}
}

// GetLevel2BusinessDetail 空实现，返回 nil 表示未查询到明细
func (noopClient) GetLevel2BusinessDetail(_ context.Context, _ int64) (*Level2BusinessDetail, error) {
	return nil, nil
}

// ListLevel2BusinessDetails 空实现，返回空切片
func (noopClient) ListLevel2BusinessDetails(_ context.Context, _ []int64) ([]Level2BusinessDetail, error) {
	return nil, nil
}
