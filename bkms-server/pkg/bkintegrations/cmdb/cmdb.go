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

// Package cmdb 提供 bkcc + txcmdb 查询能力
package cmdb

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cast"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkcc"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/txcmdb"
)

// ErrBusinessLevel2Missing 缺少二级业务关联
var ErrBusinessLevel2Missing = errors.New("business has no level-2 business (Level2BizID missing)")

// BusinessDetail 用户业务明细
type BusinessDetail struct {
	// BizID 业务 ID（bkcc）
	BizID string
	// BizName 业务名称（bkcc）
	BizName string

	// ObsProductID 运营产品 ID
	ObsProductID string
	// ObsProductName 运营产品名称
	ObsProductName string

	// Level1BizID txcmdb 一级业务 ID
	Level1BizID string
	// Level1BizName txcmdb 一级业务名称
	Level1BizName string
	// Level2BizID txcmdb 二级业务 ID
	Level2BizID string
	// Level2BizName txcmdb 二级业务名称
	Level2BizName string
}

// Service 聚合 bkcc + txcmdb 查询
type Service interface {
	// GetBusinessByID 按 bk_biz_id 查询有权限的业务信息
	GetBusinessByID(ctx context.Context, bizID int64) (*bkcc.Business, error)
	// ListBusinesses 查询 SRE 有权限的业务信息
	ListBusinesses(ctx context.Context) ([]bkcc.Business, error)
	// GetLevel2BusinessDetail 查询单个二级业务明细
	GetLevel2BusinessDetail(ctx context.Context, level2BizID int64) (*txcmdb.Level2BusinessDetail, error)
	// ListLevel2BusinessDetails 批量查询二级业务明细
	ListLevel2BusinessDetails(ctx context.Context, level2BizIDs []int64) ([]txcmdb.Level2BusinessDetail, error)

	// GetCMDBInfo 获取聚合信息
	GetCMDBInfo(ctx context.Context, bkCCBizID int64) (*BusinessDetail, error)
	// ListBusinessesWithLevel2Details 批量查询 SRE 有权限的业务信息(聚合 bkcc + txcmdb 信息)
	ListBusinessesWithLevel2Details(ctx context.Context) ([]BusinessDetail, error)
}

// cmdbService 聚合 bkcc + txcmdb 查询
type cmdbService struct {
	bk bkcc.Client

	tc txcmdb.Client
}

// NewService return Service
func NewService(user auth.User) (Service, error) {
	bkClient, err := bkcc.New(user)
	if err != nil {
		return nil, errors.Wrap(err, "initial bkcc client")
	}
	tcClient, err := txcmdb.New()
	if err != nil {
		return nil, errors.Wrap(err, "initial txcmdb client")
	}
	return &cmdbService{bk: bkClient, tc: tcClient}, nil
}

// GetBusinessByID 按 bk_biz_id 查询有权限的业务信息
func (s *cmdbService) GetBusinessByID(ctx context.Context, bizID int64) (*bkcc.Business, error) {
	return s.bk.GetBusinessByID(ctx, bizID)
}

// ListBusinesses 查询 SRE 有权限的业务信息
func (s *cmdbService) ListBusinesses(ctx context.Context) ([]bkcc.Business, error) {
	return s.bk.ListBusinesses(ctx)
}

// GetLevel2BusinessDetail 查询单个二级业务明细
func (s *cmdbService) GetLevel2BusinessDetail(
	ctx context.Context, level2BizID int64,
) (*txcmdb.Level2BusinessDetail, error) {
	return s.tc.GetLevel2BusinessDetail(ctx, level2BizID)
}

// ListLevel2BusinessDetails 批量查询二级业务明细
func (s *cmdbService) ListLevel2BusinessDetails(
	ctx context.Context, level2BizIDs []int64,
) ([]txcmdb.Level2BusinessDetail, error) {
	return s.tc.ListLevel2BusinessDetails(ctx, level2BizIDs)
}

// GetCMDBInfo 获取二级业务 ID、运营产品 ID、运营产品名称
func (s *cmdbService) GetCMDBInfo(ctx context.Context, bkCCBizID int64) (*BusinessDetail, error) {
	// 查询 bkcc 获取二级业务 ID
	biz, err := s.GetBusinessByID(ctx, bkCCBizID)
	if err != nil {
		return nil, errors.Wrapf(err, "get business from bkcc, bkCCBizID: %d", bkCCBizID)
	}
	level2BizID := cast.ToInt64(biz.Level2BizID)
	if level2BizID <= 0 {
		return nil, errors.Wrapf(ErrBusinessLevel2Missing, "bkCCBizID=%d", bkCCBizID)
	}

	// 获取运营产品信息
	detail, err := s.GetLevel2BusinessDetail(ctx, level2BizID)
	if err != nil {
		return nil, errors.Wrapf(err, "get business level2 details from txcmdb, level2BizID=%s", biz.Level2BizID)
	}
	// 未查询到二级业务明细（如 txcmdb noop 实现返回空）时，
	// 返回不含运营产品信息的业务明细，保证创建 Workspace 等主流程可继续
	if detail == nil {
		log.Warnf(ctx, "business level-2 detail not found, level2BizID=%s, bkCCBizID=%d", biz.Level2BizID, bkCCBizID)
		return &BusinessDetail{
			BizID:       cast.ToString(bkCCBizID),
			Level2BizID: biz.Level2BizID,
		}, nil
	}

	return &BusinessDetail{
		BizID:          cast.ToString(bkCCBizID),
		ObsProductName: detail.ObsProductName,
		ObsProductID:   detail.ObsProductID,
		Level2BizID:    detail.Level2BizID,
	}, nil
}

// ListBusinessesWithLevel2Details 批量查询 SRE 有权限的业务信息(聚合 bkcc + txcmdb 信息)
func (s *cmdbService) ListBusinessesWithLevel2Details(ctx context.Context) ([]BusinessDetail, error) {
	businesses, err := s.ListBusinesses(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "list user maintained businesses")
	}
	level2BizIDSet := lo.Uniq(lo.FilterMap(businesses, func(biz bkcc.Business, _ int) (int64, bool) {
		id := cast.ToInt64(biz.Level2BizID)
		return id, id > 0
	}))
	if len(level2BizIDSet) == 0 {
		return []BusinessDetail{}, nil
	}

	details, err := s.ListLevel2BusinessDetails(ctx, level2BizIDSet)
	if err != nil {
		return nil, errors.Wrap(err, "list txcmdb business level2 details")
	}
	detailMap := lo.SliceToMap(
		details,
		func(detail txcmdb.Level2BusinessDetail) (string, txcmdb.Level2BusinessDetail) {
			return detail.Level2BizID, detail
		},
	)

	results := lo.Map(businesses, func(biz bkcc.Business, _ int) BusinessDetail {
		detail, ok := detailMap[biz.Level2BizID]
		if !ok {
			// txcmdb 明细缺失时保留 bkcc 基础信息，避免授权业务列表被误过滤
			return BusinessDetail{
				BizID:       biz.BizID,
				BizName:     biz.BizName,
				Level2BizID: biz.Level2BizID,
			}
		}
		return BusinessDetail{
			BizID:          biz.BizID,
			BizName:        biz.BizName,
			Level1BizID:    detail.Level1BizID,
			Level1BizName:  detail.Level1BizName,
			Level2BizID:    detail.Level2BizID,
			Level2BizName:  detail.Level2BizName,
			ObsProductID:   detail.ObsProductID,
			ObsProductName: detail.ObsProductName,
		}
	})

	return results, nil
}
