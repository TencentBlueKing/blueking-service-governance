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

	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	k8smanifest "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/manifest"
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
	store ResourceSnapshotStore,
	appID string,
	deployID string,
	gameDeploy *tkex.GameDeployment,
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

	gdUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(gameDeploy)
	if err != nil {
		log.Errorf(ctx, "convert game deployment to unstructured for app %s deploy %s: %v", appID, deployID, err)
	}

	// 将主工作负载和额外资源合并
	allObjs := append(extraObjs, unstructured.Unstructured{Object: gdUnstructured})

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

	if err = store.CreateResources(ctx, snapshots); err != nil {
		log.Errorf(ctx, "create resource snapshots for app %s deploy %s: %v", appID, deployID, err)
	}
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
