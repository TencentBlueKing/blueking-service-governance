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

package component

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"gopkg.in/yaml.v3"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// builtinCompCreator is the creator/updater name for built-in components.
const builtinCompCreator = "admin"

// LoadBuiltinFromFolder loads component definitions from files in the specified folder.
//
// Args:
//   - defs: The component definition store to save loaded component definitions.
//   - folderPath: The path to the folder containing component definition files, it can also be a path
//     of a single file, in that case, only the file will be loaded.
func LoadBuiltinFromFolder(ctx context.Context, defs *ComponentDefStore, folderPath string) error {
	compDefs := make([]*ComponentDef, 0)

	fileInfo, err := os.Stat(folderPath)
	if err != nil {
		return errors.Wrap(err, "stating path")
	}
	if !fileInfo.IsDir() {
		// The folderPath is a single file path, parse and load it directly
		compDef, err := parseCompFile(folderPath)
		if err != nil {
			return errors.Wrapf(err, "parsing component file %s", folderPath)
		}
		compDefs = append(compDefs, compDef)
	} else {
		// Walk through the folder to load all component definition files
		err := filepath.WalkDir(folderPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			// Read the content of .yaml/.yml files, parse it into a map object.
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".yaml" && ext != ".yml" {
				return nil
			}

			compDef, err := parseCompFile(path)
			if err != nil {
				// Abort the walking process if failed to parse a file
				return errors.Wrapf(err, "parsing component file %s", path)
			}
			compDefs = append(compDefs, compDef)
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "walking through files")
		}
	}

	if err := checkCustomDefConflicts(ctx, defs, compDefs); err != nil {
		return err
	}
	// 内置定义写入平台租户，由 SharedFromPlatform 共享给所有租户；只覆盖已有的内置定义
	platform, tenantErr := tenant.PlatformTenantID()
	if tenantErr != nil {
		return tenantErr
	}
	for _, compDef := range compDefs {
		err := UpsertDef(ctx, defs, compDef, store.TenantOf(platform), store.Where("isBuiltin", true))
		if err != nil {
			return errors.Wrap(err, "upserting builtin componentDef")
		}
	}

	compDefNames := lo.Map(compDefs, func(c *ComponentDef, _ int) string { return c.Name })
	log.Infof(ctx, "Loaded builtin components successfully, total=%d, names=%v", len(compDefNames), compDefNames)
	return nil
}

// checkCustomDefConflicts 检查各租户是否已有与待加载内置定义同名的自建定义。
// 内置定义对所有租户可见，与自建定义同名会让按名称解析定义产生歧义
func checkCustomDefConflicts(ctx context.Context, defs *ComponentDefStore, builtins []*ComponentDef) error {
	if len(builtins) == 0 {
		return nil
	}
	keys := lo.SliceToMap(builtins, func(c *ComponentDef) (string, struct{}) { return c.Key(), struct{}{} })
	names := lo.Uniq(lo.Map(builtins, func(c *ComponentDef, _ int) string { return c.Name }))
	customs, err := defs.List(ctx, store.TenantAll(),
		store.Where("isBuiltin", false), store.Where("name", bson.M{"$in": names}),
		store.SelectFields("tenantID", "name", "version"),
	)
	if err != nil {
		return errors.Wrap(err, "listing custom component defs")
	}
	conflicts := lo.FilterMap(customs, func(c ComponentDef, _ int) (string, bool) {
		_, ok := keys[c.Key()]
		return c.TenantID + "/" + c.Key(), ok
	})
	if len(conflicts) > 0 {
		return errors.Errorf(
			"builtin component defs conflict with custom defs of the same name %v, rename them first", conflicts,
		)
	}
	return nil
}

// parseCompFile reads and parses a component file into a ComponentDef object.
func parseCompFile(path string) (*ComponentDef, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rawData map[string]any
	if err := yaml.Unmarshal(content, &rawData); err != nil {
		return nil, err
	}

	// Make a ComponentDef object from the content
	var compDef *ComponentDef
	if err := mapstructure.Decode(rawData, &compDef); err != nil {
		return nil, err
	}

	// Set the Creator and Updater to "admin" as system reserved
	compDef.Creator = builtinCompCreator
	compDef.Updater = builtinCompCreator
	// 共享读取以 isBuiltin 为准，不依赖文件中的字段，漏写会让内置定义对其他租户不可见
	compDef.IsBuiltin = true
	return compDef, nil
}
