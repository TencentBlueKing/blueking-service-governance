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
	"fmt"
	"path/filepath"
	"slices"
	"text/template"
	"text/template/parse"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n/extract"
)

var _ = Describe("Translation keys in source", func() {
	var keys []extract.Key

	BeforeEach(func() {
		var err error
		keys, err = extract.Keys(filepath.Join("..", ".."))
		Expect(err).NotTo(HaveOccurred())
		Expect(keys).NotTo(BeEmpty())
	})

	It("should have a localized message for every key referenced in source", func() {
		for _, key := range keys {
			Expect(
				HasKey(key.Name),
			).To(BeTrue(), "missing message for key %q used at %s, run `make i18n`", key.Name, key.Pos)
		}
	})

	It("should keep template placeholders consistent with the params passed in source", func() {
		var problems []string
		for _, key := range keys {
			if !HasKey(key.Name) {
				continue
			}
			problems = append(problems, placeholderProblems(key)...)
		}
		Expect(problems).To(BeEmpty())
	})
})

// placeholderProblems 检查一个 key 的模板占位符：各语言须一致；不得使用调用处未传入的参数；
// i18n.T 传入的参数须在模板中用到（Define 的参数同时作为错误详情的附加信息，允许只传不用）
func placeholderProblems(key extract.Key) []string {
	var problems []string
	fields := templateFields(messages[key.Name][supportedLangs[0]])
	for _, lang := range supportedLangs[1:] {
		if other := templateFields(messages[key.Name][lang]); !slices.Equal(fields, other) {
			problems = append(problems, fmt.Sprintf("%s: placeholders differ between %s %v and %s %v",
				key.Name, supportedLangs[0], fields, lang, other))
		}
	}
	if key.ParamsUnknown {
		return problems
	}
	for _, f := range fields {
		if !slices.Contains(key.Params, f) {
			problems = append(problems, fmt.Sprintf("%s: placeholder {{.%s}} is not passed at any call site (%s)",
				key.Name, f, key.Pos))
		}
	}
	if key.Kind == extract.KindT {
		for _, p := range key.Params {
			if !slices.Contains(fields, p) {
				problems = append(problems, fmt.Sprintf("%s: param %q is passed at %s but not used in the template",
					key.Name, p, key.Pos))
			}
		}
	}
	return problems
}

// templateFields 返回模板中引用的顶层字段名（如 {{.appID}} 中的 appID），已排序去重
func templateFields(tmpl *template.Template) []string {
	var fields []string
	var walk func(n parse.Node)
	walkBranch := func(b *parse.BranchNode) {
		walk(b.Pipe)
		walk(b.List)
		walk(b.ElseList)
	}
	walk = func(n parse.Node) {
		switch n := n.(type) {
		case *parse.ListNode:
			if n != nil {
				for _, c := range n.Nodes {
					walk(c)
				}
			}
		case *parse.PipeNode:
			if n != nil {
				for _, c := range n.Cmds {
					walk(c)
				}
			}
		case *parse.ActionNode:
			walk(n.Pipe)
		case *parse.CommandNode:
			for _, a := range n.Args {
				walk(a)
			}
		case *parse.FieldNode:
			fields = append(fields, n.Ident[0])
		case *parse.IfNode:
			walkBranch(&n.BranchNode)
		case *parse.RangeNode:
			walkBranch(&n.BranchNode)
		case *parse.WithNode:
			walkBranch(&n.BranchNode)
		}
	}
	walk(tmpl.Root)
	slices.Sort(fields)
	return slices.Compact(fields)
}
