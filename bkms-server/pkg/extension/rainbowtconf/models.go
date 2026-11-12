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

// Package rainbowtconf 保存应用的七彩石入口（页面地址），受空间配置开关 rainbowTConf 控制。
package rainbowtconf

import (
	"context"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	cfgswitchmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/cfgswitch/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

const maxURLLength = 2048

var (
	// ErrEntryNotFound 应用未配置七彩石入口
	ErrEntryNotFound = errors.New("rainbow tconf entry not found")
	// ErrEntryAlreadyExists 应用已有七彩石入口
	ErrEntryAlreadyExists = errors.New("rainbow tconf entry already exists")
	// ErrInvalidURL 七彩石页面地址不合法
	ErrInvalidURL = errors.New("url validation failed")
	// ErrNotEnabled 空间未开启七彩石
	ErrNotEnabled = errors.New("rainbow tconf is not enabled for workspace")
)

// Entry 应用的七彩石入口（页面地址），一个应用至多一条，没有记录表示未配置
type Entry struct {
	AppID     string    `bson:"appID"`
	TenantID  string    `json:"-" bson:"tenantID"`
	URL       string    `bson:"url"`
	Updater   string    `bson:"updater"`
	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

// EntryStore 应用七彩石入口的存储
type EntryStore = store.GenericStore[Entry]

// EntryMeta 应用七彩石入口的存储元数据，唯一索引：appID（由 golang-migrate 维护）
var EntryMeta = store.Meta[Entry]{
	Collection:   "rainbow_tconf_entries",
	TenantScoped: true,
	NotFound:     ErrEntryNotFound,
	Duplicate:    ErrEntryAlreadyExists,
}

// ByApp 按应用 ID 匹配
func ByApp(appID string) store.Opt {
	return store.Where("appID", appID)
}

// NormalizeURL 去掉首尾空白并校验地址：须为 https、有主机名、不带用户信息，不限制域名
func NormalizeURL(raw string) (string, error) {
	u := strings.TrimSpace(raw)
	if u == "" {
		return "", errors.Wrap(ErrInvalidURL, "url must not be blank")
	}
	if n := utf8.RuneCountInString(u); n > maxURLLength {
		return "", errors.Wrapf(ErrInvalidURL, "url is too long (%d > %d)", n, maxURLLength)
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return "", errors.Wrapf(ErrInvalidURL, "parse url: %v", err)
	}
	if parsed.Scheme != "https" {
		return "", errors.Wrap(ErrInvalidURL, "only https url is supported")
	}
	if parsed.Hostname() == "" {
		return "", errors.Wrap(ErrInvalidURL, "url host must not be empty")
	}
	if parsed.User != nil {
		return "", errors.Wrap(ErrInvalidURL, "url must not contain user info")
	}
	return u, nil
}

// GetEntry 获取应用的七彩石入口，未配置时返回 nil, nil
func GetEntry(ctx context.Context, db *mongo.Database, appID string) (*Entry, error) {
	b, err := store.Of(db, EntryMeta).Get(ctx, ByApp(appID))
	if errors.Is(err, ErrEntryNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "get rainbow tconf entry of app %s", appID)
	}
	return b, nil
}

// SaveURL 设置应用七彩石入口的页面地址，不存在时创建，地址不变时不写库
func SaveURL(ctx context.Context, db *mongo.Database, appID, rawURL, operator string) (*Entry, error) {
	u, err := NormalizeURL(rawURL)
	if err != nil {
		return nil, err
	}

	entry, err := saveURL(ctx, db, appID, u, operator)
	if errors.Is(err, ErrEntryAlreadyExists) {
		entry, err = saveURL(ctx, db, appID, u, operator)
	}
	return entry, err
}

// saveURL 按已校验的地址写入，地址不变时不写库
func saveURL(ctx context.Context, db *mongo.Database, appID, u, operator string) (*Entry, error) {
	current, err := GetEntry(ctx, db, appID)
	if err != nil {
		return nil, err
	}
	if current != nil && current.URL == u {
		return current, nil
	}

	target := &Entry{AppID: appID, URL: u, Updater: operator}
	if _, err = store.Of(db, EntryMeta).Upsert(
		ctx, target, ByApp(appID), store.UpdateSelected("url", "updater"),
	); err != nil {
		return nil, errors.Wrapf(err, "save rainbow tconf entry of app %s", appID)
	}
	return GetEntry(ctx, db, appID)
}

// DeleteByApp 删除应用的七彩石入口，不存在时不报错
func DeleteByApp(ctx context.Context, db *mongo.Database, appID string) error {
	if err := store.Of(db, EntryMeta).DeleteIfExists(ctx, ByApp(appID)); err != nil {
		return errors.Wrapf(err, "delete rainbow tconf entry of app %s", appID)
	}
	return nil
}

// RequireEnabled 空间未开启七彩石时返回 ErrNotEnabled
func RequireEnabled(ctx context.Context, db *mongo.Database, workspaceID string) error {
	enabled, err := cfgswitchmodel.IsEnabled(ctx, db, workspaceID, cfgswitchmodel.SwitchRainbowTConf)
	if err != nil {
		return errors.Wrapf(err, "check rainbow tconf cfg switch of workspace %s", workspaceID)
	}
	if !enabled {
		return errors.Wrapf(ErrNotEnabled, "workspace %s", workspaceID)
	}
	return nil
}
