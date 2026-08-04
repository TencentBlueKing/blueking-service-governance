package importer

import (
	"context"

	pkgerrors "github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	parserpkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/parser"
	previewpkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/preview"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// Service imports env vars into public/env/app targets using the same parsing and
// scope validation semantics as the preview flow.
type Service struct {
	scopedEnvVarStore envvars.ScopedEnvVarStore
	appModelStore     appmodel.AppModelStore
}

// NewService creates an import service.
func NewService(
	scopedEnvVarStore envvars.ScopedEnvVarStore,
	appModelStore appmodel.AppModelStore,
) *Service {
	return &Service{
		scopedEnvVarStore: scopedEnvVarStore,
		appModelStore:     appModelStore,
	}
}

// ImportPublic imports vars into workspace/envType scopes.
func (s *Service) ImportPublic(ctx context.Context, workspaceID, content string) error {
	records, err := parserpkg.ParseEnvFileRecords(content)
	if err != nil {
		return err
	}

	// 公共导入允许在同一个文件里混用 workspace / envType 两类 scope，
	// 因此先按解析后的实际生效 scope 分组，再分别批量写入。
	grouped := map[envvartypes.ScopedEnvVarScope][]envvars.ScopedEnvVar{}
	scopeOrder := make([]envvartypes.ScopedEnvVarScope, 0, len(records))
	seenScopes := make(map[envvartypes.ScopedEnvVarScope]struct{}, len(records))
	for _, record := range records {
		resolution, err := resolveRecord(record, previewpkg.ResolvePublicRecord)
		if err != nil {
			return err
		}
		scope := resolution.EffectiveScope
		if _, ok := seenScopes[scope]; !ok {
			scopeOrder = append(scopeOrder, scope)
			seenScopes[scope] = struct{}{}
		}
		grouped[scope] = append(grouped[scope], envvars.ScopedEnvVar{
			Key:         record.Key,
			Value:       record.Value,
			Description: record.Description,
			IsBuiltin:   false,
			IsSensitive: false,
		})
	}

	for i, scope := range scopeOrder {
		vars := grouped[scope]
		if err := s.scopedEnvVarStore.BatchUpsertByKey(ctx, workspaceID, scope, vars); err != nil {
			if i > 0 {
				return pkgerrors.Wrapf(
					err,
					"public env var import partially succeeded before failing at scope %s",
					scope,
				)
			}
			return pkgerrors.Wrapf(err, "import public env vars for scope %s", scope)
		}
	}
	return nil
}

func resolveRecord(
	record parserpkg.ParsedEnvVarRecord,
	resolve previewpkg.RecordResolver,
) (*previewpkg.RecordResolution, error) {
	// 把 resolver 的校验失败统一包装成与 preview 相同的 invalid-content 错误，
	// 预览和正式导入返回给用户的错误形态保持一致
	resolution, err := resolve(record)
	if err != nil {
		return nil, pkgerrors.Wrapf(
			parserpkg.ErrInvalidEnvFileContent,
			"line %d: %s",
			record.Line,
			err.Error(),
		)
	}
	if resolution == nil {
		return nil, pkgerrors.New("record resolver returned nil result without error")
	}
	return resolution, nil
}

// ImportEnv imports vars into the target env scope.
// 目标环境由页面上下文指定，因此输入文件不允许声明任何 scope 元数据。
func (s *Service) ImportEnv(ctx context.Context, environment envmodel.Environment, content string) error {
	records, err := parserpkg.ParseEnvFileRecords(content)
	if err != nil {
		return err
	}

	scope := envvartypes.ScopeEnv(environment.Name)
	var vars []envvars.ScopedEnvVar
	resolve := previewpkg.NewEnvRecordResolver(environment)
	for _, record := range records {
		// 单环境导入与应用导入保持一致：目标 scope 完全由页面上下文决定
		if _, err := resolveRecord(record, resolve); err != nil {
			return err
		}
		vars = append(vars, envvars.ScopedEnvVar{
			Key:         record.Key,
			Value:       record.Value,
			Description: record.Description,
			IsBuiltin:   false,
			IsSensitive: false,
		})
	}

	return s.scopedEnvVarStore.BatchUpsertByKey(ctx, environment.WorkspaceID, scope, vars)
}

// ImportApp imports vars into app-defined env vars.
func (s *Service) ImportApp(ctx context.Context, app *bkmsapp.Application, content string) error {
	records, err := parserpkg.ParseEnvFileRecords(content)
	if err != nil {
		return err
	}

	var vars []appmodel.Variable
	for _, record := range records {
		// 应用直接定义变量的导入场景明确禁止任何 scope 元数据，这样导入文件
		// 只能表达 workload.envVars 本身。
		if _, err := resolveRecord(record, previewpkg.ResolveAppRecord); err != nil {
			return err
		}
		vars = append(vars, appmodel.Variable{
			Key:         record.Key,
			Value:       record.Value,
			Description: record.Description,
			IsSensitive: false,
		})
	}

	return appmodel.NewAppEnvVarService(s.appModelStore).BatchUpsert(ctx, app.ID, vars)
}
