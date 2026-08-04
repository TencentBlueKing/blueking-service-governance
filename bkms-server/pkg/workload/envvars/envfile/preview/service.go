package preview

import (
	"context"

	pkgerrors "github.com/pkg/errors"
	"github.com/samber/lo"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	parserpkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/parser"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// ImportPreviewService 提供环境变量导入预览的共享逻辑，
// 支持公共（workspace/envType）、单环境、应用三种导入上下文
type ImportPreviewService struct {
	scopedEnvVarStore envvars.ScopedEnvVarStore
	appModelStore     appmodel.AppModelStore
}

type existingVarSnapshot struct {
	Value       string
	IsSensitive bool
}

type existingVarLookup func(
	record parserpkg.ParsedEnvVarRecord,
	resolution RecordResolution,
) (existingVarSnapshot, bool)

type publicScopeKey struct {
	ScopeType  envvartypes.ScopeType
	ScopeValue string
	Key        string
}

// NewPreviewService 创建一个环境变量导入预览服务实例
func NewPreviewService(
	scopedEnvVarStore envvars.ScopedEnvVarStore,
	appModelStore appmodel.AppModelStore,
) *ImportPreviewService {
	return &ImportPreviewService{
		scopedEnvVarStore: scopedEnvVarStore,
		appModelStore:     appModelStore,
	}
}

// PreviewPublic 预览将环境变量导入到公共作用域（workspace / envType 级别）。
// 导入文件必须显式声明结构化 scope 元数据。
func (s *ImportPreviewService) PreviewPublic(
	ctx context.Context,
	workspaceID, content string,
) (*ImportPreview, error) {
	// 解析并校验导入内容，遇到格式错误/非法 key/重复 key 立即返回错误
	records, err := parserpkg.ParseEnvFileRecords(content)
	if err != nil {
		return nil, err
	}

	// 查询工作空间下已有的公共环境变量，用于判断新增还是覆盖
	existingVars, err := s.scopedEnvVarStore.ListPublic(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	// 组装预览结果，使用公共 scope 解析策略。
	return buildImportPreview(
		records,
		buildPublicExistingVarLookup(existingVars),
		ResolvePublicRecord,
	)
}

// PreviewEnv 预览将环境变量导入到指定的单一环境作用域。
// 导入目标由当前页面上下文决定，文件中不允许声明任何 scope 元数据。
// 注意：overwrite 仅表示目标 env scope 下已直接定义过同名 key；
// 若只是 workspace / envType 公共变量在该环境中生效，不视为 overwrite。
func (s *ImportPreviewService) PreviewEnv(
	ctx context.Context,
	environment envmodel.Environment,
	content string,
) (*ImportPreview, error) {
	// 解析并校验导入内容
	records, err := parserpkg.ParseEnvFileRecords(content)
	if err != nil {
		return nil, err
	}

	// 仅查询目标 env scope 下直接定义的变量，用于判断 overwrite。
	// 这里不读取当前环境的“最终生效变量”，避免把继承自 workspace / envType
	// 的公共变量误判成 overwrite。
	existingVars, err := s.scopedEnvVarStore.List(
		ctx,
		environment.WorkspaceID,
		envvars.WithScopes(envvartypes.ScopeEnv(environment.Name)),
	)
	if err != nil {
		return nil, err
	}

	// 组装预览结果，scope 固定为目标环境
	return buildImportPreview(
		records,
		buildExistingVarLookupFromScopedVars(existingVars),
		NewEnvRecordResolver(environment),
	)
}

// PreviewApp 预览将环境变量导入到指定应用的自定义环境变量。
// 此场景下不允许声明任何 scope 元数据。
// 注意：overwrite 仅表示应用已直接定义过同名自定义环境变量；
// 若只是背景变量（如内置变量、公共变量、依赖服务变量）中存在同名 key，不视为 overwrite。
func (s *ImportPreviewService) PreviewApp(
	ctx context.Context,
	app *bkmsapp.Application,
	content string,
) (*ImportPreview, error) {
	// 解析并校验导入内容。
	records, err := parserpkg.ParseEnvFileRecords(content)
	if err != nil {
		return nil, err
	}

	// 仅查询应用自身已定义的环境变量，用于判断 overwrite。
	appEnvVars, err := s.appModelStore.ListAppDefinedEnvVars(ctx, app.ID)
	if err != nil && !pkgerrors.Is(err, appmodel.ErrAppModelNotFound) {
		return nil, err
	}

	// 组装预览结果，app 导入不接受 scope 元数据。
	return buildImportPreview(records, buildExistingVarLookupFromAppVars(appEnvVars), ResolveAppRecord)
}

// buildImportPreview 将已通过校验的解析记录转换为预览条目，并统计汇总信息。
// 每条记录通过 resolve 策略确定 scope 生效情况，再与目标作用域下已直接存在的变量比对。
// overwrite 判断只针对本次目标写入范围内直接存在的数据，而不是最终生效的全部变量。
func buildImportPreview(
	records []parserpkg.ParsedEnvVarRecord,
	lookup existingVarLookup,
	resolve RecordResolver,
) (*ImportPreview, error) {
	items := make([]ImportPreviewItem, 0, len(records))
	summary := ImportPreviewSummary{}

	for _, record := range records {
		// 根据导入环境解析scope
		res, err := resolve(record)
		if err != nil {
			return nil, pkgerrors.Wrapf(parserpkg.ErrInvalidEnvFileContent, "line %d: %s", record.Line, err.Error())
		}
		if res == nil {
			return nil, pkgerrors.Wrapf(
				parserpkg.ErrInvalidEnvFileContent,
				"line %d: resolver returned nil result",
				record.Line,
			)
		}

		item := ImportPreviewItem{
			Key:           record.Key,
			Value:         record.Value,
			Description:   record.Description,
			DeclaredScope: buildDeclaredPreviewScope(record),
			Action:        ImportActionNew,
			EffectScope:   res.EffectStatus,
			Messages:      res.Messages,
		}
		if res.EffectiveScope.ScopeType != "" {
			item.EffectiveScope = &ImportPreviewScope{
				Type:  string(res.EffectiveScope.ScopeType),
				Value: res.EffectiveScope.ScopeValue,
			}
		}

		// 判断是新增还是覆盖，如果覆盖则记录原值。
		if existing, ok := lookup(record, *res); ok {
			item.Action = ImportActionOverwrite
			if existing.IsSensitive {
				item.OriginalValue = envvartypes.SensitiveValueMask
			} else {
				item.OriginalValue = existing.Value
			}
		}

		// 更新汇总计数
		switch item.Action {
		case ImportActionNew:
			summary.New++
		case ImportActionOverwrite:
			summary.Overwrite++
		}

		items = append(items, item)
	}

	summary.Total = len(items)
	return &ImportPreview{
		Items:   items,
		Summary: summary,
	}, nil
}

// buildPublicExistingVarLookup 构造一个用于“公共作用域”导入预览的现有变量查询器。
// 公共变量按 (scopeType, scopeValue, key) 三元组唯一定位，因此这里以 publicScopeKey
// 为索引构建查找表，查询时根据记录解析出的生效 scope 与 key 进行匹配。
// 适用于 PreviewPublic：工作空间/环境类型级别的变量覆盖判断。
func buildPublicExistingVarLookup(vars []envvars.ScopedEnvVar) existingVarLookup {
	byScopeAndKey := lo.SliceToMap(vars, func(v envvars.ScopedEnvVar) (publicScopeKey, existingVarSnapshot) {
		return publicScopeKey{
				ScopeType:  v.ScopeType,
				ScopeValue: v.ScopeValue,
				Key:        v.Key,
			}, existingVarSnapshot{
				Value:       v.Value,
				IsSensitive: v.IsSensitive,
			}
	})
	return func(record parserpkg.ParsedEnvVarRecord, resolution RecordResolution) (existingVarSnapshot, bool) {
		key := publicScopeKey{
			ScopeType:  resolution.EffectiveScope.ScopeType,
			ScopeValue: resolution.EffectiveScope.ScopeValue,
			Key:        record.Key,
		}
		existing, ok := byScopeAndKey[key]
		return existing, ok
	}
}

// buildExistingVarLookupFromScopedVars 构造一个用于“单环境作用域”导入预览的现有变量查询器。
// 目标 env scope 下直接定义的变量仅按 key 唯一，因此以 key 为索引构建查找表，忽略 resolution。
// 适用于 PreviewEnv：只判断目标环境下直接定义过的同名 key，不把继承自公共变量的同名 key 视为覆盖。
func buildExistingVarLookupFromScopedVars(vars []envvars.ScopedEnvVar) existingVarLookup {
	byKey := lo.SliceToMap(vars, func(v envvars.ScopedEnvVar) (string, existingVarSnapshot) {
		return v.Key, existingVarSnapshot{
			Value:       v.Value,
			IsSensitive: v.IsSensitive,
		}
	})
	return func(record parserpkg.ParsedEnvVarRecord, _ RecordResolution) (existingVarSnapshot, bool) {
		existing, ok := byKey[record.Key]
		return existing, ok
	}
}

// buildExistingVarLookupFromAppVars 构造一个用于“应用自定义变量”导入预览的现有变量查询器。
// 应用自身定义的变量仅按 key 唯一，因此以 key 为索引构建查找表，忽略 resolution。
// 适用于 PreviewApp：只判断应用已直接定义过的同名自定义变量，不把背景变量（内置/公共/依赖服务）中的同名 key 视为覆盖。
func buildExistingVarLookupFromAppVars(vars []appmodel.Variable) existingVarLookup {
	byKey := lo.SliceToMap(vars, func(v appmodel.Variable) (string, existingVarSnapshot) {
		return v.Key, existingVarSnapshot{
			Value:       v.Value,
			IsSensitive: v.IsSensitive,
		}
	})
	return func(record parserpkg.ParsedEnvVarRecord, _ RecordResolution) (existingVarSnapshot, bool) {
		existing, ok := byKey[record.Key]
		return existing, ok
	}
}

// buildDeclaredPreviewScope 根据解析记录中显式声明的 scope 元数据构造预览用的 scope 结构。
// 仅当记录声明了 scopeType 或 scopeValue 时才返回非 nil 的 *ImportPreviewScope；
// 若两者均未声明（即未指定任何 scope 元数据），返回 nil，表示导入文件未声明结构化 scope。
func buildDeclaredPreviewScope(record parserpkg.ParsedEnvVarRecord) *ImportPreviewScope {
	if !record.ScopeTypeSpecified() && !record.ScopeValueSpecified() {
		return nil
	}

	scope := &ImportPreviewScope{}
	if record.DeclaredScopeType != nil {
		scope.Type = *record.DeclaredScopeType
	}
	if record.DeclaredScopeValue != nil {
		scope.Value = *record.DeclaredScopeValue
	}
	return scope
}
