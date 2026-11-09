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

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// DashboardDirectoryItem 仪表盘目录树中的仪表盘
type DashboardDirectoryItem struct {
	// Folder 所属目录标题
	Folder string `json:"folder" yaml:"folder"`
	// UID 仪表盘 uid
	UID string `json:"uid" yaml:"uid"`
	// Title 仪表盘标题
	Title string `json:"title" yaml:"title"`
	// URL 仪表盘访问 URL
	URL string `json:"url" yaml:"url" table:"-"`
}

// AppDashboard 应用绑定的仪表盘
type AppDashboard struct {
	// UID 仪表盘 uid
	UID string `json:"uid" yaml:"uid"`
	// Title 仪表盘标题
	Title string `json:"title" yaml:"title"`
	// URL 仪表盘访问 URL
	URL string `json:"url" yaml:"url" table:"-"`
}

// UpdateAppDashboardOptions 更新应用仪表盘绑定的选项
type UpdateAppDashboardOptions struct {
	// NewUID 新的仪表盘 uid
	NewUID string `json:"uid,omitempty"`
}

// ListAppDashboardsRespData 获取应用绑定仪表盘返回数据
type ListAppDashboardsRespData struct {
	Data []AppDashboard `json:"data"`
}

// dashboardDirectoryNode 目录树节点
type dashboardDirectoryNode struct {
	Dashboards []dashboardDirectoryRawItem `json:"dashboards"`
	Title      string                      `json:"title"`
}

// dashboardDirectoryRawItem 目录树中的仪表盘原始项（仅内部用于反序列化）
type dashboardDirectoryRawItem struct {
	UID   string `json:"uid"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// ListDashboardDirectoryTree 获取工作空间下的仪表盘目录树
func (c *SvcBasedClient) ListDashboardDirectoryTree(
	ctx context.Context,
	workspaceID string,
) ([]DashboardDirectoryItem, error) {
	path := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s/bkmonitor/dashboards", url.PathEscape(workspaceID))

	var respData struct {
		Data []dashboardDirectoryNode `json:"data"`
	}
	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, formatAPIError(resp, "list dashboard directory tree")
	}

	return flattenDashboardDirectoryTree(respData.Data), nil
}

// flattenDashboardDirectoryTree 将目录树拍平为仪表盘列表。
func flattenDashboardDirectoryTree(nodes []dashboardDirectoryNode) []DashboardDirectoryItem {
	items := make([]DashboardDirectoryItem, 0)
	for _, node := range nodes {
		for _, item := range node.Dashboards {
			items = append(items, DashboardDirectoryItem{
				Folder: node.Title,
				UID:    item.UID,
				Title:  item.Title,
				URL:    item.URL,
			})
		}
	}
	return items
}

// ListAppDashboards 获取应用绑定的仪表盘列表
func (c *SvcBasedClient) ListAppDashboards(ctx context.Context, appID string) ([]AppDashboard, error) {
	path := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/bkmonitor/dashboards", url.PathEscape(appID))

	var respData ListAppDashboardsRespData
	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, formatAPIError(resp, "list app dashboards")
	}

	return respData.Data, nil
}

// CreateAppDashboard 创建应用仪表盘绑定，标题由服务端根据 uid 解析。
func (c *SvcBasedClient) CreateAppDashboard(ctx context.Context, appID, uid string) error {
	path := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/bkmonitor/dashboards", url.PathEscape(appID))
	body := map[string]string{"uid": uid}

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Post(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return formatAPIError(resp, "create app dashboard")
	}

	return nil
}

// UpdateAppDashboard 更新应用仪表盘绑定
func (c *SvcBasedClient) UpdateAppDashboard(
	ctx context.Context,
	appID, uid string,
	opts UpdateAppDashboardOptions,
) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/bkmonitor/dashboards/%s",
		url.PathEscape(appID), url.PathEscape(uid),
	)

	resp, err := c.cli.R().SetContext(ctx).SetBody(opts).Put(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return formatAPIError(resp, "update app dashboard")
	}

	return nil
}

// DeleteAppDashboard 删除应用仪表盘绑定
func (c *SvcBasedClient) DeleteAppDashboard(ctx context.Context, appID, uid string) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/bkmonitor/dashboards/%s",
		url.PathEscape(appID), url.PathEscape(uid),
	)

	resp, err := c.cli.R().SetContext(ctx).Delete(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return formatAPIError(resp, "delete app dashboard")
	}

	return nil
}
