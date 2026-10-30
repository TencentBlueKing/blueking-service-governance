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

package tenancy_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const modulePath = "github.com/TencentBlueKing/blueking-service-governance/bkms-server"

// serverRoot 返回 bkms-server 目录的绝对路径，测试的工作目录是本包所在目录
func serverRoot() (string, error) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	return root, errors.Wrap(err, "resolve bkms-server root")
}

// sourceFile 一个已解析的非测试 Go 源文件
type sourceFile struct {
	// rel 相对 bkms-server 的路径，使用 "/" 分隔，用于白名单匹配与报错
	rel  string
	fset *token.FileSet
	file *ast.File
}

// pos 返回节点的 "相对路径:行号"，便于在失败信息中直接定位
func (f *sourceFile) pos(n ast.Node) string {
	return f.rel + ":" + strconv.Itoa(f.fset.Position(n.Pos()).Line)
}

// importName 返回文件中导入 importPath 时使用的包名，未导入时返回空字符串
func (f *sourceFile) importName(importPath string) string {
	for _, imp := range f.file.Imports {
		if strings.Trim(imp.Path.Value, `"`) != importPath {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return filepath.Base(importPath)
	}
	return ""
}

// isPkgSelector 判断 expr 是否为 "<pkgName>.<sel>"
func isPkgSelector(expr ast.Expr, pkgName, sel string) bool {
	s, ok := expr.(*ast.SelectorExpr)
	if !ok || s.Sel.Name != sel {
		return false
	}
	id, ok := s.X.(*ast.Ident)
	return ok && pkgName != "" && id.Name == pkgName
}

// loadSources 解析 bkms-server 下 pkg、cmd 目录的全部非测试 Go 文件。
// 生成的文件（swag 文档、mockery mock）不含租户相关逻辑，一并跳过
func loadSources() ([]*sourceFile, error) {
	root, err := serverRoot()
	if err != nil {
		return nil, err
	}
	var files []*sourceFile
	fset := token.NewFileSet()
	walk := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "read %s", path)
		}
		if strings.Contains(string(content), "Code generated") && strings.Contains(string(content), "DO NOT EDIT") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, content, 0)
		if err != nil {
			return errors.Wrapf(err, "parse %s", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return errors.Wrapf(err, "relative path of %s", path)
		}
		files = append(files, &sourceFile{rel: filepath.ToSlash(rel), fset: fset, file: file})
		return nil
	}
	for _, sub := range []string{"pkg", "cmd"} {
		if err := filepath.WalkDir(filepath.Join(root, sub), walk); err != nil {
			return nil, errors.Wrapf(err, "walk %s", sub)
		}
	}
	return files, nil
}

// matchesAny 判断相对路径是否命中白名单：以 "/" 结尾的条目匹配目录前缀，否则精确匹配文件
func matchesAny(rel string, allowlist map[string]string) bool {
	for entry := range allowlist {
		if strings.HasSuffix(entry, "/") && strings.HasPrefix(rel, entry) || rel == entry {
			return true
		}
	}
	return false
}
