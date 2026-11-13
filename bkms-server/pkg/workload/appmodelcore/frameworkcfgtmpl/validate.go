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

package frameworkcfgtmpl

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod/mods"
)

const (
	// MaxNameLength 模板名称的最大字符数。
	MaxNameLength = 64
	// MaxFileNameLength 配置文件名的最大长度。
	MaxFileNameLength = 64
	// MaxContentSize 模板内容的最大字节数。
	MaxContentSize = 64 * 1024
)

// 与应用配置文件名的规则保持一致，模板的文件名会原样成为应用的框架配置文件名。
var fileNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

// ResolveGroup 校验框架与语言的组合是否支持框架配置文件，并返回文件格式。
func ResolveGroup(framework, language string) (Group, appcfg.FileFormat, error) {
	group := Group{Framework: framework, Language: language}
	stack := appruntime.Stack{Type: appruntime.AppTypeBKMSApp, Framework: framework, Language: language}
	// mods.Resolve 为兼容存量应用允许缺语言，模板必须落到具体语言，先做严格校验。
	if err := appruntime.ValidateStack(stack); err != nil {
		return group, "", errors.Wrapf(ErrInvalidTemplate, "framework %q language %q: %v", framework, language, err)
	}
	set, err := mods.Resolve(stack)
	if err != nil {
		return group, "", errors.Wrapf(ErrInvalidTemplate, "framework %q language %q: %v", framework, language, err)
	}
	if !set.FrameworkFile.Supported {
		return group, "", errors.Wrapf(ErrInvalidTemplate, "framework %q has no framework config file", framework)
	}
	return group, appcfg.FileFormat(set.FrameworkFile.Props.Format), nil
}

// normalize 去掉名称与文件名首尾的空白，并按分组的文件格式校验内容。
func (d Definition) normalize(format appcfg.FileFormat) (Definition, error) {
	d.Name = strings.TrimSpace(d.Name)
	d.FileName = strings.TrimSpace(d.FileName)
	switch {
	case d.Name == "":
		return d, errors.Wrap(ErrInvalidTemplate, "name is required")
	case utf8.RuneCountInString(d.Name) > MaxNameLength:
		return d, errors.Wrapf(ErrInvalidTemplate, "name exceeds %d characters", MaxNameLength)
	case d.Name == BuiltinName:
		return d, errors.Wrapf(ErrInvalidTemplate, "name %q is reserved for the builtin template", BuiltinName)
	case len(d.FileName) > MaxFileNameLength || !fileNamePattern.MatchString(d.FileName):
		return d, errors.Wrapf(ErrInvalidTemplate,
			"file name must be 1-%d letters, digits, '_', '-' or '.'", MaxFileNameLength)
	case strings.TrimSpace(d.Content) == "":
		return d, errors.Wrap(ErrInvalidTemplate, "content is required")
	case len(d.Content) > MaxContentSize:
		return d, errors.Wrapf(ErrInvalidTemplate, "content exceeds %d bytes", MaxContentSize)
	}
	if err := (appcfg.FrameworkPolicy{}).ValidateContent(d.Content, format); err != nil {
		return d, errors.Wrapf(ErrInvalidTemplate, "%v", err)
	}
	return d, nil
}
