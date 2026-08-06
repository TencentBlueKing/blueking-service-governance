package types

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EnvVariableList", func() {
	Describe("ToDeduplicatedList", func() {
		It("should keep only the last item for duplicate keys while preserving effective item order", func() {
			vars := EnvVariableList{
				{Key: "SHARED_KEY", Value: "workspace-value"},
				{Key: "WORKSPACE_ONLY_KEY", Value: "workspace-only-value"},
				{Key: "SHARED_KEY", Value: "envtype-value"},
				{Key: "ENV_TYPE_ONLY_KEY", Value: "envtype-only-value"},
				{Key: "SHARED_KEY", Value: "env-value"},
				{Key: "ENV_ONLY_KEY", Value: "env-only-value"},
				{Key: "SHARED_KEY", Value: "app-value"},
				{Key: "APP_ONLY_KEY", Value: "app-only-value"},
			}

			Expect(vars.ToDeduplicatedList()).To(Equal(EnvVariableList{
				{Key: "WORKSPACE_ONLY_KEY", Value: "workspace-only-value"},
				{Key: "ENV_TYPE_ONLY_KEY", Value: "envtype-only-value"},
				{Key: "ENV_ONLY_KEY", Value: "env-only-value"},
				{Key: "SHARED_KEY", Value: "app-value"},
				{Key: "APP_ONLY_KEY", Value: "app-only-value"},
			}))
		})
	})
})

var _ = Describe("EnvVariableRichList", func() {
	Describe("ToDeduplicatedList", func() {
		It("should keep the last rich item for duplicate keys while preserving effective item order", func() {
			vars := EnvVariableRichList{
				Vars: []EnvVariableRichItem{
					{
						Obj:    EnvVariableObj{Key: "SHARED_KEY", Value: "workspace-value"},
						Source: ConflictedSource{Source: EnvVarSourceScopedWorkspace},
					},
					{
						Obj:    EnvVariableObj{Key: "WORKSPACE_ONLY_KEY", Value: "workspace-only-value"},
						Source: ConflictedSource{Source: EnvVarSourceScopedWorkspace},
					},
					{
						Obj:    EnvVariableObj{Key: "SHARED_KEY", Value: "app-value"},
						Source: ConflictedSource{Source: EnvVarSourceApp},
					},
					{
						Obj:    EnvVariableObj{Key: "APP_ONLY_KEY", Value: "app-only-value"},
						Source: ConflictedSource{Source: EnvVarSourceApp},
					},
				},
			}

			Expect(vars.ToDeduplicatedList()).To(Equal(EnvVariableRichList{
				Vars: []EnvVariableRichItem{
					{
						Obj:    EnvVariableObj{Key: "WORKSPACE_ONLY_KEY", Value: "workspace-only-value"},
						Source: ConflictedSource{Source: EnvVarSourceScopedWorkspace},
					},
					{
						Obj:    EnvVariableObj{Key: "SHARED_KEY", Value: "app-value"},
						Source: ConflictedSource{Source: EnvVarSourceApp},
					},
					{
						Obj:    EnvVariableObj{Key: "APP_ONLY_KEY", Value: "app-only-value"},
						Source: ConflictedSource{Source: EnvVarSourceApp},
					},
				},
			}))
		})
	})
})
