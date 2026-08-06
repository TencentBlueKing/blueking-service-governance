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

// Package deploy 部署，分流处理不同类型应用的部署需求。
package deploy

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
)

// CreateDeploy 创建部署，支持多环境（逗号分隔）
func CreateDeploy(ctx context.Context, workspaceID, appID, envName, deploySpecFile string) error {
	// 解析多环境名称
	envNames := parseEnvNames(envName)
	if len(envNames) == 0 {
		return errors.New("env name is required")
	}

	cli := client.New()

	// 校验所有环境名称合法性
	if err := validateEnvNames(ctx, cli, workspaceID, envNames); err != nil {
		return err
	}

	// 获取应用信息（只需一次）
	app, err := cli.GetAppMinimal(ctx, workspaceID, appID)
	if err != nil {
		return err
	}

	// 检查文件是否存在
	if _, err = os.Stat(deploySpecFile); err != nil {
		return errors.Wrapf(err, "deploy spec file %s not found", deploySpecFile)
	}

	file, err := os.ReadFile(deploySpecFile)
	if err != nil {
		return errors.Wrapf(err, "read deploy spec file %s failed", deploySpecFile)
	}

	// 对每个环境依次执行部署
	var errs []string
	for _, env := range envNames {
		if deployErr := createDeployForEnv(ctx, cli, app, env, file, deploySpecFile); deployErr != nil {
			errs = append(errs, fmt.Sprintf("env %s: %v", env, deployErr))
			fmt.Printf("✗ Deploy failed for env: %s (%v)\n", env, deployErr)
		} else {
			fmt.Printf("✓ Deploy created successfully for env: %s\n", env)
		}
	}

	if len(errs) > 0 {
		return errors.Errorf("deploy failed for some envs:\n  %s", strings.Join(errs, "\n  "))
	}

	return nil
}

// createDeployForEnv 对单个环境执行部署
func createDeployForEnv(
	ctx context.Context,
	cli client.Client,
	app *client.AppMinimal,
	envName string,
	file []byte,
	deploySpecFile string,
) error {
	switch app.Type {
	case constant.AppTypeHelm:
		opts := new(client.HelmDeployOptions)
		if err := yaml.Unmarshal(file, opts); err != nil {
			return errors.Wrapf(err, "parse deploy spec file %s failed", deploySpecFile)
		}
		if err := opts.Validate(); err != nil {
			return err
		}
		return cli.CreateAppHelmDeploy(ctx, app.ID, envName, *opts)

	case constant.AppTypeTrpc:
		opts := new(client.AppModelDeployOptions)
		if err := yaml.Unmarshal(file, opts); err != nil {
			return errors.Wrapf(err, "parse deploy spec file %s failed", deploySpecFile)
		}
		if err := opts.Validate(); err != nil {
			return err
		}
		return cli.CreateAppTrpcDeploy(ctx, app.ID, envName, *opts)

	case constant.AppTypeTaf:
		opts := new(client.AppModelDeployOptions)
		if err := yaml.Unmarshal(file, opts); err != nil {
			return errors.Wrapf(err, "parse deploy spec file %s failed", deploySpecFile)
		}
		if err := opts.Validate(); err != nil {
			return err
		}
		return cli.CreateAppTafDeploy(ctx, app.ID, envName, *opts)
	default:
		return errors.Errorf("unknown app type: %s", app.Type)
	}
}
