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

package handler

import (
	"strconv"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	buildserializer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/build/serializer"
	imagebuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
)

// buildConfigFromInput converts the request payload into a persisted build
// config and applies the business validation needed by downstream build flows.
func buildConfigFromInput(
	appID string,
	input buildserializer.UpdateBuildConfigInput,
	dataBefore *imagebuild.Config,
) (*imagebuild.Config, error) {
	cfg := &imagebuild.Config{
		AppID:      appID,
		SourceType: imagebuild.SourceType(input.SourceType),
	}

	// TagConfig is optional, but once provided the custom tag options must still
	// form a usable image tag template.
	if tagCfg := input.TagConfig; tagCfg != nil {
		cfg.TagConfig = imagebuild.TagConfig{Type: imagebuild.VersionType(tagCfg.Type)}
		if tagCfg.Type == string(imagebuild.VersionTypeCustom) && tagCfg.CustomOpts != nil {
			opts := tagCfg.CustomOpts
			if len(opts.Prefix) > imagebuild.MaxCustomTagPrefixLength {
				return nil, errCustomTagPrefixTooLong.With("max", strconv.Itoa(imagebuild.MaxCustomTagPrefixLength))
			}
			if !opts.WithRevision && !opts.WithBuildTime && opts.Prefix == "" {
				return nil, errCustomTagEmpty
			}
			cfg.TagConfig.CustomOpts = &imagebuild.CustomTagOpts{
				Prefix:        opts.Prefix,
				WithRevision:  opts.WithRevision,
				WithBuildTime: opts.WithBuildTime,
			}
		}
	}

	switch cfg.SourceType {
	case imagebuild.SourceTypeImageRegistry:
		if input.Image == nil {
			return nil, errImageFieldRequired
		}
		cfg.Image = &imagebuild.ImageConfig{Name: input.Image.Name}
		var existingImage *imagebuild.ImageConfig
		if dataBefore != nil {
			existingImage = dataBefore.Image
		}
		if err := cfg.Image.SetUserPass(existingImage, input.Image.Username, input.Image.Password); err != nil {
			return nil, errInvalidImageCredentials.WithCause(err)
		}
	case imagebuild.SourceTypeCodeRepository:
		if input.CodeRepo == nil {
			return nil, errCodeRepoFieldRequired
		}
		codeRepo, err := input.CodeRepo.ToModel()
		if err != nil {
			return nil, errInvalidCodeRepoConfig.WithCause(err)
		}
		cfg.PipelineType = string(bkci.PipelineTypeDockerfile)
		cfg.CodeRepo = codeRepo
	case imagebuild.SourceTypePipeline:
		if input.Pipeline == nil {
			return nil, errPipelineFieldRequired
		}
		cfg.PipelineType = input.Pipeline.PipelineID
		cfg.Pipeline = &imagebuild.PipelineConfig{
			PipelineID: input.Pipeline.PipelineID,
			Params:     input.Pipeline.Params,
		}
	default:
		return nil, errInvalidSourceType
	}

	return cfg, nil
}
