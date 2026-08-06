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

package postrenderer

import (
	"bytes"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/postrender"
)

// mockPostRenderer 用于测试的 mock PostRenderer
type mockPostRenderer struct {
	suffix string
	err    error
}

func (m *mockPostRenderer) Run(renderedManifests *bytes.Buffer) (*bytes.Buffer, error) {
	if m.err != nil {
		return nil, m.err
	}
	return bytes.NewBufferString(renderedManifests.String() + m.suffix), nil
}

var _ postrender.PostRenderer = (*mockPostRenderer)(nil)

var _ = Describe("ChainPostRenderer", func() {
	Describe("NewChainPostRenderer", func() {
		It("should return nil when all renderers are nil", func() {
			chain := NewChainPostRenderer(nil, nil, nil)
			Expect(chain).To(BeNil())
		})

		It("should return nil when no renderers provided", func() {
			chain := NewChainPostRenderer()
			Expect(chain).To(BeNil())
		})

		It("should filter nil renderers and return non-nil chain", func() {
			r := &mockPostRenderer{suffix: "-a"}
			chain := NewChainPostRenderer(nil, r, nil)
			Expect(chain).NotTo(BeNil())
			Expect(chain.renderers).To(HaveLen(1))
		})

		It("should keep all non-nil renderers in order", func() {
			r1 := &mockPostRenderer{suffix: "-a"}
			r2 := &mockPostRenderer{suffix: "-b"}
			chain := NewChainPostRenderer(r1, r2)
			Expect(chain).NotTo(BeNil())
			Expect(chain.renderers).To(HaveLen(2))
		})
	})

	Describe("Run", func() {
		It("should execute renderers in order", func() {
			r1 := &mockPostRenderer{suffix: "-first"}
			r2 := &mockPostRenderer{suffix: "-second"}
			chain := NewChainPostRenderer(r1, r2)

			input := bytes.NewBufferString("base")
			output, err := chain.Run(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(output.String()).To(Equal("base-first-second"))
		})

		It("should stop and return error when a renderer fails", func() {
			r1 := &mockPostRenderer{suffix: "-ok"}
			r2 := &mockPostRenderer{err: fmt.Errorf("render failed")}
			r3 := &mockPostRenderer{suffix: "-never"}
			chain := NewChainPostRenderer(r1, r2, r3)

			input := bytes.NewBufferString("base")
			_, err := chain.Run(input)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("render failed"))
		})

		It("should pass through input when only one renderer", func() {
			r := &mockPostRenderer{suffix: "-only"}
			chain := NewChainPostRenderer(r)

			input := bytes.NewBufferString("data")
			output, err := chain.Run(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(output.String()).To(Equal("data-only"))
		})
	})
})
