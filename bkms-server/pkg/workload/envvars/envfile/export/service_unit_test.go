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

package export

import (
	"testing"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	envvarsstore "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

func TestFilterOutSensitiveScopedVars(t *testing.T) {
	vars := []envvarsstore.ScopedEnvVar{
		{Key: "PUBLIC", Value: "visible", IsSensitive: false},
		{Key: "SECRET", Value: "hidden", IsSensitive: true},
	}

	filtered := filterOutSensitiveScopedVars(vars)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 non-sensitive scoped var, got %d", len(filtered))
	}
	if filtered[0].Key != "PUBLIC" {
		t.Fatalf("expected PUBLIC to remain, got %s", filtered[0].Key)
	}
}

func TestFilterOutSensitiveAppVars(t *testing.T) {
	vars := []appmodel.Variable{
		{Key: "APP_MODE", Value: "prod", IsSensitive: false},
		{Key: "APP_SECRET", Value: "hidden", IsSensitive: true},
	}

	filtered := filterOutSensitiveAppVars(vars)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 non-sensitive app var, got %d", len(filtered))
	}
	if filtered[0].Key != "APP_MODE" {
		t.Fatalf("expected APP_MODE to remain, got %s", filtered[0].Key)
	}
}

func TestFilterOutSensitiveEffectiveVars(t *testing.T) {
	vars := envvartypes.EnvVariableList{
		{Key: "PUBLIC", Value: "visible", IsSensitive: false},
		{Key: "SECRET", Value: "hidden", IsSensitive: true},
	}

	filtered := filterOutSensitiveEffectiveVars(vars)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 non-sensitive effective var, got %d", len(filtered))
	}
	if filtered[0].Key != "PUBLIC" {
		t.Fatalf("expected PUBLIC to remain, got %s", filtered[0].Key)
	}
}

func TestRenderRecordsEscapesMultilineDescription(t *testing.T) {
	content := renderRecords([]envFileRecord{
		{
			Key:         "KEY",
			Value:       "value",
			Description: "line1\nline2",
		},
	})
	if got, want := content, "# desc: \"line1\\nline2\"\nKEY=value\n"; got != want {
		t.Fatalf("unexpected rendered content:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestEnvVariableListToDeduplicatedListKeepsLastOccurrence(t *testing.T) {
	vars := envvartypes.EnvVariableList{
		{Key: "SHARED", Value: "workspace"},
		{Key: "OTHER", Value: "other"},
		{Key: "SHARED", Value: "app"},
	}

	deduped := vars.ToDeduplicatedList()
	if len(deduped) != 2 {
		t.Fatalf("expected 2 items after deduplication, got %d", len(deduped))
	}
	if deduped[0].Key != "OTHER" || deduped[0].Value != "other" {
		t.Fatalf("expected OTHER to remain first, got %+v", deduped[0])
	}
	if deduped[1].Key != "SHARED" || deduped[1].Value != "app" {
		t.Fatalf("expected last SHARED value to win, got %+v", deduped[1])
	}
}
