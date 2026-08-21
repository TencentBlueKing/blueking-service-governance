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

package appcfg

import (
	"context"
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeAppConfigFileStore struct {
	files map[bson.ObjectID]AppConfigFile
}

func newFakeAppConfigFileStore(items ...AppConfigFile) *fakeAppConfigFileStore {
	store := &fakeAppConfigFileStore{files: make(map[bson.ObjectID]AppConfigFile, len(items))}
	for _, item := range items {
		store.files[item.ID] = item
	}
	return store
}

func (s *fakeAppConfigFileStore) Add(_ context.Context, acf AppConfigFile) (bson.ObjectID, error) {
	if acf.ID == bson.NilObjectID {
		acf.ID = bson.NewObjectID()
	}
	s.files[acf.ID] = acf
	return acf.ID, nil
}

func (s *fakeAppConfigFileStore) GetByID(_ context.Context, id bson.ObjectID) (*AppConfigFile, error) {
	item, ok := s.files[id]
	if !ok {
		return nil, nil
	}
	cloned := item
	return &cloned, nil
}

func (s *fakeAppConfigFileStore) IsOwnedByApp(_ context.Context, id bson.ObjectID, appID string) (bool, error) {
	item, ok := s.files[id]
	return ok && item.AppID == appID, nil
}

func (s *fakeAppConfigFileStore) List(
	_ context.Context,
	appID string,
	listOpts ...AcfListOption,
) ([]AppConfigFile, error) {
	opts := &ListOptions{}
	for _, opt := range listOpts {
		opt.ApplyToOptions(opts)
	}

	result := make([]AppConfigFile, 0)
	for _, item := range s.files {
		if item.AppID != appID {
			continue
		}
		if opts.filterType != "" && item.Type != opts.filterType {
			continue
		}
		if opts.filterEnvName != nil && item.EnvName != *opts.filterEnvName {
			continue
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == result[j].Name {
			return result[i].ID.Hex() < result[j].ID.Hex()
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func (s *fakeAppConfigFileStore) GetByAppAndMountPath(
	ctx context.Context,
	appID string,
	mountPath string,
) (*AppConfigFile, error) {
	items, err := s.ListByAppAndMountPath(ctx, appID, mountPath)
	if err != nil || len(items) == 0 {
		return nil, err
	}
	cloned := items[0]
	return &cloned, nil
}

func (s *fakeAppConfigFileStore) ListByAppAndMountPath(
	_ context.Context,
	appID string,
	mountPath string,
) ([]AppConfigFile, error) {
	result := make([]AppConfigFile, 0)
	for _, item := range s.files {
		if item.AppID == appID && item.MountPath == mountPath {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID.Hex() < result[j].ID.Hex()
	})
	return result, nil
}

func (s *fakeAppConfigFileStore) Update(_ context.Context, acf AppConfigFile) (int64, error) {
	s.files[acf.ID] = acf
	return 1, nil
}

func (s *fakeAppConfigFileStore) UpdateIfVersionMatches(
	_ context.Context,
	acf AppConfigFile,
	expectedVersion int64,
) (int64, error) {
	current, ok := s.files[acf.ID]
	if !ok || current.CurrentVersion != expectedVersion {
		return 0, ErrAppConfigFileVersionConflict
	}
	s.files[acf.ID] = acf
	return 1, nil
}

func (s *fakeAppConfigFileStore) DeleteByID(_ context.Context, appID string, id bson.ObjectID) (int64, error) {
	item, ok := s.files[id]
	if !ok || item.AppID != appID {
		return 0, nil
	}
	delete(s.files, id)
	return 1, nil
}

func (s *fakeAppConfigFileStore) DeleteByApp(_ context.Context, appID string) (int64, error) {
	var deleted int64
	for id, item := range s.files {
		if item.AppID == appID {
			delete(s.files, id)
			deleted++
		}
	}
	return deleted, nil
}

func (s *fakeAppConfigFileStore) IsReferencedByOther(_ context.Context, id bson.ObjectID) (bool, error) {
	for _, item := range s.files {
		if item.BaseAppConfigFileID != nil && *item.BaseAppConfigFileID == id {
			return true, nil
		}
	}
	return false, nil
}

type fakeAppConfigFileVersionStore struct {
	versionsByFile map[bson.ObjectID][]AppConfigFileVersion
	deletedFileIDs []bson.ObjectID
}

func newFakeAppConfigFileVersionStore() *fakeAppConfigFileVersionStore {
	return &fakeAppConfigFileVersionStore{
		versionsByFile: make(map[bson.ObjectID][]AppConfigFileVersion),
	}
}

func (s *fakeAppConfigFileVersionStore) Add(_ context.Context, version AppConfigFileVersion) (bson.ObjectID, error) {
	if version.ID == bson.NilObjectID {
		version.ID = bson.NewObjectID()
	}
	s.versionsByFile[version.AppConfigFileID] = append(s.versionsByFile[version.AppConfigFileID], version)
	return version.ID, nil
}

func (s *fakeAppConfigFileVersionStore) BatchGetByAppAndIDs(
	_ context.Context,
	_ string,
	_ []bson.ObjectID,
) ([]AppConfigFileVersion, error) {
	return nil, nil
}

func (s *fakeAppConfigFileVersionStore) GetByFileAndVersion(
	_ context.Context,
	appConfigFileID bson.ObjectID,
	version int64,
) (*AppConfigFileVersion, error) {
	for _, item := range s.versionsByFile[appConfigFileID] {
		if item.Version == version {
			cloned := item
			return &cloned, nil
		}
	}
	return nil, ErrAppCfgFileVersionNotFound
}

func (s *fakeAppConfigFileVersionStore) List(
	_ context.Context,
	_ AppConfigFileVersionListOptions,
) ([]AppConfigFileVersion, int64, error) {
	var result []AppConfigFileVersion
	for _, items := range s.versionsByFile {
		result = append(result, items...)
	}
	return result, int64(len(result)), nil
}

func (s *fakeAppConfigFileVersionStore) SoftDeleteByID(
	_ context.Context,
	_ bson.ObjectID,
	_ string,
) (int64, error) {
	return 0, nil
}

func (s *fakeAppConfigFileVersionStore) DeleteByFileID(
	_ context.Context,
	appConfigFileID bson.ObjectID,
) (int64, error) {
	s.deletedFileIDs = append(s.deletedFileIDs, appConfigFileID)
	count := int64(len(s.versionsByFile[appConfigFileID]))
	delete(s.versionsByFile, appConfigFileID)
	return count, nil
}

var _ = Describe("AppConfigFileService env config", func() {
	var (
		ctx          context.Context
		fileStore    *fakeAppConfigFileStore
		versionStore *fakeAppConfigFileVersionStore
		service      *AppConfigFileService
		defaultFile  AppConfigFile
	)

	BeforeEach(func() {
		ctx = context.Background()
		versionStore = newFakeAppConfigFileVersionStore()
		defaultContent := "feature.enabled=true"
		defaultFile = AppConfigFile{
			ID: bson.NewObjectID(),
			AppConfigFileContentSpec: AppConfigFileContentSpec{
				AppID:             "app-1",
				EnvName:           EnvNameDefault,
				Name:              "custom-env",
				Type:              AppConfigFileTypeNormal,
				ContentSourceType: ContentSourceTypeLocal,
				Format:            FileFormat("env"),
				ConfigKind:        ConfigKindPlain,
				MountPath:         "/data/app/conf/custom.env",
				IsUnifiedConfig:   true,
				Content:           &defaultContent,
			},
			CurrentVersion: 1,
			Updater:        "tester",
		}
		fileStore = newFakeAppConfigFileStore(defaultFile)
		service = NewAppConfigFileService(fileStore, versionStore)
	})

	It("enables independent env config without creating env instances (reference model)", func() {
		err := service.UpdateEnvConfig(ctx, &defaultFile, UpdateEnvConfigParams{
			IsUnifiedConfig: false,
			MountedEnvNames: []string{"prod", "stag"},
			Operator:        "tester",
			Description:     "enable independent env config",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(defaultFile.IsUnifiedConfig).To(BeFalse())
		Expect(defaultFile.MountedEnvNames).To(Equal([]string{"prod", "stag"}))

		storedDefault := fileStore.files[defaultFile.ID]
		Expect(storedDefault.CurrentVersion).To(Equal(int64(2)))

		envFiles, err := service.listEnvInstanceFiles(ctx, defaultFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(envFiles).To(BeEmpty())
	})

	It("enables independent env config for all envs", func() {
		err := service.UpdateEnvConfig(ctx, &defaultFile, UpdateEnvConfigParams{
			IsUnifiedConfig: false,
			Operator:        "tester",
			Description:     "enable independent env config for all envs",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(defaultFile.IsUnifiedConfig).To(BeFalse())
		Expect(defaultFile.MountedEnvNames).To(BeNil())

		envFiles, err := service.listEnvInstanceFiles(ctx, defaultFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(envFiles).To(BeEmpty())
	})

	It("creates plain file always with unified config", func() {
		content := "feature.enabled=true"
		created, err := service.Create(ctx, CreateCfgFileParams{
			AppID:             "app-1",
			EnvName:           EnvNameDefault,
			Name:              "feature-flags",
			Type:              AppConfigFileTypeNormal,
			ContentSourceType: ContentSourceTypeLocal,
			Format:            FileFormat("env"),
			ConfigKind:        ConfigKindPlain,
			MountPath:         "/data/app/conf/feature-flags.env",
			IsUnifiedConfig:   true,
			Content:           &content,
			Creator:           "tester",
			Description:       "create plain file",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(created.IsUnifiedConfig).To(BeTrue())

		envFiles, err := service.listEnvInstanceFiles(ctx, *created)
		Expect(err).NotTo(HaveOccurred())
		Expect(envFiles).To(BeEmpty())
	})

	It("switches back to unified config and deletes all env instances", func() {
		prodContent := "feature.enabled=false"
		prodFile := AppConfigFile{
			ID: bson.NewObjectID(),
			AppConfigFileContentSpec: AppConfigFileContentSpec{
				AppID:                  defaultFile.AppID,
				EnvName:                "prod",
				Name:                   "custom-env--prod",
				Type:                   AppConfigFileTypeNormal,
				ContentSourceType:      ContentSourceTypeLocal,
				Format:                 FileFormat("env"),
				ConfigKind:             ConfigKindPlain,
				MountPath:              defaultFile.MountPath,
				DefaultAppConfigFileID: &defaultFile.ID,
				Content:                &prodContent,
			},
			CurrentVersion: 1,
		}
		stagContent := "feature.enabled=gray"
		stagFile := AppConfigFile{
			ID: bson.NewObjectID(),
			AppConfigFileContentSpec: AppConfigFileContentSpec{
				AppID:                  defaultFile.AppID,
				EnvName:                "stag",
				Name:                   "custom-env--stag",
				Type:                   AppConfigFileTypeNormal,
				ContentSourceType:      ContentSourceTypeLocal,
				Format:                 FileFormat("env"),
				ConfigKind:             ConfigKindPlain,
				MountPath:              defaultFile.MountPath,
				DefaultAppConfigFileID: &defaultFile.ID,
				Content:                &stagContent,
			},
			CurrentVersion: 1,
		}
		defaultFile.IsUnifiedConfig = false
		defaultFile.MountedEnvNames = []string{"prod", "stag"}
		fileStore.files[defaultFile.ID] = defaultFile
		fileStore.files[prodFile.ID] = prodFile
		fileStore.files[stagFile.ID] = stagFile
		_, _ = versionStore.Add(ctx, AppConfigFileVersion{AppConfigFileID: prodFile.ID, Version: 1})
		_, _ = versionStore.Add(ctx, AppConfigFileVersion{AppConfigFileID: stagFile.ID, Version: 1})

		err := service.UpdateEnvConfig(ctx, &defaultFile, UpdateEnvConfigParams{
			IsUnifiedConfig: true,
			Operator:        "tester",
			Description:     "switch to unified config",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(defaultFile.IsUnifiedConfig).To(BeTrue())
		Expect(defaultFile.MountedEnvNames).To(BeNil())

		_, prodExists := fileStore.files[prodFile.ID]
		_, stagExists := fileStore.files[stagFile.ID]
		Expect(prodExists).To(BeFalse())
		Expect(stagExists).To(BeFalse())
		Expect(versionStore.deletedFileIDs).To(ContainElements(prodFile.ID, stagFile.ID))
	})

	It("restores specified env instances to reference state", func() {
		prodContent := "feature.enabled=false"
		prodFile := AppConfigFile{
			ID: bson.NewObjectID(),
			AppConfigFileContentSpec: AppConfigFileContentSpec{
				AppID:                  defaultFile.AppID,
				EnvName:                "prod",
				Name:                   "custom-env--prod",
				Type:                   AppConfigFileTypeNormal,
				ContentSourceType:      ContentSourceTypeLocal,
				Format:                 FileFormat("env"),
				ConfigKind:             ConfigKindPlain,
				MountPath:              defaultFile.MountPath,
				DefaultAppConfigFileID: &defaultFile.ID,
				Content:                &prodContent,
			},
			CurrentVersion: 1,
		}
		stagContent := "feature.enabled=gray"
		stagFile := AppConfigFile{
			ID: bson.NewObjectID(),
			AppConfigFileContentSpec: AppConfigFileContentSpec{
				AppID:                  defaultFile.AppID,
				EnvName:                "stag",
				Name:                   "custom-env--stag",
				Type:                   AppConfigFileTypeNormal,
				ContentSourceType:      ContentSourceTypeLocal,
				Format:                 FileFormat("env"),
				ConfigKind:             ConfigKindPlain,
				MountPath:              defaultFile.MountPath,
				DefaultAppConfigFileID: &defaultFile.ID,
				Content:                &stagContent,
			},
			CurrentVersion: 1,
		}
		defaultFile.IsUnifiedConfig = false
		defaultFile.MountedEnvNames = []string{"prod", "stag"}
		fileStore.files[defaultFile.ID] = defaultFile
		fileStore.files[prodFile.ID] = prodFile
		fileStore.files[stagFile.ID] = stagFile
		_, _ = versionStore.Add(ctx, AppConfigFileVersion{AppConfigFileID: prodFile.ID, Version: 1})
		_, _ = versionStore.Add(ctx, AppConfigFileVersion{AppConfigFileID: stagFile.ID, Version: 1})

		err := service.UpdateEnvConfig(ctx, &defaultFile, UpdateEnvConfigParams{
			IsUnifiedConfig:   false,
			MountedEnvNames:   []string{"prod", "stag"},
			FallbackConfigEnv: "prod",
			Operator:          "tester",
			Description:       "fallback prod to reference",
		})

		Expect(err).NotTo(HaveOccurred())
		_, prodExists := fileStore.files[prodFile.ID]
		Expect(prodExists).To(BeFalse())
		_, stagExists := fileStore.files[stagFile.ID]
		Expect(stagExists).To(BeTrue())
	})

	It("rejects fallback on unified config", func() {
		err := service.UpdateEnvConfig(ctx, &defaultFile, UpdateEnvConfigParams{
			IsUnifiedConfig:   false,
			FallbackConfigEnv: "prod",
			Operator:          "tester",
			Description:       "fallback on unified",
		})

		Expect(errors.Is(err, ErrFallbackRequiresIndependentConfig)).To(BeTrue())
	})

	It("creates independent env instance on first modification", func() {
		defaultFile.IsUnifiedConfig = false
		defaultFile.MountedEnvNames = []string{"prod", "stag"}
		fileStore.files[defaultFile.ID] = defaultFile

		prodContent := "feature.enabled=false"
		envFile, err := service.CreatePlainEnvInstance(
			ctx, defaultFile, "prod", &prodContent, "tester", "create independent env instance",
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(envFile).NotTo(BeNil())
		Expect(envFile.EnvName).To(Equal("prod"))
		Expect(envFile.DefaultAppConfigFileID).NotTo(BeNil())
		Expect(*envFile.DefaultAppConfigFileID).To(Equal(defaultFile.ID))
		Expect(*envFile.Content).To(Equal("feature.enabled=false"))
	})

	It("zero-value spec is treated as independent (safe: no env instances exist for old data)", func() {
		spec := AppConfigFileContentSpec{}
		Expect(spec.HasIndependentEnvConfig()).To(BeTrue())
	})

	It("explicitly unified spec is not independent", func() {
		spec := AppConfigFileContentSpec{IsUnifiedConfig: true}
		Expect(spec.HasIndependentEnvConfig()).To(BeFalse())
	})

	It("keeps IsUnifiedConfig unchanged on generic update", func() {
		content := "feature.enabled=true"
		file := defaultFile
		file.Content = &content
		file.IsUnifiedConfig = true

		err := service.UpdateFile(ctx, &file, "tester", UpdateCfgFileOptions{
			OperationType: AppConfigFileVersionOperationTypeUpdate,
			Description:   "keep unified config unchanged",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(file.IsUnifiedConfig).To(BeTrue())

		storedFile := fileStore.files[file.ID]
		Expect(storedFile.IsUnifiedConfig).To(BeTrue())
	})

	It("cleans up stale env instances when narrowing mount scope", func() {
		prodContent := "feature.enabled=false"
		prodFile := AppConfigFile{
			ID: bson.NewObjectID(),
			AppConfigFileContentSpec: AppConfigFileContentSpec{
				AppID:                  defaultFile.AppID,
				EnvName:                "prod",
				Name:                   "custom-env--prod",
				Type:                   AppConfigFileTypeNormal,
				ContentSourceType:      ContentSourceTypeLocal,
				Format:                 FileFormat("env"),
				ConfigKind:             ConfigKindPlain,
				MountPath:              defaultFile.MountPath,
				DefaultAppConfigFileID: &defaultFile.ID,
				Content:                &prodContent,
			},
			CurrentVersion: 1,
		}
		stagContent := "feature.enabled=gray"
		stagFile := AppConfigFile{
			ID: bson.NewObjectID(),
			AppConfigFileContentSpec: AppConfigFileContentSpec{
				AppID:                  defaultFile.AppID,
				EnvName:                "stag",
				Name:                   "custom-env--stag",
				Type:                   AppConfigFileTypeNormal,
				ContentSourceType:      ContentSourceTypeLocal,
				Format:                 FileFormat("env"),
				ConfigKind:             ConfigKindPlain,
				MountPath:              defaultFile.MountPath,
				DefaultAppConfigFileID: &defaultFile.ID,
				Content:                &stagContent,
			},
			CurrentVersion: 1,
		}
		defaultFile.IsUnifiedConfig = false
		defaultFile.MountedEnvNames = []string{"prod", "stag"}
		fileStore.files[defaultFile.ID] = defaultFile
		fileStore.files[prodFile.ID] = prodFile
		fileStore.files[stagFile.ID] = stagFile
		_, _ = versionStore.Add(ctx, AppConfigFileVersion{AppConfigFileID: prodFile.ID, Version: 1})
		_, _ = versionStore.Add(ctx, AppConfigFileVersion{AppConfigFileID: stagFile.ID, Version: 1})

		err := service.UpdateEnvConfig(ctx, &defaultFile, UpdateEnvConfigParams{
			IsUnifiedConfig: false,
			MountedEnvNames: []string{"prod"},
			Operator:        "tester",
			Description:     "narrow to prod only",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(defaultFile.MountedEnvNames).To(Equal([]string{"prod"}))
		_, prodExists := fileStore.files[prodFile.ID]
		Expect(prodExists).To(BeTrue())
		_, stagExists := fileStore.files[stagFile.ID]
		Expect(stagExists).To(BeFalse())
	})

	It("rejects creating a plain env record without defaultAppConfigFileID", func() {
		content := "feature.enabled=true"
		_, err := service.Create(ctx, CreateCfgFileParams{
			AppID:             defaultFile.AppID,
			EnvName:           "prod",
			Name:              "orphan-plain",
			Type:              AppConfigFileTypeNormal,
			ContentSourceType: ContentSourceTypeLocal,
			Format:            FileFormat("env"),
			ConfigKind:        ConfigKindPlain,
			MountPath:         "/data/app/conf/orphan.env",
			IsUnifiedConfig:   true,
			Content:           &content,
			Creator:           "tester",
		})

		Expect(errors.Is(err, ErrInvalidConfigSpec)).To(BeTrue())
		Expect(err).To(MatchError(ContainSubstring("plain env instance requires defaultAppConfigFileID")))
	})

	It("keeps mountedEnvNames when switching back to unified config", func() {
		defaultFile.IsUnifiedConfig = false
		defaultFile.MountedEnvNames = []string{"prod", "stag"}
		fileStore.files[defaultFile.ID] = defaultFile

		err := service.UpdateEnvConfig(ctx, &defaultFile, UpdateEnvConfigParams{
			IsUnifiedConfig: true,
			MountedEnvNames: []string{"prod"},
			Operator:        "tester",
			Description:     "switch to unified with scoped envs",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(defaultFile.IsUnifiedConfig).To(BeTrue())
		Expect(defaultFile.MountedEnvNames).To(Equal([]string{"prod"}))
	})

	It("updates mountedEnvNames while already in unified config", func() {
		err := service.UpdateEnvConfig(ctx, &defaultFile, UpdateEnvConfigParams{
			IsUnifiedConfig: true,
			MountedEnvNames: []string{"prod"},
			Operator:        "tester",
			Description:     "scope unified config to prod",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(defaultFile.IsUnifiedConfig).To(BeTrue())
		Expect(defaultFile.MountedEnvNames).To(Equal([]string{"prod"}))
		Expect(fileStore.files[defaultFile.ID].MountedEnvNames).To(Equal([]string{"prod"}))
	})

	It("rejects mountPath change on plain env instance", func() {
		prodContent := "feature.enabled=false"
		prodFile := AppConfigFile{
			ID: bson.NewObjectID(),
			AppConfigFileContentSpec: AppConfigFileContentSpec{
				AppID:                  defaultFile.AppID,
				EnvName:                "prod",
				Name:                   "custom-env--prod",
				Type:                   AppConfigFileTypeNormal,
				ContentSourceType:      ContentSourceTypeLocal,
				Format:                 FileFormat("env"),
				ConfigKind:             ConfigKindPlain,
				MountPath:              defaultFile.MountPath,
				DefaultAppConfigFileID: &defaultFile.ID,
				Content:                &prodContent,
			},
			CurrentVersion: 1,
		}
		fileStore.files[prodFile.ID] = prodFile
		prodFile.MountPath = "/data/app/conf/other.env"

		err := service.UpdateFile(ctx, &prodFile, "tester", UpdateCfgFileOptions{
			OperationType: AppConfigFileVersionOperationTypeUpdate,
			Description:   "change env instance mountPath",
		})

		Expect(errors.Is(err, ErrInvalidConfigSpec)).To(BeTrue())
		Expect(err).To(MatchError(ContainSubstring("plain env instance cannot change mountPath")))
	})

	It("finds existing plain env instance for content update", func() {
		prodContent := "feature.enabled=false"
		prodFile := AppConfigFile{
			ID: bson.NewObjectID(),
			AppConfigFileContentSpec: AppConfigFileContentSpec{
				AppID:                  defaultFile.AppID,
				EnvName:                "prod",
				Name:                   "custom-env--prod",
				Type:                   AppConfigFileTypeNormal,
				ContentSourceType:      ContentSourceTypeLocal,
				Format:                 FileFormat("env"),
				ConfigKind:             ConfigKindPlain,
				MountPath:              defaultFile.MountPath,
				DefaultAppConfigFileID: &defaultFile.ID,
				Content:                &prodContent,
			},
			CurrentVersion: 1,
		}
		fileStore.files[prodFile.ID] = prodFile

		found, err := service.FindPlainEnvInstance(ctx, defaultFile, "prod")
		Expect(err).NotTo(HaveOccurred())
		Expect(found).NotTo(BeNil())
		Expect(found.ID).To(Equal(prodFile.ID))
		Expect(*found.Content).To(Equal("feature.enabled=false"))
	})

	It("cleans mountedEnvNames for roots without env instances when env is deleted", func() {
		defaultFile.IsUnifiedConfig = false
		defaultFile.MountedEnvNames = []string{"prod", "stag"}
		fileStore.files[defaultFile.ID] = defaultFile

		err := service.CleanupPlainEnvInstancesByEnv(ctx, defaultFile.AppID, "prod")
		Expect(err).NotTo(HaveOccurred())

		stored := fileStore.files[defaultFile.ID]
		Expect(stored.MountedEnvNames).To(Equal([]string{"stag"}))
	})

	It("clears mountedEnvNames to empty slice when the last scoped env is deleted", func() {
		defaultFile.IsUnifiedConfig = true
		defaultFile.MountedEnvNames = []string{"prod"}
		fileStore.files[defaultFile.ID] = defaultFile

		err := service.CleanupPlainEnvInstancesByEnv(ctx, defaultFile.AppID, "prod")
		Expect(err).NotTo(HaveOccurred())

		stored := fileStore.files[defaultFile.ID]
		Expect(stored.MountedEnvNames).NotTo(BeNil())
		Expect(stored.MountedEnvNames).To(Equal([]string{}))
	})

	It("rejects overlay whose base is a plain config file", func() {
		overlayContent := "patches:\n- foo: 1\n"
		_, err := service.Create(ctx, CreateCfgFileParams{
			AppID:               defaultFile.AppID,
			EnvName:             EnvNameDefault,
			Name:                "plain-overlay",
			Type:                AppConfigFileTypeOverlay,
			ContentSourceType:   ContentSourceTypeLocal,
			Format:              FileFormatYAML,
			ConfigKind:          ConfigKindFramework,
			BaseAppConfigFileID: &defaultFile.ID,
			OverlayContent:      &overlayContent,
			Creator:             "tester",
		})

		Expect(errors.Is(err, ErrInvalidConfigSpec)).To(BeTrue())
		Expect(err).To(MatchError(ContainSubstring("overlay base must be a framework config file")))
	})
})
