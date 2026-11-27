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

package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
)

// GitRepo 单测 Git 服务中的仓库
type GitRepo struct {
	// Name 仓库名
	Name string
	// URL 仓库 HTTP 地址，以 .git 结尾
	URL string
	// DefaultBranch 默认分支
	DefaultBranch string
}

// CleanupTB *testing.T 与 GinkgoT() 共有的方法
type CleanupTB interface {
	Helper()
	Cleanup(func())
	Fatalf(format string, args ...any)
}

// NewGitRepo 创建带初始提交与指定 Tag 的私有仓库，测试结束后删除
func NewGitRepo(t CleanupTB, tags ...string) *GitRepo {
	t.Helper()
	name := fmt.Sprintf("ut-%d", time.Now().UnixNano())
	repo := &GitRepo{Name: name, DefaultBranch: "main"}

	err := giteaRequest(http.MethodPost, "/api/v1/user/repos", map[string]any{
		"name": name, "private": true, "auto_init": true, "default_branch": repo.DefaultBranch,
	}, nil)
	if err != nil {
		t.Fatalf("create git repo %s: %v", name, err)
	}
	t.Cleanup(func() {
		_ = giteaRequest(http.MethodDelete, "/api/v1/repos/"+GitServerUsername()+"/"+name, nil, nil)
	})
	// gitea 返回的 clone_url 取自 ROOT_URL，可能与单测访问地址不同
	repo.URL = GitServerURL() + "/" + GitServerUsername() + "/" + name + ".git"

	for _, tag := range tags {
		path := "/api/v1/repos/" + GitServerUsername() + "/" + name + "/tags"
		if err = giteaRequest(http.MethodPost, path, map[string]any{
			"tag_name": tag, "target": repo.DefaultBranch,
		}, nil); err != nil {
			t.Fatalf("create tag %s of git repo %s: %v", tag, name, err)
		}
	}
	return repo
}

// NewGitBranch 从默认分支创建分支
func (r *GitRepo) NewGitBranch(t CleanupTB, branch string) {
	t.Helper()
	path := "/api/v1/repos/" + GitServerUsername() + "/" + r.Name + "/branches"
	if err := giteaRequest(http.MethodPost, path, map[string]any{
		"new_branch_name": branch, "old_branch_name": r.DefaultBranch,
	}, nil); err != nil {
		t.Fatalf("create branch %s of git repo %s: %v", branch, r.Name, err)
	}
}

// giteaRequest 以管理员身份调用 gitea API
func giteaRequest(method, path string, body, result any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return errors.Wrap(err, "marshal request body")
		}
		reader = bytes.NewReader(data)
	}
	reqURL, err := url.JoinPath(GitServerURL(), path)
	if err != nil {
		return errors.Wrapf(err, "join url %s", path)
	}
	req, err := http.NewRequest(method, reqURL, reader)
	if err != nil {
		return errors.Wrapf(err, "new request %s %s", method, path)
	}
	req.SetBasicAuth(GitServerUsername(), GitServerPassword())
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return errors.Wrapf(err, "request %s %s", method, path)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= http.StatusBadRequest {
		return errors.Errorf("request %s %s: status %d, body %s", method, path, resp.StatusCode, data)
	}
	if result != nil {
		if err = json.Unmarshal(data, result); err != nil {
			return errors.Wrapf(err, "unmarshal response of %s %s", method, path)
		}
	}
	return nil
}
