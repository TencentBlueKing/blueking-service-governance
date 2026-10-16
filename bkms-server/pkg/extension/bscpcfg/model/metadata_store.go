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

// Package model 定义了应用配置管理相关的纯数据模型。
package model

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// 错误定义
var (
	// ErrMetadataNotFound 配置管理元信息不存在
	ErrMetadataNotFound = errors.New("bscpcfg metadata not found")

	// ErrMetadataAlreadyExists 配置管理元信息已存在（同一 app 只能有一条记录）
	ErrMetadataAlreadyExists = errors.New("bscpcfg metadata already exists")
)

// MetadataStore Metadata 存储，一个 app 一条记录
type MetadataStore = store.GenericStore[Metadata]

// MetadataMeta Metadata 的存储元数据，写入完整文档前会校验必填字段。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID
var MetadataMeta = store.Meta[Metadata]{
	Collection: "bscpcfg_metadata",
	NotFound:   ErrMetadataNotFound,
	Duplicate:  ErrMetadataAlreadyExists,
	BeforeSave: []func(*Metadata) error{validateMetadata},
}

func validateMetadata(meta *Metadata) error {
	return errors.Wrap(validate.Struct(meta), "metadata validation failed")
}

// ByApp 按应用 ID 匹配，适用于 Metadata 与 EnvBinding
func ByApp(appID string) store.Opt {
	return store.Where("appID", appID)
}
