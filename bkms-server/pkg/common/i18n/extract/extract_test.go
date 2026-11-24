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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n/extract"
)

func TestExtract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Extract Suite")
}

var _ = Describe("Messages", func() {
	var root string

	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		Expect(os.MkdirAll(filepath.Dir(path), 0o755)).To(Succeed())
		Expect(os.WriteFile(path, []byte(content), 0o644)).To(Succeed())
	}

	ids := func() []string {
		msgs, err := extract.Messages(root, "pkg")
		Expect(err).NotTo(HaveOccurred())
		var res []string
		for _, m := range msgs {
			res = append(res, m.ID)
		}
		return res
	}

	BeforeEach(func() {
		root = GinkgoT().TempDir()
		write("go.mod", "module example.com/demo\n\ngo 1.25\n")
	})

	It("should extract literal messages of non-internal bkerrs constructors and rules", func() {
		write("pkg/a/a.go", `package a

func f(err error, name string, msg string) {
	_ = bkerrs.New(bkerrs.ErrCodeInvalidArgument, "new msg")
	_ = bkerrs.Errorf(bkerrs.ErrCodeNotFound, "app %s not found", name)
	_ = bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "wrap msg")
	_ = bkerrs.Wrapf(err, bkerrs.ErrCodeInvalidRequest, "wrapf %d", 1)
	_ = bkerrs.On(errX, bkerrs.ErrCodeInvalidArgument, "on msg")
	_ = bkerrs.Otherwise(bkerrs.ErrCodeInvalidArgument, "otherwise msg")
	_ = bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "internal msg")
	_ = bkerrs.Errorf(bkerrs.ErrCodeInternalError, "internal %s", name)
	_ = bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, msg)
}
`)
		Expect(ids()).To(Equal([]string{
			"app %s not found", "new msg", "on msg", "otherwise msg", "wrap msg", "wrapf %d",
		}))
	})

	It("should treat unqualified calls inside package bkerrs and i18n as package calls", func() {
		write("pkg/bkerrs/gin.go", `package bkerrs

var x = i18n.Noop("noop msg")

func f(err error) { _ = Wrap(err, ErrCodeNotFound, "resource not found") }
`)
		write("pkg/b/b.go", `package b

func f(ctx context.Context) { _ = i18n.T(ctx, "hello %s", "x") }
`)
		Expect(ids()).To(Equal([]string{"hello %s", "noop msg", "resource not found"}))
	})

	It("should resolve store meta domain sentinels in the same or another package", func() {
		write("pkg/model/errors.go", `package model

import "errors"

var ErrWidgetNotFound = errors.New("widget not found")
`)
		write("pkg/model/store.go", `package model

var ErrWidgetExists = errors.New("widget already exists")

var WidgetMeta = store.Meta[Widget]{NotFound: ErrWidgetNotFound, Duplicate: ErrWidgetExists}
`)
		write("pkg/other/store.go", `package other

import (
	m "example.com/demo/pkg/model"
)

var OtherMeta = store.Meta[Other]{NotFound: m.ErrWidgetNotFound, Collection: "others"}
`)
		Expect(ids()).To(Equal([]string{"widget already exists", "widget not found"}))
	})

	It("should report the first position relative to the module root", func() {
		write("pkg/a/a.go", "package a\n\nvar x = i18n.Noop(\"msg\")\n")
		msgs, err := extract.Messages(root, "pkg")
		Expect(err).NotTo(HaveOccurred())
		Expect(msgs).To(Equal([]extract.Message{{ID: "msg", Pos: "pkg/a/a.go:3"}}))
	})
})
