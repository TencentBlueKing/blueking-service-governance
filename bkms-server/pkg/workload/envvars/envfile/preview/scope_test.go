package preview_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	parserpkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/parser"
	previewpkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/preview"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ func(parserpkg.ParsedEnvVarRecord) (*previewpkg.RecordResolution, error) = previewpkg.ResolvePublicRecord

var _ = Describe("Record resolvers", func() {
	It("resolves public record scope from explicit workspace metadata", func() {
		res, err := previewpkg.ResolvePublicRecord(parserpkg.ParsedEnvVarRecord{
			DeclaredScopeType: lo.ToPtr(string(envvartypes.ScopeTypeWorkspace)),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(res).NotTo(BeNil())
		Expect(res.EffectiveScope.ScopeType).To(Equal(envvartypes.ScopeTypeWorkspace))
		Expect(res.EffectStatus).To(Equal(previewpkg.ImportEffectScopeApplied))
	})

	It("rejects env scope in public import", func() {
		_, err := previewpkg.ResolvePublicRecord(parserpkg.ParsedEnvVarRecord{
			DeclaredScopeType:  lo.ToPtr(string(envvartypes.ScopeTypeEnv)),
			DeclaredScopeValue: lo.ToPtr("prod-env"),
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`scopeType "env" is not allowed in public import`))
	})

	It("uses page context for env import without requiring metadata", func() {
		resolve := previewpkg.NewEnvRecordResolver(envmodel.Environment{Name: "prod-env"})
		res, err := resolve(parserpkg.ParsedEnvVarRecord{})
		Expect(err).NotTo(HaveOccurred())
		Expect(res).NotTo(BeNil())
		Expect(res.EffectiveScope.ScopeType).To(Equal(envvartypes.ScopeTypeEnv))
		Expect(res.EffectiveScope.ScopeValue).To(Equal("prod-env"))
	})

	It("rejects scope metadata in env import records", func() {
		resolve := previewpkg.NewEnvRecordResolver(envmodel.Environment{Name: "prod-env"})
		_, err := resolve(parserpkg.ParsedEnvVarRecord{
			DeclaredScopeType: lo.ToPtr(string(envvartypes.ScopeTypeWorkspace)),
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("env import does not allow scope metadata"))
	})

	It("rejects scope metadata in app import records", func() {
		_, err := previewpkg.ResolveAppRecord(parserpkg.ParsedEnvVarRecord{
			DeclaredScopeType: lo.ToPtr(string(envvartypes.ScopeTypeWorkspace)),
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("app import does not allow scope metadata"))
	})

	It("surfaces invalid env file content errors through the exported sentinel", func() {
		_, err := parserpkg.ParseEnvFileRecords("BAD LINE")
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
	})
})
