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

package appmodel

import (
	"context"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
	k8smanifest "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/manifest"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

const (
	// maxManifestBytes 单资源 manifest 最大字节数（5MB）
	maxManifestBytes = 5 << 20

	// manifestTruncatedSuffix 超长时在截断正文后追加
	manifestTruncatedSuffix = "\n# Manifest truncated (exceeds storage limit).\n"
)

// saveResourceSnapshots 持久化本次部署下发的资源清单快照
func saveResourceSnapshots(
	ctx context.Context,
	db *mongo.Database,
	appID string,
	deployID string,
	mainWorkload any,
	extraObjs []unstructured.Unstructured,
	sensitiveEnvVarValues map[string]string,
) {
	// 预防来自上游的非标准 Unstructured 导致的 panic，虽然理论上不应该，但以防万一
	defer func() {
		if r := recover(); r != nil {
			log.Errorf(ctx, "panic while saving resource snapshots for app %s deploy %s: %v", appID, deployID, r)
		}
	}()

	deployOID, _ := bson.ObjectIDFromHex(deployID)

	wlUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(mainWorkload)
	if err != nil {
		log.Errorf(ctx, "convert main workload to unstructured for app %s deploy %s: %v", appID, deployID, err)
	}

	// 将主工作负载和额外资源合并
	allObjs := append(extraObjs, unstructured.Unstructured{Object: wlUnstructured})

	timeNow := time.Now()

	// 构造 ResourceSnapshot，过滤掉创建失败的对象
	snapshots := lo.FilterMap(allObjs, func(obj unstructured.Unstructured, _ int) (ResourceSnapshot, bool) {
		snap, newErr := NewResourceSnapshot(obj, sensitiveEnvVarValues, appID, deployOID, timeNow)
		if newErr != nil {
			log.Errorf(ctx, "create resource snapshot for app %s deploy %s: %v", appID, deployID, newErr)
			return ResourceSnapshot{}, false
		}
		return *snap, true
	})

	if err = store.Of(db, ResourceSnapshotMeta).CreateMany(ctx, lo.ToSlicePtr(snapshots)); err != nil {
		log.Errorf(ctx, "create resource snapshots for app %s deploy %s: %v", appID, deployID, err)
	}
}

// copyInstancesUpdateResourceSnapshots 为全量实例更新的部署记录生成资源清单快照
// 全量实例更新仅 Patch 主工作负载的主容器镜像与更新策略，因此复制源部署记录的快照，
// 并在主工作负载快照中同步修改这两处；源快照已脱敏，复制不会泄露敏感信息。
// 实例更新前已删除遗留的 Rollout，因此不复制其快照
func copyInstancesUpdateResourceSnapshots(
	ctx context.Context,
	db *mongo.Database,
	appID, srcDeployID, dstDeployID string,
	mainWorkload ResourceKey,
	image, updateStrategy string,
) {
	srcOID, err := bson.ObjectIDFromHex(srcDeployID)
	if err != nil {
		log.Errorf(ctx, "invalid source deploy id %s for app %s: %v", srcDeployID, appID, err)
		return
	}
	snapshotStore := store.Of(db, ResourceSnapshotMeta)
	srcSnapshots, err := snapshotStore.List(ctx, ByDeployRecord(appID, srcOID), InKindNameOrder())
	if err != nil {
		log.Errorf(ctx, "list resource snapshots for app %s deploy %s: %v", appID, srcDeployID, err)
		return
	}
	// 清理遗留的 Rollout 快照（实例更新时不使用 Rollout，会提前清理）
	srcSnapshots = lo.Reject(srcSnapshots, func(snap ResourceSnapshot, _ int) bool {
		return snap.APIVersion == batchrelease.APIVersion && snap.Kind == batchrelease.Kind
	})
	// 历史部署记录可能没有快照，此时无需复制
	if len(srcSnapshots) == 0 {
		log.Warnf(ctx, "no resource snapshots for app %s deploy %s, skip copy", appID, srcDeployID)
		return
	}

	dstOID, _ := bson.ObjectIDFromHex(dstDeployID)
	timeNow := time.Now()
	snapshots := lo.Map(srcSnapshots, func(snap ResourceSnapshot, _ int) ResourceSnapshot {
		// 截断的清单已不是合法 YAML，原样复制
		if snap.Kind == mainWorkload.Kind && snap.Name == mainWorkload.Name && !snap.IsTruncated {
			manifest, truncated, uErr := instancesUpdateManifest(snap.Manifest, image, updateStrategy)
			if uErr != nil {
				log.Errorf(ctx, "update %s snapshot for app %s deploy %s: %v", mainWorkload, appID, dstDeployID, uErr)
			} else {
				snap.Manifest, snap.IsTruncated = manifest, truncated
			}
		}
		snap.ID = bson.ObjectID{}
		snap.DeployRecordID = dstOID
		snap.CreatedAt = timeNow
		return snap
	})

	if err = snapshotStore.CreateMany(ctx, lo.ToSlicePtr(snapshots)); err != nil {
		log.Errorf(ctx, "create resource snapshots for app %s deploy %s: %v", appID, dstDeployID, err)
	}
}

// instancesUpdateManifest 由 GameDeployment 清单生成全量实例更新后的清单，返回新的清单及是否被截断。
// 与 UpdateInstances 对工作负载的修改保持一致：修改主容器镜像（image 为空时不修改）与更新策略类型，
// partition 置为 0，并移除实例更新前已从工作负载上清理的 Rollout 标签
func instancesUpdateManifest(manifest, image, updateStrategy string) (string, bool, error) {
	var obj map[string]any
	if err := yaml.Unmarshal([]byte(manifest), &obj); err != nil {
		return "", false, errors.Wrap(err, "unmarshal manifest")
	}
	unstructured.RemoveNestedField(obj, "metadata", "labels", batchrelease.WorkloadLabelKey)

	if image != "" {
		containers, _, err := unstructured.NestedSlice(obj, "spec", "template", "spec", "containers")
		if err != nil {
			return "", false, errors.Wrap(err, "get containers")
		}
		for _, c := range containers {
			if container, ok := c.(map[string]any); ok && container["name"] == defaults.WorkloadMainContainerName {
				container["image"] = image
			}
		}
		if err = unstructured.SetNestedSlice(obj, containers, "spec", "template", "spec", "containers"); err != nil {
			return "", false, errors.Wrap(err, "set containers")
		}
	}

	if err := unstructured.SetNestedField(obj, updateStrategy, "spec", "updateStrategy", "type"); err != nil {
		return "", false, errors.Wrap(err, "set update strategy type")
	}
	if err := unstructured.SetNestedField(obj, int64(0), "spec", "updateStrategy", "partition"); err != nil {
		return "", false, errors.Wrap(err, "set update strategy partition")
	}
	return UnstructuredToYaml(unstructured.Unstructured{Object: obj})
}

// NewResourceSnapshot 通过 Unstructured 创建资源清单快照
func NewResourceSnapshot(
	obj unstructured.Unstructured,
	sensitiveEnvVarValues map[string]string,
	appID string,
	deployOID bson.ObjectID,
	createdAt time.Time,
) (*ResourceSnapshot, error) {
	maskedObj := *obj.DeepCopy()
	k8smanifest.NewMasker(sensitiveEnvVarValues, envvartypes.SensitiveValueMask).Mask(&maskedObj)

	manifest, truncated, err := UnstructuredToYaml(maskedObj)
	if err != nil {
		return nil, errors.Wrapf(err, "build manifest for %s/%s", obj.GetKind(), obj.GetName())
	}
	return &ResourceSnapshot{
		DeployRecordID: deployOID,
		AppID:          appID,
		APIVersion:     obj.GetAPIVersion(),
		Kind:           obj.GetKind(),
		Name:           obj.GetName(),
		Manifest:       manifest,
		IsTruncated:    truncated,
		CreatedAt:      createdAt,
	}, nil
}

// UnstructuredToYaml 将 Unstructured 序列化为 YAML，如果超长则截断并追加 manifestTruncatedSuffix。
func UnstructuredToYaml(obj unstructured.Unstructured) (out string, isTruncated bool, err error) {
	yamlBytes, err := yaml.Marshal(obj.Object)
	if err != nil {
		return "", false, errors.Wrap(err, "marshal object to yaml")
	}
	if len(yamlBytes) <= maxManifestBytes {
		return string(yamlBytes), false, nil
	}
	end := maxManifestBytes
	for end > 0 && !utf8.RuneStart(yamlBytes[end]) {
		end--
	}
	return string(yamlBytes[:end]) + manifestTruncatedSuffix, true, nil
}
