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

package topology

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

var _ = Describe("Builder", func() {
	var builder *Builder
	var ctx context.Context

	BeforeEach(func() {
		builder = NewBuilder()
		ctx = context.Background()
	})

	Describe("Build", func() {
		It("should return error when snapshot is nil", func() {
			graph, err := builder.Build(ctx, nil)
			Expect(err).To(HaveOccurred())
			Expect(graph).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("snapshot is nil"))
		})
	})

	Describe("buildNodes", func() {
		It("should build nodes with correct IDs and displayName", func() {
			entries := []ResourceEntry{
				{
					Kind:       "Deployment",
					Namespace:  "default",
					Name:       "nginx",
					IsManaged:  true,
					SourceType: SourceTypeHelmManifest,
				},
				{
					Kind:       "Service",
					Namespace:  "default",
					Name:       "nginx-svc",
					IsManaged:  true,
					SourceType: SourceTypeHelmManifest,
				},
			}
			clusterResources := map[string]*unstructured.Unstructured{}

			nodes, nodeIDSet := builder.buildNodes(entries, clusterResources, false)

			Expect(nodes).To(HaveLen(2))
			Expect(nodeIDSet).To(HaveLen(2))

			// 验证 Deployment 节点
			Expect(nodes[0].Kind).To(Equal("Deployment"))
			Expect(nodes[0].Name).To(Equal("nginx"))
			Expect(nodes[0].DisplayName).To(Equal("Deployment/nginx"))
			Expect(nodes[0].IsManaged).To(BeTrue())
			Expect(nodes[0].ID).To(Equal(EncodeNodeID("Deployment", "default", "nginx")))

			// 验证 Service 节点
			Expect(nodes[1].Kind).To(Equal("Service"))
		})

		It("should mark nodes as NotFound when not in cluster resources", func() {
			entries := []ResourceEntry{
				{
					Kind:       "ConfigMap",
					Namespace:  "default",
					Name:       "missing-cm",
					IsManaged:  true,
					SourceType: SourceTypeHelmManifest,
				},
			}
			clusterResources := map[string]*unstructured.Unstructured{}

			nodes, _ := builder.buildNodes(entries, clusterResources, false)

			Expect(nodes).To(HaveLen(1))
			Expect(nodes[0].Status).To(Equal(k8sstatus.NotFound))
		})

		It("should deduplicate nodes with same kind/namespace/name", func() {
			entries := []ResourceEntry{
				{
					Kind:       "Pod",
					Namespace:  "default",
					Name:       "nginx-pod",
					IsManaged:  false,
					SourceType: SourceTypeOwnerReference,
				},
				{
					Kind:       "Pod",
					Namespace:  "default",
					Name:       "nginx-pod",
					IsManaged:  false,
					SourceType: SourceTypeOwnerReference,
				},
			}
			clusterResources := map[string]*unstructured.Unstructured{}

			nodes, nodeIDSet := builder.buildNodes(entries, clusterResources, false)

			Expect(nodes).To(HaveLen(1))
			Expect(nodeIDSet).To(HaveLen(1))
		})
	})

	Describe("buildPrimaryEdges", func() {
		It("should build ownerRef edges from cluster resources", func() {
			rsObj := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "ReplicaSet",
				"metadata": map[string]any{
					"name":      "nginx-abc123",
					"namespace": "default",
					"ownerReferences": []any{
						map[string]any{
							"apiVersion": "apps/v1",
							"kind":       "Deployment",
							"name":       "nginx",
							"uid":        "12345",
						},
					},
				},
			}}

			deployNodeID := EncodeNodeID("Deployment", "default", "nginx")
			rsNodeID := EncodeNodeID("ReplicaSet", "default", "nginx-abc123")

			nodeIDSet := map[string]bool{
				deployNodeID: true,
				rsNodeID:     true,
			}

			clusterResources := map[string]*unstructured.Unstructured{
				"ReplicaSet/default/nginx-abc123": rsObj,
			}

			entries := []ResourceEntry{
				{Kind: "Deployment", Namespace: "default", Name: "nginx", IsManaged: true},
				{Kind: "ReplicaSet", Namespace: "default", Name: "nginx-abc123", IsManaged: false},
			}

			edges := builder.buildPrimaryEdges(clusterResources, nodeIDSet, entries)

			Expect(edges).To(HaveLen(1))
			Expect(edges[0].SourceID).To(Equal(deployNodeID))
			Expect(edges[0].TargetID).To(Equal(rsNodeID))
			Expect(edges[0].Relation).To(Equal(EdgeRelationCreates))
			Expect(edges[0].IsPrimary).To(BeTrue())
			Expect(edges[0].Reason.Type).To(Equal(RelationTypeOwnerReference))
		})

		It("should skip edges when parent node is not in snapshot", func() {
			podObj := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]any{
					"name":      "nginx-pod",
					"namespace": "default",
					"ownerReferences": []any{
						map[string]any{
							"apiVersion": "apps/v1",
							"kind":       "ReplicaSet",
							"name":       "nginx-rs-out-of-snapshot",
							"uid":        "99",
						},
					},
				},
			}}

			podNodeID := EncodeNodeID("Pod", "default", "nginx-pod")
			nodeIDSet := map[string]bool{
				podNodeID: true,
			}

			clusterResources := map[string]*unstructured.Unstructured{
				"Pod/default/nginx-pod": podObj,
			}

			edges := builder.buildPrimaryEdges(clusterResources, nodeIDSet, nil)

			Expect(edges).To(BeEmpty())
		})
	})

	Describe("buildAppRootNodeAndEdges", func() {
		It("should create APP virtual root node with correct properties", func() {
			nodes := []Node{
				{
					ID:        EncodeNodeID("Deployment", "default", "nginx"),
					Kind:      "Deployment",
					Namespace: "default",
					Name:      "nginx",
					IsManaged: true,
				},
			}

			rootNode, rootEdges := builder.buildAppRootNodeAndEdges("my-app", nodes, nil)

			Expect(rootNode.Kind).To(Equal(NodeKindApp))
			Expect(rootNode.Name).To(Equal("my-app"))
			Expect(rootNode.DisplayName).To(Equal("my-app"))
			Expect(rootNode.Status).To(Equal(k8sstatus.Active))
			Expect(rootNode.IsManaged).To(BeTrue())
			Expect(rootNode.ID).To(Equal(EncodeNodeID(NodeKindApp, "", "my-app")))
			Expect(rootEdges).To(HaveLen(1))
		})

		It("should create MANAGES edges to all top-level nodes without incoming primary edges", func() {
			deployNodeID := EncodeNodeID("Deployment", "default", "nginx")
			svcNodeID := EncodeNodeID("Service", "default", "nginx-svc")
			rsNodeID := EncodeNodeID("ReplicaSet", "default", "nginx-rs")

			nodes := []Node{
				{ID: deployNodeID, Kind: "Deployment", Namespace: "default", Name: "nginx", IsManaged: true},
				{ID: svcNodeID, Kind: "Service", Namespace: "default", Name: "nginx-svc", IsManaged: true},
				{ID: rsNodeID, Kind: "ReplicaSet", Namespace: "default", Name: "nginx-rs", IsManaged: false},
			}
			edges := []Edge{
				{SourceID: deployNodeID, TargetID: rsNodeID, IsPrimary: true},
			}

			rootNode, rootEdges := builder.buildAppRootNodeAndEdges("my-app", nodes, edges)

			// deploy 和 svc 没有入边，应各有一条从 root 出发的 MANAGES 边
			Expect(rootEdges).To(HaveLen(2))
			for _, e := range rootEdges {
				Expect(e.SourceID).To(Equal(rootNode.ID))
				Expect(e.Relation).To(Equal(EdgeRelationManages))
				Expect(e.IsPrimary).To(BeTrue())
				Expect(e.Reason.Type).To(Equal(RelationTypeAppRoot))
			}
		})

		It("should create MANAGES edges to all nodes when no edges exist", func() {
			nodes := []Node{
				{ID: EncodeNodeID("Deployment", "default", "a"), Kind: "Deployment", Namespace: "default", Name: "a"},
				{ID: EncodeNodeID("Service", "default", "b"), Kind: "Service", Namespace: "default", Name: "b"},
			}

			_, rootEdges := builder.buildAppRootNodeAndEdges("my-app", nodes, nil)

			Expect(rootEdges).To(HaveLen(2))
		})

		It("should return no edges when nodes list is empty", func() {
			rootNode, rootEdges := builder.buildAppRootNodeAndEdges("my-app", nil, nil)

			Expect(rootNode.Kind).To(Equal(NodeKindApp))
			Expect(rootEdges).To(BeEmpty())
		})
	})

	DescribeTable("ownerRefToRelation",
		func(parentKind, childKind string, expected EdgeRelation) {
			Expect(ownerRefToRelation(parentKind, childKind)).To(Equal(expected))
		},
		Entry("Deployment->ReplicaSet => CREATES", "Deployment", "ReplicaSet", EdgeRelationCreates),
		Entry("ReplicaSet->Pod => CREATES", "ReplicaSet", "Pod", EdgeRelationCreates),
		Entry("Unknown->Pod => MANAGES", "Unknown", "Pod", EdgeRelationManages),
	)

	Describe("buildAuxiliaryEdges", func() {
		It("should build SELECTS edge from label_selector relation", func() {
			sourceNodeID := EncodeNodeID("Service", "default", "nginx-svc")
			targetNodeID := EncodeNodeID("Deployment", "default", "nginx")
			nodeIDSet := map[string]bool{
				sourceNodeID: true,
				targetNodeID: true,
			}

			relations := []ResourceRelation{
				{
					RelationType:    RelationTypeLabelSelector,
					SourceKind:      "Service",
					SourceNamespace: "default",
					SourceName:      "nginx-svc",
					TargetKind:      "Deployment",
					TargetNamespace: "default",
					TargetName:      "nginx",
					SourceFieldPath: "spec.selector",
					Summary:         "Service/nginx-svc selects Deployment/nginx",
					MatchedLabels:   map[string]string{"app": "nginx"},
				},
			}

			edges := builder.buildAuxiliaryEdges(relations, nodeIDSet, nil, nil)

			Expect(edges).To(HaveLen(1))
			Expect(edges[0].Relation).To(Equal(EdgeRelationSelects))
			Expect(edges[0].IsPrimary).To(BeFalse())
			Expect(edges[0].Reason.Type).To(Equal(RelationTypeLabelSelector))
			Expect(edges[0].Reason.MatchedLabels).To(HaveKeyWithValue("app", "nginx"))
		})

		It("should build MOUNTS edge from volume_mount relation", func() {
			sourceNodeID := EncodeNodeID("Deployment", "default", "web")
			targetNodeID := EncodeNodeID("ConfigMap", "default", "app-config")
			nodeIDSet := map[string]bool{
				sourceNodeID: true,
				targetNodeID: true,
			}

			relations := []ResourceRelation{
				{
					RelationType:    RelationTypeVolumeMount,
					SourceKind:      "Deployment",
					SourceNamespace: "default",
					SourceName:      "web",
					TargetKind:      "ConfigMap",
					TargetNamespace: "default",
					TargetName:      "app-config",
					Summary:         "Deployment/web mounts ConfigMap/app-config",
				},
			}

			edges := builder.buildAuxiliaryEdges(relations, nodeIDSet, nil, nil)

			Expect(edges).To(HaveLen(1))
			Expect(edges[0].Relation).To(Equal(EdgeRelationMounts))
			Expect(edges[0].IsPrimary).To(BeFalse())
		})

		It("should build ROUTES_TO edge from backend_ref relation", func() {
			sourceNodeID := EncodeNodeID("Ingress", "default", "web-ingress")
			targetNodeID := EncodeNodeID("Service", "default", "web-svc")
			nodeIDSet := map[string]bool{
				sourceNodeID: true,
				targetNodeID: true,
			}

			relations := []ResourceRelation{
				{
					RelationType:    RelationTypeBackendRef,
					SourceKind:      "Ingress",
					SourceNamespace: "default",
					SourceName:      "web-ingress",
					TargetKind:      "Service",
					TargetNamespace: "default",
					TargetName:      "web-svc",
					Summary:         "Ingress/web-ingress routes to Service/web-svc",
				},
			}

			edges := builder.buildAuxiliaryEdges(relations, nodeIDSet, nil, nil)

			Expect(edges).To(HaveLen(1))
			Expect(edges[0].Relation).To(Equal(EdgeRelationRoutes))
			Expect(edges[0].IsPrimary).To(BeFalse())
		})

		It("should skip edge when target not in snapshot", func() {
			sourceNodeID := EncodeNodeID("Service", "default", "svc")
			nodeIDSet := map[string]bool{
				sourceNodeID: true,
				// target NOT in snapshot
			}

			relations := []ResourceRelation{
				{
					RelationType:    RelationTypeVolumeMount,
					SourceKind:      "Service",
					SourceNamespace: "default",
					SourceName:      "svc",
					TargetKind:      "ConfigMap",
					TargetNamespace: "default",
					TargetName:      "missing-cm",
					Summary:         "should be skipped",
				},
			}

			edges := builder.buildAuxiliaryEdges(relations, nodeIDSet, nil, nil)

			Expect(edges).To(BeEmpty())
		})

		It("should skip self-referencing edges", func() {
			nodeID := EncodeNodeID("Service", "default", "svc")
			nodeIDSet := map[string]bool{
				nodeID: true,
			}

			relations := []ResourceRelation{
				{
					RelationType:    RelationTypeLabelSelector,
					SourceKind:      "Service",
					SourceNamespace: "default",
					SourceName:      "svc",
					TargetKind:      "Service",
					TargetNamespace: "default",
					TargetName:      "svc",
					Summary:         "self-ref should be skipped",
				},
			}

			edges := builder.buildAuxiliaryEdges(relations, nodeIDSet, nil, nil)

			Expect(edges).To(BeEmpty())
		})

		It("should deduplicate edges with same source/target/relation", func() {
			sourceNodeID := EncodeNodeID("Deployment", "default", "web")
			targetNodeID := EncodeNodeID("ConfigMap", "default", "cfg")
			nodeIDSet := map[string]bool{
				sourceNodeID: true,
				targetNodeID: true,
			}

			relations := []ResourceRelation{
				{
					RelationType:    RelationTypeVolumeMount,
					SourceKind:      "Deployment",
					SourceNamespace: "default",
					SourceName:      "web",
					TargetKind:      "ConfigMap",
					TargetNamespace: "default",
					TargetName:      "cfg",
					Summary:         "first",
				},
				{
					RelationType:    RelationTypeVolumeMount,
					SourceKind:      "Deployment",
					SourceNamespace: "default",
					SourceName:      "web",
					TargetKind:      "ConfigMap",
					TargetNamespace: "default",
					TargetName:      "cfg",
					Summary:         "duplicate",
				},
			}

			edges := builder.buildAuxiliaryEdges(relations, nodeIDSet, nil, nil)

			Expect(edges).To(HaveLen(1))
		})

		It("should skip owner_reference type (already handled in primary edges)", func() {
			sourceNodeID := EncodeNodeID("Deployment", "default", "nginx")
			targetNodeID := EncodeNodeID("ReplicaSet", "default", "nginx-rs")
			nodeIDSet := map[string]bool{
				sourceNodeID: true,
				targetNodeID: true,
			}

			relations := []ResourceRelation{
				{
					RelationType:    RelationTypeOwnerReference,
					SourceKind:      "Deployment",
					SourceNamespace: "default",
					SourceName:      "nginx",
					TargetKind:      "ReplicaSet",
					TargetNamespace: "default",
					TargetName:      "nginx-rs",
					Summary:         "owner_ref should be skipped in auxiliary",
				},
			}

			edges := builder.buildAuxiliaryEdges(relations, nodeIDSet, nil, nil)

			Expect(edges).To(BeEmpty())
		})
	})

	Describe("expandWildcardRelation", func() {
		It("should only match Pods whose labels match MatchedLabels", func() {
			svcNodeID := EncodeNodeID("Service", "default", "web-svc")
			matchedPodNodeID := EncodeNodeID("Pod", "default", "web-pod-abc")
			unmatchedPodNodeID := EncodeNodeID("Pod", "default", "job-pod-xyz")

			nodeIDSet := map[string]bool{
				svcNodeID:          true,
				matchedPodNodeID:   true,
				unmatchedPodNodeID: true,
			}

			clusterResources := map[string]*unstructured.Unstructured{
				"Pod/default/web-pod-abc": {Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]any{
						"name":      "web-pod-abc",
						"namespace": "default",
						"labels":    map[string]any{"app": "web", "tier": "frontend"},
					},
				}},
				"Pod/default/job-pod-xyz": {Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]any{
						"name":      "job-pod-xyz",
						"namespace": "default",
						"labels":    map[string]any{"batch": "migration"},
					},
				}},
			}

			rel := ResourceRelation{
				RelationType:    RelationTypeLabelSelector,
				SourceKind:      "Service",
				SourceNamespace: "default",
				SourceName:      "web-svc",
				TargetKind:      "Pod",
				TargetNamespace: "default",
				TargetName:      TargetNameWildcard,
				MatchedLabels:   map[string]string{"app": "web"},
				Summary:         "Service selects Pods with app=web",
			}

			edgeIDSet := make(map[string]bool)
			edges := builder.expandWildcardRelation(rel, nodeIDSet, edgeIDSet, clusterResources, nil)

			// 只应该匹配到 web-pod-abc，不应该匹配 job-pod-xyz
			Expect(edges).To(HaveLen(1))
			Expect(edges[0].TargetID).To(Equal(matchedPodNodeID))
			Expect(edges[0].SourceID).To(Equal(svcNodeID))
			Expect(edges[0].Relation).To(Equal(EdgeRelationSelects))
		})

		It("should skip Pods not found in clusterResources", func() {
			svcNodeID := EncodeNodeID("Service", "default", "web-svc")
			podNodeID := EncodeNodeID("Pod", "default", "ghost-pod")

			nodeIDSet := map[string]bool{
				svcNodeID: true,
				podNodeID: true,
			}

			// clusterResources 中没有这个 Pod
			clusterResources := map[string]*unstructured.Unstructured{}

			rel := ResourceRelation{
				RelationType:    RelationTypeLabelSelector,
				SourceKind:      "Service",
				SourceNamespace: "default",
				SourceName:      "web-svc",
				TargetKind:      "Pod",
				TargetNamespace: "default",
				TargetName:      TargetNameWildcard,
				MatchedLabels:   map[string]string{"app": "web"},
			}

			edgeIDSet := make(map[string]bool)
			edges := builder.expandWildcardRelation(rel, nodeIDSet, edgeIDSet, clusterResources, nil)

			Expect(edges).To(BeEmpty())
		})

		It("should match all Pods when MatchedLabels is empty", func() {
			svcNodeID := EncodeNodeID("Service", "default", "web-svc")
			pod1NodeID := EncodeNodeID("Pod", "default", "pod-1")
			pod2NodeID := EncodeNodeID("Pod", "default", "pod-2")

			nodeIDSet := map[string]bool{
				svcNodeID:  true,
				pod1NodeID: true,
				pod2NodeID: true,
			}

			rel := ResourceRelation{
				RelationType:    RelationTypeLabelSelector,
				SourceKind:      "Service",
				SourceNamespace: "default",
				SourceName:      "web-svc",
				TargetKind:      "Pod",
				TargetNamespace: "default",
				TargetName:      TargetNameWildcard,
				MatchedLabels:   nil, // 无 labels 约束
			}

			edgeIDSet := make(map[string]bool)
			edges := builder.expandWildcardRelation(rel, nodeIDSet, edgeIDSet, nil, nil)

			// 无 labels 约束时，应匹配所有同命名空间的 Pod
			Expect(edges).To(HaveLen(2))
		})
	})

	DescribeTable("isLabelsMatch",
		func(selector, labels map[string]string, expected bool) {
			Expect(isLabelsMatch(selector, labels)).To(Equal(expected))
		},
		Entry(
			"selector is subset of labels",
			map[string]string{"app": "web"},
			map[string]string{"app": "web", "tier": "frontend", "version": "v1"},
			true,
		),
		Entry(
			"selector has key not in labels",
			map[string]string{"app": "web", "env": "prod"},
			map[string]string{"app": "web"},
			false,
		),
		Entry(
			"values mismatch",
			map[string]string{"app": "web"},
			map[string]string{"app": "api"},
			false,
		),
		Entry(
			"empty selector matches any labels",
			map[string]string{},
			map[string]string{"app": "web"},
			true,
		),
		Entry(
			"nil labels with non-empty selector",
			map[string]string{"app": "web"}, nil, false,
		),
	)

	DescribeTable("relationTypeToEdgeRelation",
		func(relType RelationType, expected EdgeRelation) {
			Expect(relationTypeToEdgeRelation(relType)).To(Equal(expected))
		},
		Entry("label_selector => SELECTS", RelationTypeLabelSelector, EdgeRelationSelects),
		Entry("unknown => MANAGES", RelationType("unknown"), EdgeRelationManages),
	)

	DescribeTable("isDynamicResource",
		func(kind, namespace, name string, expected bool) {
			staticSet := buildStaticResourceSet([]ResourceEntry{
				{Kind: "Job", Namespace: "default", Name: "migration-job"},
				{Kind: "Deployment", Namespace: "default", Name: "web"},
			})
			Expect(isDynamicResource(staticSet, kind, namespace, name)).To(Equal(expected))
		},
		Entry("Pod not in static entries", "Pod", "default", "web-abc-123", true),
		Entry("ReplicaSet not in static entries", "ReplicaSet", "default", "web-rs-abc", true),
		Entry("Job not in static entries (created by CronJob)", "Job", "default", "cj-triggered-job-12345", true),
		Entry("Job in static entries (Helm Chart declared)", "Job", "default", "migration-job", false),
		Entry("non-dynamic kinds", "Deployment", "default", "web", false),
	)

	DescribeTable("involveDynamicResource",
		func(rel ResourceRelation, expected bool) {
			staticSet := buildStaticResourceSet([]ResourceEntry{
				{Kind: "Job", Namespace: "default", Name: "migration-job"},
			})
			Expect(involveDynamicResource(staticSet, rel)).To(Equal(expected))
		},
		Entry("source is dynamic Pod", ResourceRelation{
			SourceKind: "Pod", SourceNamespace: "default", SourceName: "web-abc",
			TargetKind: "ConfigMap", TargetNamespace: "default", TargetName: "app-config",
		}, true),
		Entry("target is dynamic Pod", ResourceRelation{
			SourceKind: "Service", SourceNamespace: "default", SourceName: "web-svc",
			TargetKind: "Pod", TargetNamespace: "default", TargetName: "web-abc",
		}, true),
		Entry("source is static Job", ResourceRelation{
			SourceKind: "Job", SourceNamespace: "default", SourceName: "migration-job",
			TargetKind: "ConfigMap", TargetNamespace: "default", TargetName: "app-config",
		}, false),
		Entry("neither is dynamic", ResourceRelation{
			SourceKind: "Deployment", SourceNamespace: "default", SourceName: "web",
			TargetKind: "ConfigMap", TargetNamespace: "default", TargetName: "app-config",
		}, false),
	)

	Describe("mergeExtensionRelations", func() {
		// staticEntries 模拟 snapshot.Resources：Helm Chart 声明的静态资源
		var staticEntries []ResourceEntry

		BeforeEach(func() {
			staticEntries = []ResourceEntry{
				{Kind: "Deployment", Namespace: "default", Name: "web"},
				{Kind: "Service", Namespace: "default", Name: "web-svc"},
				{Kind: "ConfigMap", Namespace: "default", Name: "app-config"},
				{Kind: "Secret", Namespace: "default", Name: "db-secret"},
			}
		})

		It("should keep static persisted relations and replace dynamic ones with realtime", func() {
			persisted := []ResourceRelation{
				// 静态关系：Deployment -> ConfigMap（volume_mount），应保留
				{
					RelationType: RelationTypeVolumeMount,
					SourceKind:   "Deployment", SourceNamespace: "default", SourceName: "web",
					TargetKind: "ConfigMap", TargetNamespace: "default", TargetName: "app-config",
					Summary: "persisted static volume_mount",
				},
				// 动态关系：Pod(旧) -> ConfigMap（volume_mount），应丢弃
				{
					RelationType: RelationTypeVolumeMount,
					SourceKind:   "Pod", SourceNamespace: "default", SourceName: "web-old-pod-abc",
					TargetKind: "ConfigMap", TargetNamespace: "default", TargetName: "app-config",
					Summary: "persisted stale pod volume_mount",
				},
				// 动态关系：Service -> Pod（label_selector, target=*），TargetKind=Pod 是动态的
				// Pod TargetNameWildcard 不在 staticEntries → involveDynamicResource=true → 丢弃
				{
					RelationType: RelationTypeLabelSelector,
					SourceKind:   "Service", SourceNamespace: "default", SourceName: "web-svc",
					TargetKind: "Pod", TargetNamespace: "default", TargetName: TargetNameWildcard,
					Summary: "persisted label_selector wildcard",
				},
				// 动态关系：Pod(旧) -> Secret（env_ref），应丢弃
				{
					RelationType: RelationTypeEnvRef,
					SourceKind:   "Pod", SourceNamespace: "default", SourceName: "web-old-pod-abc",
					TargetKind: "Secret", TargetNamespace: "default", TargetName: "db-secret",
					Summary: "persisted stale pod env_ref",
				},
			}

			realtime := []ResourceRelation{
				// owner_reference 类型，应被丢弃（由 buildPrimaryEdges 处理）
				{
					RelationType: RelationTypeOwnerReference,
					SourceKind:   "ReplicaSet", SourceNamespace: "default", SourceName: "web-rs-new",
					TargetKind: "Pod", TargetNamespace: "default", TargetName: "web-new-pod-xyz",
					Summary: "realtime owner_ref",
				},
				// 动态关系：Pod(新) -> ConfigMap（volume_mount），应采纳
				{
					RelationType: RelationTypeVolumeMount,
					SourceKind:   "Pod", SourceNamespace: "default", SourceName: "web-new-pod-xyz",
					TargetKind: "ConfigMap", TargetNamespace: "default", TargetName: "app-config",
					Summary: "realtime new pod volume_mount",
				},
				// 动态关系：Pod(新) -> Secret（env_ref），应采纳
				{
					RelationType: RelationTypeEnvRef,
					SourceKind:   "Pod", SourceNamespace: "default", SourceName: "web-new-pod-xyz",
					TargetKind: "Secret", TargetNamespace: "default", TargetName: "db-secret",
					Summary: "realtime new pod env_ref",
				},
				// 动态关系：Service -> Pod（label_selector, target=*），应采纳
				{
					RelationType: RelationTypeLabelSelector,
					SourceKind:   "Service", SourceNamespace: "default", SourceName: "web-svc",
					TargetKind: "Pod", TargetNamespace: "default", TargetName: TargetNameWildcard,
					Summary: "realtime label_selector wildcard",
				},
				// 静态关系（实时收集到的），不涉及动态资源 → 不采纳（由 persisted 提供）
				{
					RelationType: RelationTypeVolumeMount,
					SourceKind:   "Deployment", SourceNamespace: "default", SourceName: "web",
					TargetKind: "ConfigMap", TargetNamespace: "default", TargetName: "app-config",
					Summary: "realtime static volume_mount (should not be added)",
				},
			}

			merged := mergeExtensionRelations(persisted, realtime, staticEntries)

			// 应保留：persisted 静态 volume_mount
			// 应采纳：realtime label_selector + realtime new Pod volume_mount + realtime new Pod env_ref
			// 总计 4 条
			Expect(merged).To(HaveLen(4))

			summaries := make([]string, 0, len(merged))
			for _, r := range merged {
				summaries = append(summaries, r.Summary)
			}
			Expect(summaries).To(ContainElement("persisted static volume_mount"))
			Expect(summaries).To(ContainElement("realtime label_selector wildcard"))
			Expect(summaries).To(ContainElement("realtime new pod volume_mount"))
			Expect(summaries).To(ContainElement("realtime new pod env_ref"))

			Expect(summaries).NotTo(ContainElement("persisted stale pod volume_mount"))
			Expect(summaries).NotTo(ContainElement("persisted stale pod env_ref"))
			Expect(summaries).NotTo(ContainElement("persisted label_selector wildcard"))
			Expect(summaries).NotTo(ContainElement("realtime owner_ref"))
			Expect(summaries).NotTo(ContainElement("realtime static volume_mount (should not be added)"))
		})

		It("should return empty when both inputs are empty", func() {
			merged := mergeExtensionRelations(nil, nil, nil)
			Expect(merged).To(BeEmpty())
		})

		It("should return only persisted static relations when realtime is empty", func() {
			persisted := []ResourceRelation{
				{
					RelationType: RelationTypeVolumeMount,
					SourceKind:   "Deployment", SourceNamespace: "default", SourceName: "web",
					TargetKind: "ConfigMap", TargetNamespace: "default", TargetName: "app-config",
					Summary: "static",
				},
				{
					RelationType: RelationTypeVolumeMount,
					SourceKind:   "Pod", SourceNamespace: "default", SourceName: "web-abc",
					TargetKind: "ConfigMap", TargetNamespace: "default", TargetName: "app-config",
					Summary: "dynamic-stale",
				},
			}

			merged := mergeExtensionRelations(persisted, nil, staticEntries)
			Expect(merged).To(HaveLen(1))
			Expect(merged[0].Summary).To(Equal("static"))
		})

		It("should handle label_selector with wildcard target Pod as dynamic relation", func() {
			// label_selector 中 TargetKind=Pod, TargetName=* 的关系
			// Pod/* 不在 staticEntries → involveDynamicResource=true → persisted 丢弃
			persisted := []ResourceRelation{
				{
					RelationType: RelationTypeLabelSelector,
					SourceKind:   "Service", SourceNamespace: "default", SourceName: "web-svc",
					TargetKind: "Pod", TargetNamespace: "default", TargetName: TargetNameWildcard,
					Summary: "persisted svc selects pods",
				},
			}
			realtime := []ResourceRelation{
				{
					RelationType: RelationTypeLabelSelector,
					SourceKind:   "Service", SourceNamespace: "default", SourceName: "web-svc",
					TargetKind: "Pod", TargetNamespace: "default", TargetName: TargetNameWildcard,
					Summary: "realtime svc selects pods",
				},
			}

			merged := mergeExtensionRelations(persisted, realtime, staticEntries)

			// persisted 被丢弃（Pod/* 不在 staticEntries → involveDynamicResource=true）
			// realtime 被采纳（involveDynamicResource=true 且非 owner_reference）
			Expect(merged).To(HaveLen(1))
			Expect(merged[0].Summary).To(Equal("realtime svc selects pods"))
		})

		It("should preserve persisted relations for static Job declared in Helm Chart", func() {
			// 当 Job 是 Helm Chart 直接声明的静态资源时，其持久化关系应被保留
			staticEntriesWithJob := append(staticEntries, ResourceEntry{
				Kind: "Job", Namespace: "default", Name: "migration-job",
			})

			persisted := []ResourceRelation{
				// 静态 Job -> ConfigMap（volume_mount），应保留
				{
					RelationType: RelationTypeVolumeMount,
					SourceKind:   "Job", SourceNamespace: "default", SourceName: "migration-job",
					TargetKind: "ConfigMap", TargetNamespace: "default", TargetName: "app-config",
					Summary: "static job volume_mount",
				},
				// 动态 Job（CronJob 创建的） -> Secret，应丢弃
				{
					RelationType: RelationTypeEnvRef,
					SourceKind:   "Job", SourceNamespace: "default", SourceName: "cj-triggered-12345",
					TargetKind: "Secret", TargetNamespace: "default", TargetName: "db-secret",
					Summary: "dynamic job env_ref",
				},
			}

			realtime := []ResourceRelation{
				// 动态 Job（新的 CronJob 创建的） -> Secret，应采纳
				{
					RelationType: RelationTypeEnvRef,
					SourceKind:   "Job", SourceNamespace: "default", SourceName: "cj-triggered-67890",
					TargetKind: "Secret", TargetNamespace: "default", TargetName: "db-secret",
					Summary: "realtime dynamic job env_ref",
				},
			}

			merged := mergeExtensionRelations(persisted, realtime, staticEntriesWithJob)

			Expect(merged).To(HaveLen(2))
			summaries := make([]string, 0, len(merged))
			for _, r := range merged {
				summaries = append(summaries, r.Summary)
			}
			// 静态 Job 关系保留
			Expect(summaries).To(ContainElement("static job volume_mount"))
			// 动态 Job 新关系采纳
			Expect(summaries).To(ContainElement("realtime dynamic job env_ref"))
			// 旧动态 Job 关系丢弃
			Expect(summaries).NotTo(ContainElement("dynamic job env_ref"))
		})
	})
})
