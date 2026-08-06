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

// Package helm deploy provides deploy related functions.
package helm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/lint/support"
	"helm.sh/helm/v3/pkg/repo"
)

// PullChart 从 Helm 仓库拉取指定版本的 Chart，并在加载后自动进行 Lint 校验
// 返回加载后的 *chart.Chart 对象和 Lint 校验结果
func PullChart(repoURL, chartName, chartVersion, username, password string) (*chart.Chart, *LintResult, error) {
	// 1. 创建临时目录用于存放下载的 Chart 包
	tmpDir, err := os.MkdirTemp("", "helm-chart-*")
	if err != nil {
		return nil, nil, errors.Wrap(err, "create temp dir for chart pull")
	}
	defer os.RemoveAll(tmpDir)

	// 2. 查找 Chart 在仓库中的下载地址（带认证，支持跳过 TLS 校验）
	chartURL, err := repo.FindChartInAuthAndTLSRepoURL(
		repoURL, username, password,
		chartName, chartVersion, "", "", "", true,
		getter.All(&cli.EnvSettings{}),
	)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "find chart %s version %s in repo %s", chartName, chartVersion, repoURL)
	}

	// 3. 使用 Helm SDK 的 Pull action 拉取 Chart
	pull := action.NewPullWithOpts(action.WithConfig(&action.Configuration{}))
	pull.Settings = &cli.EnvSettings{}
	pull.DestDir = tmpDir
	pull.Version = chartVersion
	pull.Username = username
	pull.Password = password
	pull.Untar = true
	pull.UntarDir = tmpDir

	if _, err = pull.Run(chartURL); err != nil {
		return nil, nil, errors.Wrapf(err, "pull chart %s version %s from %s", chartName, chartVersion, repoURL)
	}

	// 4. 加载解压后的 Chart 目录
	chartDir := filepath.Join(tmpDir, chartName)
	loadedChart, err := loader.Load(chartDir)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "load chart from %s", chartDir)
	}

	// 5. 在临时目录中进行 Lint 校验（此时 Chart 文件仍然存在于磁盘上）
	lintResult, err := LintChart(chartDir)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "lint chart %s", chartName)
	}

	return loadedChart, lintResult, nil
}

// LintChart 对指定路径的 Chart 进行语法和结构校验
func LintChart(chartPath string) (*LintResult, error) {
	lint := action.NewLint()
	result := lint.Run([]string{chartPath}, nil)

	lintResult := &LintResult{}
	for _, msg := range result.Messages {
		switch msg.Severity {
		case support.ErrorSev:
			lintResult.Errors = append(lintResult.Errors, fmt.Sprintf("[ERROR] %s", msg.Error()))
		case support.WarningSev:
			lintResult.Warnings = append(lintResult.Warnings, fmt.Sprintf("[WARN] %s", msg.Error()))
		default:
			lintResult.Infos = append(lintResult.Infos, fmt.Sprintf("[INFO] %s", msg.Error()))
		}
	}

	return lintResult, nil
}
