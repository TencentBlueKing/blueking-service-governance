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

package helm

import (
	"context"
	"errors"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"helm.sh/helm/v3/pkg/action"
	helmrelease "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/secret"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	bkmshelm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/arrangement"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

var _ = Describe("genHelmReleaseSpec", func() {
	var (
		app *bkmsapp.Application
		env *bkmsenv.Environment
	)

	BeforeEach(func() {
		// 初始化有效的 Application
		app = &bkmsapp.Application{
			WorkspaceID: "test-workspace",
			Name:        "test-app",
			DisplayName: "Test Application",
			Type:        bkmsapp.AppTypeHelm,
			HelmSpec: &bkmsapp.HelmSpec{
				HelmSource: &bkmsapp.HelmSource{
					RepoType: "HelmRepo",
					HelmRepoConfig: &bkmsapp.HelmRepoConfig{
						RepoURL:   "https://charts.example.com",
						ChartName: "my-chart",
					},
				},
			},
		}

		// 初始化有效的 Env
		env = &bkmsenv.Environment{
			Name:        "dev",
			DisplayName: "Development",
			Type:        "development",
			Cluster: bkmsenv.BizCluster{
				ProjectCode: "bkms-test-workspace",
				ClusterID:   "BCS-K8S-00000",
				Namespace:   "test-namespace",
			},
		}
	})

	Context("when parameters are valid", func() {
		Context("without traffic lane", func() {
			It("should successfully generate ReleaseSpec", func() {
				spec, err := genHelmReleaseSpec(app, env, "", "my-chart")
				Expect(err).NotTo(HaveOccurred())
				Expect(spec).NotTo(BeNil())

				// 验证基本字段
				Expect(spec.ProjectCode).To(Equal("bkms-test-workspace"))
				Expect(spec.ClusterID).To(Equal("BCS-K8S-00000"))
				Expect(spec.Namespace).To(Equal("test-namespace"))
				Expect(spec.ReleaseName).To(Equal("dev-test-app"))
				Expect(spec.ChartRepoName).To(Equal("bkms-test-workspace"))
				Expect(spec.ChartName).To(Equal("my-chart"))
				Expect(spec.TrafficLaneName).To(Equal(""))
			})
		})

		Context("with GitRepo chart name", func() {
			It("should use resolved chart name from caller", func() {
				app.HelmSpec.HelmSource = &bkmsapp.HelmSource{
					RepoType: bkmsapp.HelmSourceRepoTypeGit,
					GitRepoConfig: &bkmsapp.GitRepoConfig{
						Type:      bkmsapp.GitRepoTypeTGit,
						RepoURL:   "https://git.example.com/group/repo.git",
						Revision:  "main",
						SourceDir: "charts/test-app",
					},
				}

				spec, err := genHelmReleaseSpec(app, env, "", "test-app")
				Expect(err).NotTo(HaveOccurred())
				Expect(spec.ChartName).To(Equal("test-app"))
				Expect(spec.ReleaseName).To(Equal("dev-test-app"))
			})
		})

		Context("with traffic lane", func() {
			It("should include traffic lane name in ReleaseName", func() {
				trafficLaneName := "lane-01"
				spec, err := genHelmReleaseSpec(app, env, trafficLaneName, "my-chart")
				Expect(err).NotTo(HaveOccurred())
				Expect(spec).NotTo(BeNil())

				// 验证 ReleaseName 包含泳道名称
				Expect(spec.ReleaseName).To(Equal("dev-lane-01-test-app"))
				Expect(spec.TrafficLaneName).To(Equal(trafficLaneName))
			})
		})

		Context("when ReleaseName exceeds 53 characters", func() {
			It("should truncate ReleaseName to 53 characters", func() {
				// 创建一个会导致超长 ReleaseName 的场景
				app.Name = "very-long-application-name-that-exceeds-limit"
				env.Name = "production-environment"
				trafficLaneName := "traffic-lane-with-long-name"

				spec, err := genHelmReleaseSpec(app, env, trafficLaneName, "my-chart")
				Expect(err).NotTo(HaveOccurred())
				Expect(spec).NotTo(BeNil())

				// 验证 ReleaseName 长度不超过 53
				Expect(len(spec.ReleaseName)).To(BeNumerically("<=", 53))
				Expect(spec.ReleaseName).To(HaveLen(53))
			})
		})
	})

	Context("when Application parameters are invalid", func() {
		Context("when app is nil", func() {
			It("should return an error", func() {
				spec, err := genHelmReleaseSpec(nil, env, "", "my-chart")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid app"))
				Expect(err.Error()).To(ContainSubstring("app is nil"))
				Expect(spec).To(BeNil())
			})
		})

		Context("when HelmSpec is nil", func() {
			It("should return an error", func() {
				app.HelmSpec = nil
				spec, err := genHelmReleaseSpec(app, env, "", "my-chart")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid app"))
				Expect(err.Error()).To(ContainSubstring("app helm spec is nil"))
				Expect(spec).To(BeNil())
			})
		})

		Context("when HelmSource is nil", func() {
			It("should return an error", func() {
				app.HelmSpec.HelmSource = nil
				spec, err := genHelmReleaseSpec(app, env, "", "my-chart")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid app"))
				Expect(err.Error()).To(ContainSubstring("app helm source is nil"))
				Expect(spec).To(BeNil())
			})
		})

		Context("when HelmRepoConfig is nil", func() {
			It("should return an error", func() {
				app.HelmSpec.HelmSource.HelmRepoConfig = nil
				spec, err := genHelmReleaseSpec(app, env, "", "my-chart")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid app"))
				Expect(err.Error()).To(ContainSubstring("app helm repo config is nil"))
				Expect(spec).To(BeNil())
			})
		})

		Context("when RepoURL is empty", func() {
			It("should return an error", func() {
				app.HelmSpec.HelmSource.HelmRepoConfig.RepoURL = ""
				spec, err := genHelmReleaseSpec(app, env, "", "my-chart")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid app"))
				Expect(err.Error()).To(ContainSubstring("app helm repo config repo url is empty"))
				Expect(spec).To(BeNil())
			})
		})

		Context("when ChartName is empty", func() {
			It("should return an error", func() {
				app.HelmSpec.HelmSource.HelmRepoConfig.ChartName = ""
				spec, err := genHelmReleaseSpec(app, env, "", "my-chart")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid app"))
				Expect(err.Error()).To(ContainSubstring("app helm repo config chart name is empty"))
				Expect(spec).To(BeNil())
			})
		})
	})

	Context("when Env parameters are invalid", func() {
		Context("when env is nil", func() {
			It("should return an error", func() {
				spec, err := genHelmReleaseSpec(app, nil, "", "my-chart")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid env"))
				Expect(err.Error()).To(ContainSubstring("env is nil"))
				Expect(spec).To(BeNil())
			})
		})

		Context("when BkProject is empty", func() {
			It("should return an error", func() {
				env.Cluster.ProjectCode = ""
				spec, err := genHelmReleaseSpec(app, env, "", "my-chart")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("invalid env: env cluster project code is empty"))
				Expect(spec).To(BeNil())
			})
		})

		Context("when DefaultCluster is empty", func() {
			It("should return an error", func() {
				env.Cluster.ClusterID = ""
				spec, err := genHelmReleaseSpec(app, env, "", "my-chart")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("invalid env: env cluster id is empty"))
				Expect(spec).To(BeNil())
			})
		})

		Context("when DefaultNamespace is empty", func() {
			It("should return an error", func() {
				env.Cluster.Namespace = ""
				spec, err := genHelmReleaseSpec(app, env, "", "my-chart")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("invalid env: env cluster namespace is empty"))
				Expect(spec).To(BeNil())
			})
		})
	})
})

var _ = Describe("validateApp", func() {
	var app *bkmsapp.Application

	BeforeEach(func() {
		app = &bkmsapp.Application{
			HelmSpec: &bkmsapp.HelmSpec{
				HelmSource: &bkmsapp.HelmSource{
					RepoType: bkmsapp.HelmSourceRepoTypeHelm,
					HelmRepoConfig: &bkmsapp.HelmRepoConfig{
						RepoURL:   "https://charts.example.com",
						ChartName: "my-chart",
					},
				},
			},
		}
	})

	It("should pass for HelmRepo config", func() {
		Expect(validateApp(app)).To(Succeed())
	})

	It("should pass for GitRepo config without HelmRepoConfig", func() {
		app.HelmSpec.HelmSource = &bkmsapp.HelmSource{
			RepoType: bkmsapp.HelmSourceRepoTypeGit,
			GitRepoConfig: &bkmsapp.GitRepoConfig{
				Type:      bkmsapp.GitRepoTypeTGit,
				RepoURL:   "https://git.example.com/group/repo.git",
				Revision:  "main",
				SourceDir: "charts/test-app",
			},
		}

		Expect(validateApp(app)).To(Succeed())
	})

	It("should reject GitRepo without GitRepoConfig", func() {
		app.HelmSpec.HelmSource = &bkmsapp.HelmSource{RepoType: bkmsapp.HelmSourceRepoTypeGit}

		err := validateApp(app)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("app helm source git repo config is nil"))
	})

	It("should reject unsupported BCSRepo", func() {
		app.HelmSpec.HelmSource = &bkmsapp.HelmSource{RepoType: bkmsapp.HelmSourceRepoTypeBCS}

		err := validateApp(app)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("app helm source repo type BCSRepo is not supported"))
	})

	It("should reject invalid RepoType", func() {
		app.HelmSpec.HelmSource = &bkmsapp.HelmSource{RepoType: "UnknownRepo"}

		err := validateApp(app)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("app helm source repo type UnknownRepo is invalid"))
	})
})

var _ = Describe("genValuesVariables", func() {
	var (
		ctx         context.Context
		workspaceID string
		appID       string
		appName     string
		imageTag    string
	)

	// mockDatabaseAndBuildStore mocks database client and build config store
	mockDatabaseAndBuildStore := func() *build.ConfigStoreMongo {
		mockey.Mock(database.Client).Return(&mongo.Client{}).Build()
		mockey.Mock(database.Name).Return("testdb").Build()

		mockStore := &build.ConfigStoreMongo{}
		mockey.Mock(build.NewConfigStoreMongo).Return(mockStore, nil).Build()

		return mockStore
	}

	// mockBuildConfigWithCodeRepo mocks build config with code repository source type
	mockBuildConfigWithCodeRepo := func(appID string) {
		mockBuildCfg := &build.Config{
			AppID:      appID,
			SourceType: build.SourceTypeCodeRepository,
			CodeRepo: &build.RepositoryConfig{
				Type:      build.RepositoryTypeTGit,
				RepoAlias: "test-repo",
				RepoURL:   "https://git.example.com/test/repo.git",
			},
		}
		mockey.Mock((*build.ConfigStoreMongo).Get).Return(mockBuildCfg, nil).Build()
	}

	// mockBuildConfigWithImageRegistry mocks build config with image registry source type
	mockBuildConfigWithImageRegistry := func(appID, imageName string) {
		mockBuildCfg := &build.Config{
			AppID:      appID,
			SourceType: build.SourceTypeImageRegistry,
			Image: &build.ImageConfig{
				Name:     imageName,
				Username: "user",
				Password: "pass",
			},
		}
		mockey.Mock((*build.ConfigStoreMongo).Get).Return(mockBuildCfg, nil).Build()
	}

	mockBuildConfigWithImageRegistryNoCredential := func(appID, imageName string) {
		mockBuildCfg := &build.Config{
			AppID:      appID,
			SourceType: build.SourceTypeImageRegistry,
			Image: &build.ImageConfig{
				Name: imageName,
			},
		}
		mockey.Mock((*build.ConfigStoreMongo).Get).Return(mockBuildCfg, nil).Build()
	}

	BeforeEach(func() {
		ctx = context.Background()
		workspaceID = "test-workspace"
		appID = "test-app-id-" + stringx.Random(6)
		appName = "test-app"
		imageTag = "v1.0.0"
	})

	Context("when build config source type is code repository", func() {
		It("should generate correct variables with bkrepo registry", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockDatabaseAndBuildStore()
				mockBuildConfigWithCodeRepo(appID)

				// Mock workspace.GetWorkspaceImageRegistry
				mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(&bkmsreg.ImageRegistry{
					WorkspaceID:      workspaceID,
					Type:             bkmsreg.ImageRegistryTypeBuiltin,
					Registry:         "docker.bkrepo.example.com",
					Username:         "admin",
					Password:         "blueking",
					BkCICredentialID: "abcdefgh",
				}, nil).Build()

				variables, err := genValuesVariables(ctx, workspaceID, appID, appName, imageTag)
				Expect(err).NotTo(HaveOccurred())
				Expect(variables).To(HaveLen(7))

				// Verify variables
				Expect(
					variables[arrangement.BkmsAppImagePullSecret],
				).To(Equal("bkms-image-pull-secret-test-workspace"))
				Expect(variables[arrangement.BkmsArtifactImageRepository]).To(Equal(appName))
				Expect(variables[arrangement.BkmsArtifactImageRegistry]).To(Equal("docker.bkrepo.example.com"))
				Expect(variables[arrangement.BkmsArtifactImageName]).To(Equal("docker.bkrepo.example.com/test-app"))
				Expect(variables[arrangement.BkmsArtifactImageTag]).To(Equal(imageTag))
				Expect(variables[arrangement.BkmsArtifactImage]).To(Equal("docker.bkrepo.example.com/test-app:v1.0.0"))
				Expect(variables[arrangement.BkmsNetworkingIngressDomain]).To(Equal("not-implemented.example.com"))
			})
		})
	})

	Context("when build config source type is image registry", func() {
		Context("with full registry URL (registry.example.com/group/repository)", func() {
			It("should parse registry and repository correctly", func() {
				mockey.PatchConvey("test", GinkgoT(), func() {
					mockDatabaseAndBuildStore()
					mockBuildConfigWithImageRegistry(appID, "registry.example.com/mygroup/myapp")

					variables, err := genValuesVariables(ctx, workspaceID, appID, appName, imageTag)
					Expect(err).NotTo(HaveOccurred())
					Expect(variables).To(HaveLen(7))

					// Verify registry and repository are parsed correctly
					Expect(variables[arrangement.BkmsAppImagePullSecret]).To(Equal(
						secret.ResolveImagePullSecretName(workspaceID, appID, &build.Config{
							Image: &build.ImageConfig{
								Username: "custom-user",
								Password: "custom-pass",
							},
						}),
					))
					Expect(variables[arrangement.BkmsArtifactImageRepository]).To(Equal("mygroup/myapp"))
					Expect(variables[arrangement.BkmsArtifactImageRegistry]).To(Equal("registry.example.com"))
					Expect(
						variables[arrangement.BkmsArtifactImageName],
					).To(Equal("registry.example.com/mygroup/myapp"))
					Expect(variables[arrangement.BkmsArtifactImageTag]).To(Equal(imageTag))
					Expect(
						variables[arrangement.BkmsArtifactImage],
					).To(Equal("registry.example.com/mygroup/myapp:v1.0.0"))
				})
			})
		})

		Context("with registry and port (registry.example.com:5000/repository)", func() {
			It("should parse registry with port correctly", func() {
				mockey.PatchConvey("test", GinkgoT(), func() {
					mockDatabaseAndBuildStore()
					mockBuildConfigWithImageRegistry(appID, "registry.example.com:5000/myapp")

					variables, err := genValuesVariables(ctx, workspaceID, appID, appName, imageTag)
					Expect(err).NotTo(HaveOccurred())

					// Verify registry with port is parsed correctly
					Expect(variables[arrangement.BkmsAppImagePullSecret]).To(Equal(
						secret.ResolveImagePullSecretName(workspaceID, appID, &build.Config{
							Image: &build.ImageConfig{
								Username: "custom-user",
								Password: "custom-pass",
							},
						}),
					))
					Expect(variables[arrangement.BkmsArtifactImageRepository]).To(Equal("myapp"))
					Expect(variables[arrangement.BkmsArtifactImageRegistry]).To(Equal("registry.example.com:5000"))
					Expect(variables[arrangement.BkmsArtifactImageName]).To(Equal("registry.example.com:5000/myapp"))
					Expect(variables[arrangement.BkmsArtifactImageTag]).To(Equal(imageTag))
					Expect(
						variables[arrangement.BkmsArtifactImage],
					).To(Equal("registry.example.com:5000/myapp:v1.0.0"))
				})
			})
		})

		Context("with group/repository format", func() {
			It("should use empty registry", func() {
				mockey.PatchConvey("test", GinkgoT(), func() {
					mockDatabaseAndBuildStore()
					mockBuildConfigWithImageRegistry(appID, "mygroup/myapp")

					variables, err := genValuesVariables(ctx, workspaceID, appID, appName, imageTag)
					Expect(err).NotTo(HaveOccurred())

					// Verify no registry is extracted
					Expect(variables[arrangement.BkmsAppImagePullSecret]).To(Equal(
						secret.ResolveImagePullSecretName(workspaceID, appID, &build.Config{
							Image: &build.ImageConfig{
								Username: "custom-user",
								Password: "custom-pass",
							},
						}),
					))
					Expect(variables[arrangement.BkmsArtifactImageRepository]).To(Equal("mygroup/myapp"))
					Expect(variables[arrangement.BkmsArtifactImageRegistry]).To(Equal(""))
					Expect(variables[arrangement.BkmsArtifactImageName]).To(Equal("mygroup/myapp"))
					Expect(variables[arrangement.BkmsArtifactImageTag]).To(Equal(imageTag))
					Expect(variables[arrangement.BkmsArtifactImage]).To(Equal("mygroup/myapp:v1.0.0"))
				})
			})
		})

		Context("with repository only", func() {
			It("should use empty registry", func() {
				mockey.PatchConvey("test", GinkgoT(), func() {
					mockDatabaseAndBuildStore()
					mockBuildConfigWithImageRegistry(appID, "myapp")

					variables, err := genValuesVariables(ctx, workspaceID, appID, appName, imageTag)
					Expect(err).NotTo(HaveOccurred())

					// Verify no registry is extracted
					Expect(variables[arrangement.BkmsAppImagePullSecret]).To(Equal(
						secret.ResolveImagePullSecretName(workspaceID, appID, &build.Config{
							Image: &build.ImageConfig{
								Username: "custom-user",
								Password: "custom-pass",
							},
						}),
					))
					Expect(variables[arrangement.BkmsArtifactImageRepository]).To(Equal("myapp"))
					Expect(variables[arrangement.BkmsArtifactImageRegistry]).To(Equal(""))
					Expect(variables[arrangement.BkmsArtifactImageName]).To(Equal("myapp"))
					Expect(variables[arrangement.BkmsArtifactImage]).To(Equal("myapp:v1.0.0"))
				})
			})
		})

		Context("without custom image credential", func() {
			It("should use workspace image pull secret", func() {
				mockey.PatchConvey("test", GinkgoT(), func() {
					mockDatabaseAndBuildStore()
					mockBuildConfigWithImageRegistryNoCredential(appID, "registry.example.com/mygroup/myapp")

					variables, err := genValuesVariables(ctx, workspaceID, appID, appName, imageTag)
					Expect(err).NotTo(HaveOccurred())

					Expect(
						variables[arrangement.BkmsAppImagePullSecret],
					).To(Equal(secret.ResolveImagePullSecretName(workspaceID, "", nil)))
				})
			})
		})
	})

	Context("when getting build config fails", func() {
		It("should return error", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockDatabaseAndBuildStore()
				mockey.Mock((*build.ConfigStoreMongo).Get).Return(
					nil, errors.New("build config not found"),
				).Build()

				variables, err := genValuesVariables(ctx, workspaceID, appID, appName, imageTag)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("get build config for"))
				Expect(err.Error()).To(ContainSubstring("build config not found"))
				Expect(variables).To(BeNil())
			})
		})
	})
})

var _ = Describe("genHelmDeployValues", func() {
	var (
		ctx          context.Context
		app          *bkmsapp.Application
		env          *bkmsenv.Environment
		reader       *envvars.UnifiedEnvVarsReader
		valuesFileID string
		imageTag     string
	)

	// mockDatabaseAndAppConfigFileStore mocks database client and app config file store
	mockDatabaseAndAppConfigFileStore := func() *appcfg.AppConfigFileStoreMongo {
		mockey.Mock(database.Client).Return(&mongo.Client{}).Build()
		mockey.Mock(database.Name).Return("testdb").Build()

		mockStore := &appcfg.AppConfigFileStoreMongo{}
		mockey.Mock(appcfg.NewAppConfigFileStoreMongo).Return(mockStore, nil).Build()

		return mockStore
	}

	// mockListVars mocks the env vars reader to return the given list.
	mockListVars := func(list envvartypes.EnvVariableList) {
		mockey.Mock((*envvars.UnifiedEnvVarsReader).ListVars).Return(list, nil).Build()
	}

	BeforeEach(func() {
		ctx = context.Background()
		app = &bkmsapp.Application{
			WorkspaceID: "test-workspace",
			ID:          "test-app",
			Name:        "test-app",
			Type:        bkmsapp.AppTypeHelm,
		}
		env = &bkmsenv.Environment{Name: "dev", Type: "development"}
		reader = envvars.NewUnifiedEnvVarsReader(nil, nil, nil)
		valuesFileID = "507f1f77bcf86cd799439011"
		imageTag = "v1.0.0"
	})

	Context("when all parameters are valid", func() {
		It("should successfully generate helm deploy values", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				// Mock database and app config file store
				mockDatabaseAndAppConfigFileStore()

				// Mock app config file
				content := `image:
  registry: ${{ bkms.ARTIFACT_IMAGE_REGISTRY }}
  repository: ${{ bkms.ARTIFACT_IMAGE_REPOSITORY }}
  tag: ${{ bkms.ARTIFACT_IMAGE_TAG }}
imagePullSecrets:
  - name: ${{ bkms.APP_IMAGE_PULL_SECRET }}`
				mockAppConfigFile := &appcfg.AppConfigFile{
					AppConfigFileContentSpec: appcfg.AppConfigFileContentSpec{
						AppID:             app.ID,
						Name:              "test-values",
						Type:              appcfg.AppConfigFileTypeNormal,
						ContentSourceType: appcfg.ContentSourceTypeLocal,
						Content:           &content,
					},
				}
				mockey.Mock((*appcfg.AppConfigFileStoreMongo).GetByID).Return(mockAppConfigFile, nil).Build()

				// Mock AppConfigFileEditor
				mockEditor := &mockAppConfigFileEditor{content: content}
				mockey.Mock(appcfg.NewAppConfigFileEditor).Return(mockEditor, nil).Build()
				mockListVars(nil)

				// Mock genValuesVariables
				mockVariables := map[string]string{
					arrangement.BkmsAppImagePullSecret:      "bkms-image-pull-secret-test-workspace",
					arrangement.BkmsArtifactImageRepository: "test-app",
					arrangement.BkmsArtifactImageRegistry:   "docker.example.com",
					arrangement.BkmsArtifactImageName:       "docker.example.com/test-app",
					arrangement.BkmsArtifactImageTag:        "v1.0.0",
					arrangement.BkmsArtifactImage:           "docker.example.com/test-app:v1.0.0",
					arrangement.BkmsNetworkingIngressDomain: "not-implemented.example.com",
				}
				mockey.Mock(genValuesVariables).Return(mockVariables, nil).Build()

				values, missing, missingEnv, err := genHelmDeployValues(
					ctx, reader, app, env, valuesFileID, imageTag, false,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(values).NotTo(BeEmpty())
				Expect(missing).To(BeEmpty())
				Expect(missingEnv).To(BeEmpty())

				// Verify placeholders are replaced
				Expect(values).To(ContainSubstring("registry: docker.example.com"))
				Expect(values).To(ContainSubstring("repository: test-app"))
				Expect(values).To(ContainSubstring("tag: v1.0.0"))
				Expect(values).To(ContainSubstring("name: bkms-image-pull-secret-test-workspace"))
			})
		})

		It("should handle overlay app config file type", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockDatabaseAndAppConfigFileStore()

				// Mock overlay app config file
				baseContent := `image:
  registry: base-registry
  tag: base-tag`
				overlayContent := `image:
  registry: ${{ bkms.ARTIFACT_IMAGE_REGISTRY }}
  tag: ${{ bkms.ARTIFACT_IMAGE_TAG }}`
				mockAppConfigFile := &appcfg.AppConfigFile{
					AppConfigFileContentSpec: appcfg.AppConfigFileContentSpec{
						AppID:             app.ID,
						Name:              "overlay-values",
						Type:              appcfg.AppConfigFileTypeOverlay,
						ContentSourceType: appcfg.ContentSourceTypeLocal,
						OverlayContent:    &overlayContent,
					},
				}
				mockey.Mock((*appcfg.AppConfigFileStoreMongo).GetByID).Return(mockAppConfigFile, nil).Build()

				// Mock AppConfigFileEditor to return merged content
				mergedContent := baseContent + "\n" + overlayContent
				mockEditor := &mockAppConfigFileEditor{content: mergedContent}
				mockey.Mock(appcfg.NewAppConfigFileEditor).Return(mockEditor, nil).Build()
				mockListVars(nil)

				// Mock genValuesVariables
				mockVariables := map[string]string{
					arrangement.BkmsArtifactImageRegistry: "docker.example.com",
					arrangement.BkmsArtifactImageTag:      "v1.0.0",
				}
				mockey.Mock(genValuesVariables).Return(mockVariables, nil).Build()

				values, _, _, err := genHelmDeployValues(ctx, reader, app, env, valuesFileID, imageTag, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(values).NotTo(BeEmpty())
			})
		})

		It("should render both bkms and env references in one pass", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockDatabaseAndAppConfigFileStore()

				content := `image: ${{ bkms.ARTIFACT_IMAGE }}
custom: ${{ env.MY_VAR }}`
				mockAppConfigFile := &appcfg.AppConfigFile{
					AppConfigFileContentSpec: appcfg.AppConfigFileContentSpec{
						AppID: app.ID, Name: "v", Type: appcfg.AppConfigFileTypeNormal,
						ContentSourceType: appcfg.ContentSourceTypeLocal, Content: &content,
					},
				}
				mockey.Mock((*appcfg.AppConfigFileStoreMongo).GetByID).Return(mockAppConfigFile, nil).Build()
				mockey.Mock(appcfg.NewAppConfigFileEditor).
					Return(&mockAppConfigFileEditor{content: content}, nil).
					Build()
				mockey.Mock(genValuesVariables).Return(map[string]string{
					arrangement.BkmsArtifactImage: "docker.example.com/test-app:v1.0.0",
				}, nil).Build()
				mockListVars(envvartypes.EnvVariableList{{Key: "MY_VAR", Value: "foo"}})

				values, missing, missingEnv, err := genHelmDeployValues(
					ctx, reader, app, env, valuesFileID, imageTag, false,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(missing).To(BeEmpty())
				Expect(missingEnv).To(BeEmpty())
				// AC-001 / AC-004: both namespaces replaced
				Expect(values).To(ContainSubstring("image: docker.example.com/test-app:v1.0.0"))
				Expect(values).To(ContainSubstring("custom: foo"))
			})
		})

		It("should mask sensitive env vars when maskSensitive is true and keep real value otherwise", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockDatabaseAndAppConfigFileStore()

				content := `secret: ${{ env.SECRET }}`
				mockAppConfigFile := &appcfg.AppConfigFile{
					AppConfigFileContentSpec: appcfg.AppConfigFileContentSpec{
						AppID: app.ID, Name: "v", Type: appcfg.AppConfigFileTypeNormal,
						ContentSourceType: appcfg.ContentSourceTypeLocal, Content: &content,
					},
				}
				mockey.Mock((*appcfg.AppConfigFileStoreMongo).GetByID).Return(mockAppConfigFile, nil).Build()
				mockey.Mock(appcfg.NewAppConfigFileEditor).
					Return(&mockAppConfigFileEditor{content: content}, nil).
					Build()
				mockey.Mock(genValuesVariables).Return(map[string]string{}, nil).Build()
				mockListVars(envvartypes.EnvVariableList{{Key: "SECRET", Value: "supersecret", IsSensitive: true}})

				// AC-005: preview masks the sensitive value
				preview, _, _, err := genHelmDeployValues(ctx, reader, app, env, valuesFileID, imageTag, true)
				Expect(err).NotTo(HaveOccurred())
				Expect(preview).To(ContainSubstring("secret: " + envvartypes.SensitiveValueMask))
				Expect(preview).NotTo(ContainSubstring("supersecret"))

				// AC-005: deploy uses the real value
				deployed, _, _, err := genHelmDeployValues(ctx, reader, app, env, valuesFileID, imageTag, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(deployed).To(ContainSubstring("secret: supersecret"))
			})
		})

		It("should not error on undefined env vars and report them via missing list", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockDatabaseAndAppConfigFileStore()

				content := `a: ${{ env.UNKNOWN }}`
				mockAppConfigFile := &appcfg.AppConfigFile{
					AppConfigFileContentSpec: appcfg.AppConfigFileContentSpec{
						AppID: app.ID, Name: "v", Type: appcfg.AppConfigFileTypeNormal,
						ContentSourceType: appcfg.ContentSourceTypeLocal, Content: &content,
					},
				}
				mockey.Mock((*appcfg.AppConfigFileStoreMongo).GetByID).Return(mockAppConfigFile, nil).Build()
				mockey.Mock(appcfg.NewAppConfigFileEditor).
					Return(&mockAppConfigFileEditor{content: content}, nil).
					Build()
				mockey.Mock(genValuesVariables).Return(map[string]string{}, nil).Build()
				mockListVars(nil)

				// AC-002: undefined env var does not fail, reported in missing env vars.
				values, missing, missingEnv, err := genHelmDeployValues(
					ctx, reader, app, env, valuesFileID, imageTag, true,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(missing).To(BeEmpty())
				Expect(missingEnv).To(Equal([]string{"UNKNOWN"}))
				Expect(values).To(ContainSubstring("a:"))
			})
		})
	})

	Context("when app config file ID is invalid", func() {
		It("should return error for invalid ObjectID format", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockDatabaseAndAppConfigFileStore()

				invalidID := "invalid-object-id"
				values, _, _, err := genHelmDeployValues(ctx, reader, app, env, invalidID, imageTag, false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid app config file id"))
				Expect(values).To(BeEmpty())
			})
		})
	})

	Context("when getting app config file fails", func() {
		It("should return error", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockDatabaseAndAppConfigFileStore()
				mockey.Mock((*appcfg.AppConfigFileStoreMongo).GetByID).Return(
					nil, errors.New("app config file not found"),
				).Build()

				values, _, _, err := genHelmDeployValues(ctx, reader, app, env, valuesFileID, imageTag, false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("get app config file"))
				Expect(err.Error()).To(ContainSubstring("app config file not found"))
				Expect(values).To(BeEmpty())
			})
		})
	})

	Context("when getting compiled content fails", func() {
		It("should return error", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockDatabaseAndAppConfigFileStore()

				content := "test content"
				mockAppConfigFile := &appcfg.AppConfigFile{
					AppConfigFileContentSpec: appcfg.AppConfigFileContentSpec{
						AppID:             app.ID,
						Name:              "test-values",
						Type:              appcfg.AppConfigFileTypeNormal,
						ContentSourceType: appcfg.ContentSourceTypeLocal,
						Content:           &content,
					},
				}
				mockey.Mock((*appcfg.AppConfigFileStoreMongo).GetByID).Return(mockAppConfigFile, nil).Build()

				mockEditor := &mockAppConfigFileEditor{err: errors.New("failed to compile content")}
				mockey.Mock(appcfg.NewAppConfigFileEditor).Return(mockEditor, nil).Build()

				values, _, _, err := genHelmDeployValues(ctx, reader, app, env, valuesFileID, imageTag, false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("compiling app config file"))
				Expect(err.Error()).To(ContainSubstring("failed to compile content"))
				Expect(values).To(BeEmpty())
			})
		})
	})
})

var _ = Describe("buildEnvContext", func() {
	var (
		ctx    context.Context
		app    *bkmsapp.Application
		env    *bkmsenv.Environment
		reader *envvars.UnifiedEnvVarsReader
	)

	BeforeEach(func() {
		ctx = context.Background()
		app = &bkmsapp.Application{WorkspaceID: "ws", ID: "app", Name: "app", Type: bkmsapp.AppTypeHelm}
		env = &bkmsenv.Environment{Name: "dev", Type: "development"}
		reader = envvars.NewUnifiedEnvVarsReader(nil, nil, nil)
	})

	It("should let higher-priority scope override lower-priority scope", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			// list is ordered low->high priority, so the later K wins
			mockey.Mock((*envvars.UnifiedEnvVarsReader).ListVars).Return(envvartypes.EnvVariableList{
				{Key: "K", Value: "g"},
				{Key: "K", Value: "e"},
			}, nil).Build()

			vars, err := buildEnvContext(ctx, reader, app, env, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(vars["K"]).To(Equal("e"))
		})
	})

	It("should mask sensitive values only when maskSensitive is true", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock((*envvars.UnifiedEnvVarsReader).ListVars).Return(envvartypes.EnvVariableList{
				{Key: "PLAIN", Value: "v"},
				{Key: "SECRET", Value: "s", IsSensitive: true},
			}, nil).Build()

			masked, err := buildEnvContext(ctx, reader, app, env, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(masked["PLAIN"]).To(Equal("v"))
			Expect(masked["SECRET"]).To(Equal(envvartypes.SensitiveValueMask))

			raw, err := buildEnvContext(ctx, reader, app, env, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(raw["SECRET"]).To(Equal("s"))
		})
	})

	It("should propagate reader error", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock((*envvars.UnifiedEnvVarsReader).ListVars).Return(nil, errors.New("db down")).Build()

			_, err := buildEnvContext(ctx, reader, app, env, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("list env vars"))
		})
	})
})

var _ = Describe("collectMissingVars", func() {
	It("should return env refs separately and keep other refs qualified", func() {
		content := `a: ${{ env.B }}
b: ${{ env.A }}
c: ${{ env.PRESENT }}
d: ${{ bkms.MISSING }}
e: ${{ bkms.IMAGE }}`
		contexts := map[string]map[string]string{
			"env":  {"PRESENT": "x"},
			"bkms": {"IMAGE": "img"},
		}
		missing, missingEnv, err := collectMissingVars(content, contexts)
		Expect(err).NotTo(HaveOccurred())
		Expect(missing).To(Equal([]string{"bkms.MISSING"}))
		Expect(missingEnv).To(Equal([]string{"A", "B"}))
	})

	It("should return empty when all refs are defined", func() {
		content := `a: ${{ env.A }}
b: ${{ bkms.B }}`
		contexts := map[string]map[string]string{"env": {"A": "1"}, "bkms": {"B": "2"}}
		missing, missingEnv, err := collectMissingVars(content, contexts)
		Expect(err).NotTo(HaveOccurred())
		Expect(missing).To(BeEmpty())
		Expect(missingEnv).To(BeEmpty())
	})

	It("should return empty when content has no variable references", func() {
		missing, missingEnv, err := collectMissingVars("plain: text", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(missing).To(BeEmpty())
		Expect(missingEnv).To(BeEmpty())
	})
})

var _ = Describe("UninstallHelmRelease", func() {
	var record *Record
	var ctx context.Context

	BeforeEach(func() {
		record = &Record{
			ClusterID:    "BCS-K8S-00000",
			Namespace:    "test-namespace",
			ReleaseName:  "test-release",
			ProjectCode:  "test-project",
			Revision:     "1",
			ChartName:    "test-chart",
			ChartVersion: "1.0.0",
		}
		ctx = context.Background()
	})

	Context("when release does not exist", func() {
		It("should treat uninstall as successful", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockey.Mock(bkmshelm.NewActionConfiguration).Return(&action.Configuration{}, nil).Build()
				mockey.Mock((*action.Uninstall).Run).Return(nil, driver.ErrReleaseNotFound).Build()

				err := UninstallHelmRelease(ctx, record)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Context("when uninstall returns other error", func() {
		It("should return the wrapped error", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				uninstallErr := errors.New("uninstall failed")
				mockey.Mock(bkmshelm.NewActionConfiguration).Return(&action.Configuration{}, nil).Build()
				mockey.Mock((*action.Uninstall).Run).Return(nil, uninstallErr).Build()

				err := UninstallHelmRelease(ctx, record)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("uninstall helm release test-release"))
				Expect(err.Error()).To(ContainSubstring("uninstall failed"))
			})
		})
	})

	Context("when uninstall succeeds", func() {
		It("should return nil", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockey.Mock(bkmshelm.NewActionConfiguration).Return(&action.Configuration{}, nil).Build()
				mockey.Mock((*action.Uninstall).Run).Return(&helmrelease.UninstallReleaseResponse{}, nil).Build()

				err := UninstallHelmRelease(ctx, record)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

// mockAppConfigFileEditor is a mock implementation of AppConfigFileEditor for testing
type mockAppConfigFileEditor struct {
	content string
	err     error
}

// GetEditableContentField ...
func (m *mockAppConfigFileEditor) GetEditableContentField() appcfg.EditableContentField {
	return appcfg.EditableContentFieldContent
}

// SetContent ...
func (m *mockAppConfigFileEditor) SetContent(content string) error {
	m.content = content
	return nil
}

// SetOverlayContent ...
func (m *mockAppConfigFileEditor) SetOverlayContent(content string) error {
	m.content = content
	return nil
}

// GetCompiledContent ...
func (m *mockAppConfigFileEditor) GetCompiledContent(ctx context.Context) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.content, nil
}
