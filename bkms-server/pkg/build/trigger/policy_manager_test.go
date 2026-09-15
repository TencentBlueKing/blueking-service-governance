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

package trigger

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	imagebuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
)

type memoryPolicyStore struct {
	mu       sync.Mutex
	policies map[string]Policy
}

func newMemoryPolicyStore() *memoryPolicyStore {
	return &memoryPolicyStore{policies: make(map[string]Policy)}
}

func policyKey(appID, policyID string) string {
	return appID + "/" + policyID
}

func (s *memoryPolicyStore) Create(_ context.Context, policy *Policy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.policies {
		if existing.AppID == policy.AppID && existing.Name == policy.Name {
			return ErrPolicyNameDuplicated
		}
	}
	now := time.Now()
	policy.CreatedAt = now
	policy.UpdatedAt = now
	copied := *policy
	s.policies[policyKey(policy.AppID, policy.ID)] = copied
	return nil
}

func (s *memoryPolicyStore) Update(_ context.Context, policy *Policy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := policyKey(policy.AppID, policy.ID)
	current, ok := s.policies[key]
	if !ok {
		return ErrPolicyNotFound
	}
	for _, existing := range s.policies {
		if existing.AppID == policy.AppID && existing.Name == policy.Name && existing.ID != policy.ID {
			return ErrPolicyNameDuplicated
		}
	}
	current.Name = policy.Name
	current.Event = policy.Event
	current.BranchMatchMode = policy.BranchMatchMode
	current.BranchMatchValue = policy.BranchMatchValue
	current.PathFilter = policy.PathFilter
	current.PipelineID = policy.PipelineID
	current.TriggerID = policy.TriggerID
	current.UpdatedAt = time.Now()
	s.policies[key] = current
	return nil
}

func (s *memoryPolicyStore) UpdateStatus(_ context.Context, appID, policyID string, status Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := policyKey(appID, policyID)
	current, ok := s.policies[key]
	if !ok {
		return ErrPolicyNotFound
	}
	current.Status = status
	current.UpdatedAt = time.Now()
	s.policies[key] = current
	return nil
}

func (s *memoryPolicyStore) Get(_ context.Context, appID, policyID string) (*Policy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.policies[policyKey(appID, policyID)]
	if !ok {
		return nil, ErrPolicyNotFound
	}
	copied := current
	return &copied, nil
}

func (s *memoryPolicyStore) List(_ context.Context, appID string) ([]Policy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	listed := make([]Policy, 0)
	for _, policy := range s.policies {
		if policy.AppID == appID {
			listed = append(listed, policy)
		}
	}
	return listed, nil
}

func (s *memoryPolicyStore) Delete(_ context.Context, appID, policyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := policyKey(appID, policyID)
	if _, ok := s.policies[key]; !ok {
		return ErrPolicyNotFound
	}
	delete(s.policies, key)
	return nil
}

type fakeConfigStore struct {
	cfg *imagebuild.Config
}

func (s *fakeConfigStore) Create(context.Context, *imagebuild.Config) error { return nil }
func (s *fakeConfigStore) Update(context.Context, *imagebuild.Config) error { return nil }
func (s *fakeConfigStore) Delete(context.Context, string) error             { return nil }
func (s *fakeConfigStore) Get(_ context.Context, _ string) (*imagebuild.Config, error) {
	if s.cfg == nil {
		return nil, errors.New("build config not found")
	}
	copied := *s.cfg
	if s.cfg.CodeRepo != nil {
		repo := *s.cfg.CodeRepo
		copied.CodeRepo = &repo
	}
	return &copied, nil
}

type stubPipelineOps struct {
	ensureCalls  int
	cleanupCalls int
}

var (
	_ PolicyStore            = &memoryPolicyStore{}
	_ imagebuild.ConfigStore = &fakeConfigStore{}
	_ PipelineOps            = &stubPipelineOps{}
)

func (s *stubPipelineOps) Ensure(
	_ context.Context,
	_, _ string,
) (*bkci.Pipeline, error) {
	s.ensureCalls++
	return &bkci.Pipeline{ID: "p-trigger-1"}, nil
}

func (s *stubPipelineOps) Cleanup(_ context.Context, _, _ string) error {
	s.cleanupCalls++
	return nil
}

func eligibleBuildConfig(appID string) *imagebuild.Config {
	return &imagebuild.Config{
		AppID:      appID,
		SourceType: imagebuild.SourceTypeCodeRepository,
		TagConfig:  imagebuild.TagConfig{Type: imagebuild.VersionTypeSemver},
		CodeRepo: &imagebuild.RepositoryConfig{
			Type:          imagebuild.RepositoryTypeTGit,
			RepoAlias:     "group/app",
			RepoURL:       "https://git.example.com/group/app.git",
			DefaultBranch: "master",
		},
	}
}

func eqPolicyForm(name, branch string) PolicyForm {
	return PolicyForm{
		Name:             name,
		Event:            EventPush,
		BranchMatchMode:  BranchMatchModeEq,
		BranchMatchValue: branch,
	}
}

var _ = Describe("PolicyManager", func() {
	var (
		ctx       context.Context
		app       *bkmsapp.Application
		policies  *memoryPolicyStore
		cfgStore  *fakeConfigStore
		pipelines *stubPipelineOps
		manager   *PolicyManager
	)

	BeforeEach(func() {
		ctx = context.Background()
		app = &bkmsapp.Application{
			ID:          "app-" + stringx.Random(8),
			WorkspaceID: "ws-1",
		}
		policies = newMemoryPolicyStore()
		cfgStore = &fakeConfigStore{cfg: eligibleBuildConfig(app.ID)}
		pipelines = &stubPipelineOps{}
		manager = NewPolicyManager(policies, cfgStore, pipelines)
	})

	AfterEach(func() {
		mockey.UnPatchAll()
	})

	It("does not persist when first Ensure fails", func() {
		mockey.Mock((*stubPipelineOps).Ensure).Return(nil, errors.New("ensure failed")).Build()

		_, err := manager.Create(ctx, app, "alice", eqPolicyForm("p1", "master"))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("ensure trigger pipeline"))

		listed, listErr := policies.List(ctx, app.ID)
		Expect(listErr).NotTo(HaveOccurred())
		Expect(listed).To(BeEmpty())
	})

	It("does not delete the last policy when Cleanup fails", func() {
		created, err := manager.Create(ctx, app, "alice", eqPolicyForm("p1", "master"))
		Expect(err).NotTo(HaveOccurred())
		mockey.Mock((*stubPipelineOps).Cleanup).Return(errors.New("cleanup failed")).Build()

		err = manager.Delete(ctx, app.WorkspaceID, app.ID, created.ID)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cleanup trigger pipeline"))

		_, getErr := policies.Get(ctx, app.ID, created.ID)
		Expect(getErr).NotTo(HaveOccurred())
	})

	It("rejects the sixth policy", func() {
		for i := 1; i <= MaxPoliciesPerApp; i++ {
			_, err := manager.Create(ctx, app, "alice", eqPolicyForm(
				fmt.Sprintf("p%d", i),
				fmt.Sprintf("branch-%d", i),
			))
			Expect(err).NotTo(HaveOccurred())
		}
		_, err := manager.Create(ctx, app, "alice", eqPolicyForm("overflow", "other"))
		Expect(err).To(MatchError(ErrTooManyPolicies))
		Expect(pipelines.ensureCalls).To(Equal(1))
	})

	It("reuses pipelineID for the second policy without calling Ensure again", func() {
		first, err := manager.Create(ctx, app, "alice", eqPolicyForm("p1", "master"))
		Expect(err).NotTo(HaveOccurred())
		second, err := manager.Create(ctx, app, "alice", eqPolicyForm("p2", "develop"))
		Expect(err).NotTo(HaveOccurred())
		Expect(second.PipelineID).To(Equal(first.PipelineID))
		Expect(pipelines.ensureCalls).To(Equal(1))
	})

	It("rejects unsupported app source type", func() {
		cfgStore.cfg.SourceType = imagebuild.SourceTypeImageRegistry
		cfgStore.cfg.CodeRepo = nil
		_, err := manager.Create(ctx, app, "alice", eqPolicyForm("p1", "master"))
		Expect(err).To(MatchError(ErrUnsupportedAppType))
		Expect(pipelines.ensureCalls).To(Equal(0))
	})

	It("rejects create and update when auto generate tag is disabled", func() {
		created, err := manager.Create(ctx, app, "alice", eqPolicyForm("p1", "master"))
		Expect(err).NotTo(HaveOccurred())
		cfgStore.cfg.TagConfig = imagebuild.TagConfig{}

		_, err = manager.Create(ctx, app, "alice", eqPolicyForm("p2", "develop"))
		Expect(err).To(MatchError(ErrAutoGenerateTagDisabled))

		_, err = manager.Update(ctx, app.ID, created.ID, eqPolicyForm("p1", "release"))
		Expect(err).To(MatchError(ErrAutoGenerateTagDisabled))
	})

	It("rejects all match mode with a non-empty value", func() {
		_, err := manager.Create(ctx, app, "alice", PolicyForm{
			Name:             "p-all",
			Event:            EventPush,
			BranchMatchMode:  BranchMatchModeAll,
			BranchMatchValue: "master",
		})
		Expect(err).To(MatchError(ErrInvalidBranchMatch))
		Expect(err.Error()).To(Equal("匹配方式为全部时匹配值必须为空"))
	})

	It("rejects eq match mode without values", func() {
		_, err := manager.Create(ctx, app, "alice", PolicyForm{
			Name:            "p-eq",
			Event:           EventPush,
			BranchMatchMode: BranchMatchModeEq,
		})
		Expect(err).To(MatchError(ErrInvalidBranchMatch))
	})

	Describe("GuardBuildConfigUpdate", func() {
		var before, after *imagebuild.Config

		BeforeEach(func() {
			_, err := manager.Create(ctx, app, "alice", eqPolicyForm("p1", "master"))
			Expect(err).NotTo(HaveOccurred())
			before = eligibleBuildConfig(app.ID)
			copied := *before
			repo := *before.CodeRepo
			copied.CodeRepo = &repo
			after = &copied
		})

		It("rejects changing repoAlias", func() {
			after.CodeRepo.RepoAlias = "group/other"
			Expect(manager.GuardBuildConfigUpdate(ctx, app.ID, before, after)).To(MatchError(ErrBuildConfigLocked))
		})

		It("rejects changing sourceType", func() {
			after.SourceType = imagebuild.SourceTypeImageRegistry
			Expect(manager.GuardBuildConfigUpdate(ctx, app.ID, before, after)).To(MatchError(ErrBuildConfigLocked))
		})

		It("rejects disabling auto generate tag", func() {
			after.TagConfig = imagebuild.TagConfig{}
			Expect(manager.GuardBuildConfigUpdate(ctx, app.ID, before, after)).To(MatchError(ErrBuildConfigLocked))
		})

		It("allows repoURL, default branch and semver to custom", func() {
			after.CodeRepo.RepoURL = "https://git.example.com/group/app-renamed.git"
			after.CodeRepo.DefaultBranch = "main"
			after.TagConfig = imagebuild.TagConfig{Type: imagebuild.VersionTypeCustom}
			Expect(manager.GuardBuildConfigUpdate(ctx, app.ID, before, after)).To(Succeed())
		})

		It("allows locked field changes when no policy exists", func() {
			Expect(manager.Delete(ctx, app.WorkspaceID, app.ID, mustListFirst(ctx, policies, app.ID))).To(Succeed())
			after.CodeRepo.RepoAlias = "group/other"
			Expect(manager.GuardBuildConfigUpdate(ctx, app.ID, before, after)).To(Succeed())
		})
	})
})

func mustListFirst(ctx context.Context, store PolicyStore, appID string) string {
	listed, err := store.List(ctx, appID)
	Expect(err).NotTo(HaveOccurred())
	Expect(listed).NotTo(BeEmpty())
	return listed[0].ID
}
