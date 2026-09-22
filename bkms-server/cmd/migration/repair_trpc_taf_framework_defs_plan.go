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
