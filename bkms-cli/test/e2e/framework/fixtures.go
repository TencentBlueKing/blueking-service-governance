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

// Package framework 提供 e2e 基础框架功能
package framework

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/onsi/ginkgo/v2"
	"gopkg.in/yaml.v3"
)

// testdataRoot 返回 testdata 根目录的绝对路径（基于 framework 包自身位置）
func testdataRoot() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		ginkgo.Fail("failed to get framework file path")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "testdata")
}

// TestdataPath 返回 testdata 子目录下指定文件的绝对路径。
// subDir 为 testdata 下的子目录（如 "appspec"、"app"），
// filename 为文件名（如 "resources.yaml"）。
func TestdataPath(subDir, filename string) string {
	return filepath.Join(testdataRoot(), subDir, filename)
}

// WriteFixtureFile 将任意结构体序列化为 YAML 并写入临时文件，返回文件路径。
// 调用者应在测试结束后清理文件（建议使用 DeferCleanup）。
func WriteFixtureFile(data any, prefix string) string {
	content, err := yaml.Marshal(data)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("failed to marshal fixture to YAML: %v", err))
	}

	f, err := os.CreateTemp("", prefix+"-*.yaml")
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("failed to create temp fixture file: %v", err))
	}
	defer f.Close()

	if _, err = f.Write(content); err != nil {
		ginkgo.Fail(fmt.Sprintf("failed to write fixture file: %v", err))
	}

	Logf("FIXTURE", "Created temp fixture file: %s", f.Name())
	return f.Name()
}

// CleanupFixtureFile 删除指定的临时 fixture 文件
func CleanupFixtureFile(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		Logf("FIXTURE", "Failed to clean up fixture file %s: %v", path, err)
		return
	}
	Logf("FIXTURE", "Cleaned up fixture file: %s", path)
}

// EnvVarFixture 表示一个环境变量条目
type EnvVarFixture struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// NewEnvVarFixtures 创建一组测试用环境变量 fixture
func NewEnvVarFixtures(pairs ...EnvVarFixture) []EnvVarFixture {
	return pairs
}

// PolarisConfigFixture 表示一个北极星配置 fixture
type PolarisConfigFixture struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
	Service   string `yaml:"service"`
}
