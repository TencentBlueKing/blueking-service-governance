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

package dbm

import (
	"context"
	"fmt"
	"sync"
)

// StubClient 用于本地开发的 DBM 模拟客户端
type StubClient struct {
	mu        sync.Mutex
	ticketSeq int
	// ticketID -> status
	tickets map[int]TicketStatus
}

// compile-time check
var _ Client = (*StubClient)(nil)

// NewStub 创建 Stub 客户端
func NewStub() Client {
	return &StubClient{
		ticketSeq: 1000,
		tickets:   make(map[int]TicketStatus),
	}
}

// CreateRedis simulates Redis creation and returns a completed ticket ID.
func (s *StubClient) CreateRedis(_ context.Context, params *CreateRedisParams, _ string) (int, error) {
	if params == nil {
		return 0, fmt.Errorf("params are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ticketSeq++
	s.tickets[s.ticketSeq] = TicketStatusSucceeded
	return s.ticketSeq, nil
}

// DisableRedis simulates Redis disabling and returns a completed ticket ID.
func (s *StubClient) DisableRedis(_ context.Context, params *DisableRedisParams, _ string) (int, error) {
	if params == nil {
		return 0, fmt.Errorf("params are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ticketSeq++
	s.tickets[s.ticketSeq] = TicketStatusSucceeded
	return s.ticketSeq, nil
}

// DeleteRedis simulates Redis deletion and returns a completed ticket ID.
func (s *StubClient) DeleteRedis(_ context.Context, params *DeleteRedisParams, _ string) (int, error) {
	if params == nil {
		return 0, fmt.Errorf("params are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ticketSeq++
	s.tickets[s.ticketSeq] = TicketStatusSucceeded
	return s.ticketSeq, nil
}

// GetTicketStatus returns the simulated status for a ticket.
func (s *StubClient) GetTicketStatus(_ context.Context, ticketID int, _ string) (*TicketInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status, ok := s.tickets[ticketID]
	if !ok {
		return nil, fmt.Errorf("ticket not found: %d", ticketID)
	}
	return &TicketInfo{ID: ticketID, Status: status}, nil
}

// FindClusterByName returns deterministic simulated cluster information for a name.
func (s *StubClient) FindClusterByName(
	_ context.Context,
	bkBizID int,
	clusterName string,
	_ ClusterType,
	_ string,
) (*ClusterInfo, error) {
	return &ClusterInfo{
		ID:          1,
		Domain:      fmt.Sprintf("%s.redis.db", clusterName),
		Port:        6379,
		Status:      "online",
		ClusterType: ClusterTypeTwemproxyRedis,
		BkBizID:     bkBizID,
	}, nil
}

// GetClusterInfo returns deterministic simulated cluster information for an ID.
func (s *StubClient) GetClusterInfo(_ context.Context, clusterID int, _ string) (*ClusterInfo, error) {
	return &ClusterInfo{
		ID:          clusterID,
		Domain:      "stub-cluster.redis.db",
		Port:        6379,
		Status:      "online",
		ClusterType: ClusterTypeTwemproxyRedis,
		BkBizID:     1,
	}, nil
}
