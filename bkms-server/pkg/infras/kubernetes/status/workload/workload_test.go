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

package workload

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

var _ = Describe("CombineMessage", func() {
	Context("when both reason and message are non-empty", func() {
		It("combines them with colon separator", func() {
			Expect(CombineMessage("TestReason", "test message")).To(Equal("TestReason: test message"))
		})
	})

	Context("when only reason is non-empty", func() {
		It("returns reason only", func() {
			Expect(CombineMessage("TestReason", "")).To(Equal("TestReason"))
		})
	})

	Context("when only message is non-empty", func() {
		It("returns message only", func() {
			Expect(CombineMessage("", "test message")).To(Equal("test message"))
		})
	})

	Context("when both are empty", func() {
		It("returns empty string", func() {
			Expect(CombineMessage("", "")).To(Equal(""))
		})
	})
})

var _ = Describe("GetCondition", func() {
	Context("when conditions contain the target type", func() {
		It("returns the matching condition", func() {
			manifest := map[string]any{
				"status": map[string]any{
					"conditions": []any{
						map[string]any{"type": "Available", "status": "True"},
						map[string]any{"type": "Progressing", "status": "True"},
					},
				},
			}
			cond := GetCondition(manifest, "Progressing")
			Expect(cond).NotTo(BeNil())
			Expect(cond["type"]).To(Equal("Progressing"))
		})
	})

	Context("when conditions do not contain the target type", func() {
		It("returns nil", func() {
			manifest := map[string]any{
				"status": map[string]any{
					"conditions": []any{
						map[string]any{"type": "Available", "status": "True"},
					},
				},
			}
			Expect(GetCondition(manifest, "Degraded")).To(BeNil())
		})
	})

	Context("when conditions is missing", func() {
		It("returns nil", func() {
			manifest := map[string]any{
				"status": map[string]any{},
			}
			Expect(GetCondition(manifest, "Available")).To(BeNil())
		})
	})
})

var _ = Describe("IsGenerationObserved", func() {
	Context("when observedGeneration >= metadata.generation", func() {
		It("returns true", func() {
			manifest := map[string]any{
				"metadata": map[string]any{"generation": int64(1)},
				"status":   map[string]any{"observedGeneration": int64(1)},
			}
			Expect(IsGenerationObserved(manifest)).To(BeTrue())
		})
	})

	Context("when observedGeneration < metadata.generation", func() {
		It("returns false", func() {
			manifest := map[string]any{
				"metadata": map[string]any{"generation": int64(2)},
				"status":   map[string]any{"observedGeneration": int64(1)},
			}
			Expect(IsGenerationObserved(manifest)).To(BeFalse())
		})
	})

	Context("when observedGeneration is zero", func() {
		It("returns false", func() {
			manifest := map[string]any{
				"metadata": map[string]any{"generation": int64(1)},
				"status":   map[string]any{"observedGeneration": int64(0)},
			}
			Expect(IsGenerationObserved(manifest)).To(BeFalse())
		})
	})

	Context("when metadata.generation is missing", func() {
		It("returns false", func() {
			manifest := map[string]any{
				"status": map[string]any{"observedGeneration": int64(1)},
			}
			Expect(IsGenerationObserved(manifest)).To(BeFalse())
		})
	})
})

var _ = Describe("CheckDegraded", func() {
	Context("when Degraded condition is True", func() {
		It("returns Degraded result with reason and message", func() {
			manifest := map[string]any{
				"status": map[string]any{
					"conditions": []any{
						map[string]any{
							"type":    "Degraded",
							"status":  "True",
							"reason":  "TestReason",
							"message": "something went wrong",
						},
					},
				},
			}
			result := CheckDegraded(manifest)
			Expect(result).NotTo(BeNil())
			Expect(result.Code).To(Equal(k8sstatus.Degraded))
			Expect(result.Message).To(Equal("TestReason: something went wrong"))
		})
	})

	Context("when Degraded condition is False", func() {
		It("returns nil", func() {
			manifest := map[string]any{
				"status": map[string]any{
					"conditions": []any{
						map[string]any{"type": "Degraded", "status": "False"},
					},
				},
			}
			Expect(CheckDegraded(manifest)).To(BeNil())
		})
	})

	Context("when no Degraded condition exists", func() {
		It("returns nil", func() {
			manifest := map[string]any{
				"status": map[string]any{
					"conditions": []any{
						map[string]any{"type": "Available", "status": "True"},
					},
				},
			}
			Expect(CheckDegraded(manifest)).To(BeNil())
		})
	})
})

var _ = Describe("CheckDeploymentDegraded", func() {
	Context("when ReplicaFailure condition is True", func() {
		It("returns Degraded result", func() {
			manifest := map[string]any{
				"status": map[string]any{
					"conditions": []any{
						map[string]any{
							"type":    "ReplicaFailure",
							"status":  "True",
							"reason":  "FailedCreate",
							"message": "replica creation failed",
						},
					},
				},
			}
			result := CheckDeploymentDegraded(manifest)
			Expect(result).NotTo(BeNil())
			Expect(result.Code).To(Equal(k8sstatus.Degraded))
		})
	})

	Context("when Progressing condition has ProgressDeadlineExceeded reason", func() {
		It("returns Degraded result", func() {
			manifest := map[string]any{
				"status": map[string]any{
					"conditions": []any{
						map[string]any{
							"type":    "Progressing",
							"status":  "False",
							"reason":  "ProgressDeadlineExceeded",
							"message": "deployment exceeded deadline",
						},
					},
				},
			}
			result := CheckDeploymentDegraded(manifest)
			Expect(result).NotTo(BeNil())
			Expect(result.Code).To(Equal(k8sstatus.Degraded))
		})
	})

	Context("when Progressing condition is False but not ProgressDeadlineExceeded", func() {
		It("returns nil", func() {
			manifest := map[string]any{
				"status": map[string]any{
					"conditions": []any{
						map[string]any{
							"type":   "Progressing",
							"status": "False",
							"reason": "OtherReason",
						},
					},
				},
			}
			Expect(CheckDeploymentDegraded(manifest)).To(BeNil())
		})
	})

	Context("when no degraded conditions exist", func() {
		It("returns nil", func() {
			manifest := map[string]any{
				"status": map[string]any{
					"conditions": []any{
						map[string]any{"type": "Available", "status": "True"},
						map[string]any{"type": "Progressing", "status": "True", "reason": "NewReplicaSetAvailable"},
					},
				},
			}
			Expect(CheckDeploymentDegraded(manifest)).To(BeNil())
		})
	})
})

var _ = Describe("AreReplicasConsistent", func() {
	Context("when spec.replicas == 0 and status.replicas == 0", func() {
		It("returns consistent", func() {
			manifest := map[string]any{
				"spec":   map[string]any{"replicas": int64(0)},
				"status": map[string]any{"replicas": int64(0)},
			}
			consistent, reason := AreReplicasConsistent(manifest)
			Expect(consistent).To(BeTrue())
			Expect(reason).To(BeEmpty())
		})
	})

	Context("when all replica fields match spec.replicas", func() {
		It("returns consistent", func() {
			manifest := map[string]any{
				"spec": map[string]any{"replicas": int64(3)},
				"status": map[string]any{
					"replicas":          int64(3),
					"updatedReplicas":   int64(3),
					"readyReplicas":     int64(3),
					"availableReplicas": int64(3),
				},
			}
			consistent, reason := AreReplicasConsistent(manifest, "status.availableReplicas")
			Expect(consistent).To(BeTrue())
			Expect(reason).To(BeEmpty())
		})
	})

	Context("when spec.replicas != status.replicas", func() {
		It("returns inconsistent with reason", func() {
			manifest := map[string]any{
				"spec":   map[string]any{"replicas": int64(3)},
				"status": map[string]any{"replicas": int64(2)},
			}
			consistent, reason := AreReplicasConsistent(manifest)
			Expect(consistent).To(BeFalse())
			Expect(reason).To(Equal("spec.replicas != status.replicas"))
		})
	})

	Context("when spec.replicas != status.updatedReplicas", func() {
		It("returns inconsistent with reason", func() {
			manifest := map[string]any{
				"spec": map[string]any{"replicas": int64(3)},
				"status": map[string]any{
					"replicas":        int64(3),
					"updatedReplicas": int64(2),
				},
			}
			consistent, reason := AreReplicasConsistent(manifest)
			Expect(consistent).To(BeFalse())
			Expect(reason).To(Equal("spec.replicas != status.updatedReplicas"))
		})
	})

	Context("when spec.replicas != extra status field", func() {
		It("returns inconsistent with reason", func() {
			manifest := map[string]any{
				"spec": map[string]any{"replicas": int64(3)},
				"status": map[string]any{
					"replicas":          int64(3),
					"updatedReplicas":   int64(3),
					"readyReplicas":     int64(3),
					"availableReplicas": int64(2),
				},
			}
			consistent, reason := AreReplicasConsistent(manifest, "status.availableReplicas")
			Expect(consistent).To(BeFalse())
			Expect(reason).To(Equal("spec.replicas != status.availableReplicas"))
		})
	})
})
