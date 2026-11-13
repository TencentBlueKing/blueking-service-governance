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

package frameworkcfgtmpl

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// CreateParams 创建自定义模板的参数。
type CreateParams struct {
	Framework string
	// Language 语言，如 go、cpp。
	Language string
	Definition
	// IsDefault 是否同时设为分组的默认模板。
	IsDefault bool
	Operator  string
}

// UpdateParams 编辑自定义模板的参数，分组（框架与语言）不可修改。
type UpdateParams struct {
	Definition
	// IsDefault 为 nil 时不改变默认状态；取消默认模板的默认状态后，分组回落到内置模板。
	IsDefault *bool
	Operator  string
}

// List 列出工作空间的模板，不返回模板内容。filter 中为空的字段不参与过滤。
// 每个分组先列内置模板，再按更新时间倒序列自定义模板。
func List(ctx context.Context, db *mongo.Database, workspaceID string, filter Group) ([]*View, error) {
	opts := []store.Opt{ByWorkspace(workspaceID), store.OmitFields("content"), store.Desc("updatedAt")}
	if filter.Framework != "" {
		opts = append(opts, store.Where("framework", filter.Framework))
	}
	if filter.Language != "" {
		opts = append(opts, store.Where("language", filter.Language))
	}
	customs, err := store.Of(db, TemplateMeta).List(ctx, opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "list framework config templates of workspace %s", workspaceID)
	}
	customsByGroup := lo.GroupBy(customs, func(t Template) Group { return t.Group })

	views := make([]*View, 0, len(builtins)+len(customs))
	for _, builtin := range builtins {
		if (filter.Framework != "" && builtin.Framework != filter.Framework) ||
			(filter.Language != "" && builtin.Language != filter.Language) {
			continue
		}
		group := customsByGroup[builtin.Group]
		builtin.Content = ""
		builtin.IsDefault = !lo.ContainsBy(group, func(t Template) bool { return t.IsDefault })
		views = append(views, &builtin)
		for i := range group {
			views = append(views, group[i].view())
		}
	}
	return views, nil
}

// Get 获取模板详情（含内容），id 可以是内置模板 ID。
func Get(ctx context.Context, db *mongo.Database, workspaceID, id string) (*View, error) {
	s := store.Of(db, TemplateMeta)
	if isBuiltinID(id) {
		view, ok := lookupBuiltin(id)
		if !ok {
			return nil, errors.Wrapf(ErrTemplateNotFound, "template %s", id)
		}
		hasCustomDefault, err := s.Exists(ctx, DefaultOf(workspaceID, view.Group))
		if err != nil {
			return nil, errors.Wrapf(err, "check default template of %s/%s", view.Framework, view.Language)
		}
		view.IsDefault = !hasCustomDefault
		return &view, nil
	}
	tmpl, err := getCustom(ctx, s, workspaceID, id)
	if err != nil {
		return nil, err
	}
	return tmpl.view(), nil
}

// Create 创建自定义模板。
func Create(ctx context.Context, db *mongo.Database, workspaceID string, params CreateParams) (*View, error) {
	group, format, err := ResolveGroup(params.Framework, params.Language)
	if err != nil {
		return nil, err
	}
	def, err := params.normalize(format)
	if err != nil {
		return nil, err
	}

	s := store.Of(db, TemplateMeta)
	tmpl := &Template{
		WorkspaceID: workspaceID,
		Group:       group,
		Name:        def.Name,
		FileName:    def.FileName,
		Content:     def.Content,
		IsDefault:   params.IsDefault,
		Creator:     params.Operator,
		Updater:     params.Operator,
	}
	if params.IsDefault {
		if err = prepareDefault(ctx, s, workspaceID, group, def.Name, nil); err != nil {
			return nil, err
		}
	}
	if err = s.Create(ctx, tmpl); err != nil {
		return nil, errors.Wrapf(err, "create template %q", def.Name)
	}
	return tmpl.view(), nil
}

// Update 编辑自定义模板，返回编辑前后的模板。
func Update(
	ctx context.Context, db *mongo.Database, workspaceID, id string, params UpdateParams,
) (before, after *View, err error) {
	if isBuiltinID(id) {
		return nil, nil, errors.Wrapf(ErrBuiltinReadOnly, "update template %s", id)
	}
	s := store.Of(db, TemplateMeta)
	old, err := getCustom(ctx, s, workspaceID, id)
	if err != nil {
		return nil, nil, err
	}
	_, format, err := ResolveGroup(old.Framework, old.Language)
	if err != nil {
		return nil, nil, err
	}
	def, err := params.normalize(format)
	if err != nil {
		return nil, nil, err
	}

	updated := *old
	updated.Name = def.Name
	updated.FileName = def.FileName
	updated.Content = def.Content
	updated.Updater = params.Operator
	fields := []string{"name", "fileName", "content", "updater"}
	// 默认状态与内容在同一次写入中提交；取消默认只改本模板，不能清除其它请求设置的默认。
	if params.IsDefault != nil {
		if *params.IsDefault {
			if err = prepareDefault(ctx, s, workspaceID, old.Group, def.Name, &old.ID); err != nil {
				return nil, nil, err
			}
		}
		updated.IsDefault = *params.IsDefault
		fields = append(fields, "isDefault")
	}
	if err = s.Update(ctx, &updated, store.UpdateSelected(fields...), ByID(workspaceID, old.ID)); err != nil {
		return nil, nil, errors.Wrapf(err, "update template %s", id)
	}
	return old.view(), updated.view(), nil
}

// Delete 删除自定义模板。删除的是默认模板时，分组回落到内置模板。
func Delete(ctx context.Context, db *mongo.Database, workspaceID, id string) (*View, error) {
	if isBuiltinID(id) {
		return nil, errors.Wrapf(ErrBuiltinReadOnly, "delete template %s", id)
	}
	oid, err := parseCustomID(id)
	if err != nil {
		return nil, err
	}
	deleted, err := store.Of(db, TemplateMeta).GetAndDelete(ctx, ByID(workspaceID, oid))
	if err != nil {
		return nil, errors.Wrapf(err, "delete template %s", id)
	}
	return deleted.view(), nil
}

// SetDefault 把模板设为所在分组的默认模板，返回设置前后的模板（含内容）。
// 对内置模板调用表示清除分组的自定义默认；模板已是默认时不做修改。
func SetDefault(ctx context.Context, db *mongo.Database, workspaceID, id string) (before, after *View, err error) {
	before, err = Get(ctx, db, workspaceID, id)
	if err != nil {
		return nil, nil, err
	}
	if before.IsDefault {
		return before, before, nil
	}
	var target *bson.ObjectID
	if !before.Builtin {
		target = lo.ToPtr(lo.Must(bson.ObjectIDFromHex(before.ID)))
	}
	if err = setDefault(ctx, store.Of(db, TemplateMeta), workspaceID, before.Group, target); err != nil {
		return nil, nil, err
	}
	after = lo.ToPtr(*before)
	after.IsDefault = true
	return before, after, nil
}

// setDefault 把 target 设为分组的默认模板；target 为 nil 时清除自定义默认，分组回落到内置模板。
// 默认状态不属于模板内容，不刷新 updatedAt。
func setDefault(ctx context.Context, s *TemplateStore, workspaceID string, group Group, target *bson.ObjectID) error {
	if err := clearOtherDefaults(ctx, s, workspaceID, group, target); err != nil {
		return err
	}
	if target == nil {
		return nil
	}
	if err := s.Update(
		ctx, map[string]any{"isDefault": true}, ByID(workspaceID, *target), store.KeepUpdatedAt(),
	); err != nil {
		return errors.Wrapf(err, "set template %s as default", target.Hex())
	}
	return nil
}

// prepareDefault 在以默认状态写入模板 self（新建时为 nil）之前，检查名称未被占用并清除分组内其它默认。
//
// 不使用事务：清除后的写入若被部分唯一索引拒绝，说明并发请求已把另一个模板设为默认，
// 本次返回 ErrTemplateConflict 且不写入任何内容；只有清除后写入因其它原因失败时，分组才会回落到内置模板。
// 预先检查名称是为了避免模板因重名写入失败却已清掉原有默认。
func prepareDefault(
	ctx context.Context, s *TemplateStore, workspaceID string, group Group, name string, self *bson.ObjectID,
) error {
	opts := []store.Opt{ByGroup(workspaceID, group), store.Where("name", name)}
	if self != nil {
		opts = append(opts, store.Filter(bson.M{"_id": bson.M{"$ne": *self}}))
	}
	taken, err := s.Exists(ctx, opts...)
	if err != nil {
		return errors.Wrapf(err, "check name of template %q", name)
	}
	if taken {
		return errors.Wrapf(ErrTemplateConflict, "template %q already exists", name)
	}
	return clearOtherDefaults(ctx, s, workspaceID, group, self)
}

// clearOtherDefaults 清除分组内除 keep 以外的默认标记，keep 为 nil 时清除全部。
func clearOtherDefaults(
	ctx context.Context,
	s *TemplateStore,
	workspaceID string,
	group Group,
	keep *bson.ObjectID,
) error {
	opts := []store.Opt{DefaultOf(workspaceID, group), store.KeepUpdatedAt()}
	if keep != nil {
		opts = append(opts, store.Filter(bson.M{"_id": bson.M{"$ne": *keep}}))
	}
	if _, err := s.UpdateMany(ctx, map[string]any{"isDefault": false}, opts...); err != nil {
		return errors.Wrapf(err, "clear default template of %s/%s", group.Framework, group.Language)
	}
	return nil
}

func getCustom(ctx context.Context, s *TemplateStore, workspaceID, id string) (*Template, error) {
	oid, err := parseCustomID(id)
	if err != nil {
		return nil, err
	}
	tmpl, err := s.Get(ctx, ByID(workspaceID, oid))
	if err != nil {
		return nil, errors.Wrapf(err, "get template %s", id)
	}
	return tmpl, nil
}

// parseCustomID 解析自定义模板 ID，非法 ID 视为模板不存在。
func parseCustomID(id string) (bson.ObjectID, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return bson.NilObjectID, errors.Wrapf(ErrTemplateNotFound, "template %s", id)
	}
	return oid, nil
}
