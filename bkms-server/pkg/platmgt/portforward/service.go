package portforward

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

var (
	// ErrPermissionDenied 表示用户无 port-forward 白名单权限。
	ErrPermissionDenied = errors.New("no port-forward permission")

	// ErrProductionEnvNotAllowed 表示不允许将 production 类型的环境加入白名单。
	ErrProductionEnvNotAllowed = errors.New("production environment is not allowed in port-forward whitelist")

	// ErrEnvNotFound 表示指定的环境 ID 不存在。
	ErrEnvNotFound = errors.New("environment not found")
)

// Service 提供 port-forward 白名单业务操作。
type Service struct {
	store    Store
	envStore envmodel.EnvironmentStore
}

// NewService 创建白名单 Service 实例。
func NewService(store Store, envStore envmodel.EnvironmentStore) *Service {
	return &Service{store: store, envStore: envStore}
}

// Add 将环境 ID 添加到白名单中，添加前校验环境存在性和非 production 类型。
func (s *Service) Add(ctx context.Context, envIDs []string) error {
	envIDs = lo.Uniq(envIDs)
	if err := s.validateEnvIDs(ctx, envIDs); err != nil {
		return err
	}
	return s.store.Add(ctx, envIDs)
}

// Remove 从白名单中移除指定的环境 ID。
func (s *Service) Remove(ctx context.Context, envIDs []string) error {
	return s.store.Remove(ctx, lo.Uniq(envIDs))
}

// List 返回当前白名单中所有环境 ID。
func (s *Service) List(ctx context.Context) ([]string, error) {
	return s.store.List(ctx)
}

// CheckPermission 校验指定环境 ID 是否在白名单中。
// 返回 nil 表示有权限，返回 ErrPermissionDenied 表示无权限。
func (s *Service) CheckPermission(ctx context.Context, envID string) error {
	ok, err := s.store.Contains(ctx, envID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrPermissionDenied
	}
	return nil
}

// validateEnvIDs 校验 envIDs 中每个环境：
// 1. 环境是否存在（通过 envID 查询数据库）
// 2. 环境类型是否为非 production
func (s *Service) validateEnvIDs(ctx context.Context, envIDs []string) error {
	if len(envIDs) == 0 {
		return nil
	}

	for _, envID := range envIDs {
		objID, err := bson.ObjectIDFromHex(envID)
		if err != nil {
			return errors.Wrapf(err, "invalid environment ID %q", envID)
		}

		env, err := s.envStore.Get(ctx, objID)
		if err != nil {
			if errors.Is(err, envmodel.ErrEnvNotFound) {
				return errors.Wrapf(ErrEnvNotFound, "environment %q", envID)
			}
			return err
		}

		// 校验环境类型是否为非 production
		if bkmsenv.IsProductionType(bkmsenv.Type(env.Type)) {
			return ErrProductionEnvNotAllowed
		}
	}

	return nil
}
