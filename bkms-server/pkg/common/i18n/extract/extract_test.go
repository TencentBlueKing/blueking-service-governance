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

package extract_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n/extract"
)

func writeFile(path, content string) {
	Expect(os.MkdirAll(filepath.Dir(path), 0o755)).To(Succeed())
	Expect(os.WriteFile(path, []byte(content), 0o600)).To(Succeed())
}

var _ = Describe("Keys", func() {
	var root string

	BeforeEach(func() {
		root = GinkgoT().TempDir()
		writeFile(filepath.Join(root, "common", "bkerrs", "codes.go"), `package bkerrs
const ErrDetailCodeShared ErrDetailCode = "SHARED_CODE"
const builtinMsg = "builtin msg"
var errBuiltin = Define(ErrCodeNotFound, "BUILTIN_CODE", builtinMsg)
`)
		writeFile(filepath.Join(root, "foo", "errors.go"), `package foo
const localCode = "FOO_LOCAL"
var (
	errLiteral = bkerrs.Define(bkerrs.ErrCodeInvalidArgument, "FOO_LITERAL", "foo literal")
	errConverted = bkerrs.Define(bkerrs.ErrCodeInvalidArgument, bkerrs.ErrDetailCode("FOO_CONVERTED"), "foo converted")
	errLocal = bkerrs.Define(bkerrs.ErrCodeInvalidArgument, localCode, "foo local")
	errShared = bkerrs.Define(bkerrs.ErrCodeInvalidArgument, bkerrs.ErrDetailCodeShared, "shared")
	errUnresolvable = bkerrs.Define(bkerrs.ErrCodeInvalidArgument, codeFromFunc(), "skipped")
)
func texts(ctx context.Context, key string) []string {
	return []string{i18n.T(ctx, "foo.title"), i18n.T(ctx, "foo.title"), i18n.T(ctx, key)}
}
`)
		writeFile(filepath.Join(root, "foo", "errors_test.go"), `package foo
var errInTest = bkerrs.Define(bkerrs.ErrCodeInvalidArgument, "FOO_IN_TEST", "in test")
`)
	})

	It("should extract keys with defaults, directories and positions", func() {
		keys, err := extract.Keys(root)
		Expect(err).NotTo(HaveOccurred())

		names := make([]string, 0, len(keys))
		byName := map[string]extract.Key{}
		for _, k := range keys {
			names = append(names, k.Name)
			byName[k.Name] = k
		}
		Expect(names).To(Equal([]string{
			"BUILTIN_CODE", "FOO_CONVERTED", "FOO_LITERAL", "FOO_LOCAL", "SHARED_CODE", "foo.title",
		}))

		Expect(byName["BUILTIN_CODE"].Default).To(Equal("builtin msg"))
		Expect(byName["FOO_LOCAL"].Default).To(Equal("foo local"))
		Expect(byName["foo.title"].Default).To(BeEmpty())
		Expect(byName["FOO_LITERAL"].Dir).To(Equal(filepath.Join(root, "foo")))
		Expect(byName["foo.title"].Pos).To(HavePrefix(filepath.Join(root, "foo", "errors.go") + ":"))
	})

	It("should collect template params passed at call sites", func() {
		writeFile(filepath.Join(root, "bar", "bar.go"), `package bar
const paramName = "name"
var (
	errChain = bkerrs.Define(bkerrs.ErrCodeNotFound, "BAR_CHAIN", "chain")
	errAlias = bkerrs.Define(bkerrs.ErrCodeNotFound, "BAR_ALIAS", "alias")
	errDynamic = bkerrs.Define(bkerrs.ErrCodeNotFound, "BAR_DYNAMIC", "dynamic")
	errNone = bkerrs.Define(bkerrs.ErrCodeNotFound, "BAR_NONE", "none")
)
func use(ctx context.Context, cause error, k string, m map[string]string) {
	_ = errChain.WithCause(cause).With("appID", "1").With(paramName, "x")
	_ = errChain.With("appID", "2")
	_ = bkerrs.Define(bkerrs.ErrCodeNotFound, "BAR_INLINE", "inline").With("id", "3")
	e := errAlias
	e = e.With("envName", "prod")
	_ = errDynamic.With(k, "v")
	_ = bkerrs.ErrShared.With("shared", "s")
	_ = i18n.T(ctx, "bar.title", map[string]string{"release": "r", paramName: "n"})
	_ = i18n.T(ctx, "bar.title", map[string]string{"extra": "e"})
	_ = i18n.T(ctx, "bar.dynamic", m)
	_ = i18n.T(ctx, "bar.plain")
}
`)
		writeFile(filepath.Join(root, "common", "bkerrs", "shared.go"), `package bkerrs
var ErrShared = Define(ErrCodeNotFound, "BAR_SHARED", "shared")
`)
		keys, err := extract.Keys(root)
		Expect(err).NotTo(HaveOccurred())
		byName := map[string]extract.Key{}
		for _, k := range keys {
			byName[k.Name] = k
		}

		Expect(byName["BAR_CHAIN"].Kind).To(Equal(extract.KindDefine))
		Expect(byName["BAR_CHAIN"].Params).To(Equal([]string{"appID", "name"}))
		Expect(byName["BAR_INLINE"].Params).To(Equal([]string{"id"}))
		Expect(byName["BAR_ALIAS"].Params).To(Equal([]string{"envName"}))
		Expect(byName["BAR_SHARED"].Params).To(Equal([]string{"shared"}))
		Expect(byName["BAR_NONE"].Params).To(BeEmpty())
		Expect(byName["BAR_DYNAMIC"].ParamsUnknown).To(BeTrue())
		Expect(byName["BAR_CHAIN"].ParamsUnknown).To(BeFalse())

		Expect(byName["bar.title"].Kind).To(Equal(extract.KindT))
		Expect(byName["bar.title"].Params).To(Equal([]string{"extra", "name", "release"}))
		Expect(byName["bar.dynamic"].ParamsUnknown).To(BeTrue())
		Expect(byName["bar.plain"].Params).To(BeEmpty())
		Expect(byName["bar.plain"].ParamsUnknown).To(BeFalse())
	})
})
