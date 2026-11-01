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
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// 规则名，豁免注释通过它指明豁免哪条规则
const (
	ruleRawCollection = "raw-collection"
	ruleDefaultTenant = "default-tenant"
	ruleSetTenant     = "set-tenant"
	ruleDetachedCtx   = "detached-ctx"
)

var knownRules = []string{ruleRawCollection, ruleDefaultTenant, ruleSetTenant, ruleDetachedCtx}

// sourceRule 一条针对业务代码的静态约束。
// 确有需要的调用点在源码中写 "//tenancy:allow <规则名>[,<规则名>] <理由>" 豁免，
// 注释放在违规所在行的行尾，或违规所在语句的上一行
type sourceRule struct {
	name string
	// matches 判断节点是否违反约束
	matches func(f *sourceFile, n ast.Node) bool
	// inScope 限定约束生效的文件，为 nil 时对全部文件生效
	inScope func(rel string) bool
}

// check 返回未被豁免的违规位置，以及指明本规则却没有豁免到任何违规的注释位置（豁免过期）
func (r sourceRule) check(files []*sourceFile) (violations, stale []string) {
	for _, f := range files {
		used := make([]bool, len(f.allows))
		if r.inScope == nil || r.inScope(f.rel) {
			// stack 为当前节点及其祖先，用于找到违规所在的语句
			var stack []ast.Node
			ast.Inspect(f.file, func(n ast.Node) bool {
				if n == nil {
					stack = stack[:len(stack)-1]
					return true
				}
				stack = append(stack, n)
				if !r.matches(f, n) {
					return true
				}
				line, stmtLine := f.line(n), f.line(n)
				for i := len(stack) - 1; i >= 0; i-- {
					if stmt, ok := stack[i].(ast.Stmt); ok {
						stmtLine = f.line(stmt)
						break
					}
				}
				allowed := false
				for i, a := range f.allows {
					if slices.Contains(a.rules, r.name) && (a.line == line || a.line == stmtLine-1) {
						used[i], allowed = true, true
					}
				}
				if !allowed {
					violations = append(violations, f.pos(line))
				}
				return true
			})
		}
		for i, a := range f.allows {
			if slices.Contains(a.rules, r.name) && !used[i] {
				stale = append(stale, f.pos(a.line))
			}
		}
	}
	return violations, stale
}

// pkgSelectorRule 匹配对 importPath 包中 name 的引用，例如 tenant.DefaultTenantID
func pkgSelectorRule(importPath, name string) func(f *sourceFile, n ast.Node) bool {
	return func(f *sourceFile, n ast.Node) bool {
		return isPkgSelector(asExpr(n), f.importName(importPath), name)
	}
}

func asExpr(n ast.Node) ast.Expr {
	e, _ := n.(ast.Expr)
	return e
}

var _ = Describe("Tenant isolation source rules", func() {
	var files []*sourceFile

	BeforeEach(func() {
		var err error
		files, err = cachedSources()
		Expect(err).NotTo(HaveOccurred())
		Expect(files).NotTo(BeEmpty())
	})

	expectRule := func(rule sourceRule) {
		violations, stale := rule.check(files)
		Expect(violations).To(BeEmpty(), "violations without a //tenancy:allow %s comment", rule.name)
		Expect(stale).To(BeEmpty(), "//tenancy:allow %s comments that allow nothing", rule.name)
	}

	// 测的是：豁免注释格式正确，规则名都存在且写了理由。
	// 拼错的规则名不会豁免任何违规，但也不会被各规则的过期检查发现
	It("requires every tenancy:allow comment to name known rules with a reason", func() {
		var invalid []string
		for _, f := range files {
			for _, a := range f.allows {
				unknown := slices.ContainsFunc(a.rules, func(name string) bool {
					return !slices.Contains(knownRules, name)
				})
				if len(a.rules) == 0 || unknown {
					invalid = append(invalid, f.pos(a.line))
				}
			}
		}
		Expect(invalid).To(BeEmpty(), "rule names must be one of %v", knownRules)
	})

	// 测的是：业务代码不绕过 GenericStore 直接操作 mongo 集合。
	// 租户过滤与 tenantID 写入都在 GenericStore 里完成，直接调用 Database.Collection(name) 或
	// GenericStore.Collection() 拿到原生集合后，读会看到所有租户的数据，写不会带 tenantID。
	// 只有不经过请求租户、自行处理租户的代码（GenericStore 自身、迁移、测试工具）可以豁免
	It("forbids raw collection access outside the store and migrations", func() {
		expectRule(sourceRule{
			name: ruleRawCollection,
			matches: func(_ *sourceFile, n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return false
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				return ok && sel.Sel.Name == "Collection"
			},
		})
	})

	// 测的是：业务代码不把 default 租户当作兜底。
	// 多租户模式下 default 不是任何真实租户，用它兜底会把数据写进一个谁都看不到的租户，
	// 或者让请求读到单租户时期的数据；拿不到租户时应当报错（tenant.GetTenantID 无兜底）
	It("forbids falling back to the default tenant in business code", func() {
		expectRule(sourceRule{
			name:    ruleDefaultTenant,
			matches: pkgSelectorRule(modulePath+"/pkg/infras/tenant", "DefaultTenantID"),
		})
	})

	// 测的是：业务代码不直接覆盖 ctx 中的租户。
	// 请求租户只能由中间件根据请求头写入，异步任务由 taskq 从消息中恢复；
	// 其余代码需要切换租户时（平台管理、迁移遍历），统一使用 store.ScopeToTenant，方便审计所有切换点
	It("forbids setting the tenant on ctx outside the middleware, taskq and store", func() {
		expectRule(sourceRule{
			name:    ruleSetTenant,
			matches: pkgSelectorRule(modulePath+"/pkg/infras/tenant", "WithTenantID"),
		})
	})

	// 测的是：handler 中不用 context.Background() / context.TODO() 新建上下文。
	// 新建的上下文不含请求租户，传给 store 会报 ErrTenantIDRequired，传给下游可能被当作无租户调用；
	// 需要脱离请求生命周期时应使用 context.WithoutCancel(ctx)，它保留租户等请求值
	It("forbids detaching from the request context in handlers", func() {
		expectRule(sourceRule{
			name:    ruleDetachedCtx,
			inScope: func(rel string) bool { return strings.Contains(rel, "/handler/") },
			matches: func(f *sourceFile, n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return false
				}
				ctxPkg := f.importName("context")
				return isPkgSelector(call.Fun, ctxPkg, "Background") || isPkgSelector(call.Fun, ctxPkg, "TODO")
			},
		})
	})
})
