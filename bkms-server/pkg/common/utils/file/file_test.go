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

package file_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/file"
)

var _ = Describe("SafeReadFile", func() {
	var (
		baseDir     string
		testContent []byte
	)

	BeforeEach(func() {
		var err error
		baseDir, err = os.MkdirTemp("", "safe-read-test-*")
		Expect(err).NotTo(HaveOccurred())

		testContent = []byte("hello world")
		err = os.WriteFile(filepath.Join(baseDir, "test.txt"), testContent, 0o644)
		Expect(err).NotTo(HaveOccurred())

		// 创建子目录和嵌套文件
		subDir := filepath.Join(baseDir, "subdir")
		Expect(os.MkdirAll(subDir, 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested content"), 0o644)).To(Succeed())
	})

	AfterEach(func() {
		os.RemoveAll(baseDir)
	})

	It("should read file in base directory", func() {
		content, err := file.SafeReadFile(baseDir, "test.txt")
		Expect(err).NotTo(HaveOccurred())
		Expect(content).To(Equal(testContent))
	})

	It("should read file in subdirectory", func() {
		content, err := file.SafeReadFile(baseDir, "subdir/nested.txt")
		Expect(err).NotTo(HaveOccurred())
		Expect(content).To(Equal([]byte("nested content")))
	})

	It("should reject path with ../ escaping base directory", func() {
		_, err := file.SafeReadFile(baseDir, "../../etc/passwd")
		Expect(err).To(HaveOccurred())
	})

	It("should reject bare .. path", func() {
		_, err := file.SafeReadFile(baseDir, "..")
		Expect(err).To(HaveOccurred())
	})

	It("should return not exist error for missing file", func() {
		_, err := file.SafeReadFile(baseDir, "nonexistent.txt")
		Expect(err).To(HaveOccurred())
		Expect(os.IsNotExist(err)).To(BeTrue())
	})
})
