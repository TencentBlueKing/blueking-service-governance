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

package customruntime

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// Store 工作空间自定义运行时镜像存储。
//
// 记录归属工作空间而非应用，因此不提供任何以 appID 为条件的具名 Opt
type Store = store.GenericStore[Image]

// Meta 自定义运行时镜像的存储元数据，写入完整文档前会校验记录并去除唯一键字段的首尾空白。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：workspaceID + type + name
var Meta = store.Meta[Image]{
	Collection: "custom_runtime_images",
	NotFound:   ErrCustomRuntimeImageNotFound,
	BeforeSave: []func(*Image) error{normalizeImage},
}

// normalizeImage 校验记录，并按 trim 后的值落库：后续查询的具名 Opt 也用同一口径，避免空白差异造成查不到
func normalizeImage(image *Image) error {
	if err := image.Validate(); err != nil {
		return err
	}
	image.WorkspaceID = strings.TrimSpace(image.WorkspaceID)
	image.Name = strings.TrimSpace(image.Name)
	return nil
}

// ByWorkspace 按所属工作空间匹配
func ByWorkspace(workspaceID string) store.Opt {
	return store.Where("workspaceID", strings.TrimSpace(workspaceID))
}

// ByType 按镜像类型匹配，imageType 为空时不过滤
func ByType(imageType ImageType) store.Opt {
	if imageType == "" {
		return nil
	}
	return store.Where("type", imageType)
}

// ByWorkspaceAndName 按工作空间与仓库名称匹配，不区分 builder / runner。
//
// tag 查询只拿到镜像名，而唯一键含类型，因此判断「是否已落库」时用它而不是 ByKey
func ByWorkspaceAndName(workspaceID, name string) store.Opt {
	return store.Combine(ByWorkspace(workspaceID), store.Where("name", strings.TrimSpace(name)))
}

// ByKey 按唯一键（工作空间 + 镜像类型 + 仓库名称）匹配
func ByKey(workspaceID string, imageType ImageType, name string) store.Opt {
	return store.Combine(ByWorkspaceAndName(workspaceID, name), store.Where("type", imageType))
}

// NameContains 按仓库名称模糊匹配（不区分大小写），keyword 为空白时不过滤
func NameContains(keyword string) store.Opt {
	return store.ContainsIgnoreCase(keyword, "name")
}

// InListOrder 列表的稳定排序：先按类型，再按名称
func InListOrder() store.Opt {
	return store.Combine(store.Asc("type"), store.Asc("name"))
}

// Register 幂等登记自定义运行时镜像：以工作空间 + 类型 + 名称为唯一键，
// 不存在时插入，已存在时只刷新 updatedAt，其余字段保持不变，调用方无需在写入前先查重
func Register(ctx context.Context, images *Store, image *Image) error {
	// 先规范化再构造过滤条件，保证过滤条件与落库值口径一致
	if err := normalizeImage(image); err != nil {
		return err
	}
	// 显式选中 updatedAt 时写入的是对象中的值，因此需要先赋值
	image.UpdatedAt = time.Now()
	_, err := images.Upsert(
		ctx, image, store.UpdateSelected("updatedAt"), ByKey(image.WorkspaceID, image.Type, image.Name),
	)
	if err != nil {
		return errors.Wrapf(err, "register custom runtime image %s/%s/%s", image.WorkspaceID, image.Type, image.Name)
	}
	return nil
}
