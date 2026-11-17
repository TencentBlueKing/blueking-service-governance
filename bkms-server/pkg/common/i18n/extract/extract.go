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

// Package extract 静态扫描 Go 源码，提取需要提供翻译的 key：bkerrs.Define 的错误详情码与 i18n.T 的字面量 key
//
// 本包不能依赖 i18n：翻译资源不完整时 i18n 加载即 panic，而补全翻译的工具恰恰要在这种状态下运行
package extract

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Kind 翻译 key 的来源
type Kind int

const (
	// KindDefine bkerrs.Define 的错误详情码，参数经 .With 传入，同时作为错误详情的附加信息
	KindDefine Kind = iota
	// KindT i18n.T 的字面量 key，参数经 map 字面量传入，仅用于渲染文案
	KindT
)

// Key 源码中引用的一个翻译 key
type Key struct {
	Name string
	Kind Kind
	// Default 为 Define 时给开发者看的英文描述；i18n.T 的 key 没有默认文案，为空
	Default string
	// Params 为各调用处传入的模板参数名的并集，已排序
	Params []string
	// ParamsUnknown 表示有调用处的参数无法静态解析（如参数名不是常量、map 以变量传入），此时 Params 可能不全
	ParamsUnknown bool
	// Dir 为 key 所在源文件的目录
	Dir string
	// Pos 为 key 在源码中的位置，形如 file:line:col
	Pos string
}

type sourceFile struct {
	dir string
	ast *ast.File
}

// Keys 扫描 roots 下的非测试 Go 文件，返回按名称排序、去重后的 key（同一 key 多处引用时保留首次出现的位置）
//
// 详情码与描述支持字符串字面量、ErrDetailCode("...") 转换以及字符串常量（同包或 bkerrs 包）；
// 无法静态解析的 Define 会被跳过，由运行时遍历 bkerrs.Definitions() 的测试兜底。
// Define 的参数取自包级错误变量、Define 调用本身或由它们赋值的局部变量上的 .With 调用链；
// 以函数参数等其他方式传递的错误上的 .With 无法识别
func Keys(roots ...string) ([]Key, error) {
	fset := token.NewFileSet()
	var files []sourceFile
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
			if err != nil {
				return errors.Wrapf(err, "parse %s", path)
			}
			files = append(files, sourceFile{dir: filepath.Dir(path), ast: f})
			return nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "walk %s", root)
		}
	}

	consts := collectConsts(files)
	defines := collectDefines(files, consts)
	found := map[string]*Key{}
	params := map[string]map[string]bool{}
	unknown := map[string]bool{}
	addParams := func(key string, names []string, ok bool) {
		if params[key] == nil {
			params[key] = map[string]bool{}
		}
		for _, name := range names {
			params[key][name] = true
		}
		if !ok {
			unknown[key] = true
		}
	}

	for _, f := range files {
		pkgName := f.ast.Name.Name
		errorKey := func(expr ast.Expr, locals map[string]string) string {
			root := receiverRoot(expr, pkgName)
			if isDefineExpr(root, pkgName) {
				return consts.resolve(root.(*ast.CallExpr).Args[1], f.dir)
			}
			if id, ok := root.(*ast.Ident); ok && locals[id.Name] != "" {
				return locals[id.Name]
			}
			return defines.lookup(root, f.dir)
		}
		locals := collectLocals(f.ast, errorKey)
		ast.Inspect(f.ast, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if key, param, ok := withParam(call, f.dir, consts, func(e ast.Expr) string {
				return errorKey(e, locals)
			}); key != "" {
				addParams(key, []string{param}, ok)
				return true
			}

			var key Key
			switch {
			case isDefine(call, pkgName):
				key.Kind = KindDefine
				key.Name = consts.resolve(call.Args[1], f.dir)
				key.Default = consts.resolve(call.Args[2], f.dir)
			case isCall(call, pkgName, "i18n", "T") && len(call.Args) >= 2:
				key.Kind = KindT
				// T 约定 key 只能是字面量，非字面量的调用无法静态检查
				if lit, ok := call.Args[1].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					key.Name, _ = strconv.Unquote(lit.Value)
				}
				if key.Name != "" {
					var names []string
					ok := true
					if len(call.Args) >= 3 {
						names, ok = consts.mapKeys(call.Args[2], f.dir)
					}
					addParams(key.Name, names, ok)
				}
			default:
				return true
			}
			if _, exists := found[key.Name]; key.Name == "" || exists {
				return true
			}
			key.Dir = f.dir
			key.Pos = fset.Position(call.Args[1].Pos()).String()
			found[key.Name] = &key
			return true
		})
	}

	keys := make([]Key, 0, len(found))
	for name, k := range found {
		for p := range params[name] {
			k.Params = append(k.Params, p)
		}
		sort.Strings(k.Params)
		k.ParamsUnknown = unknown[name]
		keys = append(keys, *k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Name < keys[j].Name })
	return keys, nil
}

// withParam 识别 err.With("param", v) 形式的调用，返回 err 对应的错误详情码与参数名；
// ok 为 false 表示参数名无法静态解析。不是 With 调用或无法定位到 Define 时 key 为空
func withParam(
	call *ast.CallExpr,
	dir string,
	consts constTable,
	errorKey func(ast.Expr) string,
) (key, param string, ok bool) {
	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel || sel.Sel.Name != "With" || len(call.Args) != 2 {
		return "", "", false
	}
	if key = errorKey(sel.X); key == "" {
		return "", "", false
	}
	param = consts.resolve(call.Args[0], dir)
	return key, param, param != ""
}

// collectLocals 收集文件内由错误变量赋值而来的局部变量（如 bkErr := errX），值为其错误详情码；
// 不区分函数作用域，按出现顺序处理以支持多次中转
func collectLocals(f *ast.File, errorKey func(ast.Expr, map[string]string) string) map[string]string {
	locals := map[string]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != len(assign.Rhs) {
			return true
		}
		for i, lhs := range assign.Lhs {
			if id, ok := lhs.(*ast.Ident); ok {
				if key := errorKey(assign.Rhs[i], locals); key != "" {
					locals[id.Name] = key
				}
			}
		}
		return true
	})
	return locals
}

// receiverRoot 沿方法调用链（如 errX.WithCause(err).With(...)）回溯到最初的接收者
func receiverRoot(expr ast.Expr, pkgName string) ast.Expr {
	for {
		call, ok := expr.(*ast.CallExpr)
		if !ok || isDefine(call, pkgName) {
			return expr
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return expr
		}
		expr = sel.X
	}
}

func isDefineExpr(expr ast.Expr, pkgName string) bool {
	call, ok := expr.(*ast.CallExpr)
	return ok && isDefine(call, pkgName)
}

func isDefine(call *ast.CallExpr, pkgName string) bool {
	return isCall(call, pkgName, "bkerrs", "Define") && len(call.Args) == 3
}

// isCall 判断是否调用了 pkg.fn，未带包名的调用视为当前包
func isCall(call *ast.CallExpr, currentPkg, pkg, fn string) bool {
	switch f := call.Fun.(type) {
	case *ast.Ident:
		return currentPkg == pkg && f.Name == fn
	case *ast.SelectorExpr:
		x, ok := f.X.(*ast.Ident)
		return ok && x.Name == pkg && f.Sel.Name == fn
	}
	return false
}

// constTable 按名称索引的字符串值，byDir 用于解析同包引用，byPkg 用于解析 pkg.Name 形式的引用
type constTable struct {
	byDir map[string]map[string]string
	byPkg map[string]map[string]string
}

func newConstTable() constTable {
	return constTable{byDir: map[string]map[string]string{}, byPkg: map[string]map[string]string{}}
}

func (t constTable) put(f sourceFile, name, value string) {
	put(t.byDir, f.dir, name, value)
	put(t.byPkg, f.ast.Name.Name, name, value)
}

func put(m map[string]map[string]string, scope, name, value string) {
	if m[scope] == nil {
		m[scope] = map[string]string{}
	}
	m[scope][name] = value
}

// collectConsts 收集包级字符串常量
func collectConsts(files []sourceFile) constTable {
	t := newConstTable()
	forEachValueSpec(files, token.CONST, func(f sourceFile, name string, value ast.Expr) {
		if v := literal(value); v != "" {
			t.put(f, name, v)
		}
	})
	return t
}

// collectDefines 收集由 bkerrs.Define 初始化的包级错误变量，值为其错误详情码
func collectDefines(files []sourceFile, consts constTable) constTable {
	t := newConstTable()
	forEachValueSpec(files, token.VAR, func(f sourceFile, name string, value ast.Expr) {
		if call, ok := value.(*ast.CallExpr); ok && isDefine(call, f.ast.Name.Name) {
			if key := consts.resolve(call.Args[1], f.dir); key != "" {
				t.put(f, name, key)
			}
		}
	})
	return t
}

func forEachValueSpec(files []sourceFile, tok token.Token, fn func(f sourceFile, name string, value ast.Expr)) {
	for _, f := range files {
		for _, decl := range f.ast.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != tok {
				continue
			}
			for _, spec := range gen.Specs {
				vs := spec.(*ast.ValueSpec)
				for i, name := range vs.Names {
					if i < len(vs.Values) {
						fn(f, name.Name, vs.Values[i])
					}
				}
			}
		}
	}
}

// lookup 解析同包标识符或 pkg.Name 引用，未找到时返回空串
func (t constTable) lookup(expr ast.Expr, dir string) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return t.byDir[dir][e.Name]
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok {
			return t.byPkg[x.Name][e.Sel.Name]
		}
	}
	return ""
}

func (t constTable) resolve(expr ast.Expr, dir string) string {
	if call, ok := expr.(*ast.CallExpr); ok && len(call.Args) == 1 {
		return t.resolve(call.Args[0], dir)
	}
	if v := t.lookup(expr, dir); v != "" {
		return v
	}
	return literal(expr)
}

// mapKeys 解析 map 字面量的键，ok 为 false 表示不是 map 字面量或存在无法解析的键
func (t constTable) mapKeys(expr ast.Expr, dir string) (names []string, ok bool) {
	lit, isLit := expr.(*ast.CompositeLit)
	if !isLit {
		return nil, false
	}
	for _, elt := range lit.Elts {
		kv, isKV := elt.(*ast.KeyValueExpr)
		if !isKV {
			return names, false
		}
		name := t.resolve(kv.Key, dir)
		if name == "" {
			return names, false
		}
		names = append(names, name)
	}
	return names, true
}

// literal 解析字符串字面量及其类型转换（如 ErrDetailCode("X")），其他表达式返回空串
func literal(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			s, _ := strconv.Unquote(e.Value)
			return s
		}
	case *ast.CallExpr:
		if len(e.Args) == 1 {
			return literal(e.Args[0])
		}
	}
	return ""
}
