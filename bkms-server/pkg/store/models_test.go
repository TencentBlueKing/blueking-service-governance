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

package store_test

import (
	"context"
	"strings"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// 本文件定义测试用的数据模型，它们分别模拟了现有 Store 中的几种典型形态。

// Widget 模拟最常见的模型：ObjectID 主键、自然键唯一索引（ownerID + name）、
// 创建/更新时间戳、值类型嵌套结构体、指针嵌套的敏感字段。
type Widget struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	OwnerID   string        `bson:"ownerID"`
	Name      string        `bson:"name"`
	Priority  int           `bson:"priority"`
	Tags      []string      `bson:"tags"`
	Spec      WidgetSpec    `bson:"spec"`
	Secret    *Credential   `bson:"secret,omitempty"`
	CreatedAt time.Time     `bson:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"`
}

// WidgetSpec 值类型嵌套结构体
type WidgetSpec struct {
	Color string `bson:"color"`
	Size  int    `bson:"size"`
}

// Credential 指针嵌套结构体，其中 Password 需要加密存储
type Credential struct {
	Username string `bson:"username"`
	Password string `bson:"password"`
}

const (
	widgetCollection = "store_test_widgets"
	encPrefix        = "enc:"
	defaultColor     = "gray"
	plainPassword    = "plain"
)

// longAgo 是 mustCreateStaleWidget 使用的时间戳，早于任何写操作的当前时间
var longAgo = time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)

var (
	ErrWidgetNotFound = errors.New("widget not found")
	ErrWidgetExists   = errors.New("widget already exists")
	errNameRequired   = errors.New("widget name is required")
	errNotEncrypted   = errors.New("password is not encrypted")
)

func requireWidgetName(w *Widget) error {
	if w.Name == "" {
		return errNameRequired
	}
	return nil
}

func defaultWidgetColor(w *Widget) error {
	if w.Spec.Color == "" {
		w.Spec.Color = defaultColor
	}
	return nil
}

// encryptWidgetPassword 模拟加密：加上固定前缀，便于直接断言库中实际存储的内容
func encryptWidgetPassword(w *Widget) error {
	if w.Secret != nil && w.Secret.Password != "" {
		w.Secret.Password = encPrefix + w.Secret.Password
	}
	return nil
}

func decryptWidgetPassword(w *Widget) error {
	if w.Secret == nil || w.Secret.Password == "" {
		return nil
	}
	plain, ok := strings.CutPrefix(w.Secret.Password, encPrefix)
	if !ok {
		return errNotEncrypted
	}
	w.Secret.Password = plain
	return nil
}

// widgetMeta 模拟带有校验、默认值与加解密逻辑的模型元信息
var widgetMeta = store.Meta[Widget]{
	Collection: widgetCollection,
	NotFound:   ErrWidgetNotFound,
	Duplicate:  ErrWidgetExists,
	BeforeSave: []func(*Widget) error{requireWidgetName, defaultWidgetColor},
	Encode:     []func(*Widget) error{encryptWidgetPassword},
	Decode:     []func(*Widget) error{decryptWidgetPassword},
}

// Item 模拟以字符串作为 _id、没有时间戳的模型（如 port-forward 白名单）
type Item struct {
	ID    string `bson:"_id"`
	Value string `bson:"value"`
}

const itemCollection = "store_test_items"

var itemMeta = store.Meta[Item]{Collection: itemCollection}

// Timestamps 以 inline 方式嵌入的公共时间戳
type Timestamps struct {
	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

// Note 模拟通过 inline 嵌入时间戳的模型
type Note struct {
	ID         bson.ObjectID `bson:"_id,omitempty"`
	Timestamps `bson:",inline"`
	Body       string `bson:"body"`
}

const noteCollection = "store_test_notes"

var noteMeta = store.Meta[Note]{Collection: noteCollection}

// Ambiguous 模拟 Go 字段名与另一个字段的 bson key 相同的模型："Label" 无法判断指的是哪个字段
type Ambiguous struct {
	Title string `bson:"Label"`
	Label string `bson:"title"`
}

var ambiguousMeta = store.Meta[Ambiguous]{Collection: "store_test_ambiguous"}

// Board 模拟以 map 保存各环境状态（如北极星配置的 envStates）、以结构体数组保存子元素（如应用模型的 components）
// 并带有计数器的模型，用于验证嵌套 Upsert、删除 map 条目与 Modify
type Board struct {
	ID        bson.ObjectID         `bson:"_id,omitempty"`
	OwnerID   string                `bson:"ownerID"`
	Name      string                `bson:"name"`
	States    map[string]BoardState `bson:"states,omitempty"`
	Cards     []Card                `bson:"cards"`
	Hits      int64                 `bson:"hits"`
	CreatedAt time.Time             `bson:"createdAt"`
	UpdatedAt time.Time             `bson:"updatedAt"`
}

// BoardState 一个环境的状态
type BoardState struct {
	Status  string `bson:"status"`
	Message string `bson:"message"`
}

// Card Board 中的子元素，以 Name 区分
type Card struct {
	Name  string `bson:"name"`
	Title string `bson:"title"`
}

const boardCollection = "store_test_boards"

var boardMeta = store.Meta[Board]{Collection: boardCollection}

// Profile 模拟不声明 _id 字段、以业务字段作为唯一键的模型（如工作空间以 id 字段为唯一键），
// 文档的 _id 由 MongoDB 生成，对模型不可见
type Profile struct {
	Key       string    `bson:"key"`
	Title     string    `bson:"title"`
	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

const profileCollection = "store_test_profiles"

var ErrProfileNotFound = errors.New("profile not found")

var profileMeta = store.Meta[Profile]{Collection: profileCollection, NotFound: ErrProfileNotFound}

// storeFixture 汇总测试用例需要的依赖
type storeFixture struct {
	ctx      context.Context
	app      *fxtest.App
	db       *mongo.Database
	widgets  *store.GenericStore[Widget]
	items    *store.GenericStore[Item]
	notes    *store.GenericStore[Note]
	boards   *store.GenericStore[Board]
	profiles *store.GenericStore[Profile]
	// owner 当前用例的 ownerID，让过滤条件贴近领域包中 byXxx 的真实用法；
	// 用例之间的隔离依靠每个用例开始前清空测试集合
	owner string
}

// setUpStores 通过 fx 提供测试模型的 Store，同时清空测试集合、确保唯一索引存在
func setUpStores() *storeFixture {
	f := &storeFixture{ctx: context.Background(), owner: "owner-" + stringx.Random(8)}
	f.app = fxtest.New(
		GinkgoT(),
		store.FxModule,
		store.Provide(widgetMeta),
		store.Provide(itemMeta),
		store.Provide(noteMeta),
		store.Provide(boardMeta),
		store.Provide(profileMeta),
		fx.Populate(&f.db, &f.widgets, &f.items, &f.notes, &f.boards, &f.profiles),
	)
	f.app.RequireStart()

	for _, coll := range []*mongo.Collection{
		f.widgets.Collection(), f.items.Collection(), f.notes.Collection(), f.boards.Collection(),
		f.profiles.Collection(),
	} {
		_, err := coll.DeleteMany(f.ctx, bson.M{})
		Expect(err).NotTo(HaveOccurred())
	}
	// 测试集合不在数据库迁移中，这里补上唯一索引，用于验证唯一键冲突的处理
	_, err := f.widgets.Collection().Indexes().CreateOne(f.ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "ownerID", Value: 1}, {Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	Expect(err).NotTo(HaveOccurred())
	_, err = f.profiles.Collection().Indexes().CreateOne(f.ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "key", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	Expect(err).NotTo(HaveOccurred())
	return f
}

func (f *storeFixture) tearDown() {
	f.app.RequireStop()
}

// newWidget 构造一个属于当前用例 owner 的 Widget
func (f *storeFixture) newWidget(name string) *Widget {
	return &Widget{OwnerID: f.owner, Name: name, Priority: 1, Tags: []string{"a"}}
}

// newSecretWidget 构造一个带凭证的 Widget，凭证密码为 plainPassword
func (f *storeFixture) newSecretWidget(name string) *Widget {
	w := f.newWidget(name)
	w.Secret = &Credential{Username: "u", Password: plainPassword}
	return w
}

// mustCreateWidget 创建并返回一个 Widget
func (f *storeFixture) mustCreateWidget(name string) *Widget {
	w := f.newWidget(name)
	Expect(f.widgets.Create(f.ctx, w)).To(Succeed())
	return w
}

// mustCreateStaleWidget 创建一个 createdAt、updatedAt 均为 longAgo 的 Widget，
// 用于断言写操作刷新了 updatedAt，而不必 sleep 等待时间前进
func (f *storeFixture) mustCreateStaleWidget(name string) *Widget {
	w := f.newWidget(name)
	w.CreatedAt = longAgo
	Expect(f.widgets.Create(f.ctx, w)).To(Succeed())
	Expect(w.UpdatedAt).To(Equal(longAgo))
	return w
}

// mustGetWidget 读取当前 owner 下指定名称的 Widget
func (f *storeFixture) mustGetWidget(name string) *Widget {
	w, err := f.widgets.Get(f.ctx, f.byName(name))
	Expect(err).NotTo(HaveOccurred())
	return w
}

// widgetExists 判断当前 owner 下是否存在指定名称的 Widget
func (f *storeFixture) widgetExists(name string) bool {
	ok, err := f.widgets.Exists(f.ctx, f.byName(name))
	Expect(err).NotTo(HaveOccurred())
	return ok
}

// countWidgets 统计当前 owner 下的 Widget 数量
func (f *storeFixture) countWidgets() int64 {
	n, err := f.widgets.Count(f.ctx, f.byOwner())
	Expect(err).NotTo(HaveOccurred())
	return n
}

// rawWidget 绕过 GenericStore 直接读取数据库中的原始文档，用于断言实际存储的内容。
// 返回 bson.Raw 而非 bson.M：驱动 v2 解码到 bson.M 时，嵌套文档默认是 bson.D，按路径取值更直接
func (f *storeFixture) rawWidget(name string) bson.Raw {
	raw, err := f.widgets.Collection().FindOne(f.ctx, bson.M{"ownerID": f.owner, "name": name}).Raw()
	Expect(err).NotTo(HaveOccurred())
	return raw
}

// rawPassword 读取库中实际存储的凭证密码（即 Encode 之后的值）
func (f *storeFixture) rawPassword(name string) string {
	return rawString(f.rawWidget(name), "secret", "password")
}

// rawString 读取原始文档中指定路径的字符串值
func rawString(raw bson.Raw, path ...string) string {
	v, err := raw.LookupErr(path...)
	Expect(err).NotTo(HaveOccurred())
	return v.StringValue()
}

// rawHas 判断原始文档中是否存在指定路径，用于断言字段被 $unset 而不是写入了零值
func rawHas(raw bson.Raw, path ...string) bool {
	_, err := raw.LookupErr(path...)
	return err == nil
}

// widgetNames 按顺序返回 Widget 的名称，用于断言列表内容与排序
func widgetNames(list []Widget) []string {
	return lo.Map(list, func(w Widget, _ int) string { return w.Name })
}

// beStoredTime 匹配写入数据库后再读出的时间：MongoDB 只保存毫秒精度
func beStoredTime(t time.Time) types.GomegaMatcher {
	return BeTemporally("~", t, time.Millisecond)
}

// byOwner 匹配当前 owner 下的全部 Widget，模拟领域包中定义的具名 Opt
func (f *storeFixture) byOwner() store.Opt {
	return store.Where("ownerID", f.owner)
}

// byName 匹配当前 owner 下指定名称的 Widget，演示在已有 Opt 上组合新的 Opt
func (f *storeFixture) byName(name string) store.Opt {
	return store.Combine(f.byOwner(), store.Where("name", name))
}

// mustCreateBoard 创建一个属于当前 owner、带有 prod 环境状态与一张卡片的 Board，
// createdAt、updatedAt 均为 longAgo
func (f *storeFixture) mustCreateBoard(name string) *Board {
	b := &Board{
		OwnerID:   f.owner,
		Name:      name,
		States:    map[string]BoardState{"prod": {Status: "ok"}},
		Cards:     []Card{{Name: "c1", Title: "first"}},
		CreatedAt: longAgo,
	}
	Expect(f.boards.Create(f.ctx, b)).To(Succeed())
	return b
}

// mustGetBoard 读取当前 owner 下指定名称的 Board
func (f *storeFixture) mustGetBoard(name string) *Board {
	b, err := f.boards.Get(f.ctx, f.byName(name))
	Expect(err).NotTo(HaveOccurred())
	return b
}
