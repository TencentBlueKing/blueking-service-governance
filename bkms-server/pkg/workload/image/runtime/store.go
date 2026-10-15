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

package runtime

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// Store 平台运行时镜像存储
type Store = store.GenericStore[Image]

// Meta 平台运行时镜像的存储元数据，写入完整文档前会去除名称首尾空白并校验各字段。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：type + name
var Meta = store.Meta[Image]{
	Collection: "runtime_images",
	NotFound:   ErrRuntimeImageNotFound,
	Duplicate:  ErrRuntimeImageAlreadyExists,
	BeforeSave: []func(*Image) error{normalizeImage},
}

// normalizeImage 校验镜像记录并去除名称首尾空白
func normalizeImage(image *Image) error {
	if err := image.Validate(); err != nil {
		return err
	}
	image.Name = strings.TrimSpace(image.Name)
	return nil
}

// ByType 按镜像类型匹配
func ByType(imageType ImageType) store.Opt {
	return store.Where("type", imageType)
}

// ByTypeAndName 按镜像类型和仓库名称匹配，名称会先去除首尾空白
func ByTypeAndName(imageType ImageType, name string) store.Opt {
	return store.Combine(ByType(imageType), store.Where("name", strings.TrimSpace(name)))
}

// KeywordMatches 按名称或描述模糊匹配（不区分大小写），keyword 为空白时不过滤
func KeywordMatches(keyword string) store.Opt {
	return store.ContainsIgnoreCase(keyword, "name", "description")
}

// UpdateDescription 按镜像类型和仓库名称更新运行时镜像描述，其余字段保持不变；
// 描述超长时返回校验错误，记录不存在时返回 ErrRuntimeImageNotFound
func UpdateDescription(ctx context.Context, images *Store, imageType ImageType, name, description string) error {
	if err := (&Image{Description: description}).validateDescription(); err != nil {
		return err
	}
	if err := images.Update(ctx, bson.M{"description": description}, ByTypeAndName(imageType, name)); err != nil {
		return errors.Wrapf(err, "update runtime image description %s/%s", imageType, name)
	}
	return nil
}
