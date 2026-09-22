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

// Package app defines the application model and store implementations
package app

import (
	"fmt"
	"time"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/credentials"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
)

const (
	// AppTypeHelm helm 应用：https://helm.sh/
	AppTypeHelm = appruntime.AppTypeHelm
	// AppTypeAgones agones 应用：https://agones.dev
	AppTypeAgones = appruntime.AppTypeAgones
	// AppTypeBKMSApp 使用 AppModel 管理的默认应用类型，底层使用 AppModel 应用模型。
	AppTypeBKMSApp = appruntime.AppTypeBKMSApp
	// AppTypeTRPC trpc 应用（桥接期旧持久化值）
	AppTypeTRPC = appruntime.AppTypeTRPC
	// AppTypeTAF taf 应用（桥接期旧持久化值）
	AppTypeTAF = appruntime.AppTypeTAF
)

// Application is the main type for the project. An application is a deployable unit which can be defined in various
// ways including a helm chart or other types.
type Application struct {
	// ID is the unique identifier of the application
	ID string `json:"id" bson:"id,omitempty" validate:"required"`

	// WorkspaceID is the workspace of the application
	WorkspaceID string `json:"workspaceID" bson:"workspaceID"`

	// Name is the name of the application
	Name string `json:"name" bson:"name"`
	// DisplayName is the display name of the application
	DisplayName string `json:"displayName" bson:"displayName"`
	// Type is the type of the application
	Type string `json:"type" bson:"type"`
	// Language is the programming language of a BKMSApp. Helm/Agones leave this empty.
	Language string `json:"language" bson:"language,omitempty"`
	// Framework is the application framework of a BKMSApp: trpc, taf or blank.
	Framework string `json:"framework" bson:"framework,omitempty"`

	Creator   string    `json:"creator" bson:"creator"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`

	// **This field is only present for legacy trpc applications.**
	// TrpcSpec is the compatibility DTO of the old trpc application spec.
	TrpcSpec *TrpcSpec `json:"trpcSpec" bson:"trpcSpec,omitempty"`

	// **This field is only present for helm applications.**
	// HelmSpec is the spec of the helm application
	HelmSpec *HelmSpec `json:"helmSpec" bson:"helmSpec,omitempty"`
}

func (a *Application) String() string {
	return fmt.Sprintf("%s/%s", a.WorkspaceID, a.Name)
}

// TrpcSpec is the spec of the trpc application
type TrpcSpec struct {
	// Language is the programming language of the trpc application
	// Currently supported languages are: "go", "cpp"
	Language string `json:"language" bson:"language"`
}

// HelmSpec is the spec of the helm application
type HelmSpec struct {
	HelmSource *HelmSource `json:"helmSource" bson:"helmSource"`
}

type HelmSourceRepoType string

const (
	// HelmSourceRepoTypeHelm Helm 仓库
	HelmSourceRepoTypeHelm HelmSourceRepoType = "HelmRepo"
	// HelmSourceRepoTypeBCS BCS 仓库
	HelmSourceRepoTypeBCS HelmSourceRepoType = "BCSRepo"
	// HelmSourceRepoTypeGit Git 仓库
	HelmSourceRepoTypeGit HelmSourceRepoType = "GitRepo"
)

// HelmSource contains the source information of a helm application. This information can be used to:
//
// - Retrieve the source files of the application chart
// - Deploy the application to an environment using the configured values
type HelmSource struct {
	// RepoType is the type of the repository, the value can be "HelmRepo", "BCSRepo" or "Git"
	RepoType HelmSourceRepoType `json:"repoType" bson:"repoType"`
	// GitRepoConfig, applicable when RepoType is "GitRepo"
	GitRepoConfig *GitRepoConfig `json:"gitRepoConfig" bson:"gitRepoConfig,omitempty"`
	// HelmRepoConfig, applicable when RepoType is "HelmRepo"
	HelmRepoConfig *HelmRepoConfig `json:"helmRepoConfig" bson:"helmRepoConfig,omitempty"`
	// BCSRepoConfig, applicable when RepoType is "BCSRepo"
	BCSRepoConfig *BCSRepoConfig `json:"bcsRepoConfig" bson:"bcsRepoConfig,omitempty"`

	// ValueFiles is a list of Helm value files to use when generating a template
	ValueFiles []string `json:"valueFiles" bson:"valueFiles"`
}

// GitRepoType is the type of the Git repository
type GitRepoType string

const (
	// GitRepoTypeTGit TGit（工蜂）仓库
	GitRepoTypeTGit GitRepoType = "TGit"
	// GitRepoTypeGitHub GitHub 仓库
	GitRepoTypeGitHub GitRepoType = "GitHub"
)

// GitRepoConfig is the config for using a Git repository as the helm chart source.
// It supports both public repositories and private repositories authenticated via OAuth token.
type GitRepoConfig struct {
	// Type is the type of the Git repository, e.g. "TGit", "GitHub"
	Type GitRepoType `json:"type" bson:"type"`
	// RepoAlias is the alias of the repository, used for display purposes
	// Optional
	RepoAlias string `json:"repoAlias" bson:"repoAlias"`
	// RepoURL is the URL to the repository
	RepoURL string `json:"repoURL" bson:"repoURL"`
	// Revision is the branch or tag name to use when generating chart
	Revision string `json:"revision" bson:"revision"`
	// SourceDir is the directory in the repository to use as the helm chart source
	// Important: the directory may include a valid helm chart or something
	//	can be transformed into a helm chart (e.g. yaml file, kustomize config)
	// Helm chart packed and uploaded by bkci plugin in bkci pipeline execute process
	SourceDir string `json:"sourceDir" bson:"sourceDir"`
}

// HelmRepoConfig is config for using standard helm repo
type HelmRepoConfig struct {
	// RepoURL is the URL to the repository
	RepoURL string `json:"repoURL" bson:"repoURL"`
	// ChartName is the name of the helm chart
	ChartName string `json:"chartName" bson:"chartName"`
	// Username is the username to use when authenticating to the repository
	Username string `json:"username" bson:"username,omitempty"`
	// Password is the password to use when authenticating to the repository
	Password string `json:"password" bson:"password,omitempty"`
}

// BCSRepoConfig is config for using bcs-repo as helm repo
type BCSRepoConfig struct {
	ProjectCode string `json:"projectCode" bson:"projectCode"`
	RepoName    string `json:"repoName" bson:"repoName"`
	ChartName   string `json:"chartName" bson:"chartName"`
}

// IsAppModelType checks if the given app type is an appmodel type.
// During the bridge period this includes legacy trpc/taf and bkmsApp.
func IsAppModelType(appType string) bool {
	return appruntime.IsAppModelType(appType)
}

// IsHelmBasedType checks if the given app type is a helm-based type
// Helm-based types include Helm and Agones, which are both based on Helm charts
func IsHelmBasedType(appType string) bool {
	return appruntime.IsHelmBasedType(appType)
}

// DisplayLanguage returns the list-safe language without reading AppModel.
func (a *Application) DisplayLanguage() string {
	language, _ := a.displayStack()
	return language
}

// DisplayFramework returns the list-safe framework without reading AppModel.
func (a *Application) DisplayFramework() string {
	_, framework := a.displayStack()
	return framework
}

// RuntimeSnapshot copies Application-level tech-stack fields into a compatibility DTO.
func (a *Application) RuntimeSnapshot() appruntime.Snapshot {
	if a == nil {
		return appruntime.Snapshot{}
	}
	trpcSpecLanguage := ""
	if a.TrpcSpec != nil {
		trpcSpecLanguage = a.TrpcSpec.Language
	}
	return appruntime.Snapshot{
		AppID:            a.ID,
		AppType:          a.Type,
		Language:         a.Language,
		Framework:        a.Framework,
		TrpcSpecLanguage: trpcSpecLanguage,
	}
}

func (a *Application) displayStack() (language, framework string) {
	if a == nil {
		return "", ""
	}
	trpcSpecLanguage := ""
	if a.TrpcSpec != nil {
		trpcSpecLanguage = a.TrpcSpec.Language
	}
	return appruntime.ProjectDisplay(a.Type, a.Language, a.Framework, trpcSpecLanguage)
}

// SetUserPass 设置 HelmRepoConfig 的 username 和 password，值为空时用 existingConfig 的值
// 设置完成后自动校验 Username 和 Password 的合法性
func (h *HelmRepoConfig) SetUserPass(existingConfig *HelmRepoConfig, username, password *string) error {
	if existingConfig != nil {
		h.Username = existingConfig.Username
		h.Password = existingConfig.Password
	}
	if username != nil {
		h.Username = *username
	}
	if password != nil {
		h.Password = *password
	}

	return credentials.ValidateOptionalUserPass(h.Username, h.Password)
}
