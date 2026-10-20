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

	"github.com/pkg/errors"
)

// UpsertEnvContent 处理一次内容更新的完整用例：准备目标文件、校验编译后内容，并创建或更新文件版本。
func (s *AppCfgFileDefService) UpsertEnvContent(
	ctx context.Context,
	def *AppConfigFileDef,
	params UpsertEnvContentParams,
) (*UpsertEnvContentResult, error) {
	targetFile, compiledContent, isNewFile, oldFile, err := s.prepareEnvContentUpdate(
		ctx, def, params.EnvName, params.Content, params.Operator,
	)
	if err != nil {
		return nil, errors.Wrap(err, "preparing env content update")
	}

	if params.ValidateCompiledContent != nil {
		if err = params.ValidateCompiledContent(targetFile, compiledContent); err != nil {
			return nil, errors.Wrap(err, "validating compiled content")
		}
	}

	result := &UpsertEnvContentResult{
		File:            targetFile,
		CompiledContent: compiledContent,
		IsNewFile:       isNewFile,
		OldFile:         oldFile,
	}
	if isNewFile {
		targetFile, err = s.CreateFileWithVersion(
			ctx, *targetFile, def.Name, params.Description, params.Operator,
		)
		if err != nil {
			return nil, errors.Wrap(err, "creating env instance")
		}
		result.File = targetFile
	} else {
		if err = s.UpdateFile(ctx, targetFile, def.Name, params.Operator, UpdateCfgFileOptions{
			OperationType:          AppConfigFileVersionOperationTypeUpdate,
			Description:            params.Description,
			ExpectedCurrentVersion: params.ExpectedCurrentVersion,
		}); err != nil {
			return result, errors.Wrap(err, "updating target file")
		}
	}

	return result, nil
}

func (s *AppCfgFileDefService) prepareEnvContentUpdate(
	ctx context.Context,
	def *AppConfigFileDef,
	envName string,
	content string,
	operator string,
) (*AppConfigFile, string, bool, *AppConfigFile, error) {
	if def == nil {
		return nil, "", false, nil, errors.New("def is required")
	}

	policy, err := s.policyFor(def.ConfigKind)
	if err != nil {
		return nil, "", false, nil, err
	}

	defaultFileWithDef, err := s.GetDefaultFileWithDef(ctx, def.ID)
	if err != nil {
		return nil, "", false, nil, err
	}

	if err = policy.ValidateContent(content, defaultFileWithDef.GetConfigFormat()); err != nil {
		return nil, "", false, nil, errors.Wrap(err, "kind-specific content validation")
	}

	targetFile, isNewFile, err := s.resolveContentUpdateTarget(
		ctx, def, envName, policy, *defaultFileWithDef,
	)
	if err != nil {
		return nil, "", false, nil, err
	}
	if isNewFile {
		targetFile, err = s.buildContentUpdateNewFile(defaultFileWithDef, content, operator, policy, envName)
		if err != nil {
			return nil, "", false, nil, err
		}
	}

	var oldFile *AppConfigFile
	if !isNewFile && targetFile != nil {
		copied := *targetFile
		oldFile = &copied
	}

	compiledContent, err := s.applyContentUpdate(ctx, targetFile, content)
	if err != nil {
		return nil, "", false, nil, err
	}
	return targetFile, compiledContent, isNewFile, oldFile, nil
}

func (s *AppCfgFileDefService) resolveContentUpdateTarget(
	ctx context.Context,
	def *AppConfigFileDef,
	envName string,
	policy ConfigKindPolicy,
	defaultFile AppConfigFileWithDef,
) (*AppConfigFile, bool, error) {
	if envName == EnvNameDefault {
		return &defaultFile.AppConfigFile, false, nil
	}

	// 先检查挂载范围，再决定是否回落默认文件
	if !policy.IsEffectiveForEnv(def, envName) {
		return nil, false, errors.Wrapf(ErrInvalidConfigSpec, "file %s is not effective for env %s", def.Name, envName)
	}

	// 统一配置：内容写到默认文件
	if !def.EnvConfigMode.IsIndependent() {
		return &defaultFile.AppConfigFile, false, nil
	}

	existing, err := s.FindEnvInstance(ctx, def.ID, envName)
	if err != nil {
		return nil, false, errors.Wrap(err, "finding env instance")
	}
	if existing != nil {
		return existing, false, nil
	}

	return nil, true, nil
}

func (s *AppCfgFileDefService) buildContentUpdateNewFile(
	defaultFile *AppConfigFileWithDef,
	content string,
	operator string,
	policy ConfigKindPolicy,
	envName string,
) (*AppConfigFile, error) {
	params := CreateEnvInstanceParams{
		EnvName:  envName,
		Operator: operator,
	}
	strategy, err := policy.GetEnvContentStrategy(defaultFile.Def)
	if err != nil {
		return nil, err
	}
	if strategy == EnvContentStrategyOverwrite {
		params.Content = &content
	} else {
		params.OverlayContent = &content
	}

	acf, err := s.buildEnvInstance(*defaultFile, params)
	if err != nil {
		return nil, err
	}
	return acf, nil
}

func (s *AppCfgFileDefService) applyContentUpdate(
	ctx context.Context,
	targetFile *AppConfigFile,
	content string,
) (string, error) {
	editor, err := NewAppConfigFileEditor(s.db, targetFile)
	if err != nil {
		return "", errors.Wrap(err, "creating app config file editor")
	}

	switch editor.GetEditableContentField() {
	case EditableContentFieldContent:
		if err = editor.SetContent(content); err != nil {
			return "", errors.Wrap(err, "setting content")
		}
	case EditableContentFieldOverlayContent:
		if err = editor.SetOverlayContent(content); err != nil {
			return "", errors.Wrap(err, "setting overlay content")
		}
	default:
		return "", errors.Wrap(ErrInvalidConfigSpec, "target file has no editable content field")
	}

	compiledContent, err := editor.GetCompiledContent(ctx)
	if err != nil {
		return "", errors.Wrap(err, "compiling content")
	}
	return compiledContent, nil
}
