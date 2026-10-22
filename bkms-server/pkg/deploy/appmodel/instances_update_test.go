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

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("createInstancesUpdateRecord", func() {
	var ctx context.Context
	var db *mongo.Database
	var recordStore *RecordStore
	var appID string

	BeforeEach(func() {
		ctx = auth.WithUser(context.Background(), auth.User{ID: "tester"})
		appID = "inst-update-app-" + stringx.Random(6)
		db = testutil.NewDB(GinkgoT())
		recordStore = store.Of(db, RecordMeta)
		DeferCleanup(func() { _, _ = recordStore.DeleteMany(ctx, store.Where("appID", appID)) })
	})

	Context("createInstancesUpdateRecord", func() {
		It("should create a deploying record inheriting the base record", func() {
			base := &Record{
				WorkspaceID:     "ws-1",
				AppID:           appID,
				EnvName:         "prod",
				TrafficLaneName: "lane-1",
				ClusterID:       "BCS-K8S-00001",
				Namespace:       "ns-1",
				ImageTag:        "v1",
				UpdateStrategy:  "RollingUpdate",
				Replicas:        3,
				DeletionCost:    -2,
				LabelSelector:   map[string]string{"app": "gd1"},
				WorkloadKind:    k8skind.GameDeploy,
				ResourceKeys: ResourceKeys{
					{Kind: k8skind.GameDeploy, Name: "gd1"},
					{Kind: "ConfigMap", Name: "cm1"},
				},
				Status:  StatusDeployed,
				Extras:  map[string]string{ExtraKeyDeploySource: DeploySourceBuildAutoDeploy},
				Creator: "creator",
			}
			Expect(recordStore.Create(ctx, base)).To(Succeed())
			baseID := base.ID.Hex()

			deployer := &Deployer{db: db, app: &bkmsapp.Application{ID: appID}}
			deployID, err := deployer.createInstancesUpdateRecord(ctx, base, "v2", "InplaceUpdate", 4)
			Expect(err).NotTo(HaveOccurred())
			Expect(deployID).NotTo(Equal(baseID))

			latest, err := GetLatest(ctx, db, appID, "prod", "lane-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(latest.ID.Hex()).To(Equal(deployID))
			Expect(latest.Status).To(Equal(StatusDeploying))
			Expect(latest.ImageTag).To(Equal("v2"))
			Expect(latest.UpdateStrategy).To(Equal("InplaceUpdate"))
			Expect(latest.Replicas).To(Equal(int32(4)))
			Expect(latest.WorkspaceID).To(Equal("ws-1"))
			Expect(latest.ClusterID).To(Equal("BCS-K8S-00001"))
			Expect(latest.Namespace).To(Equal("ns-1"))
			Expect(latest.DeletionCost).To(Equal(int64(-2)))
			Expect(latest.LabelSelector).To(Equal(base.LabelSelector))
			Expect(latest.WorkloadKind).To(Equal(k8skind.GameDeploy))
			Expect(latest.ResourceKeys).To(Equal(base.ResourceKeys))
			// 不继承构建信息，按直接部署展示
			Expect(latest.Extras).To(BeEmpty())
			Expect(latest.Creator).To(Equal("tester"))
			Expect(latest.Updater).To(Equal("tester"))
			Expect(latest.StartedAt.IsZero()).To(BeFalse())

			// 源记录保持不变
			old, err := GetRecord(ctx, db, appID, baseID)
			Expect(err).NotTo(HaveOccurred())
			Expect(old.Status).To(Equal(StatusDeployed))
			Expect(old.ImageTag).To(Equal("v1"))
		})
	})
})

// newPodForTest 创建用于测试的 Pod
func newPodForTest(name, phase string) unstructured.Unstructured {
	return unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": name},
		"status":   map[string]any{"phase": phase},
	}}
}

var _ = Describe("splitPodNamesByStatus", func() {
	It("should split pod names by terminated status and skip pods without name", func() {
		terminated, running := (&podManager{}).splitPodNamesByStatus([]unstructured.Unstructured{
			newPodForTest("pod-1", "Running"),
			newPodForTest("pod-2", "Succeeded"),
			newPodForTest("pod-3", "Pending"),
			newPodForTest("pod-4", "Failed"),
			newPodForTest("", "Running"),
		})

		Expect(terminated).To(Equal([]string{"pod-2", "pod-4"}))
		Expect(running).To(Equal([]string{"pod-1", "pod-3"}))
	})
})

// newPodWithImageForTest 创建带容器镜像的测试 Pod
func newPodWithImageForTest(name, phase string, containers map[string]string) unstructured.Unstructured {
	po := newPodForTest(name, phase)
	var cs []any
	for cName, image := range containers {
		cs = append(cs, map[string]any{"name": cName, "image": image})
	}
	po.Object["spec"] = map[string]any{"containers": cs}
	return po
}

var _ = Describe("FilterShouldGrayscalePodNames", func() {
	It("should only keep specified running pods whose main container image tag differs", func() {
		workloadPods := []unstructured.Unstructured{
			newPodWithImageForTest("pod-1", "Running", map[string]string{"main": "app:v1"}),
			// 已是灰度镜像
			newPodWithImageForTest("pod-2", "Running", map[string]string{"main": "app:v2"}),
			// 终止态
			newPodWithImageForTest("pod-3", "Failed", map[string]string{"main": "app:v1"}),
			// 未指定
			newPodWithImageForTest("pod-4", "Running", map[string]string{"main": "app:v1"}),
			// 仅 sidecar 镜像不同
			newPodWithImageForTest("pod-5", "Running", map[string]string{"main": "app:v2", "sidecar": "sc:v1"}),
		}

		names := (&podManager{}).FilterShouldGrayscalePodNames(
			workloadPods, []string{"pod-1", "pod-2", "pod-3", "pod-5"}, "v2",
		)

		Expect(names).To(Equal([]string{"pod-1"}))
	})
})

var _ = Describe("coversAllRunningPods", func() {
	m := &podManager{}
	pods := []unstructured.Unstructured{
		newPodForTest("pod-1", "Running"),
		newPodForTest("pod-2", "Pending"),
		newPodForTest("pod-3", "Failed"),
	}

	It("should return true when all running pods are specified", func() {
		Expect(m.coversAllRunningPods(pods, []string{"pod-1", "pod-2"})).To(BeTrue())
		// 包含终止态 Pod 也视为全量
		Expect(m.coversAllRunningPods(pods, []string{"pod-1", "pod-2", "pod-3"})).To(BeTrue())
	})

	It("should return false when some running pods are not specified", func() {
		Expect(m.coversAllRunningPods(pods, []string{"pod-1"})).To(BeFalse())
		Expect(m.coversAllRunningPods(pods, []string{"pod-1", "pod-3"})).To(BeFalse())
	})

	It("should return false when there is no running pod", func() {
		Expect(m.coversAllRunningPods(nil, []string{"pod-1"})).To(BeFalse())
		Expect(m.coversAllRunningPods(
			[]unstructured.Unstructured{newPodForTest("pod-3", "Succeeded")}, []string{"pod-3"},
		)).To(BeFalse())
	})
})

const gameDeployManifestForTest = `apiVersion: tkex.tencent.com/v1alpha1
kind: GameDeployment
metadata:
  name: gd1
spec:
  replicas: 3
  template:
    spec:
      containers:
      - image: mirrors.example.com/app:v1
        name: main
      - image: mirrors.example.com/sidecar:v1
        name: sidecar
  updateStrategy:
    maxUnavailable: 25%
    type: RollingUpdate
`

// gameDeploySpecForTest 解析 GameDeployment 清单的 spec，便于断言
func gameDeploySpecForTest(manifest string) map[string]any {
	var obj map[string]any
	Expect(yaml.Unmarshal([]byte(manifest), &obj)).To(Succeed())
	return obj["spec"].(map[string]any)
}

// containerImagesForTest 返回 spec 中各容器的镜像
func containerImagesForTest(spec map[string]any) []any {
	containers := spec["template"].(map[string]any)["spec"].(map[string]any)["containers"].([]any)
	images := make([]any, 0, len(containers))
	for _, c := range containers {
		images = append(images, c.(map[string]any)["image"])
	}
	return images
}

var _ = Describe("instancesUpdateManifest", func() {
	It("should update main container image and set inplace update strategy", func() {
		manifest, truncated, err := instancesUpdateManifest(
			gameDeployManifestForTest, "mirrors.example.com/app:v2", "InplaceUpdate",
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(truncated).To(BeFalse())
		spec := gameDeploySpecForTest(manifest)
		Expect(containerImagesForTest(spec)).To(Equal([]any{
			"mirrors.example.com/app:v2", "mirrors.example.com/sidecar:v1",
		}))
		Expect(spec["updateStrategy"]).To(Equal(map[string]any{
			"type": "InplaceUpdate", "partition": float64(0), "maxUnavailable": "25%",
		}))
		Expect(spec["replicas"]).To(Equal(float64(3)))
	})

	It("should set rolling update strategy and reset the partition", func() {
		manifest, _, err := instancesUpdateManifest(
			gameDeployManifestForTest, "mirrors.example.com/app:v2", "RollingUpdate",
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(gameDeploySpecForTest(manifest)["updateStrategy"]).To(Equal(map[string]any{
			"type": "RollingUpdate", "partition": float64(0), "maxUnavailable": "25%",
		}))
	})

	It("should create update strategy when it is missing and keep image when image is empty", func() {
		manifest, _, err := instancesUpdateManifest(`kind: GameDeployment
spec:
  template:
    spec:
      containers:
      - image: app:v1
        name: main
`, "", "InplaceUpdate")

		Expect(err).NotTo(HaveOccurred())
		spec := gameDeploySpecForTest(manifest)
		Expect(containerImagesForTest(spec)).To(Equal([]any{"app:v1"}))
		Expect(spec["updateStrategy"]).To(Equal(map[string]any{"type": "InplaceUpdate", "partition": float64(0)}))
	})

	It("should remove the rollout label and keep other labels", func() {
		manifest, _, err := instancesUpdateManifest(`kind: GameDeployment
metadata:
  labels:
    app: gd1
    rollout: "true"
  name: gd1
spec:
  template:
    spec:
      containers:
      - image: app:v1
        name: main
`, "app:v2", "InplaceUpdate")

		Expect(err).NotTo(HaveOccurred())
		var obj map[string]any
		Expect(yaml.Unmarshal([]byte(manifest), &obj)).To(Succeed())
		Expect(obj["metadata"].(map[string]any)["labels"]).To(Equal(map[string]any{"app": "gd1"}))
	})

	It("should return error when the manifest is invalid", func() {
		_, _, err := instancesUpdateManifest("spec: [", "app:v2", "InplaceUpdate")

		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("copyInstancesUpdateResourceSnapshots", func() {
	var ctx context.Context
	var db *mongo.Database
	var snapStore *ResourceSnapshotStore
	var appID string
	mainWorkload := ResourceKey{Kind: k8skind.GameDeploy, Name: "gd1"}

	// listByDeployRecord 按 kind、name 升序列出某次部署的全部快照
	listByDeployRecord := func(deployOID bson.ObjectID) []ResourceSnapshot {
		snapshots, err := snapStore.List(ctx, ByDeployRecord(appID, deployOID), InKindNameOrder())
		Expect(err).NotTo(HaveOccurred())
		return snapshots
	}

	BeforeEach(func() {
		ctx = context.Background()
		appID = "inst-update-snap-" + stringx.Random(6)
		db = testutil.NewDB(GinkgoT())
		snapStore = store.Of(db, ResourceSnapshotMeta)
		DeferCleanup(func() { _, _ = snapStore.DeleteMany(ctx, store.Where("appID", appID)) })
	})

	It("should copy snapshots to the new deploy record and update the main workload", func() {
		srcOID, dstOID := bson.NewObjectID(), bson.NewObjectID()
		Expect(snapStore.CreateMany(ctx, []*ResourceSnapshot{
			{
				AppID: appID, DeployRecordID: srcOID, APIVersion: "tkex.tencent.com/v1alpha1",
				Kind: k8skind.GameDeploy, Name: "gd1", Manifest: gameDeployManifestForTest,
			},
			{
				AppID: appID, DeployRecordID: srcOID, APIVersion: "v1",
				Kind: "ConfigMap", Name: "cm1", Manifest: "cm1-yaml",
			},
		})).To(Succeed())

		copyInstancesUpdateResourceSnapshots(
			ctx, db, appID, srcOID.Hex(), dstOID.Hex(), mainWorkload,
			"mirrors.example.com/app:v2", "InplaceUpdate",
		)

		dst := listByDeployRecord(dstOID)
		Expect(dst).To(HaveLen(2))
		// 按 kind、name 升序：ConfigMap 在前
		Expect(dst[0].Kind).To(Equal("ConfigMap"))
		Expect(dst[0].Manifest).To(Equal("cm1-yaml"))
		Expect(dst[0].DeployRecordID).To(Equal(dstOID))
		Expect(dst[1].Kind).To(Equal(k8skind.GameDeploy))
		Expect(dst[1].APIVersion).To(Equal("tkex.tencent.com/v1alpha1"))
		Expect(dst[1].DeployRecordID).To(Equal(dstOID))
		Expect(containerImagesForTest(gameDeploySpecForTest(dst[1].Manifest))).
			To(ContainElement("mirrors.example.com/app:v2"))

		// 源快照保持不变
		src := listByDeployRecord(srcOID)
		Expect(src[1].Manifest).To(Equal(gameDeployManifestForTest))
	})

	It("should not copy the rollout deleted before the instances update", func() {
		srcOID, dstOID := bson.NewObjectID(), bson.NewObjectID()
		Expect(snapStore.CreateMany(ctx, []*ResourceSnapshot{
			{
				AppID: appID, DeployRecordID: srcOID, APIVersion: "tkex.tencent.com/v1alpha1",
				Kind: k8skind.GameDeploy, Name: "gd1", Manifest: gameDeployManifestForTest,
			},
			{
				AppID: appID, DeployRecordID: srcOID, APIVersion: batchrelease.APIVersion,
				Kind: batchrelease.Kind, Name: "gd1", Manifest: "rollout-yaml",
			},
		})).To(Succeed())

		copyInstancesUpdateResourceSnapshots(
			ctx, db, appID, srcOID.Hex(), dstOID.Hex(), mainWorkload, "app:v2", "InplaceUpdate",
		)

		dst := listByDeployRecord(dstOID)
		Expect(dst).To(HaveLen(1))
		Expect(dst[0].Kind).To(Equal(k8skind.GameDeploy))
		// 源快照保持不变
		Expect(listByDeployRecord(srcOID)).To(HaveLen(2))
	})

	It("should copy the main workload as is when the manifest is truncated", func() {
		srcOID, dstOID := bson.NewObjectID(), bson.NewObjectID()
		Expect(snapStore.Create(ctx, &ResourceSnapshot{
			AppID: appID, DeployRecordID: srcOID, Kind: k8skind.GameDeploy, Name: "gd1",
			Manifest: "truncated-yaml", IsTruncated: true,
		})).To(Succeed())

		copyInstancesUpdateResourceSnapshots(
			ctx, db, appID, srcOID.Hex(), dstOID.Hex(), mainWorkload, "app:v2", "InplaceUpdate",
		)

		dst := listByDeployRecord(dstOID)
		Expect(dst).To(HaveLen(1))
		Expect(dst[0].Manifest).To(Equal("truncated-yaml"))
		Expect(dst[0].IsTruncated).To(BeTrue())
	})

	It("should skip when the source deploy record has no snapshots", func() {
		dstOID := bson.NewObjectID()

		copyInstancesUpdateResourceSnapshots(
			ctx, db, appID, bson.NewObjectID().Hex(), dstOID.Hex(), mainWorkload, "app:v2", "InplaceUpdate",
		)

		Expect(listByDeployRecord(dstOID)).To(BeEmpty())
	})
})
