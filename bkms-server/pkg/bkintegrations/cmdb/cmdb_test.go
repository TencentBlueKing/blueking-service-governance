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

package cmdb

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkcc"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/txcmdb"
)

type fakeBKCCClient struct {
	businesses []bkcc.Business
}

func (f fakeBKCCClient) ListBusinesses(context.Context) ([]bkcc.Business, error) {
	return f.businesses, nil
}

func (f fakeBKCCClient) GetBusinessByID(context.Context, int64) (*bkcc.Business, error) {
	return nil, bkcc.ErrBusinessNotFound
}

type fakeTxCMDBClient struct {
	details []txcmdb.Level2BusinessDetail
	// queried 记录批量查询收到的二级业务 ID，nil 表示未发起查询
	queried *[]int64
}

func (f fakeTxCMDBClient) GetLevel2BusinessDetail(context.Context, int64) (*txcmdb.Level2BusinessDetail, error) {
	return nil, nil
}

func (f fakeTxCMDBClient) ListLevel2BusinessDetails(
	_ context.Context, level2BizIDs []int64,
) ([]txcmdb.Level2BusinessDetail, error) {
	*f.queried = level2BizIDs
	return f.details, nil
}

var _ = Describe("ListBusinessesWithLevel2Details", func() {
	var queried []int64

	BeforeEach(func() {
		queried = nil
	})

	newService := func(businesses []bkcc.Business, details []txcmdb.Level2BusinessDetail) Service {
		return &cmdbService{
			bk: fakeBKCCClient{businesses: businesses},
			tc: fakeTxCMDBClient{details: details, queried: &queried},
		}
	}

	It("keeps bkcc businesses when none of them has a level-2 business", func() {
		// 社区版 CMDB 没有 bs2_name_id 字段，bkcc client 解析为 "0"
		svc := newService([]bkcc.Business{
			{BizID: "2", BizName: "biz-a", Level2BizID: "0"},
			{BizID: "3", BizName: "biz-b", Level2BizID: "0"},
		}, nil)

		results, err := svc.ListBusinessesWithLevel2Details(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(Equal([]BusinessDetail{
			{BizID: "2", BizName: "biz-a"},
			{BizID: "3", BizName: "biz-b"},
		}))
		Expect(queried).To(BeNil())
	})

	It("fills level-2 details only for businesses that have a level-2 business", func() {
		svc := newService(
			[]bkcc.Business{
				{BizID: "2", BizName: "biz-a", Level2BizID: "2001"},
				{BizID: "3", BizName: "biz-b", Level2BizID: "0"},
				{BizID: "4", BizName: "biz-c", Level2BizID: "2002"},
			},
			[]txcmdb.Level2BusinessDetail{
				{
					Level1BizID: "1001", Level1BizName: "l1", Level2BizID: "2001", Level2BizName: "l2",
					ObsProductID: "p1", ObsProductName: "product",
				},
			},
		)

		results, err := svc.ListBusinessesWithLevel2Details(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(queried).To(ConsistOf(int64(2001), int64(2002)))
		Expect(results).To(Equal([]BusinessDetail{
			{
				BizID: "2", BizName: "biz-a", Level1BizID: "1001", Level1BizName: "l1",
				Level2BizID: "2001", Level2BizName: "l2", ObsProductID: "p1", ObsProductName: "product",
			},
			{BizID: "3", BizName: "biz-b"},
			// txcmdb 未返回明细时保留 bkcc 基础信息与二级业务 ID
			{BizID: "4", BizName: "biz-c", Level2BizID: "2002"},
		}))
	})

	It("returns an empty list when the user maintains no business", func() {
		results, err := newService(nil, nil).ListBusinessesWithLevel2Details(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(BeEmpty())
	})
})
