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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// sourceRule 一条针对业务代码的静态约束。
// allowlist 的 key 为相对 bkms-server 的路径（以 "/" 结尾表示目录），value 为允许的理由
type sourceRule struct {
	// matches 判断节点是否违反约束
	matches   func(f *sourceFile, n ast.Node) bool
	allowlist map[string]string
	// inScope 限定约束生效的文件，为 nil 时对全部文件生效
	inScope func(rel string) bool
}

// check 返回不在白名单内的违规位置，以及已经没有命中的白名单条目（白名单过期）
func (r sourceRule) check(files []*sourceFile) (violations, stale []string) {
	used := map[string]bool{}
	for _, f := range files {
		if r.inScope != nil && !r.inScope(f.rel) {
			continue
		}
		ast.Inspect(f.file, func(n ast.Node) bool {
			if n == nil || !r.matches(f, n) {
				return true
			}
			allowed := false
			for entry := range r.allowlist {
				if matchesAny(f.rel, map[string]string{entry: ""}) {
					used[entry], allowed = true, true
				}
			}
			if !allowed {
				violations = append(violations, f.pos(n))
			}
			return true
		})
	}
	for entry := range r.allowlist {
		if !used[entry] {
			stale = append(stale, entry)
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
		files, err = loadSources()
		Expect(err).NotTo(HaveOccurred())
		Expect(files).NotTo(BeEmpty())
	})

	expectRule := func(rule sourceRule) {
		violations, stale := rule.check(files)
		Expect(violations).To(BeEmpty(), "violations outside the allowlist")
		Expect(stale).To(BeEmpty(), "allowlist entries without any match")
	}

	// 测的是：业务代码不绕过 GenericStore 直接操作 mongo 集合。
	// 租户过滤与 tenantID 写入都在 GenericStore 里完成，直接调用 Database.Collection(name) 或
	// GenericStore.Collection() 拿到原生集合后，读会看到所有租户的数据，写不会带 tenantID。
	// 只有不经过请求租户、自行处理租户的代码（迁移、测试工具）可以直接访问
	It("forbids raw collection access outside the store and migrations", func() {
		expectRule(sourceRule{
			matches: func(_ *sourceFile, n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return false
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				return ok && sel.Sel.Name == "Collection"
			},
			allowlist: map[string]string{
				"pkg/store/":                                 "GenericStore 自身，负责注入租户条件",
				"pkg/common/testutil/":                       "测试工具按集合清理数据",
				"cmd/migration/":                             "一次性迁移命令，遍历所有租户的存量数据，写入前显式 ScopeToTenant",
				"pkg/extension/component/migrate/":           "组件定义迁移，按旧格式读取所有租户的数据",
				"pkg/workload/appmodelcore/appspec/migrate/": "应用描述迁移，按旧格式读取所有租户的数据，写回时 ScopeToTenant",
			},
		})
	})

	// 测的是：业务代码不把 default 租户当作兜底。
	// 多租户模式下 default 不是任何真实租户，用它兜底会把数据写进一个谁都看不到的租户，
	// 或者让请求读到单租户时期的数据；拿不到租户时应当报错（tenant.GetTenantID 无兜底）
	It("forbids falling back to the default tenant in business code", func() {
		expectRule(sourceRule{
			matches: pkgSelectorRule(modulePath+"/pkg/infras/tenant", "DefaultTenantID"),
			allowlist: map[string]string{
				"pkg/common/testutil/":         "测试默认在 default 租户下运行",
				"pkg/infras/taskq/envelope.go": "兼容升级前投递、没有 _tenantID 的旧任务消息，并打告警日志",
				"cmd/taskq_example.go":         "本地调试用的示例命令",
			},
		})
	})

	// 测的是：业务代码不直接覆盖 ctx 中的租户。
	// 请求租户只能由中间件根据请求头写入，异步任务由 taskq 从消息中恢复；
	// 其余代码需要切换租户时（平台管理、迁移遍历），统一使用 store.ScopeToTenant，方便审计所有切换点
	It("forbids setting the tenant on ctx outside the middleware, taskq and store", func() {
		expectRule(sourceRule{
			matches: pkgSelectorRule(modulePath+"/pkg/infras/tenant", "WithTenantID"),
			allowlist: map[string]string{
				"pkg/store/tenant.go":          "ScopeToTenant 的实现",
				"pkg/infras/taskq/envelope.go": "消费任务时从消息恢复投递方的租户",
				"pkg/common/testutil/":         "测试构造租户上下文",
				"cmd/taskq_example.go":         "本地调试用的示例命令",
			},
		})
	})

	// 测的是：handler 中不用 context.Background() / context.TODO() 新建上下文。
	// 新建的上下文不含请求租户，传给 store 会报 ErrTenantIDRequired，传给下游可能被当作无租户调用；
	// 需要脱离请求生命周期时应使用 context.WithoutCancel(ctx)，它保留租户等请求值
	It("forbids detaching from the request context in handlers", func() {
		expectRule(sourceRule{
			inScope: func(rel string) bool { return strings.Contains(rel, "/handler/") },
			matches: func(f *sourceFile, n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return false
				}
				ctxPkg := f.importName("context")
				return isPkgSelector(call.Fun, ctxPkg, "Background") || isPkgSelector(call.Fun, ctxPkg, "TODO")
			},
			allowlist: map[string]string{
				"pkg/workload/instance/handler/port_forward.go": "只用于控制 WebSocket 隧道的读写生命周期，不访问 store 与下游服务",
			},
		})
	})
})
