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
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	depsvcmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	depprovider "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// platformDepServicePlan 平台托管接入点 OpenAPI 地址所在的 depservice 方案
const platformDepServicePlan = "default"

// PolarisConfigRef 解析北极星配置所用接入点时需要的字段，由 polaris.PolarisConfig 实现。
// addon/polaris 依赖本包，因此这里不能直接引用 polaris.PolarisConfig
type PolarisConfigRef interface {
	// GetAppID 配置所属应用 ID
	GetAppID() string
	// GetAccessPointID 配置创建时固定的接入点 ID，存量配置为空
	GetAccessPointID() string
}

// Resolver 解析接入点，各调用链路只通过它获取接入信息。
//
// 涉及的模型：polaris_access_points、polaris_cluster_bindings、applications、depservice_services。
// 数据库查询失败时返回包装后的错误，不静默回退到平台托管接入点
type Resolver interface {
	// ResolveByID 按 ID 解析接入点：PlatformID 返回平台托管接入点，其余 ID 必须是该空间的自建接入点，
	// 否则返回 ErrAccessPointNotFound
	ResolveByID(ctx context.Context, workspaceID, accessPointID string) (*AccessPoint, error)
	// ResolveByCluster 解析集群使用的接入点：不允许空间自建接入点时所有集群使用平台托管接入点；
	// 否则按集群绑定解析，未绑定时返回 ErrClusterNotBound
	ResolveByCluster(ctx context.Context, workspaceID, clusterID string) (*AccessPoint, error)
	// ResolveForPolarisConfig 解析北极星配置创建时固定的接入点：接入点 ID 为空（存量数据）时按平台托管处理，
	// 其余按配置所属应用的空间调用 ResolveByID
	ResolveForPolarisConfig(ctx context.Context, cfg PolarisConfigRef) (*AccessPoint, error)
}

// NewResolver 创建接入点解析器
func NewResolver(db *mongo.Database) Resolver {
	return &resolver{db: db}
}

// PlatformAvailable 平台托管接入点是否可用：只在不允许空间自建接入点时可用，此时它是唯一的接入点，所有集群都使用它；
// 允许空间自建接入点时集群只能使用绑定的自建接入点，平台托管接入点不出现在接入点列表中，解析方法返回 ErrAccessPointNotFound
func PlatformAvailable() bool {
	return !config.G.Polaris.AllowWorkspaceAccessPoints
}

type resolver struct {
	db *mongo.Database
}

// ResolveByID 按 ID 解析接入点：PlatformID 返回平台托管接入点（不可用时返回 ErrAccessPointNotFound）；
// 其余 ID 在空间内查自建接入点，
// ID 不合法、不存在或不属于该空间时返回 ErrAccessPointNotFound
func (r *resolver) ResolveByID(ctx context.Context, workspaceID, accessPointID string) (*AccessPoint, error) {
	if accessPointID == PlatformID {
		return r.platform(ctx)
	}
	opt, err := ByWorkspaceAndID(workspaceID, accessPointID)
	if err != nil {
		return nil, err
	}
	p, err := store.Of(r.db, AccessPointMeta).Get(ctx, opt)
	if err != nil {
		return nil, errors.Wrapf(err, "get polaris access point %s in workspace %s", accessPointID, workspaceID)
	}
	return p.ToAccessPoint(), nil
}

// ResolveByCluster 解析集群使用的接入点：不允许空间自建接入点时不查绑定，直接返回平台托管接入点；
// 否则按空间内的集群绑定解析，未绑定时返回 ErrClusterNotBound，不回退到平台托管接入点
func (r *resolver) ResolveByCluster(ctx context.Context, workspaceID, clusterID string) (*AccessPoint, error) {
	if !config.G.Polaris.AllowWorkspaceAccessPoints {
		return r.platform(ctx)
	}
	binding, err := store.Of(r.db, BindingMeta).Get(ctx, ByCluster(workspaceID, clusterID))
	if errors.Is(err, ErrBindingNotFound) {
		return nil, errors.Wrapf(ErrClusterNotBound, "cluster %s in workspace %s", clusterID, workspaceID)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "get polaris binding of cluster %s in workspace %s", clusterID, workspaceID)
	}
	ap, err := r.ResolveByID(ctx, workspaceID, binding.AccessPointID)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve polaris access point bound to cluster %s", clusterID)
	}
	return ap, nil
}

// ResolveForPolarisConfig 解析北极星配置固定的接入点：接入点 ID 为空（内部版存量数据）或为 PlatformID 时返回平台托管接入点；
// 其余先查配置所属应用得到空间，再调用 ResolveByID
func (r *resolver) ResolveForPolarisConfig(ctx context.Context, cfg PolarisConfigRef) (*AccessPoint, error) {
	accessPointID := cfg.GetAccessPointID()
	if accessPointID == "" || accessPointID == PlatformID {
		return r.platform(ctx)
	}
	app, err := store.Of(r.db, bkmsapp.AppMeta).Get(ctx, bkmsapp.ByID(cfg.GetAppID()))
	if err != nil {
		return nil, errors.Wrapf(err, "get app %s of polaris config", cfg.GetAppID())
	}
	ap, err := r.ResolveByID(ctx, app.WorkspaceID, accessPointID)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve polaris access point of app %s", app.ID)
	}
	return ap, nil
}

// platform 由平台配置组装平台托管接入点：SDK 地址取自 config.Polaris，
// OpenAPI 地址取自 depservice polaris 服务的 default 方案，服务或方案不存在时为空。
// 平台托管接入点不可用（见 PlatformAvailable）时返回 ErrAccessPointNotFound
func (r *resolver) platform(ctx context.Context) (*AccessPoint, error) {
	if !PlatformAvailable() {
		return nil, errors.Wrap(ErrAccessPointNotFound, "platform polaris access point is not configured")
	}
	baseURL, err := r.platformOpenAPIBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	cfg := config.G.Polaris
	sdkAddresses := lo.Compact(lo.Map(strings.Split(cfg.Address, ","), func(addr string, _ int) string {
		return strings.TrimSpace(addr)
	}))
	return &AccessPoint{
		ID:             PlatformID,
		Kind:           KindPlatform,
		Name:           PlatformName,
		OpenAPIBaseURL: baseURL,
		JoinPoint:      cfg.JoinPoint,
		SDKAddresses:   sdkAddresses,
		Capabilities:   CapabilitiesOf(KindPlatform),
	}, nil
}

// platformOpenAPIBaseURL 读取 depservice polaris 服务 default 方案的 baseUrl，服务或方案不存在时返回空字符串
func (r *resolver) platformOpenAPIBaseURL(ctx context.Context) (string, error) {
	svc, err := store.Of(r.db, depsvcmodel.ServiceMeta).Get(ctx, depsvcmodel.ByName(depprovider.ServiceNamePolaris))
	if errors.Is(err, depsvcmodel.ErrServiceNotFound) {
		return "", nil
	}
	if err != nil {
		return "", errors.Wrap(err, "get polaris dependency service")
	}
	plan, ok := lo.Find(svc.Plans, func(p depsvcmodel.ServicePlan) bool { return p.Name == platformDepServicePlan })
	if !ok {
		return "", nil
	}
	return cast.ToString(plan.Config["baseUrl"]), nil
}
