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

package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkhcm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
)

// ListBkHCMRegions 查询 bk-hcm 地域列表
//
//	@ID			ListBkHCMRegions
//	@Summary	查询云地域列表
//	@Tags		bkintegrations-bkhcm
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		body	body		serializer.BkHCMListInput	true	"查询参数"
//	@Success	200		{object}	serializer.ListBkHCMRegionsOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/bkhcm/regions [post]
func (h *Handler) ListBkHCMRegions(c *gin.Context) {
	var bodyInput slz.BkHCMListInput
	if err := ginutils.BindJSON(c, &bodyInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	client, err := bkhcm.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bkhcm client"))
		return
	}

	regions, err := client.ListRegions(ctx, bodyInput.Filter, bodyInput.Page)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list bkhcm regions"))
		return
	}

	ginutils.OK(c, &slz.ListBkHCMRegionsOutput{
		Data: lo.Map(regions, func(item bkhcm.Region, _ int) *slz.RegionOutput {
			return &slz.RegionOutput{
				ID:         item.ID,
				Vendor:     item.Vendor,
				RegionID:   item.RegionID,
				RegionName: item.RegionName,
				Status:     item.Status,
			}
		}),
	})
}

// ListBkHCMSubnets 查询 bk-hcm 子网列表
//
//	@ID			ListBkHCMSubnets
//	@Summary	查询子网列表
//	@Tags		bkintegrations-bkhcm
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		bkBizID	path		int64				true	"业务 ID"
//	@Param		body	body		serializer.BkHCMListInput	true	"查询参数"
//	@Success	200		{object}	serializer.ListBkHCMSubnetsOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/bkhcm/bizs/{bkBizID}/subnets [post]
func (h *Handler) ListBkHCMSubnets(c *gin.Context) {
	var uriInput slz.BkHCMBizURIInput
	var bodyInput slz.BkHCMListInput
	if err := ginutils.BindURIJSON(c, &uriInput, &bodyInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	client, err := bkhcm.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bkhcm client"))
		return
	}

	subnets, err := client.ListSubnets(ctx, uriInput.BkBizID, bodyInput.Filter, bodyInput.Page)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list bkhcm subnets"))
		return
	}

	ginutils.OK(c, &slz.ListBkHCMSubnetsOutput{
		Data: lo.Map(subnets, func(item bkhcm.Subnet, _ int) *slz.SubnetOutput {
			return &slz.SubnetOutput{
				ID:         item.ID,
				Vendor:     item.Vendor,
				AccountID:  item.AccountID,
				CloudVpcID: item.CloudVpcID,
				CloudID:    item.CloudID,
				Name:       item.Name,
				Region:     item.Region,
				Zone:       item.Zone,
				Ipv4Cidr:   item.Ipv4Cidr,
				Ipv6Cidr:   item.Ipv6Cidr,
				Memo:       item.Memo,
				VpcID:      item.VpcID,
				BkBizID:    item.BkBizID,
			}
		}),
	})
}

// ListBkHCMVPCs 查询 bk-hcm VPC 列表
//
//	@ID			ListBkHCMVPCs
//	@Summary	查询 VPC 列表
//	@Tags		bkintegrations-bkhcm
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		bkBizID	path		int64				true	"业务 ID"
//	@Param		body	body		serializer.BkHCMListInput	true	"查询参数"
//	@Success	200		{object}	serializer.ListBkHCMVPCsOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/bkhcm/bizs/{bkBizID}/vpcs [post]
func (h *Handler) ListBkHCMVPCs(c *gin.Context) {
	var uriInput slz.BkHCMBizURIInput
	var bodyInput slz.BkHCMListInput
	if err := ginutils.BindURIJSON(c, &uriInput, &bodyInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	client, err := bkhcm.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bkhcm client"))
		return
	}

	vpcs, err := client.ListVPCs(ctx, uriInput.BkBizID, bodyInput.Filter, bodyInput.Page)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list bkhcm vpcs"))
		return
	}

	ginutils.OK(c, &slz.ListBkHCMVPCsOutput{
		Data: lo.Map(vpcs, func(item bkhcm.VPC, _ int) *slz.VPCOutput {
			return &slz.VPCOutput{
				ID:        item.ID,
				Vendor:    item.Vendor,
				AccountID: item.AccountID,
				CloudID:   item.CloudID,
				Name:      item.Name,
				Region:    item.Region,
				Category:  item.Category,
				Memo:      item.Memo,
				BkBizID:   item.BkBizID,
				Extension: item.Extension,
			}
		}),
	})
}

// ListBkHCMZones 查询 bk-hcm 可用区列表
//
//	@ID			ListBkHCMZones
//	@Summary	查询可用区列表
//	@Tags		bkintegrations-bkhcm
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		region	path		string				true	"地域 ID"
//	@Param		body	body		serializer.BkHCMListInput	true	"查询参数"
//	@Success	200		{object}	serializer.ListBkHCMZonesOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/bkhcm/regions/{region}/zones [post]
func (h *Handler) ListBkHCMZones(c *gin.Context) {
	var uriInput slz.BkHCMRegionURIInput
	var bodyInput slz.BkHCMListInput
	if err := ginutils.BindURIJSON(c, &uriInput, &bodyInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	client, err := bkhcm.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bkhcm client"))
		return
	}

	zones, err := client.ListZones(ctx, uriInput.Region, bodyInput.Filter, bodyInput.Page)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list bkhcm zones"))
		return
	}

	ginutils.OK(c, &slz.ListBkHCMZonesOutput{
		Data: lo.Map(zones, func(item bkhcm.Zone, _ int) *slz.ZoneOutput {
			return &slz.ZoneOutput{
				ID:      item.ID,
				Vendor:  item.Vendor,
				CloudID: item.CloudID,
				Name:    item.Name,
				NameCN:  item.NameCN,
				Region:  item.Region,
				State:   item.State,
			}
		}),
	})
}

// CreateBkHCMLoadBalancerApplication 创建负载均衡申请
//
//	@ID			CreateBkHCMLoadBalancerApplication
//	@Summary	创建负载均衡申请
//	@Tags		bkintegrations-bkhcm
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		body	body		serializer.BkHCMCreateLoadBalancerInput	true	"创建负载均衡参数"
//	@Success	200		{object}	serializer.CreateBkHCMLoadBalancerOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/bkhcm/load-balancers [post]
func (h *Handler) CreateBkHCMLoadBalancerApplication(c *gin.Context) {
	var bodyInput slz.BkHCMCreateLoadBalancerInput
	if err := ginutils.BindJSON(c, &bodyInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	client, err := bkhcm.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bkhcm client"))
		return
	}

	applicationID, err := client.CreateBizApplicationForCreateLoadBalancer(ctx, bodyInput.ToReq())
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create load balancer application"))
		return
	}

	ginutils.OK(c, &slz.CreateBkHCMLoadBalancerOutput{
		Data: &slz.CreateBkHCMLoadBalancerData{ID: applicationID},
	})
}
