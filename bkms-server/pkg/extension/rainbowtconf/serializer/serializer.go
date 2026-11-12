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

// Package serializer defines Gin input and output serializers for rainbow tconf APIs.
package serializer

import (
	"time"

	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/rainbowtconf"
)

// AppURIInput binds the appID path parameter.
type AppURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required"`
}

// PutRainbowTConfInput is the request body for setting the rainbow tconf page of an app.
type PutRainbowTConfInput struct {
	// 七彩石页面地址，仅支持 https
	URL string `json:"url" binding:"required"`
}

// RainbowTConfOutput is the rainbow tconf page bound to an app.
type RainbowTConfOutput struct {
	// 七彩石页面地址，未配置时为空
	URL string `json:"url"`
	// 最近修改人，未配置时为空
	Updater string `json:"updater"`
	// 最近修改时间，未配置时为 null
	UpdatedAt *time.Time `json:"updatedAt"`
}

// NewRainbowTConfOutput builds output from the entry; nil means not configured.
func NewRainbowTConfOutput(e *rainbowtconf.Entry) *RainbowTConfOutput {
	if e == nil {
		return &RainbowTConfOutput{}
	}
	return &RainbowTConfOutput{URL: e.URL, Updater: e.Updater, UpdatedAt: lo.EmptyableToPtr(e.UpdatedAt)}
}

// RainbowTConfResponse is the JSON response carrying the rainbow tconf page of an app.
type RainbowTConfResponse struct {
	Data *RainbowTConfOutput `json:"data"`
}
