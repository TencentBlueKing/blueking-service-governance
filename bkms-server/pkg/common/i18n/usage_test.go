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

package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// literalTKeys 扫描 root 下的非测试 Go 文件，返回 i18n.T 调用中以字符串字面量传入的 key 及其位置
func literalTKeys(root string) (map[string]string, error) {
	keys := map[string]string{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "T" {
				return true
			}
			if pkg, isIdent := sel.X.(*ast.Ident); !isIdent || pkg.Name != "i18n" {
				return true
			}
			lit, ok := call.Args[1].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			key, err := strconv.Unquote(lit.Value)
			if err == nil {
				keys[key] = fset.Position(lit.Pos()).String()
			}
			return true
		})
		return nil
	})
	return keys, err
}

var _ = Describe("i18n.T usages", func() {
	It("should find literal keys passed to i18n.T", func() {
		dir := GinkgoT().TempDir()
		src := "package demo\n\nfunc f() { _ = i18n.T(ctx, \"demo.key\"); _ = i18n.T(ctx, prefix+\"x\") }\n"
		Expect(os.WriteFile(filepath.Join(dir, "demo.go"), []byte(src), 0o600)).To(Succeed())

		keys, err := literalTKeys(dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(keys).To(HaveLen(1))
		Expect(keys).To(HaveKey("demo.key"))
	})

	It("should have a localized message for every literal key", func() {
		keys, err := literalTKeys(filepath.Join("..", ".."))
		Expect(err).NotTo(HaveOccurred())
		for key, pos := range keys {
			Expect(HasKey(key)).To(BeTrue(), "missing message for key %q used at %s", key, pos)
		}
	})
})
