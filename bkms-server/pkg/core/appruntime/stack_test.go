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

package appruntime_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
)

var _ = Describe("Normalize", func() {
	It("maps consistent old tRPC fields to the same stack as new fields", func() {
		legacy := appruntime.Snapshot{
			AppID:              "app-1",
			AppType:            appruntime.AppTypeTRPC,
			TrpcSpecLanguage:   appruntime.LanguageGo,
			WorkloadType:       appruntime.FrameworkTRPC,
			TrpcConfigLanguage: appruntime.LanguageGo,
			HasTrpcConfig:      true,
			TrpcFileName:       "trpc_go.yaml",
			TrpcFilePath:       "/usr/local/trpc/bin/",
		}
		modern := appruntime.Snapshot{
			AppID:                  "app-1",
			AppType:                appruntime.AppTypeBKMSApp,
			Language:               appruntime.LanguageGo,
			Framework:              appruntime.FrameworkTRPC,
			FrameworkConfig:        map[string]any{"fileName": "trpc_go.yaml", "filePath": "/usr/local/trpc/bin/"},
			FrameworkConfigVersion: appruntime.FrameworkConfigVersionV1,
		}

		legacyStack, err := appruntime.Normalize(legacy)
		Expect(err).NotTo(HaveOccurred())
		modernStack, err := appruntime.Normalize(modern)
		Expect(err).NotTo(HaveOccurred())
		Expect(legacyStack).To(Equal(modernStack))
		Expect(legacyStack).To(Equal(appruntime.Stack{
			Type:      appruntime.AppTypeBKMSApp,
			Framework: appruntime.FrameworkTRPC,
			Language:  appruntime.LanguageGo,
		}))

		legacyReport := appruntime.Inspect(legacy)
		modernReport := appruntime.Inspect(modern)
		Expect(legacyReport.FrameworkConfig).To(Equal(modernReport.FrameworkConfig))
		Expect(legacyReport.FrameworkConfigVersion).To(Equal(appruntime.FrameworkConfigVersionV1))
	})

	It("uses the single valid language and records its source", func() {
		report := appruntime.Inspect(appruntime.Snapshot{
			AppType:            appruntime.AppTypeTRPC,
			TrpcConfigLanguage: appruntime.LanguageCpp,
			WorkloadType:       appruntime.FrameworkTRPC,
		})
		Expect(report.Blocking()).To(BeFalse())
		Expect(report.Stack.Language).To(Equal(appruntime.LanguageCpp))
		Expect(report.Sources.LanguageFrom).To(Equal("workload.trpcConfig.language"))
	})

	It("does not silently overwrite conflicting languages", func() {
		report := appruntime.Inspect(appruntime.Snapshot{
			AppType:            appruntime.AppTypeTRPC,
			TrpcSpecLanguage:   appruntime.LanguageGo,
			TrpcConfigLanguage: appruntime.LanguageCpp,
		})
		Expect(report.Blocking()).To(BeTrue())
		Expect(issueCodes(report)).To(ContainElement(appruntime.IssueLanguageConflict))
		_, err := appruntime.Normalize(appruntime.Snapshot{
			AppType:            appruntime.AppTypeTRPC,
			TrpcSpecLanguage:   appruntime.LanguageGo,
			TrpcConfigLanguage: appruntime.LanguageCpp,
		})
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, appruntime.ErrNormalizeFailed)).To(BeTrue())
	})

	It("does not treat empty workload type as tRPC", func() {
		report := appruntime.Inspect(appruntime.Snapshot{
			AppType:           appruntime.AppTypeTAF,
			WorkloadType:      "",
			HasTafConfig:      true,
			TafFileName:       "taf_config.conf",
			TafFilePath:       "/usr/local/taf/conf/",
			HasTafFileContent: true,
		})
		Expect(report.Stack.Framework).To(Equal(appruntime.FrameworkTAF))
		Expect(report.Stack.Language).To(BeEmpty())
		Expect(issueCodes(report)).To(ContainElement(appruntime.IssueIncompleteLanguage))
		Expect(report.Blocking()).To(BeFalse())
		Expect(report.FrameworkConfig).To(Equal(map[string]any{
			"fileName": "taf_config.conf",
			"filePath": "/usr/local/taf/conf/",
		}))
	})

	It("reports type and framework conflicts", func() {
		report := appruntime.Inspect(appruntime.Snapshot{
			AppType:      appruntime.AppTypeTRPC,
			Framework:    appruntime.FrameworkTAF,
			Language:     appruntime.LanguageCpp,
			WorkloadType: appruntime.FrameworkTRPC,
		})
		Expect(report.Blocking()).To(BeTrue())
		Expect(issueCodes(report)).To(ContainElement(appruntime.IssueTypeFrameworkConflict))
	})

	It("reports unknown leftover languages instead of coercing them", func() {
		report := appruntime.Inspect(appruntime.Snapshot{
			AppType:          appruntime.AppTypeTRPC,
			TrpcSpecLanguage: "java",
		})
		Expect(issueCodes(report)).To(ContainElement(appruntime.IssueUnknownLanguage))
		Expect(report.Stack.Language).To(BeEmpty())
	})

	It("keeps helm stacks unchanged and flags unexpected tech-stack fields", func() {
		ok, err := appruntime.Normalize(appruntime.Snapshot{AppType: appruntime.AppTypeHelm})
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(appruntime.Stack{Type: appruntime.AppTypeHelm}))

		report := appruntime.Inspect(appruntime.Snapshot{
			AppType:  appruntime.AppTypeHelm,
			Language: appruntime.LanguageGo,
		})
		Expect(issueCodes(report)).To(ContainElement(appruntime.IssueUnexpectedStackFields))
	})

	It("does not apply empty-string framework as blank", func() {
		report := appruntime.Inspect(appruntime.Snapshot{
			AppType:   appruntime.AppTypeBKMSApp,
			Language:  appruntime.LanguageGo,
			Framework: "",
		})
		Expect(issueCodes(report)).To(ContainElement(appruntime.IssueMissingFramework))
	})

	It("rejects dual legacy framework configs", func() {
		report := appruntime.Inspect(appruntime.Snapshot{
			AppType:       appruntime.AppTypeTRPC,
			Language:      appruntime.LanguageGo,
			HasTrpcConfig: true,
			HasTafConfig:  true,
		})
		Expect(issueCodes(report)).To(ContainElement(appruntime.IssueDualFrameworkConfig))
	})

	It("rejects conflicting new and old file metadata", func() {
		report := appruntime.Inspect(appruntime.Snapshot{
			AppType:                appruntime.AppTypeTRPC,
			Language:               appruntime.LanguageGo,
			HasTrpcConfig:          true,
			TrpcFileName:           "old.yaml",
			TrpcFilePath:           "/old/",
			FrameworkConfig:        map[string]any{"fileName": "new.yaml", "filePath": "/new/"},
			FrameworkConfigVersion: appruntime.FrameworkConfigVersionV1,
		})
		Expect(issueCodes(report)).To(ContainElement(appruntime.IssueFrameworkConfigConflict))
	})

	It("marks missing AppModel rows as orphan applications", func() {
		report := appruntime.Inspect(appruntime.Snapshot{
			AppType:         appruntime.AppTypeTRPC,
			Language:        appruntime.LanguageGo,
			AppModelMissing: true,
		})
		Expect(issueCodes(report)).To(ContainElement(appruntime.IssueOrphanApplication))
	})

	It("summarizes issues without copying file content", func() {
		snap := appruntime.Snapshot{
			AppID:              "app-conflict",
			AppType:            appruntime.AppTypeTRPC,
			TrpcSpecLanguage:   appruntime.LanguageGo,
			TrpcConfigLanguage: appruntime.LanguageCpp,
			HasTrpcFileContent: true,
		}
		summary := appruntime.Summarize([]appruntime.Report{appruntime.Inspect(snap)}, []appruntime.Snapshot{snap}, 5)
		Expect(summary.Blocking).To(Equal(1))
		Expect(summary.ByIssue[appruntime.IssueLanguageConflict]).To(Equal(1))
		Expect(summary.Samples).To(HaveLen(1))
		Expect(summary.Samples[0].HasTrpcFileContent).To(BeTrue())
		Expect(summary.Samples[0].IssueCodes).To(ContainElement(appruntime.IssueLanguageConflict))
	})

	It("rejects unknown frameworkConfig versions", func() {
		report := appruntime.Inspect(appruntime.Snapshot{
			AppType:                appruntime.AppTypeBKMSApp,
			Language:               appruntime.LanguageGo,
			Framework:              appruntime.FrameworkTRPC,
			FrameworkConfig:        map[string]any{"fileName": "trpc_go.yaml", "filePath": "/etc/"},
			FrameworkConfigVersion: 9,
		})
		Expect(issueCodes(report)).To(ContainElement(appruntime.IssueUnknownConfigVersion))
	})
})

var _ = Describe("ProjectDisplay and list query", func() {
	It("projects list language and framework without loading AppModel", func() {
		language, framework := appruntime.ProjectDisplay(appruntime.AppTypeTRPC, "", "", "go")
		Expect(language).To(Equal("go"))
		Expect(framework).To(Equal(appruntime.FrameworkTRPC))

		language, framework = appruntime.ProjectDisplay(appruntime.AppTypeTRPC, "cpp", "trpc", "go")
		Expect(language).To(Equal("cpp"))
		Expect(framework).To(Equal("trpc"))
	})

	It("expands legacy type filters to include future bkmsApp rows", func() {
		q := appruntime.ListTypeQuery(appruntime.AppTypeTRPC)
		Expect(q.Expand).To(BeTrue())
		Expect(q.LegacyType).To(Equal(appruntime.AppTypeTRPC))
		Expect(q.Framework).To(Equal(appruntime.FrameworkTRPC))

		helm := appruntime.ListTypeQuery(appruntime.AppTypeHelm)
		Expect(helm.Expand).To(BeFalse())
		Expect(helm.ExactType).To(Equal(appruntime.AppTypeHelm))
	})
})

func issueCodes(report appruntime.Report) []string {
	codes := make([]string, 0, len(report.Issues))
	for _, issue := range report.Issues {
		codes = append(codes, issue.Code)
	}
	return codes
}
