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

package accesspoint

import (
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// AccessPointStore 空间自建接入点存储
type AccessPointStore = store.GenericStore[PolarisAccessPoint]

// AccessPointMeta 空间自建接入点的存储元数据，Token 在库中对称加密存储。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：workspaceID + name，约束空间内接入点名称唯一，同时支撑按空间列出接入点
var AccessPointMeta = store.Meta[PolarisAccessPoint]{
	Collection:   "polaris_access_points",
	TenantScoped: true,
	NotFound:     ErrAccessPointNotFound,
	Duplicate:    ErrAccessPointNameExists,
	Encode:       []func(*PolarisAccessPoint) error{encryptToken},
	Decode:       []func(*PolarisAccessPoint) error{decryptToken},
}

// BindingStore 集群绑定存储
type BindingStore = store.GenericStore[PolarisClusterBinding]

// BindingMeta 集群绑定的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：workspaceID + clusterID，约束空间内一个集群至多绑定一个接入点；不做跨空间唯一
// - workspaceID + accessPointID：列出、统计接入点绑定的集群
var BindingMeta = store.Meta[PolarisClusterBinding]{
	Collection:   "polaris_cluster_bindings",
	TenantScoped: true,
	NotFound:     ErrBindingNotFound,
	Duplicate:    ErrBindingExists,
}

// InWorkspace 匹配空间内的文档，接入点与绑定集合通用
func InWorkspace(workspaceID string) store.Opt {
	return store.Where("workspaceID", workspaceID)
}

// ByWorkspaceAndID 匹配空间内指定 ID 的自建接入点，id 不是合法的 ObjectID 时返回 ErrAccessPointNotFound
func ByWorkspaceAndID(workspaceID, id string) (store.Opt, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.Wrapf(ErrAccessPointNotFound, "invalid polaris access point id %q", id)
	}
	return store.Combine(InWorkspace(workspaceID), store.ByID(objID)), nil
}

// ByCluster 匹配空间内指定集群的绑定
func ByCluster(workspaceID, clusterID string) store.Opt {
	return store.Combine(InWorkspace(workspaceID), store.Where("clusterID", clusterID))
}

// ByAccessPoint 匹配空间内指定接入点的绑定
func ByAccessPoint(workspaceID, accessPointID string) store.Opt {
	return store.Combine(InWorkspace(workspaceID), store.Where("accessPointID", accessPointID))
}

func encryptToken(p *PolarisAccessPoint) (err error) {
	p.Token, err = crypto.OnDemand(config.G.Encrypt.Secret, crypto.AESEncrypt, p.Token)
	return errors.Wrap(err, "encrypt polaris access point token")
}

func decryptToken(p *PolarisAccessPoint) (err error) {
	p.Token, err = crypto.OnDemand(config.G.Encrypt.Secret, crypto.AESDecrypt, p.Token)
	return errors.Wrap(err, "decrypt polaris access point token")
}
