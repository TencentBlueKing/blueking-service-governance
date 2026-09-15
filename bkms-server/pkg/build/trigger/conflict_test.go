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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func policyForm(mode BranchMatchMode, value, pathFilter string) PolicyForm {
	return PolicyForm{
		Name:             "candidate",
		Event:            EventPush,
		BranchMatchMode:  mode,
		BranchMatchValue: value,
		PathFilter:       pathFilter,
	}
}

func existingPolicy(name string, mode BranchMatchMode, value, pathFilter string) Policy {
	return Policy{
		ID:               "btp-existing",
		Name:             name,
		Event:            EventPush,
		BranchMatchMode:  mode,
		BranchMatchValue: value,
		PathFilter:       pathFilter,
		Status:           StatusEnabled,
	}
}

var _ = Describe("detectOverlap", func() {
	manager := &PolicyManager{}

	type overlapCase struct {
		name      string
		candidate PolicyForm
		existing  Policy
		wantType  OverlapType
	}

	cases := []overlapCase{
		{
			name:      "AC-009 eq values intersect",
			candidate: policyForm(BranchMatchModeEq, "master", ""),
			existing:  existingPolicy("P", BranchMatchModeEq, "master", ""),
			wantType:  OverlapTypeEqEq,
		},
		{
			name:      "AC-010 eq values do not intersect",
			candidate: policyForm(BranchMatchModeEq, "develop", ""),
			existing:  existingPolicy("P", BranchMatchModeEq, "master", ""),
		},
		{
			name:      "AC-011 eq hits prefix even when pathFilter differs",
			candidate: policyForm(BranchMatchModeEq, "feature/login", "src/**"),
			existing:  existingPolicy("P", BranchMatchModePrefix, "feature/", "docs/**"),
			wantType:  OverlapTypeEqHitsPrefix,
		},
		{
			name:      "AC-012 prefix values contain each other",
			candidate: policyForm(BranchMatchModePrefix, "feature/", ""),
			existing:  existingPolicy("P", BranchMatchModePrefix, "feat", ""),
			wantType:  OverlapTypePrefixPrefix,
		},
		{
			name:      "AC-013 any match overlaps all",
			candidate: policyForm(BranchMatchModeEq, "master", ""),
			existing:  existingPolicy("P", BranchMatchModeAll, "", ""),
			wantType:  OverlapTypeAll,
		},
		{
			name:      "candidate all overlaps existing eq",
			candidate: policyForm(BranchMatchModeAll, "", ""),
			existing:  existingPolicy("P", BranchMatchModeEq, "master", ""),
			wantType:  OverlapTypeAll,
		},
		{
			name:      "comma separated eq values intersect after trim",
			candidate: policyForm(BranchMatchModeEq, "master, develop", ""),
			existing:  existingPolicy("P", BranchMatchModeEq, "develop,release", ""),
			wantType:  OverlapTypeEqEq,
		},
		{
			name:      "empty segments in match values are ignored",
			candidate: policyForm(BranchMatchModeEq, "master,, ,", ""),
			existing:  existingPolicy("P", BranchMatchModeEq, "master", ""),
			wantType:  OverlapTypeEqEq,
		},
		{
			name:      "eq match is case sensitive",
			candidate: policyForm(BranchMatchModeEq, "Master", ""),
			existing:  existingPolicy("P", BranchMatchModeEq, "master", ""),
		},
		{
			name:      "different events do not overlap",
			candidate: PolicyForm{Event: Event("tag"), BranchMatchMode: BranchMatchModeAll},
			existing:  existingPolicy("P", BranchMatchModeAll, "", ""),
		},
		{
			name:      "eq does not hit unrelated prefix",
			candidate: policyForm(BranchMatchModeEq, "master", ""),
			existing:  existingPolicy("P", BranchMatchModePrefix, "feature/", ""),
		},
		{
			name:      "prefix vs eq is still eq_hits_prefix",
			candidate: policyForm(BranchMatchModePrefix, "feature/", ""),
			existing:  existingPolicy("P", BranchMatchModeEq, "feature/login", ""),
			wantType:  OverlapTypeEqHitsPrefix,
		},
	}

	for _, tc := range cases {
		tc := tc
		It(tc.name, func() {
			hit := manager.detectOverlap(tc.candidate, tc.existing)
			if tc.wantType == "" {
				Expect(hit).To(BeNil())
				return
			}
			Expect(hit).NotTo(BeNil())
			Expect(hit.OverlapType).To(Equal(tc.wantType))
			Expect(hit.PolicyName).To(Equal(tc.existing.Name))
			Expect(hit.Message).To(ContainSubstring(tc.existing.Name))
			Expect(hit.Message).To(ContainSubstring(conflictConsequence))
		})
	}
})

var _ = Describe("collectConflicts", func() {
	manager := &PolicyManager{}

	It("AC-014 does not compare a policy against itself", func() {
		existing := existingPolicy("P", BranchMatchModeEq, "master", "")
		existing.ID = "btp-self"
		form := policyForm(BranchMatchModeEq, "master", "")
		hits := manager.collectConflicts(form, []Policy{existing}, existing.ID)
		Expect(hits).To(BeEmpty())
	})

	It("AC-015 disabled policies still occupy conflict space", func() {
		existing := existingPolicy("P", BranchMatchModeEq, "master", "")
		existing.Status = StatusDisabled
		form := policyForm(BranchMatchModeEq, "master", "")
		hits := manager.collectConflicts(form, []Policy{existing}, "")
		Expect(hits).To(HaveLen(1))
		Expect(hits[0].OverlapType).To(Equal(OverlapTypeEqEq))
	})
})
