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

// Package extract 静态扫描 Go 源码，提取显式标记为面向用户的英文原文（gettext 的 msgid），来源包括：
//
//   - i18n.Msg 标记的字符串字面量
//   - i18n.T / Translate 的字符串字面量
//   - store.Meta 中 NotFound / Duplicate 引用的、以 errors.New 字面量定义的包级错误变量
//
// 未标记的文案（如 bkerrs.New 中直接写的字面量）视为面向开发者，不提取；以变量、拼接等方式传入的文案无法静态提取
package extract

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
)

// Message 源码中一条需要翻译的原文
type Message struct {
	ID string
	// Pos 为首次出现的位置，形如 file:line（相对模块根目录）
	Pos string
}

// i18nFuncs i18n 函数（或 Msg 类型转换）名 -> 原文参数下标
var i18nFuncs = map[string]int{"T": 1, "Translate": 1, "Msg": 0}

type sourceFile struct {
	path string
	dir  string
	ast  *ast.File
	// imports 包名（或别名）-> 模块内相对目录，仅记录模块内的包
	imports map[string]string
}

// Messages 扫描模块根目录 moduleRoot 下 roots（相对 moduleRoot）中的非测试 Go 文件，返回按原文排序、去重后的消息
func Messages(moduleRoot string, roots ...string) ([]Message, error) {
	modulePath, err := readModulePath(moduleRoot)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	var files []*sourceFile
	for _, root := range roots {
		err = filepath.WalkDir(filepath.Join(moduleRoot, root), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
			if err != nil {
				return errors.Wrapf(err, "parse %s", path)
			}
			rel, _ := filepath.Rel(moduleRoot, path)
			files = append(files, &sourceFile{
				path:    rel,
				dir:     filepath.Dir(rel),
				ast:     f,
				imports: moduleImports(f, modulePath),
			})
			return nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "walk %s", root)
		}
	}

	sentinels := collectSentinels(files)
	found := map[string]Message{}
	add := func(id string, pos token.Pos) {
		if _, ok := found[id]; id == "" || ok {
			return
		}
		p := fset.Position(pos)
		rel, _ := filepath.Rel(moduleRoot, p.Filename)
		found[id] = Message{ID: id, Pos: filepath.ToSlash(rel) + ":" + strconv.Itoa(p.Line)}
	}
	for _, f := range files {
		ast.Inspect(f.ast, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				if expr := messageArg(n, f.ast.Name.Name); expr != nil {
					add(literal(expr), expr.Pos())
				}
			case *ast.CompositeLit:
				if !isStoreMeta(n.Type) {
					return true
				}
				for _, elt := range n.Elts {
					if kv, ok := elt.(*ast.KeyValueExpr); ok && isDomainErrorKey(kv.Key) {
						add(sentinels.resolve(kv.Value, f), kv.Value.Pos())
					}
				}
			}
			return true
		})
	}

	msgs := make([]Message, 0, len(found))
	for _, m := range found {
		msgs = append(msgs, m)
	}
	sort.Slice(msgs, func(i, j int) bool { return msgs[i].ID < msgs[j].ID })
	return msgs, nil
}

// messageArg 返回需要提取的原文参数，不是目标调用时返回 nil；未带包名的调用视为当前包内的调用
func messageArg(call *ast.CallExpr, pkgName string) ast.Expr {
	pkg, fn := callee(call, pkgName)
	if pkg != "i18n" {
		return nil
	}
	idx, ok := i18nFuncs[fn]
	if !ok || len(call.Args) <= idx {
		return nil
	}
	return call.Args[idx]
}

func callee(call *ast.CallExpr, pkgName string) (pkg, fn string) {
	switch f := call.Fun.(type) {
	case *ast.Ident:
		return pkgName, f.Name
	case *ast.SelectorExpr:
		if x, ok := f.X.(*ast.Ident); ok {
			return x.Name, f.Sel.Name
		}
	}
	return "", ""
}

// identName 返回标识符或 pkg.Name 选择器的名称
func identName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	}
	return ""
}

// isStoreMeta 判断类型表达式是否为 store.Meta[T]
func isStoreMeta(expr ast.Expr) bool {
	if idx, ok := expr.(*ast.IndexExpr); ok {
		expr = idx.X
	}
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == "store" && sel.Sel.Name == "Meta"
}

// isDomainErrorKey 判断是否为 Meta 中配置领域错误的字段
func isDomainErrorKey(expr ast.Expr) bool {
	name := identName(expr)
	return name == "NotFound" || name == "Duplicate"
}

// sentinelTable 目录 -> 变量名 -> errors.New 的字面量
type sentinelTable map[string]map[string]string

// collectSentinels 收集以 errors.New("...") 初始化的包级变量
func collectSentinels(files []*sourceFile) sentinelTable {
	t := sentinelTable{}
	for _, f := range files {
		for _, decl := range f.ast.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				vs := spec.(*ast.ValueSpec)
				for i, name := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					call, ok := vs.Values[i].(*ast.CallExpr)
					if !ok || len(call.Args) != 1 {
						continue
					}
					if pkg, fn := callee(call, ""); pkg != "errors" || fn != "New" {
						continue
					}
					if t[f.dir] == nil {
						t[f.dir] = map[string]string{}
					}
					t[f.dir][name.Name] = literal(call.Args[0])
				}
			}
		}
	}
	return t
}

// resolve 解析同包变量或模块内 pkg.Var 引用的错误文本，无法解析时返回空串
func (t sentinelTable) resolve(expr ast.Expr, f *sourceFile) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return t[f.dir][e.Name]
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok {
			return t[f.imports[x.Name]][e.Sel.Name]
		}
	}
	return ""
}

func literal(expr ast.Expr) string {
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		s, _ := strconv.Unquote(lit.Value)
		return s
	}
	return ""
}

func readModulePath(moduleRoot string) (string, error) {
	data, err := os.ReadFile(filepath.Join(moduleRoot, "go.mod"))
	if err != nil {
		return "", errors.Wrap(err, "read go.mod")
	}
	return modfile.ModulePath(data), nil
}

// moduleImports 返回文件中模块内导入的 包名（或别名）-> 相对目录；未起别名时以路径最后一段作为包名
func moduleImports(f *ast.File, modulePath string) map[string]string {
	imports := map[string]string{}
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		rel, ok := strings.CutPrefix(path, modulePath+"/")
		if !ok {
			continue
		}
		name := filepath.Base(rel)
		if imp.Name != nil {
			name = imp.Name.Name
		}
		imports[name] = filepath.FromSlash(rel)
	}
	return imports
}
