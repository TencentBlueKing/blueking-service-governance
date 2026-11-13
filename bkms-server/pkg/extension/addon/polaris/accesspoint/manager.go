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
	"net"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// maxNameLength 接入点名称的最大长度（按字符计）
const maxNameLength = 64

var hostValidator = validator.New()

// CreateParams 创建空间自建接入点的参数
type CreateParams struct {
	Name        string
	ConsoleURL  string
	ClientHosts []string
	// Token 选填
	Token string
}

// UpdateParams 更新空间自建接入点的参数，nil 表示不修改
type UpdateParams struct {
	Name        *string
	ConsoleURL  *string
	ClientHosts *[]string
	// Token 为空字符串表示清除
	Token *string
}

// ConnectivityParams 连通测试的参数
type ConnectivityParams struct {
	// ID 已有接入点的 ID，选填；有 ID 时记录测试结果
	ID          string
	ClientHosts []string
	// Token 为 nil 且有 ID 时使用已保存的 Token；为空字符串时不带 Token 测试
	Token *string
}

// ListResult 空间自建接入点列表
type ListResult struct {
	// AccessPoints 空间自建接入点，按创建时间排序
	AccessPoints []PolarisAccessPoint
	// BoundClusterCounts 接入点 ID 到绑定集群数量的映射
	BoundClusterCounts map[string]int
}

// Manager 空间北极星接入点的增删改查、连通测试与命名空间查询。
//
// 只处理空间自建接入点：接入点接口只在允许空间自建接入点时注册，此时平台托管接入点不可用，
// 传入 PlatformID 时与其他不存在的 ID 一样返回 ErrAccessPointNotFound。
//
// 涉及的模型：polaris_access_points、polaris_cluster_bindings，以及删除时由调用方传入的北极星配置查询
type Manager struct {
	db           *mongo.Database
	accessPoints *AccessPointStore
	bindings     *BindingStore
}

// NewManager 创建 Manager
func NewManager(db *mongo.Database) *Manager {
	return &Manager{
		db:           db,
		accessPoints: store.Of(db, AccessPointMeta),
		bindings:     store.Of(db, BindingMeta),
	}
}

// List 返回空间自建接入点以及各接入点绑定的集群数量
func (m *Manager) List(ctx context.Context, workspaceID string) (*ListResult, error) {
	aps, err := m.accessPoints.List(ctx, InWorkspace(workspaceID), store.Asc("createdAt"))
	if err != nil {
		return nil, errors.Wrapf(err, "list polaris access points of workspace %s", workspaceID)
	}
	bindings, err := m.bindings.List(ctx, InWorkspace(workspaceID), store.SelectFields("accessPointID"))
	if err != nil {
		return nil, errors.Wrapf(err, "list polaris cluster bindings of workspace %s", workspaceID)
	}
	return &ListResult{
		AccessPoints: aps,
		BoundClusterCounts: lo.CountValuesBy(bindings, func(b PolarisClusterBinding) string {
			return b.AccessPointID
		}),
	}, nil
}

// CountBoundClusters 返回空间内绑定到指定接入点的集群数量
func (m *Manager) CountBoundClusters(ctx context.Context, workspaceID, accessPointID string) (int, error) {
	n, err := m.bindings.Count(ctx, ByAccessPoint(workspaceID, accessPointID))
	if err != nil {
		return 0, errors.Wrapf(err, "count clusters bound to polaris access point %s", accessPointID)
	}
	return int(n), nil
}

// Create 创建空间自建接入点：校验参数与名称后重跑连通测试，未通过时返回 ErrConnectivityFailed
func (m *Manager) Create(ctx context.Context, workspaceID string, params CreateParams) (*PolarisAccessPoint, error) {
	p := &PolarisAccessPoint{WorkspaceID: workspaceID, Token: strings.TrimSpace(params.Token)}
	var err error
	if p.Name, err = normalizeName(params.Name); err != nil {
		return nil, err
	}
	if p.ConsoleURL, err = normalizeConsoleURL(params.ConsoleURL); err != nil {
		return nil, err
	}
	if p.ClientHosts, err = normalizeClientHosts(params.ClientHosts); err != nil {
		return nil, err
	}
	// 连通测试最长 10 秒，先排除名称重复；并发创建同名接入点仍由唯一索引兜底
	if err = m.ensureNameAvailable(ctx, workspaceID, p.Name, bson.NilObjectID); err != nil {
		return nil, err
	}
	if p.LastConnectivity, err = mustConnect(ctx, p.ClientHosts, p.Token); err != nil {
		return nil, err
	}

	username := auth.MustGetUser(ctx).ID
	p.Creator, p.Updater = username, username
	if err = m.accessPoints.Create(ctx, p); err != nil {
		return nil, errors.Wrapf(err, "create polaris access point %q in workspace %s", p.Name, workspaceID)
	}

	go audit.AddOperationRecordAsync(
		ctx, audit.OperationTypeCreate, audit.ResourceTypeWorkspace, workspaceID,
		audit.WithAttribute(audit.AttributePolarisAccessPoint),
		audit.WithDataAfter(buildAuditData(p)),
		audit.WithWorkspaceID(workspaceID),
	)
	return p, nil
}

// Update 更新空间自建接入点。客户端地址或 Token 变化时重跑连通测试，未通过时返回 ErrConnectivityFailed；
// 接入点被北极星配置使用时同样允许修改地址
func (m *Manager) Update(
	ctx context.Context, workspaceID, id string, params UpdateParams,
) (*PolarisAccessPoint, error) {
	before, err := m.get(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}

	after := *before
	if params.Name != nil {
		if after.Name, err = normalizeName(*params.Name); err != nil {
			return nil, err
		}
		if after.Name != before.Name {
			if err = m.ensureNameAvailable(ctx, workspaceID, after.Name, before.ID); err != nil {
				return nil, err
			}
		}
	}
	if params.ConsoleURL != nil {
		if after.ConsoleURL, err = normalizeConsoleURL(*params.ConsoleURL); err != nil {
			return nil, err
		}
	}
	if params.ClientHosts != nil {
		if after.ClientHosts, err = normalizeClientHosts(*params.ClientHosts); err != nil {
			return nil, err
		}
	}
	if params.Token != nil {
		after.Token = strings.TrimSpace(*params.Token)
	}
	if !slices.Equal(after.ClientHosts, before.ClientHosts) || after.Token != before.Token {
		if after.LastConnectivity, err = mustConnect(ctx, after.ClientHosts, after.Token); err != nil {
			return nil, err
		}
	}

	after.Updater = auth.MustGetUser(ctx).ID
	fields := store.UpdateSelected("name", "consoleURL", "clientHosts", "token", "lastConnectivity", "updater")
	if err = m.accessPoints.Update(ctx, &after, fields, store.ByID(before.ID)); err != nil {
		return nil, errors.Wrapf(err, "update polaris access point %s in workspace %s", id, workspaceID)
	}

	go audit.AddOperationRecordAsync(
		ctx, audit.OperationTypeUpdate, audit.ResourceTypeWorkspace, workspaceID,
		audit.WithAttribute(audit.AttributePolarisAccessPoint),
		audit.WithDataBefore(buildAuditData(before)),
		audit.WithDataAfter(buildAuditData(&after)),
		audit.WithWorkspaceID(workspaceID),
	)
	return &after, nil
}

// Delete 删除空间自建接入点。仍绑定集群、或仍有北极星配置固定使用它时返回 *AccessPointInUseError。
//
// listConfigUsages 查询固定使用该接入点的北极星配置，见 ConfigUsageLister
func (m *Manager) Delete(ctx context.Context, workspaceID, id string, listConfigUsages ConfigUsageLister) error {
	p, err := m.get(ctx, workspaceID, id)
	if err != nil {
		return err
	}

	bindings, err := m.bindings.List(ctx, ByAccessPoint(workspaceID, id), store.Asc("clusterID"))
	if err != nil {
		return errors.Wrapf(err, "list clusters bound to polaris access point %s", id)
	}
	usages, err := listConfigUsages(ctx, m.db, id)
	if err != nil {
		return errors.Wrapf(err, "list polaris configs using access point %s", id)
	}
	if len(bindings) > 0 || len(usages) > 0 {
		return &AccessPointInUseError{
			ClusterIDs: lo.Map(bindings, func(b PolarisClusterBinding, _ int) string { return b.ClusterID }),
			Configs:    usages,
		}
	}

	if err = m.accessPoints.Delete(ctx, store.ByID(p.ID)); err != nil {
		return errors.Wrapf(err, "delete polaris access point %s in workspace %s", id, workspaceID)
	}

	go audit.AddOperationRecordAsync(
		ctx, audit.OperationTypeDelete, audit.ResourceTypeWorkspace, workspaceID,
		audit.WithAttribute(audit.AttributePolarisAccessPoint),
		audit.WithDataBefore(buildAuditData(p)),
		audit.WithWorkspaceID(workspaceID),
	)
	return nil
}

// TestConnectivity 测试接入地址的连通性。连通测试未通过不是错误，结果在返回值中；
// 传入已有接入点 ID 时把结果记录到该接入点上
func (m *Manager) TestConnectivity(
	ctx context.Context, workspaceID string, params ConnectivityParams,
) (*ConnectivityReport, error) {
	hosts, err := normalizeClientHosts(params.ClientHosts)
	if err != nil {
		return nil, err
	}

	var existing *PolarisAccessPoint
	if params.ID != "" {
		if existing, err = m.get(ctx, workspaceID, params.ID); err != nil {
			return nil, err
		}
	}
	var token string
	switch {
	case params.Token != nil:
		token = strings.TrimSpace(*params.Token)
	case existing != nil:
		token = existing.Token
	}

	report := probeConnectivity(ctx, OpenAPIBaseURL(hosts[0]), token)
	if existing != nil {
		changes := map[string]any{"lastConnectivity": &report.ConnectivityResult}
		if err = m.accessPoints.Update(ctx, changes, store.ByID(existing.ID), store.KeepUpdatedAt()); err != nil {
			return nil, errors.Wrapf(err, "record connectivity of polaris access point %s", params.ID)
		}
	}
	return report, nil
}

// ListNamespaces 实时读取空间自建接入点上的北极星命名空间并过滤系统命名空间，读取失败时返回 ErrAccessPointUnreachable
func (m *Manager) ListNamespaces(ctx context.Context, workspaceID, id string) ([]string, error) {
	p, err := m.get(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, connectivityTimeout)
	defer cancel()
	namespaces, err := newOpenAPIClient(OpenAPIBaseURL(p.ClientHosts[0]), p.Token).listNamespaces(ctx)
	if err != nil {
		return nil, errors.Wrapf(ErrAccessPointUnreachable, "polaris access point %q: %s", p.Name, describeErr(err))
	}
	return namespaces, nil
}

// get 取空间内的自建接入点，平台托管接入点或其他空间的接入点返回 ErrAccessPointNotFound
func (m *Manager) get(ctx context.Context, workspaceID, id string) (*PolarisAccessPoint, error) {
	opt, err := ByWorkspaceAndID(workspaceID, id)
	if err != nil {
		return nil, err
	}
	p, err := m.accessPoints.Get(ctx, opt)
	if err != nil {
		return nil, errors.Wrapf(err, "get polaris access point %s in workspace %s", id, workspaceID)
	}
	return p, nil
}

// ensureNameAvailable 检查空间内除 excludeID 外没有同名接入点，否则返回 ErrAccessPointNameExists
func (m *Manager) ensureNameAvailable(ctx context.Context, workspaceID, name string, excludeID bson.ObjectID) error {
	exists, err := m.accessPoints.Exists(
		ctx,
		InWorkspace(workspaceID),
		store.Where("name", name),
		store.Filter(bson.M{"_id": bson.M{"$ne": excludeID}}),
	)
	if err != nil {
		return errors.Wrapf(err, "check polaris access point name %q in workspace %s", name, workspaceID)
	}
	if exists {
		return errors.Wrapf(ErrAccessPointNameExists, "name %q", name)
	}
	return nil
}

// mustConnect 保存前重跑连通测试，未通过时返回 ErrConnectivityFailed，失败原因写在错误信息里
func mustConnect(ctx context.Context, hosts []string, token string) (*ConnectivityResult, error) {
	report := probeConnectivity(ctx, OpenAPIBaseURL(hosts[0]), token)
	if !report.OK {
		return nil, errors.Wrapf(ErrConnectivityFailed, "reason %s, %s", report.Reason, report.Message)
	}
	return &report.ConnectivityResult, nil
}

func normalizeName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" || utf8.RuneCountInString(name) > maxNameLength {
		return "", errors.Wrapf(ErrAccessPointInvalid, "name must be 1-%d characters", maxNameLength)
	}
	return name, nil
}

// normalizeConsoleURL 控制台地址可以是 IP:Port 或域名，缺省协议时补 http://，去掉末尾的 /
func normalizeConsoleURL(raw string) (string, error) {
	s := strings.TrimRight(strings.TrimSpace(raw), "/")
	if s != "" && !strings.Contains(s, "://") {
		s = "http://" + s
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return "", errors.Wrapf(ErrAccessPointInvalid, "invalid console url %q", raw)
	}
	return s, nil
}

// normalizeClientHosts 客户端地址只能是 IP 或域名（不带端口与协议），去重后数量为 1 ~ MaxClientHosts
func normalizeClientHosts(raw []string) ([]string, error) {
	hosts := lo.Uniq(lo.Compact(lo.Map(raw, func(h string, _ int) string { return strings.TrimSpace(h) })))
	if len(hosts) == 0 || len(hosts) > MaxClientHosts {
		return nil, errors.Wrapf(ErrAccessPointInvalid, "client hosts must be 1-%d addresses", MaxClientHosts)
	}
	for _, host := range hosts {
		if net.ParseIP(host) == nil && hostValidator.Var(host, "hostname_rfc1123") != nil {
			return nil, errors.Wrapf(ErrAccessPointInvalid, "client host %q must be an IP or a domain name", host)
		}
	}
	return hosts, nil
}

// buildAuditData 审计记录接入点的地址等信息，不写入 Token
func buildAuditData(p *PolarisAccessPoint) map[string]any {
	return map[string]any{
		"id":          p.ID.Hex(),
		"name":        p.Name,
		"consoleURL":  p.ConsoleURL,
		"clientHosts": p.ClientHosts,
		"hasToken":    p.Token != "",
	}
}
