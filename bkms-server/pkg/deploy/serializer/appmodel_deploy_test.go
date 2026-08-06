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
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	deploypkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
)

var _ = Describe("AppModel deploy serializers", func() {
	Describe("EnvVarPreCheckOutput", func() {
		It("converts undefined vars and their sources", func() {
			output := new(serializer.EnvVarPreCheckOutput).FromModel(&deploypkg.EnvVarPreCheckResult{
				UndefinedVars: []envvarrefs.UndefinedEnvVar{
					{
						Key: "DB_HOST",
						Sources: []envvarrefs.Source{
							{Type: envvarrefs.SourceAppConfigFile, Name: "production"},
							{Type: envvarrefs.SourceComponent, Name: "redis-config"},
						},
					},
				},
			})

			Expect(output).To(Equal(&serializer.EnvVarPreCheckOutput{
				UndefinedVars: []serializer.UndefinedEnvVarOutput{
					{
						Key: "DB_HOST",
						Sources: []serializer.EnvVarReferenceSourceOutput{
							{Type: "appConfigFile", Name: "production"},
							{Type: "component", Name: "redis-config"},
						},
					},
				},
			}))
		})

		It("serializes empty undefined vars as an empty array", func() {
			output := new(serializer.EnvVarPreCheckOutput).FromModel(&deploypkg.EnvVarPreCheckResult{})

			Expect(output.UndefinedVars).To(BeEmpty())
			Expect(output.UndefinedVars).NotTo(BeNil())
			payload, err := json.Marshal(output)
			Expect(err).NotTo(HaveOccurred())
			Expect(payload).To(MatchJSON(`{"undefinedVars":[]}`))
		})
	})

	Describe("LatestDeployStatus conversions", func() {
		It("preserves source-specific fields for build auto deploy and direct deploy", func() {
			startedAt := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
			buildStatus := new(serializer.LatestDeployStatus).FromBuildAutoDeployRecord(&autodeploy.Record{
				Stage:      autodeploy.StageBuild,
				Status:     "success",
				BuildID:    "build-1",
				DeployID:   "deploy-1",
				Branch:     "main",
				ImageTag:   "v2.0.0",
				Operator:   "builder",
				PipelineID: "pipeline-1",
				StartedAt:  startedAt,
				EndedAt:    startedAt.Add(2 * time.Minute),
			})

			Expect(buildStatus.Stage).To(Equal(string(autodeploy.StageBuild)))
			Expect(buildStatus.IsBuildAutoDeploy).To(BeTrue())
			Expect(buildStatus.DeploySource).To(Equal(appmodeldeploy.DeploySourceBuildAutoDeploy))
			Expect(buildStatus.BuildID).To(Equal("build-1"))
			Expect(buildStatus.DeployID).To(Equal("deploy-1"))
			Expect(buildStatus.Branch).To(Equal("main"))
			Expect(buildStatus.ImageTag).To(Equal("v2.0.0"))
			Expect(buildStatus.Operator).To(Equal("builder"))
			Expect(buildStatus.PipelineID).To(Equal("pipeline-1"))

			recordID := bson.NewObjectID()
			directStatus := new(serializer.LatestDeployStatus).FromDeployRecord(&appmodeldeploy.Record{
				ID:        recordID,
				Status:    appmodeldeploy.StatusDeployed,
				Message:   "deploy finished",
				StartedAt: startedAt,
				EndedAt:   startedAt.Add(time.Minute),
			})

			Expect(directStatus.Stage).To(Equal(string(autodeploy.StageDeploy)))
			Expect(directStatus.IsBuildAutoDeploy).To(BeFalse())
			Expect(directStatus.DeploySource).To(Equal(appmodeldeploy.DeploySourceDirectDeploy))
			Expect(directStatus.DeployID).To(Equal(recordID.Hex()))
			Expect(directStatus.BuildID).To(BeEmpty())
			Expect(directStatus.ImageTag).To(BeEmpty())
		})
	})

	Describe("AppModelDeployRecordOutputObj", func() {
		It("reads build info directly from deploy record extras", func() {
			output := new(serializer.AppModelDeployRecordOutputObj).FromModel(appmodeldeploy.Record{
				ID:        bson.NewObjectID(),
				ClusterID: "cls-1",
				Namespace: "default",
				ImageTag:  "v1.0.0",
				Replicas:  3,
				Message:   "deploying",
				Status:    appmodeldeploy.StatusDeploying,
				Updater:   "tester",
				Extras: map[string]string{
					appmodeldeploy.ExtraKeyDeploySource:  appmodeldeploy.DeploySourceBuildAutoDeploy,
					appmodeldeploy.ExtraKeyBuildBranch:   "release",
					appmodeldeploy.ExtraKeyBuildCommitID: "commit-123",
				},
			})

			Expect(output.IsBuildAutoDeploy).To(BeTrue())
			Expect(output.DeploySource).To(Equal(appmodeldeploy.DeploySourceBuildAutoDeploy))
			Expect(output.Branch).To(Equal("release"))
			Expect(output.CommitID).To(Equal("commit-123"))
		})
	})

	Describe("AppModelResourceSnapshot FromModel", func() {
		It("only includes manifest for detail output", func() {
			snapshot := appmodeldeploy.ResourceSnapshot{
				ID:          bson.NewObjectID(),
				APIVersion:  "apps/v1",
				Kind:        "Deployment",
				Name:        "bk-app",
				Manifest:    "kind: Deployment",
				IsTruncated: true,
				CreatedAt:   time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC),
			}

			listOutput := new(serializer.AppModelResourceSnapshot).FromModel(snapshot, false)
			listPayload, err := json.Marshal(listOutput)
			Expect(err).NotTo(HaveOccurred())
			Expect(listOutput.Manifest).To(BeNil())
			Expect(string(listPayload)).NotTo(ContainSubstring("manifest"))

			detailOutput := new(serializer.AppModelResourceSnapshot).FromModel(snapshot, true)
			Expect(detailOutput.Manifest).NotTo(BeNil())
			Expect(*detailOutput.Manifest).To(Equal("kind: Deployment"))
		})
	})
})
