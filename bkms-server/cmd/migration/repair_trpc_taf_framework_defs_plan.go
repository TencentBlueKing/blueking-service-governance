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

package migration

import (
	"cmp"
	"slices"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

// preferredFrameworkDefNames 选主 def 时优先认这些名字。
// default 是历史默认文件名；trpc_go.yaml / taf.conf 是创建应用时写入的真实框架文件名。
var preferredFrameworkDefNames = map[string]struct{}{
	appcfg.DefaultAppConfigFileName: {},
	"trpc_go.yaml":                  {},
	"taf.conf":                      {},
}

func isDefaultEnvName(envName string) bool {
	return envName == appcfg.EnvNameDefault
}

func fileContentLen(s *string) int {
	return len(lo.FromPtr(s))
}

// pickPrimaryFrameworkDef 选出「真正的框架文件」那条 def，后续环境 overlay 都改挂到它下面。
//
// 只从带默认文件（envName=""）的 def 里选，避免把只有 overlay、部署会跳过的孤儿当主 def。
// 排序：preferred 名字 > 更早 createdAt > 更小 _id。没有默认文件则返回 nil。
func pickPrimaryFrameworkDef(
	defs []appcfg.AppConfigFileDef,
	filesByDef map[bson.ObjectID][]appcfg.AppConfigFile,
) *appcfg.AppConfigFileDef {
	if len(defs) == 0 {
		return nil
	}

	candidates := make([]appcfg.AppConfigFileDef, 0, len(defs))
	for _, def := range defs {
		if defHasDefaultFile(filesByDef[def.ID]) {
			candidates = append(candidates, def)
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	slices.SortFunc(candidates, func(a, b appcfg.AppConfigFileDef) int {
		aPreferred := isPreferredFrameworkDefName(a.Name)
		bPreferred := isPreferredFrameworkDefName(b.Name)
		if aPreferred != bPreferred {
			if aPreferred {
				return -1
			}
			return 1
		}
		if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
			return c
		}
		return cmp.Compare(a.ID.Hex(), b.ID.Hex())
	})
	return &candidates[0]
}

func isPreferredFrameworkDefName(name string) bool {
	_, ok := preferredFrameworkDefNames[name]
	return ok
}

func defHasDefaultFile(files []appcfg.AppConfigFile) bool {
	for _, file := range files {
		if isDefaultEnvName(file.EnvName) {
			return true
		}
	}
	return false
}

// planRepairTrpcTafFrameworkDefs 根据已加载的 defs / files 填写 plan，不访问数据库。
//
//   - 非主 def 上 envName 非空的 file → moves（可改挂）或 conflicts（同环境已占用）
//   - 非主 def 上的默认文件 → leftovers，不挪也不把该 def 列入删除
//   - 迁完后文件数为 0 的孤儿 def → deleteDefIds
//   - 修完后主 def 会有环境实例且仍是统一配置 → setIndependent
//   - 出现 conflict 时清空删除列表，整单视为不可 apply
func planRepairTrpcTafFrameworkDefs(
	plan *repairTrpcTafFrameworkDefsAppPlan,
	defs []appcfg.AppConfigFileDef,
	files []appcfg.AppConfigFile,
) *repairTrpcTafFrameworkDefsAppPlan {
	filesByDef := groupFilesByDefID(files)
	primary := pickPrimaryFrameworkDef(defs, filesByDef)
	if primary == nil {
		plan.Skipped = true
		plan.SkipReason = skipReasonNoPrimaryDefault
		return plan
	}

	plan.PrimaryDefID = primary.ID.Hex()
	plan.PrimaryDefName = primary.Name

	occupiedEnvs := occupiedEnvNames(filesByDef[primary.ID])
	claimedEnvs := map[string]string{}

	defByID := make(map[bson.ObjectID]appcfg.AppConfigFileDef, len(defs))
	for _, def := range defs {
		defByID[def.ID] = def
	}

	remainingAfterMove := make(map[bson.ObjectID]int, len(defs))
	for defID, defFiles := range filesByDef {
		remainingAfterMove[defID] = len(defFiles)
	}

	for _, file := range files {
		if file.DefID == primary.ID {
			continue
		}
		if file.DefID == bson.NilObjectID {
			continue
		}
		fromDef := defByID[file.DefID]
		if isDefaultEnvName(file.EnvName) {
			plan.Leftovers = append(plan.Leftovers, repairFrameworkDefLeftover{
				FileID:  file.ID.Hex(),
				DefID:   file.DefID.Hex(),
				EnvName: file.EnvName,
				Reason:  leftoverReasonDefaultOnOrphan,
			})
			continue
		}

		if occupiedEnvs[file.EnvName] {
			plan.Conflicts = append(plan.Conflicts, repairFrameworkDefConflict{
				FileID:      file.ID.Hex(),
				EnvName:     file.EnvName,
				FromDefID:   file.DefID.Hex(),
				FromDefName: fromDef.Name,
				Reason:      "primary def already has this envName",
			})
			continue
		}
		if firstFileID, taken := claimedEnvs[file.EnvName]; taken {
			plan.Conflicts = append(plan.Conflicts, repairFrameworkDefConflict{
				FileID:      file.ID.Hex(),
				EnvName:     file.EnvName,
				FromDefID:   file.DefID.Hex(),
				FromDefName: fromDef.Name,
				Reason:      "another orphan file already claims envName (file " + firstFileID + ")",
			})
			continue
		}

		claimedEnvs[file.EnvName] = file.ID.Hex()
		plan.Moves = append(plan.Moves, repairFrameworkDefMove{
			FileID:      file.ID.Hex(),
			EnvName:     file.EnvName,
			Type:        string(file.Type),
			FromDefID:   file.DefID.Hex(),
			FromDefName: fromDef.Name,
			ToDefID:     primary.ID.Hex(),
			ContentLen:  fileContentLen(file.Content),
			OverlayLen:  fileContentLen(file.OverlayContent),
		})
		remainingAfterMove[file.DefID]--
	}

	for _, def := range defs {
		if def.ID == primary.ID {
			continue
		}
		if remainingAfterMove[def.ID] == 0 {
			plan.DeleteDefIDs = append(plan.DeleteDefIDs, def.ID.Hex())
			plan.DeleteDefNames = append(plan.DeleteDefNames, def.Name)
		}
	}

	hasEnvAfterRepair := len(occupiedEnvs) > 0 || len(plan.Moves) > 0
	plan.SetIndependent = hasEnvAfterRepair && !primary.EnvConfigMode.IsIndependent()

	if len(plan.Conflicts) > 0 {
		plan.SkipReason = skipReasonConflicts
		plan.DeleteDefIDs = nil
		plan.DeleteDefNames = nil
		return plan
	}

	if len(plan.Moves) == 0 && !plan.SetIndependent && len(plan.DeleteDefIDs) == 0 {
		plan.Skipped = true
		plan.SkipReason = skipReasonAlreadyHealthy
	}
	return plan
}

func groupFilesByDefID(files []appcfg.AppConfigFile) map[bson.ObjectID][]appcfg.AppConfigFile {
	grouped := make(map[bson.ObjectID][]appcfg.AppConfigFile)
	for _, file := range files {
		if file.DefID == bson.NilObjectID {
			continue
		}
		grouped[file.DefID] = append(grouped[file.DefID], file)
	}
	return grouped
}

func occupiedEnvNames(files []appcfg.AppConfigFile) map[string]bool {
	occupied := make(map[string]bool)
	for _, file := range files {
		if isDefaultEnvName(file.EnvName) {
			continue
		}
		occupied[file.EnvName] = true
	}
	return occupied
}
