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
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	deploypkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	releasestrategyserializer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
)

var _ = Describe("AppModel deploy serializers", func() {
	Describe("DeployPreCheckOutput", func() {
		It("converts undefined vars and their sources", func() {
			output := new(serializer.DeployPreCheckOutput).FromModel(&deploypkg.DeployPreCheckResult{
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

			Expect(output).To(Equal(&serializer.DeployPreCheckOutput{
				UndefinedVars: []serializer.UndefinedEnvVarOutput{
					{
						Key: "DB_HOST",
						Sources: []serializer.EnvVarReferenceSourceOutput{
							{Type: "appConfigFile", Name: "production"},
							{Type: "component", Name: "redis-config"},
						},
					},
				},
				MissingRequiredClusterAddons: []serializer.ClusterAddonReferenceOutput{},
			}))
		})

		It("converts missing required cluster addon names", func() {
			output := new(serializer.DeployPreCheckOutput).FromModel(&deploypkg.DeployPreCheckResult{
				MissingRequiredClusterAddons: []clusteraddon.AddonReference{
					{Name: "game", DisplayName: "Gamedeploy"},
					{Name: "hook", DisplayName: "Hook-operator"},
					{Name: "other"},
				},
			})
			payload, err := json.Marshal(output)
			Expect(err).NotTo(HaveOccurred())
			Expect(payload).To(MatchJSON(`{"undefinedVars":[],"missingRequiredClusterAddons":[
				{"name":"game","displayName":"Gamedeploy"},
				{"name":"hook","displayName":"Hook-operator"},
				{"name":"other","displayName":""}
			]}`))
		})

		It("serializes empty undefined vars as an empty array", func() {
			output := new(serializer.DeployPreCheckOutput).FromModel(&deploypkg.DeployPreCheckResult{})

			payload, err := json.Marshal(output)
			Expect(err).NotTo(HaveOccurred())
			Expect(payload).To(MatchJSON(`{"undefinedVars":[],"missingRequiredClusterAddons":[]}`))
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
			Expect(buildStatus.RevisionType).To(Equal("branch"))
			Expect(buildStatus.Revision).To(Equal("main"))
			Expect(buildStatus.Branch).To(Equal("main"))
			Expect(buildStatus.ImageTag).To(Equal("v2.0.0"))
			Expect(buildStatus.Operator).To(Equal("builder"))
			Expect(buildStatus.PipelineID).To(Equal("pipeline-1"))
			Expect(buildStatus.ReleaseMode).To(Equal("all"))

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
			Expect(directStatus.ReleaseMode).To(Equal("all"))
		})

		It("exposes the release mode of a batch release deploy record", func() {
			status := new(serializer.LatestDeployStatus).FromDeployRecord(&appmodeldeploy.Record{
				ID:          bson.NewObjectID(),
				Status:      appmodeldeploy.StatusDeploying,
				ReleaseMode: appmodeldeploy.ReleaseModeCanary,
			})

			Expect(status.Status).To(Equal("deploying"))
			Expect(status.ReleaseMode).To(Equal("canary"))
		})

		It("keeps tag and commit out of the legacy branch field", func() {
			status := new(serializer.LatestDeployStatus).FromBuildAutoDeployRecord(&autodeploy.Record{
				RevisionType: "tag",
				Revision:     "v1.2.0",
				Branch:       "v1.2.0",
			})

			Expect(status.RevisionType).To(Equal("tag"))
			Expect(status.Revision).To(Equal("v1.2.0"))
			Expect(status.Branch).To(BeEmpty())
		})
	})

	Describe("AppModelDeployRecordOutputObj", func() {
		It("reads build info directly from deploy record extras", func() {
			output := new(serializer.AppModelDeployRecordOutputObj).FromModel(appmodeldeploy.Record{
				ID:           bson.NewObjectID(),
				ClusterID:    "cls-1",
				Namespace:    "default",
				ImageTag:     "v1.0.0",
				Replicas:     3,
				WorkloadKind: "Deployment",
				Message:      "deploying",
				Status:       appmodeldeploy.StatusDeploying,
				Updater:      "tester",
				Extras: map[string]string{
					appmodeldeploy.ExtraKeyDeploySource:  appmodeldeploy.DeploySourceBuildAutoDeploy,
					appmodeldeploy.ExtraKeyBuildBranch:   "release",
					appmodeldeploy.ExtraKeyBuildCommitID: "commit-123",
				},
			})

			Expect(output.IsBuildAutoDeploy).To(BeTrue())
			Expect(output.DeploySource).To(Equal(appmodeldeploy.DeploySourceBuildAutoDeploy))
			Expect(output.RevisionType).To(Equal("branch"))
			Expect(output.Revision).To(Equal("release"))
			Expect(output.Branch).To(Equal("release"))
			Expect(output.Replicas).To(Equal(int32(3)))
			Expect(output.WorkloadKind).To(Equal("Deployment"))
			Expect(output.CommitID).To(Equal("commit-123"))
			Expect(output.ReleaseMode).To(Equal("all"))
			Expect(output.ReleaseStrategyName).To(BeEmpty())
		})

		It("returns the release mode and strategy name of a batch release record", func() {
			output := new(serializer.AppModelDeployRecordOutputObj).FromModel(appmodeldeploy.Record{
				ID:              bson.NewObjectID(),
				Status:          appmodeldeploy.StatusDeploying,
				ReleaseMode:     appmodeldeploy.ReleaseModeCanary,
				ReleaseStrategy: &appmodeldeploy.ReleaseStrategySnapshot{ID: "strategy-1", Name: "careful"},
			})

			Expect(output.ReleaseMode).To(Equal("canary"))
			Expect(output.ReleaseStrategyName).To(Equal("careful"))
		})

		It("reads commit revision without presenting it as a branch", func() {
			output := new(serializer.AppModelDeployRecordOutputObj).FromModel(appmodeldeploy.Record{
				ID: bson.NewObjectID(),
				Extras: map[string]string{
					appmodeldeploy.ExtraKeyDeploySource:      appmodeldeploy.DeploySourceBuildAutoDeploy,
					appmodeldeploy.ExtraKeyBuildRevisionType: "commit",
					appmodeldeploy.ExtraKeyBuildRevision:     "a1b2c3d4e5f6",
					appmodeldeploy.ExtraKeyBuildCommitID:     "a1b2c3d4e5f6",
				},
			})

			Expect(output.RevisionType).To(Equal("commit"))
			Expect(output.Revision).To(Equal("a1b2c3d4e5f6"))
			Expect(output.Branch).To(BeEmpty())
			Expect(output.CommitID).To(Equal("a1b2c3d4e5f6"))
		})

		It("infers WorkloadKind from ResourceKeys when the field is empty", func() {
			output := new(serializer.AppModelDeployRecordOutputObj).FromModel(appmodeldeploy.Record{
				ID:        bson.NewObjectID(),
				ClusterID: "cls-1",
				Namespace: "default",
				ResourceKeys: appmodeldeploy.ResourceKeys{
					{Kind: "Service", Name: "demo"},
					{Kind: "GameDeployment", Name: "demo"},
				},
			})
			Expect(output.WorkloadKind).To(Equal("GameDeployment"))
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

	Describe("BatchReleaseProgressOutputObj FromModel", func() {
		var record *appmodeldeploy.Record

		BeforeEach(func() {
			record = &appmodeldeploy.Record{
				ID:          bson.NewObjectID(),
				ImageTag:    "v2",
				Replicas:    4,
				Status:      appmodeldeploy.StatusDeploying,
				ReleaseMode: appmodeldeploy.ReleaseModeCanary,
				ReleaseStrategy: &appmodeldeploy.ReleaseStrategySnapshot{
					ID:   "strategy-1",
					Name: "careful",
					Batches: []releasestrategy.Batch{
						{Target: "1", PauseMode: releasestrategy.PauseModePermanent},
						{Target: "50%", PauseMode: releasestrategy.PauseModeDuration, PauseSeconds: 600},
					},
				},
			}
		})

		It("returns the progress with the strategy snapshot", func() {
			pauseStartTime := time.Date(2026, 10, 22, 2, 0, 0, 0, time.UTC)
			record.BatchProgress = &appmodeldeploy.BatchProgress{
				CurrentBatchIndex: lo.ToPtr(int64(0)),
				BatchPhase:        batchrelease.BatchPhaseManualPaused,
				RolloutPhase:      "Running",
				Message:           "Rollout is running canary step 0",
				PauseStartTime:    &pauseStartTime,
				UpdatedReplicas:   1,
				Replicas:          4,
				OldImageTag:       "v1",
				NewImageTag:       "v2",
			}

			output := new(serializer.BatchReleaseProgressOutputObj).FromModel(record)
			Expect(output).To(Equal(&serializer.BatchReleaseProgressOutputObj{
				DeployID:     record.ID.Hex(),
				Status:       "deploying",
				StrategyID:   "strategy-1",
				StrategyName: "careful",
				Batches: []releasestrategyserializer.BatchOutputObj{
					{Target: "1", PauseMode: "permanent"},
					{Target: "50%", PauseMode: "duration", PauseSeconds: 600},
				},
				CurrentBatchIndex: lo.ToPtr(int64(0)),
				BatchPhase:        "manual-paused",
				RolloutPhase:      "Running",
				Message:           "Rollout is running canary step 0",
				PauseStartTime:    &pauseStartTime,
				UpdatedReplicas:   1,
				Replicas:          4,
				OldImageTag:       "v1",
				NewImageTag:       "v2",
			}))
		})

		It("keeps a missing batch index as null rather than 0", func() {
			record.BatchProgress = &appmodeldeploy.BatchProgress{BatchPhase: batchrelease.BatchPhaseUpdating}

			payload, err := json.Marshal(new(serializer.BatchReleaseProgressOutputObj).FromModel(record))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(payload)).To(ContainSubstring(`"currentBatchIndex":null`))
			Expect(string(payload)).To(ContainSubstring(`"pauseStartTime":null`))
		})

		It("falls back to a pending progress before the poller writes one", func() {
			output := new(serializer.BatchReleaseProgressOutputObj).FromModel(record)
			Expect(output.BatchPhase).To(Equal("pending"))
			Expect(output.CurrentBatchIndex).To(BeNil())
			Expect(output.Replicas).To(Equal(int32(4)))
			Expect(output.NewImageTag).To(Equal("v2"))
			Expect(output.StrategyName).To(Equal("careful"))
		})
	})
})
