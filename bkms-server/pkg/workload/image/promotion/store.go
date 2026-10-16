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

package promotion

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// Store 镜像晋级记录存储
type Store = store.GenericStore[Image]

// Meta 镜像晋级记录的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID + repoKey + tag
var Meta = store.Meta[Image]{Collection: "image_promotions"}

// ByApp 按应用 ID 匹配晋级记录
func ByApp(appID string) store.Opt {
	return store.Where("appID", appID)
}

// ByAppAndRepoKey 按应用 ID 与仓库实例唯一标识匹配晋级记录
func ByAppAndRepoKey(appID, repoKey string) store.Opt {
	return store.Combine(ByApp(appID), store.Where("repoKey", repoKey))
}

// ByTag 精确匹配应用下指定仓库实例中某个镜像标签的晋级记录（至多一条）
func ByTag(appID, repoKey, tag string) store.Opt {
	return store.Combine(ByAppAndRepoKey(appID, repoKey), store.Where("tag", tag))
}

// Promote 晋级镜像标签：以 appID + repoKey + tag 为条件幂等 upsert，
// 重复晋级时会更新 promotedBy / promotedAt 为最后一次操作的信息
func Promote(ctx context.Context, promotions *Store, appID, repoKey, tag, operator string) error {
	img := &Image{AppID: appID, RepoKey: repoKey, Tag: tag, PromotedAt: time.Now(), PromotedBy: operator}
	if _, err := promotions.Upsert(ctx, img, ByTag(appID, repoKey, tag)); err != nil {
		return errors.Wrapf(err, "upsert promotion for app %s, repo %s, tag %s", appID, repoKey, tag)
	}
	return nil
}

// ListPromotedTags 返回应用下指定仓库实例中所有已晋级的 tag
func ListPromotedTags(ctx context.Context, promotions *Store, appID, repoKey string) ([]string, error) {
	images, err := promotions.List(ctx, ByAppAndRepoKey(appID, repoKey))
	if err != nil {
		return nil, errors.Wrapf(err, "list promotions for app %s, repo %s", appID, repoKey)
	}
	return lo.Map(images, func(img Image, _ int) string { return img.Tag }), nil
}
