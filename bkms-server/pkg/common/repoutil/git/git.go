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

// Package git 远端 Git 仓库客户端（Smart HTTP）
package git

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/protocol/packp/capability"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/pkg/errors"
)

const (
	// refsHeadsPrefix 分支引用前缀
	refsHeadsPrefix = "refs/heads/"
	// refsTagsPrefix Tag 引用前缀
	refsTagsPrefix = "refs/tags/"
	// symRefHeadPrefix HEAD 符号引用前缀，如 HEAD:refs/heads/master
	symRefHeadPrefix = "HEAD:"

	// requestTimeout 请求超时
	requestTimeout = 20 * time.Second
	// maxRedirects 最大重定向次数，仅限同一主机
	maxRedirects = 3
)

var (
	// ErrAuthFailed 凭证错误或无权访问（401 / 403）
	ErrAuthFailed = errors.New("git repository authentication failed")
	// ErrRepoNotFound 仓库不存在（404）
	ErrRepoNotFound = errors.New("git repository not found")
	// ErrInvalidRepoURL 仓库地址不是合法的 HTTP(S) 地址
	ErrInvalidRepoURL = errors.New("invalid git repository url")
)

// Credential HTTP Basic 凭证
type Credential struct {
	Username string
	Password string
}

// Ref 引用
type Ref struct {
	// Name 短名称，如 master、v1.0.0
	Name string
	// Path 完整路径，如 refs/heads/master
	Path string
	// SHA 提交 ID，附注 Tag 取其指向的提交
	SHA string
}

// Refs 分支与 Tag
type Refs struct {
	// DefaultBranch 默认分支，未声明时为空
	DefaultBranch string
	// Branches 分支列表，按名称排序
	Branches []Ref
	// Tags Tag 列表，按名称排序
	Tags []Ref
}

// transportClient go-git HTTP 客户端
var transportClient = githttp.NewClient(&http.Client{
	Timeout: requestTimeout,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return errors.Errorf("stopped after %d redirects", maxRedirects)
		}
		if req.URL.Host != via[0].URL.Host {
			return errors.Errorf("redirect to another host %s is not allowed", req.URL.Host)
		}
		return nil
	},
})

// Client 远端 Git 仓库客户端
type Client struct {
	// URL 仓库地址
	URL string
	// Credential 凭证，为空时匿名访问
	Credential Credential
}

// NewClient 创建客户端，地址须为不含账号密码的 HTTP(S) 地址
func NewClient(repoURL string, cred Credential) (*Client, error) {
	if err := ValidateRepoURL(repoURL); err != nil {
		return nil, err
	}
	return &Client{URL: repoURL, Credential: cred}, nil
}

// ValidateRepoURL 校验地址为 HTTP(S) 且不含账号密码
func ValidateRepoURL(repoURL string) error {
	u, err := url.Parse(repoURL)
	if err != nil {
		return errors.Wrapf(ErrInvalidRepoURL, "parse %q: %v", repoURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.Wrapf(ErrInvalidRepoURL, "scheme of %q must be http or https", repoURL)
	}
	if u.Host == "" {
		return errors.Wrapf(ErrInvalidRepoURL, "host of %q is empty", repoURL)
	}
	if u.User != nil {
		return errors.Wrapf(ErrInvalidRepoURL, "%q must not embed credentials", repoURL)
	}
	return nil
}

// Touch 校验能否访问仓库
func (c *Client) Touch(ctx context.Context) error {
	_, err := c.ListRefs(ctx)
	return err
}

// ListRefs 列出分支与 Tag
func (c *Client) ListRefs(ctx context.Context) (*Refs, error) {
	// 去掉末尾的 /，避免拼出 repo.git//info/refs
	endpoint, err := transport.NewEndpoint(strings.TrimRight(c.URL, "/"))
	if err != nil {
		return nil, errors.Wrapf(ErrInvalidRepoURL, "parse %q: %v", c.URL, err)
	}

	var auth transport.AuthMethod
	if c.Credential.Username != "" || c.Credential.Password != "" {
		auth = &githttp.BasicAuth{Username: c.Credential.Username, Password: c.Credential.Password}
	}
	session, err := transportClient.NewUploadPackSession(endpoint, auth)
	if err != nil {
		return nil, errors.Wrapf(err, "new upload-pack session for %s", c.URL)
	}
	defer session.Close()

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	advRefs, err := session.AdvertisedReferencesContext(ctx)
	switch {
	case err == nil:
	case errors.Is(err, transport.ErrEmptyRemoteRepository):
		// 空仓库
		return &Refs{}, nil
	case errors.Is(err, transport.ErrAuthenticationRequired), errors.Is(err, transport.ErrAuthorizationFailed):
		return nil, errors.Wrapf(ErrAuthFailed, "list refs of %s: %v", c.URL, err)
	case errors.Is(err, transport.ErrRepositoryNotFound):
		return nil, errors.Wrapf(ErrRepoNotFound, "list refs of %s: %v", c.URL, err)
	default:
		return nil, errors.Wrapf(err, "list refs of %s", c.URL)
	}

	refs := &Refs{DefaultBranch: defaultBranchOf(advRefs.Capabilities)}
	for path, hash := range advRefs.References {
		switch {
		case strings.HasPrefix(path, refsHeadsPrefix):
			refs.Branches = append(refs.Branches, Ref{
				Name: strings.TrimPrefix(path, refsHeadsPrefix), Path: path, SHA: hash.String(),
			})
		case strings.HasPrefix(path, refsTagsPrefix):
			// 附注 Tag 取其指向的提交
			if peeled, ok := advRefs.Peeled[path]; ok {
				hash = peeled
			}
			refs.Tags = append(refs.Tags, Ref{
				Name: strings.TrimPrefix(path, refsTagsPrefix), Path: path, SHA: hash.String(),
			})
		}
	}
	// References 是 map，排序保证结果稳定
	sortByName(refs.Branches)
	sortByName(refs.Tags)
	return refs, nil
}

// defaultBranchOf 从能力声明解析默认分支
func defaultBranchOf(capabilities *capability.List) string {
	if capabilities == nil {
		return ""
	}
	for _, symRef := range capabilities.Get(capability.SymRef) {
		if target, ok := strings.CutPrefix(symRef, symRefHeadPrefix); ok {
			return strings.TrimPrefix(target, refsHeadsPrefix)
		}
	}
	return ""
}

// sortByName 按名称排序
func sortByName(refs []Ref) {
	slices.SortFunc(refs, func(a, b Ref) int { return strings.Compare(a.Name, b.Name) })
}
