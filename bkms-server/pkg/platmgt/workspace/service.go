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

package workspace

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
)

// Service provides platform workspace query capabilities.
type Service struct {
	workspaceStore bkmsworkspace.WorkspaceStore
	appStore       bkmsapp.ApplicationStore
	envStore       envmodel.EnvironmentStore
}

// NewService creates a platform workspace service.
func NewService(
	workspaceStore bkmsworkspace.WorkspaceStore,
	appStore bkmsapp.ApplicationStore,
	envStore envmodel.EnvironmentStore,
) *Service {
	return &Service{
		workspaceStore: workspaceStore,
		appStore:       appStore,
		envStore:       envStore,
	}
}

// List returns workspaces without per-workspace permission filtering.
func (s *Service) List(ctx context.Context, opts WorkspaceListOptions) (*WorkspaceListResult, error) {
	workspaces, total, err := s.workspaceStore.ListWithPagination(ctx, opts.ToWorkspaceStoreListPageOptions())
	if err != nil {
		return nil, err
	}

	statistics, err := s.getStateStatistics(ctx, opts.ToWorkspaceStoreStatisticsOptions())
	if err != nil {
		return nil, errors.Wrap(err, "count workspace states by filter")
	}

	workspaceIDs := lo.Map(workspaces, func(ws bkmsworkspace.Workspace, _ int) string {
		return ws.ID
	})
	appCounts, err := s.appStore.CountByWorkspaceIDs(ctx, workspaceIDs)
	if err != nil {
		return nil, errors.Wrap(err, "count applications by workspace IDs")
	}
	envCounts, err := s.envStore.CountByWorkspaceIDs(ctx, workspaceIDs)
	if err != nil {
		return nil, errors.Wrap(err, "count environments by workspace IDs")
	}

	items := make([]WorkspaceWithStats, 0, len(workspaces))
	for _, ws := range workspaces {
		items = append(items, WorkspaceWithStats{
			ID:          ws.ID,
			DisplayName: ws.DisplayName,
			Description: ws.Description,
			State:       string(ws.State),
			Creator:     ws.Creator,
			Updater:     ws.Updater,
			UpdatedAt:   ws.UpdatedAt,
			AppCount:    appCounts[ws.ID],
			EnvCount:    envCounts[ws.ID],
		})
	}

	return &WorkspaceListResult{
		Count:      total,
		Page:       opts.Page,
		PageSize:   opts.PageSize,
		Results:    items,
		Statistics: *statistics,
	}, nil
}

// Get returns the basic info of a target workspace for platform administrators.
func (s *Service) Get(ctx context.Context, workspaceID string) (*WorkspaceInfo, error) {
	ws, err := s.workspaceStore.Get(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	return &WorkspaceInfo{
		ID:          ws.ID,
		DisplayName: ws.DisplayName,
		Description: ws.Description,
		State:       string(ws.State),
		Creator:     ws.Creator,
		CreatedAt:   ws.CreatedAt,
		Updater:     ws.Updater,
		UpdatedAt:   ws.UpdatedAt,
	}, nil
}

// GetStateStatistics returns aggregated workspace state counts for all workspaces.
func (s *Service) GetStateStatistics(ctx context.Context) (*WorkspaceStateStatistics, error) {
	return s.getStateStatistics(ctx, nil)
}

func (s *Service) getStateStatistics(
	ctx context.Context,
	opts *bkmsworkspace.ListOptions,
) (*WorkspaceStateStatistics, error) {
	counts, err := s.workspaceStore.CountByState(ctx, opts)
	if err != nil {
		return nil, err
	}

	readyCount := counts[bkmsworkspace.StateReady]
	processingCount := counts[bkmsworkspace.StateProcessing]
	disabledCount := counts[bkmsworkspace.StateDisabled]
	var totalCount int64
	for _, count := range counts {
		totalCount += count
	}

	return &WorkspaceStateStatistics{
		ReadyCount:      readyCount,
		ProcessingCount: processingCount,
		DisabledCount:   disabledCount,
		TotalCount:      totalCount,
	}, nil
}
