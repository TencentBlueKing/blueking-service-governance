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

package appruntime

import (
	"slices"

	"github.com/hashicorp/go-set/v3"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

// FrameworkDef is a read-only framework catalog entry.
type FrameworkDef struct {
	Name               string
	SupportedLanguages []string
	ConfigVersion      int
}

var catalog = []FrameworkDef{
	{
		Name:               FrameworkBlank,
		SupportedLanguages: []string{LanguageCpp, LanguageGo},
		ConfigVersion:      FrameworkConfigVersionV1,
	},
	{
		Name:               FrameworkTAF,
		SupportedLanguages: []string{LanguageCpp},
		ConfigVersion:      FrameworkConfigVersionV1,
	},
	{
		Name:               FrameworkTRPC,
		SupportedLanguages: []string{LanguageCpp, LanguageGo},
		ConfigVersion:      FrameworkConfigVersionV1,
	},
}

var catalogByName = lo.SliceToMap(catalog, func(def FrameworkDef) (string, FrameworkDef) {
	return def.Name, def
})

// KnownLanguages is the product language set shared by validation and normalization.
var KnownLanguages = set.From([]string{LanguageGo, LanguageCpp})

// Catalog returns a stable-sorted deep copy of framework definitions.
func Catalog() []FrameworkDef {
	out := make([]FrameworkDef, 0, len(catalog))
	// copier only reports errors for unsupported kinds; both sides are []FrameworkDef.
	_ = copier.CopyWithOption(&out, &catalog, copier.Option{DeepCopy: true})
	return out
}

// LookupFramework returns a deep copy of the named framework definition, so that
// callers cannot mutate the catalog through the returned SupportedLanguages slice.
func LookupFramework(name string) (FrameworkDef, bool) {
	def, ok := catalogByName[name]
	if !ok {
		return FrameworkDef{}, false
	}

	var cloned FrameworkDef
	_ = copier.CopyWithOption(&cloned, &def, copier.Option{DeepCopy: true})
	return cloned, true
}

// ValidateStack checks a complete stack against the catalog.
// Helm/Agones must not carry framework or language. BKMSApp requires both.
func ValidateStack(stack Stack) error {
	switch stack.Type {
	case AppTypeHelm, AppTypeAgones:
		if stack.Framework != "" || stack.Language != "" {
			return errors.Wrapf(ErrUnexpectedStackFields, "app type %q", stack.Type)
		}
		return nil
	case AppTypeBKMSApp:
		return validateBKMSAppStack(stack)
	case AppTypeTRPC, AppTypeTAF:
		return errors.Errorf("legacy app type %q is not a normalized stack type", stack.Type)
	case "":
		return errors.New("app type is empty")
	default:
		return errors.Wrapf(ErrUnknownAppType, "app type %q", stack.Type)
	}
}

func validateBKMSAppStack(stack Stack) error {
	if stack.Framework == "" {
		return errors.New("framework is required for bkmsApp")
	}
	// Read the catalog directly: this only inspects the definition, so the copy
	// LookupFramework makes for external callers would be wasted work.
	def, ok := catalogByName[stack.Framework]
	if !ok {
		return errors.Wrapf(ErrUnknownFramework, "framework %q", stack.Framework)
	}
	if stack.Language == "" {
		return errors.New("language is required for bkmsApp")
	}
	if !KnownLanguages.Contains(stack.Language) {
		return errors.Wrapf(ErrUnknownLanguage, "language %q", stack.Language)
	}
	if !slices.Contains(def.SupportedLanguages, stack.Language) {
		return errors.Wrapf(
			ErrInvalidCombination,
			"framework %q does not support language %q",
			stack.Framework,
			stack.Language,
		)
	}
	return nil
}
