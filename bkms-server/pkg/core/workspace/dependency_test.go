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

package workspace

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
)

type fakeBCSProjectClient struct {
	*bcs.StubApiClient
	existing     *bcs.Project
	getErr       error
	created      *bcs.Project
	createErr    error
	lastCreate   bcs.CreateProjectInput
	createCalled bool
	getCalled    bool
}

func (f *fakeBCSProjectClient) GetProject(_ context.Context, _ string) (*bcs.Project, error) {
	f.getCalled = true
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.existing, nil
}

func (f *fakeBCSProjectClient) CreateProject(_ context.Context, in bcs.CreateProjectInput) (*bcs.Project, error) {
	f.createCalled = true
	f.lastCreate = in
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.created, nil
}

func TestGetExistingBCSProject(t *testing.T) {
	ctx := context.Background()

	t.Run("returns existing project and does not create", func(t *testing.T) {
		client := &fakeBCSProjectClient{
			StubApiClient: &bcs.StubApiClient{},
			existing:      &bcs.Project{ID: "exist-uid", Code: "bkms-ws", Name: "existing"},
		}

		proj, err := getExistingBCSProject(ctx, client, "bkms-ws")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if proj.ID != "exist-uid" {
			t.Fatalf("ID = %q, want exist-uid", proj.ID)
		}
		if client.createCalled {
			t.Fatal("CreateProject should not be called")
		}
	})

	t.Run("returns get error including not found", func(t *testing.T) {
		client := &fakeBCSProjectClient{
			StubApiClient: &bcs.StubApiClient{},
			getErr:        bcs.ErrProjectNotFound,
		}

		_, err := getExistingBCSProject(ctx, client, "bkms-ws")
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, bcs.ErrProjectNotFound) {
			t.Fatalf("error = %v, want ErrProjectNotFound", err)
		}
		if client.createCalled {
			t.Fatal("CreateProject should not be called")
		}
	})
}

func TestCreateBCSProject(t *testing.T) {
	ctx := context.Background()

	t.Run("creates without looking up existing project", func(t *testing.T) {
		client := &fakeBCSProjectClient{
			StubApiClient: &bcs.StubApiClient{},
			created: &bcs.Project{
				ID:   "new-uid",
				Code: "bkms-ws",
				Name: "workspace-name",
				Kind: "k8s",
			},
		}

		proj, err := createBCSProject(ctx, client, "workspace-name", "bkms-ws", 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if proj.ID != "new-uid" {
			t.Fatalf("ID = %q, want new-uid", proj.ID)
		}
		if client.getCalled {
			t.Fatal("GetProject should not be called")
		}
		want := bcs.CreateProjectInput{
			Name:        "workspace-name",
			ProjectCode: "bkms-ws",
			Kind:        "k8s",
			BusinessID:  "100",
		}
		if client.lastCreate != want {
			t.Fatalf("CreateProjectInput = %+v, want %+v", client.lastCreate, want)
		}
	})

	t.Run("falls back to project code as name", func(t *testing.T) {
		client := &fakeBCSProjectClient{
			StubApiClient: &bcs.StubApiClient{},
			created:       &bcs.Project{ID: "new-uid", Code: "bkms-ws"},
		}

		if _, err := createBCSProject(ctx, client, "", "bkms-ws", 0); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client.lastCreate.Name != "bkms-ws" {
			t.Fatalf("Name = %q, want bkms-ws", client.lastCreate.Name)
		}
		if client.lastCreate.BusinessID != "" {
			t.Fatalf("BusinessID = %q, want empty", client.lastCreate.BusinessID)
		}
		if client.lastCreate.Kind != "k8s" {
			t.Fatalf("Kind = %q, want k8s", client.lastCreate.Kind)
		}
	})

	t.Run("surfaces create errors such as duplicate project code", func(t *testing.T) {
		client := &fakeBCSProjectClient{
			StubApiClient: &bcs.StubApiClient{},
			createErr:     errors.New("project code already exists"),
		}

		_, err := createBCSProject(ctx, client, "workspace-name", "bkms-ws", 100)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("error = %v", err)
		}
	})
}
