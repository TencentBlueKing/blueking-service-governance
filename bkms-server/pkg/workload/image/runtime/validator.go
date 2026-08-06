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

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// ErrRuntimeImageTagNotFound 运行时镜像快照 tag 不存在
var ErrRuntimeImageTagNotFound = errors.New("runtime image tag not found")

// ImageReferenceValidator 校验运行时镜像引用与快照 tag 是否存在
type ImageReferenceValidator struct {
	runtimeImageStore Store
	snapshotStore     snapshot.SnapshotStore
}

// NewImageReferenceValidator 创建运行时镜像引用校验器
func NewImageReferenceValidator(
	runtimeImageStore Store,
	snapshotStore snapshot.SnapshotStore,
) *ImageReferenceValidator {
	return &ImageReferenceValidator{
		runtimeImageStore: runtimeImageStore,
		snapshotStore:     snapshotStore,
	}
}

// ValidateTaggedReference 校验完整镜像引用是否属于指定类型的运行时镜像且 tag 已存在于快照中
func (v *ImageReferenceValidator) ValidateTaggedReference(
	ctx context.Context,
	imageType ImageType,
	image string,
) (*ImageReference, error) {
	ref, err := ParseTaggedImageReference(image)
	if err != nil {
		return nil, err
	}

	if v == nil {
		return ref, errors.New("image reference validator is nil")
	}
	if v.runtimeImageStore == nil {
		return ref, errors.New("runtime image store is nil")
	}
	if v.snapshotStore == nil {
		return ref, errors.New("snapshot store is nil")
	}

	if _, err = v.runtimeImageStore.GetByTypeAndName(ctx, imageType, ref.Name); err != nil {
		return ref, err
	}

	info, err := snapshot.NewService(v.snapshotStore, nil, nil).ResolveRepoKeyForRepository(ref.Name)
	if err != nil {
		return ref, errors.Wrapf(err, "resolve repo key for runtime image %s", ref.Name)
	}
	exists, err := v.snapshotStore.HasTag(ctx, info.RepoKey, ref.Tag)
	if err != nil {
		return ref, errors.Wrapf(err, "check runtime image tag %s:%s", ref.Name, ref.Tag)
	}
	if !exists {
		return ref, ErrRuntimeImageTagNotFound
	}
	return ref, nil
}
