// Package env 提供环境相关的公共处理能力。
package env

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// ResolveEnvIDByName 通过环境名称解析出环境 ID。
// 该函数通过 ListEnvs 获取工作空间下的环境列表，按 name 匹配找到目标环境并返回其 ID。
func ResolveEnvIDByName(ctx context.Context, cli client.Client, workspaceID, envName string) (string, error) {
	envs, err := cli.ListEnvs(ctx, workspaceID)
	if err != nil {
		return "", errors.Wrapf(err, "failed to list envs for workspace %s", workspaceID)
	}

	for i := range envs {
		if envs[i].Name == envName {
			return envs[i].ID, nil
		}
	}

	return "", errors.Errorf("environment '%s' not found in workspace", envName)
}
