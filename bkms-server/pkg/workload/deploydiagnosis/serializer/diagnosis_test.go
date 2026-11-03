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

package serializer_test

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	deploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/rules"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/serializer"
)

var _ = Describe("DeployDiagnosisOutputObj.FromDiagnosis", func() {
	It("renders nil slices as empty arrays and leaves deploy fields empty when never deployed", func() {
		out := new(serializer.DeployDiagnosisOutputObj).FromDiagnosis(&deploydiagnosis.Diagnosis{
			GeneratedAt: time.Now(),
			EmptyReason: deploydiagnosis.EmptyReasonNoDeploy,
			Groups:      []*rules.Group{{RuleID: rules.IDUnknown, Instances: []*rules.GroupInstance{{ID: "pod-a"}}}},
		})

		raw, err := json.Marshal(out)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(raw)).NotTo(ContainSubstring("null"))
		Expect(out.Partial).To(BeFalse())
		Expect(out.Deploy).To(Equal(serializer.DeployOutputObj{}))
	})

	It("maps deploy, partial reasons and group fields", func() {
		out := new(serializer.DeployDiagnosisOutputObj).FromDiagnosis(&deploydiagnosis.Diagnosis{
			PartialReasons: []string{deploydiagnosis.PartialReasonEvents},
			Deploy: &rules.Deploy{
				DeployID: "d-1",
				BuildID:  "b-1",
				Stage:    autodeploy.StageBuild,
				Status:   "failed",
				Phase:    deploystatus.DeployPhaseFailed,
			},
			TotalCount:    2,
			AbnormalCount: 1,
			Groups: []*rules.Group{{
				RuleID:        rules.IDBuildFailed,
				CategoryID:    rules.CategoryBuildFailure,
				Category:      "构建失败",
				Actions:       []rules.Action{{ID: rules.ActionBuildHistory, Label: "查看构建记录"}},
				RelatedEvents: []rules.Event{{ResourceKind: "Deployment", ResourceName: "demo", Content: "quota"}},
			}},
		})

		Expect(out.Partial).To(BeTrue())
		Expect(out.Deploy).To(Equal(serializer.DeployOutputObj{
			DeployID: "d-1", BuildID: "b-1", Stage: "build", Status: "failed", Phase: "failed",
		}))
		Expect(out.Summary).To(Equal(serializer.DiagnosisSummaryObj{TotalCount: 2, AbnormalCount: 1}))
		Expect(out.Groups[0].CategoryID).To(Equal(rules.CategoryBuildFailure))
		Expect(out.Groups[0].Actions).To(Equal([]*serializer.DiagnosisActionObj{
			{ID: rules.ActionBuildHistory, Label: "查看构建记录"},
		}))
		Expect(out.Groups[0].RelatedEvents[0].ResourceName).To(Equal("demo"))
	})
})
