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
	"sync"

	"github.com/pkg/errors"
)

const modulePath = "github.com/TencentBlueKing/blueking-service-governance/bkms-server"

// serverRoot 从测试的工作目录逐级向上，返回 go.mod 中 module 为 modulePath 的目录，即 bkms-server 模块根目录。
// 核对模块名是为了不误认路径上其他模块的 go.mod
func serverRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "get working directory")
	}
	for {
		content, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && modFileDeclares(content, modulePath) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.Errorf("go.mod of module %s not found above the working directory", modulePath)
		}
		dir = parent
	}
}

// modFileDeclares 判断 go.mod 内容声明的模块是否为 module
func modFileDeclares(content []byte, module string) bool {
	for _, line := range strings.Split(string(content), "\n") {
		if fields := strings.Fields(line); len(fields) == 2 && fields[0] == "module" {
			return fields[1] == module
		}
	}
	return false
}

// allowPrefix 豁免注释的前缀，完整格式为 "//tenancy:allow <规则名>[,<规则名>] <理由>"
const allowPrefix = "//tenancy:allow"

// allowComment 源码中的一条豁免注释
type allowComment struct {
	line int
	// rules 豁免的规则名，注释缺少规则名或理由时为空
	rules []string
}

// sourceFile 一个已解析的非测试 Go 源文件
type sourceFile struct {
	// rel 相对 bkms-server 的路径，使用 "/" 分隔，用于报错
	rel    string
	fset   *token.FileSet
	file   *ast.File
	allows []allowComment
}

// line 返回节点起始位置的行号
func (f *sourceFile) line(n ast.Node) int {
	return f.fset.Position(n.Pos()).Line
}

// pos 返回 "相对路径:行号"，便于在失败信息中直接定位
func (f *sourceFile) pos(line int) string {
	return f.rel + ":" + strconv.Itoa(line)
}

// parseAllows 提取文件中的豁免注释
func parseAllows(fset *token.FileSet, file *ast.File) []allowComment {
	var allows []allowComment
	for _, group := range file.Comments {
		for _, c := range group.List {
			rest, ok := strings.CutPrefix(c.Text, allowPrefix)
			if !ok || rest != "" && !strings.HasPrefix(rest, " ") {
				continue
			}
			a := allowComment{line: fset.Position(c.Pos()).Line}
			ruleList, reason, _ := strings.Cut(strings.TrimSpace(rest), " ")
			if ruleList != "" && strings.TrimSpace(reason) != "" {
				a.rules = strings.Split(ruleList, ",")
			}
			allows = append(allows, a)
		}
	}
	return allows
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

var (
	sourcesOnce sync.Once
	sources     []*sourceFile
	sourcesErr  error
)

// cachedSources 返回 loadSources 的结果，整个测试进程只解析一次，供各条规则共用；
// 规则只读取语法树，不能修改
func cachedSources() ([]*sourceFile, error) {
	sourcesOnce.Do(func() { sources, sourcesErr = loadSources() })
	return sources, sourcesErr
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
		file, err := parser.ParseFile(fset, path, content, parser.ParseComments)
		if err != nil {
			return errors.Wrapf(err, "parse %s", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return errors.Wrapf(err, "relative path of %s", path)
		}
		files = append(files, &sourceFile{
			rel: filepath.ToSlash(rel), fset: fset, file: file, allows: parseAllows(fset, file),
		})
		return nil
	}
	for _, sub := range []string{"pkg", "cmd"} {
		if err := filepath.WalkDir(filepath.Join(root, sub), walk); err != nil {
			return nil, errors.Wrapf(err, "walk %s", sub)
		}
	}
	return files, nil
}
