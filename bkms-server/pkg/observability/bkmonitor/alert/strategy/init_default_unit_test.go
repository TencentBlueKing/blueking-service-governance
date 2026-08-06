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

package strategy

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type captureCreateStore struct {
	created []*AlertStrategy
}

func (s *captureCreateStore) Create(_ context.Context, rule *AlertStrategy) (bson.ObjectID, error) {
	copied := *rule
	copied.NoticeGroupIDs = append([]int64(nil), rule.NoticeGroupIDs...)
	s.created = append(s.created, &copied)
	return bson.NewObjectID(), nil
}

func (*captureCreateStore) Get(context.Context, bson.ObjectID) (*AlertStrategy, error) {
	return nil, nil
}

func (*captureCreateStore) ListByWorkspace(context.Context, string) ([]AlertStrategy, error) {
	return nil, nil
}

func (*captureCreateStore) ListByApp(context.Context, string, string) ([]AlertStrategy, error) {
	return nil, nil
}

func (*captureCreateStore) ListByAppAndRemoteEnv(
	context.Context, string, string, bson.ObjectID, string,
) ([]AlertStrategy, error) {
	return nil, nil
}

func (*captureCreateStore) ListEnabledByAppMatchingEnv(
	context.Context, string, string, string, bson.ObjectID,
) ([]AlertStrategy, error) {
	return nil, nil
}

func (*captureCreateStore) Update(context.Context, bson.ObjectID, bson.M) error { return nil }
func (*captureCreateStore) Delete(context.Context, bson.ObjectID) error         { return nil }

func TestInitDefaultAlertStrategiesForAppAssignsNoticeGroups(t *testing.T) {
	store := &captureCreateStore{}
	svc := &Service{store: store}

	err := svc.InitDefaultAlertStrategiesForApp(
		context.Background(),
		"default-ws",
		"app-1",
		"demo-app",
		"tester",
		[]int64{1001},
	)
	if err != nil {
		t.Fatalf("InitDefaultAlertStrategiesForApp returned error: %v", err)
	}
	if len(store.created) != len(defaultTemplates) {
		t.Fatalf("expected %d default strategies, got %d", len(defaultTemplates), len(store.created))
	}
	for _, created := range store.created {
		if len(created.NoticeGroupIDs) != 1 || created.NoticeGroupIDs[0] != 1001 {
			t.Fatalf("expected noticeGroupIDs [1001], got %v", created.NoticeGroupIDs)
		}
	}
}
