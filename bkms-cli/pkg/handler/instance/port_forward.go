package instance

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/stringx"
)

// PortForward 启动应用实例端口转发。
func PortForward(ctx context.Context, cli client.Client, opts PortForwardOptions) error {
	if err := opts.Validate(); err != nil {
		return err
	}
	if err := preflightCheckInstance(ctx, cli, opts.AppID, opts.EnvName, opts.InstanceID); err != nil {
		return err
	}

	// 预检 server 端权限（环境类型 + 白名单），避免启动 listener 后才发现无权限
	if err := cli.CheckPortForwardPermission(
		ctx, opts.AppID, opts.EnvName, opts.InstanceID, opts.RemotePort, opts.LocalPort,
	); err != nil {
		return err
	}

	return RunPortForwardListener(ctx, cli, opts)
}

// ParsePortArg 解析 kubectl 风格的端口参数。
// 支持两种格式：
//   - "REMOTE_PORT"：本地端口 = 远程端口
//   - "LOCAL_PORT:REMOTE_PORT"：指定本地端口和远程端口
func ParsePortArg(arg string) (localPort, remotePort int, err error) {
	parts := strings.SplitN(arg, ":", 2)
	switch len(parts) {
	case 1:
		// 单端口格式：REMOTE_PORT（本地端口 = 远程端口）
		port, parseErr := parsePort(parts[0])
		if parseErr != nil {
			return 0, 0, errors.Wrapf(parseErr, "invalid port %q", parts[0])
		}
		return port, port, nil
	case 2:
		// 双端口格式：LOCAL_PORT:REMOTE_PORT
		local, parseErr := parsePort(parts[0])
		if parseErr != nil {
			return 0, 0, errors.Wrapf(parseErr, "invalid local port %q", parts[0])
		}
		remote, parseErr := parsePort(parts[1])
		if parseErr != nil {
			return 0, 0, errors.Wrapf(parseErr, "invalid remote port %q", parts[1])
		}
		return local, remote, nil
	default:
		return 0, 0, errors.Errorf("invalid port argument %q: expected format [LOCAL_PORT:]REMOTE_PORT", arg)
	}
}

// ListenAddress 返回本地监听地址。
func (o *PortForwardOptions) ListenAddress() string {
	return fmt.Sprintf("%s:%d", o.LocalAddress, o.LocalPort)
}

// Validate 校验端口转发参数。
func (o *PortForwardOptions) Validate() error {
	stringx.TrimSpaceRecursive(reflect.ValueOf(o))
	if o.LocalAddress == "" {
		o.LocalAddress = defaultPortForwardLocalAddress
	}
	if err := validator.New().Struct(o); err != nil {
		return err
	}
	if !podNameRegexp.MatchString(o.InstanceID) {
		return errors.Errorf(
			"instanceID must be a valid pod name (lowercase, alphanumeric, hyphens): %s",
			o.InstanceID,
		)
	}
	return nil
}

// CheckInstanceRunning 检查指定实例是否存在且处于 Running 状态。
// TODO: 后续 API 支持服务端 status 过滤后，可减少数据传输量。
func CheckInstanceRunning(ctx context.Context, cli client.Client, appID, envName, instanceID string) (bool, error) {
	instances, err := ListInstances(ctx, cli, appID, envName, ListInstancesOptions{Status: InstanceStatusRunning})
	if err != nil {
		return false, errors.Wrap(err, "query instance list")
	}

	for _, inst := range instances {
		if inst.ID == instanceID {
			return true, nil
		}
	}
	return false, nil
}

// preflightCheckInstance 预检目标实例是否存在且处于 Running 状态。
func preflightCheckInstance(ctx context.Context, cli client.Client, appID, envName, instanceID string) error {
	running, err := CheckInstanceRunning(ctx, cli, appID, envName, instanceID)
	if err != nil {
		return errors.Wrap(err, "preflight check")
	}
	if !running {
		return errors.Errorf("instance %q not found or not running in app %q env %q", instanceID, appID, envName)
	}
	return nil
}

// preflightCheckEnvType 预检目标环境是否为非 production 类型。
// 如果环境在列表中找不到（如 feature env），则跳过检查。
func preflightCheckEnvType(ctx context.Context, cli client.Client, workspaceID, envName string) error {
	envs, err := cli.ListEnvs(ctx, workspaceID)
	if err != nil {
		return errors.Wrap(err, "list envs for env type check")
	}

	for _, env := range envs {
		if env.Name == envName {
			if env.Type == "production" {
				return errors.Errorf(
					"port-forward is only allowed in non-production environments, but %q is %q type",
					envName, env.Type,
				)
			}
			return nil
		}
	}
	// 环境不在标准列表中（可能是 feature env），跳过检查
	return nil
}

// parsePort 将字符串解析为合法端口号
func parsePort(s string) (int, error) {
	port, err := cast.ToIntE(s)
	if err != nil {
		return 0, errors.New("must be a number")
	}
	if port < 1 || port > 65535 {
		return 0, errors.Errorf("port %d must be between 1 and 65535", port)
	}
	return port, nil
}
