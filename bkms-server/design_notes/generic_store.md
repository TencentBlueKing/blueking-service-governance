# GenericStore：基于泛型的通用 Store

> 状态：阶段 3 试点已完成，并据试点反馈做了两项调整（钩子改为切片、`store.Of` + 领域函数只接收 db），
> 见「阶段 3 后续调整」；下一步是阶段 4，迁移剩余 A 类 Store。
> 维护方式：每完成一个阶段，更新「进度跟踪」与对应章节，结论变化时直接改写正文，不追加流水账。

## 背景与目标

`pkg/server/registry/registry.go` 持有 50 个 Store，绝大多数 Store 都是「一个 collection + 一组按自然键的 CRUD」，
各自手写 `InsertOne` / `FindOne` / `ErrNoDocuments` 转换 / 时间戳 / 唯一键冲突翻译等样板代码。

目标：在 `pkg/store` 下提供 `GenericStore`，模型只需声明「结构体类型 + collection 名 + 少量 meta 配置」，
即可获得通用的 CRUD 能力，不再为每个模型手写 Store。

## 阶段 1 结论：扫描与分类

扫描范围：`pkg/` 下全部 `*Store` 接口（约 60 个，其中 50 个挂在 Registry 上）。

### 共性模式（GenericStore 需要吸收的样板）

| 模式 | 出现情况 |
|---|---|
| `ErrNoDocuments` → 领域 NotFound 哨兵错误 | 几乎所有 Store |
| `MatchedCount == 0` / `DeletedCount == 0` → NotFound | 26 个文件 |
| `IsDuplicateKeyError` → 领域「已存在」错误 | 29 个文件 |
| 写入时自动填充 `createdAt` / `updatedAt` | 39 个文件；字段的 bson tag 全仓库统一为 `createdAt` / `updatedAt` |
| `Find` + `SetSort` + `cursor.All` | 27 个文件 |
| 按过滤条件 upsert（`$set` + `$setOnInsert`） | 17 个文件 |
| 读写时字段加解密（`crypto.AESEncrypt/Decrypt`） | 8 个文件 |
| 分页（`SetSkip` + `CountDocuments`） | 10 个文件 |
| 计数器（`FindOneAndUpdate` + `$inc` + upsert） | 4 处，写法完全相同 |

主键约定不统一（`_id` ObjectID、`_id` string、业务字段 `id` string 三种并存），
且绝大多数查询走自然键（`appID`、`appID + envName`、`workspaceID + type`），而非主键。
因此 GenericStore **不应以主键为中心**，而应以「过滤条件」为中心。

### 分类（Registry 中的 50 个 Store）

分类标准：

* **A 类**：全部方法都能映射到 v1 能力（见下文 API 草案），领域校验逻辑可上移到 service 或通过 hook 保留。
* **B 类**：需要 v1.1 能力（分页、计数器、投影、FindOneAndUpdate），或仅个别方法需保留自定义实现。
* **C 类**：含聚合、数组/子文档原子操作、乐观锁、多集合协作等，保留自定义 Store；后续可内嵌 GenericStore 复用基础 CRUD。

**A 类（23 个，46%）**

| Store（Registry 字段） | 方法数 | 引用数* | 备注 |
|---|---|---|---|
| BkCIProjectStore | 2 | 13 | |
| BkCIPipelineStore | 4 | 7 | `UpdateBuiltinTemplateVersion` 是 3 个字段的 `$set` |
| BkRepoProjectStore | 3 | 4 | 需要加解密 hook |
| BkRepoRepositoryStore | 2 | 0 | |
| HelmRepoCredentialStore | 2 | 11 | 需要加解密 hook |
| BuildConfigStore | 4 | 38 | 需要加解密 hook |
| ImageRegistryStore | 4 | 10 | 需要加解密 hook |
| ClusterAddonDefStore | 4 | 17 | `Create` 实为按 name upsert |
| BscpCfgStore | 10 | 10 | 是 Metadata + EnvBinding 两个 Store 的门面，`GetSnapshot` 约 10 行领域代码 |
| AppDashboardStore | 4 | 1 | |
| HelmAppComponentStore | 6 | 0 | 写入前 `EnsureName()` |
| GPAConfigStore | 6 | 13 | 默认值与校验应上移到 service |
| AppServiceStore | 7 | 6 | `GroupByAppID` = `Find($in)` + `lo.GroupBy` |
| RuntimeImageStore | 6 | 6 | 校验应上移到 service |
| AppDefaultRuleStore | 8 | 7 | 需要 `Replace`、`FindOneAndDelete` |
| DepSvcBindingStore | 6 | 2 | 写入前 `SyncInstanceIDs()` |
| DepSvcStore | 6 | 4 | 需要编解码 hook（plan config 转 JSON 串） |
| PromotionStore | 7 | 20 | upsert；`ListTags` = `Find` + `lo.Map` |
| TempAdminRecordStore | 4 | 1 | `Update` 为「整体 `$set` 但排除不可变字段」 |
| PlatAdminStore | 4 | 5 | 需要「批量插入、已存在则跳过」 |
| PortForwardWhitelistStore | 4 | 1 | 同上；文档 `_id` 即 envID |
| AppConfigFileDefStore | 6 | 78 | 能力上可覆盖，但引用多，迁移排后 |
| FeatureEnvCounterStore | 2 | 4 | 由同包的 `store.Counter` 覆盖 |

\* 引用数：Registry 字段在 `pkg/` 内被 `.Field` 引用的次数，仅作迁移面的粗略参考（不含构造函数注入）。

**B 类（10 个）**

HelmChartBuildRecordStore、BuildRecordStore（计数器 + 分页 + 关键字）、
AppModelDeployResourceSnapshotStore（分页 + 投影 + InsertMany）、
CustomRuntimeImageStore（FindOneAndUpdate upsert 并返回文档）、
ResourceSnapshotStore（拓扑；5 个方法中仅 `UpsertWithVersion` 需自定义）、
HelmChartSemverCounterStore（多字段 `$inc`，体量小可保留）、
ComponentDefStore、WorkspaceStore、AppConfigFileVersionStore、AlertStrategyStore（后两者未精读）。

**C 类（17 个）**

EnvStore、ScopedEnvVarStore、AppStore、AppModelStore、AppSpecStore、AppConfigFileStore、
PolarisConfigStore、HostPortStore、WorkspaceCompsStore、BuildAutoDeployRecordStore、HelmDeployRecordStore、
AppModelDeployRecordStore、PublishRecordStore、DepSvcInstStore、ApmInstConfigStore、OperationRecordStore、SnapshotStore。

**Registry 之外**

* `bkci.PipelineTemplateStore`、`bkci.RepositoryStore`：A 类。
* `build/trigger.PolicyStore`、`build/trigger.RecordStore`：**只有接口、尚无实现**，适合作为零迁移成本的试点。
* `bkiam/role.RoleStore`、`appcfg.AppValuesConfigStore`：未精读，预计 A/B 类。

### 覆盖率结论

* v1（基础 CRUD + upsert + hook）：覆盖 23/50 = **46%**。
* v1.1（加分页、计数器、投影）：再覆盖 3～5 个，可达 **52%～56%**。
* A 类 Store 的实现代码合计约 4000 行（含注释与 license 头），迁移后预计可删除一半以上。

## API（v1，已实现于 `pkg/store`）

```go
type Meta[T any] struct {
	Collection string // 必填
	NotFound   error  // 返回的错误同时满足 errors.Is(err, NotFound) 与 errors.Is(err, store.ErrNotFound)
	Duplicate  error  // 同上，对应 store.ErrDuplicate

	// 钩子均可配置多个，按顺序执行
	BeforeSave []func(*T) error // 作用于调用方对象：补默认值、校验；仅完整文档写入时执行
	Encode     []func(*T) error // 作用于深拷贝：加密、序列化等；所有基于对象的写入都执行（含 UpdateFields）
	Decode     []func(*T) error // 作用于每条读出的记录：解密、反序列化等（Get / List）
}

func Of[T any](db *mongo.Database, meta Meta[T]) *GenericStore[T]           // Meta 配置错误时 panic，日常使用
func New[T any](db *mongo.Database, meta Meta[T]) (*GenericStore[T], error) // 以 error 返回配置错误

// 查询条件（Opt）：多个 Opt 的条件按 AND 合并
func Where(field string, value any) Opt   // 单字段条件，value 可以是 bson.M{"$gte": 1} 等操作符文档
func Filter(m bson.M) Opt                 // 一次传入多个条件，按 key 排序以保证生成的 filter 稳定
func Combine(opts ...Opt) Opt             // 组合多个 Opt，供领域包定义 ByWorkspaceAndType 之类的复合条件
func All() Opt                            // 写操作显式声明作用于全部文档
func Asc(field string) Opt / Desc(field string) Opt / Limit(n int64) Opt // 仅 Get / List 可用

// 读（不传条件即匹配全部）
Get(ctx, opts ...Opt) (*T, error)                           // 未命中返回 NotFound；可配合 Desc 取最新一条
List(ctx, opts ...Opt) ([]T, error)                         // 无结果返回空切片
Exists(ctx, opts ...Opt) (bool, error)
Count(ctx, opts ...Opt) (int64, error)
// 写
Create(ctx, obj *T) error                                   // 回填 _id 与时间戳
CreateMany(ctx, objs []*T) error
CreateManyIfAbsent(ctx, objs []*T) error                    // 跳过唯一键冲突的文档
Update(ctx, set bson.M, opts ...Opt) error                  // $set + updatedAt，不经过钩子
UpdateFields(ctx, obj *T, fields []string, opts ...Opt) error // 从 Encode 后的 obj 中取指定字段 $set
Upsert(ctx, obj *T, opts ...Opt) (inserted bool, err error) // $set 全部字段 + $setOnInsert _id/createdAt；插入时回填二者
Replace(ctx, obj *T, opts ...Opt) error                     // 整体替换，保留原 _id；未命中返回 NotFound
// 删
Delete(ctx, opts ...Opt) error                              // 未命中返回 NotFound
DeleteIfExists(ctx, opts ...Opt) error                      // 未命中静默成功
DeleteMany(ctx, opts ...Opt) (int64, error)
// 逃生口
Collection() *mongo.Collection                              // 聚合、分页等未覆盖的查询

// 计数器
func NewCounter(db *mongo.Database, collection string) *Counter // Next(ctx, key) (int64, error)

// fx
store.FxModule          // 提供 *mongo.Database，一个 fx 应用只引入一次
store.Provide(meta)     // 提供 *GenericStore[T]，启动时即构造，Meta 配置错误在启动阶段暴露
```

自动维护的约定：

* 按 bson tag 识别 `_id`、`createdAt`、`updatedAt`（含 `,inline` 嵌入的结构体）；时间戳字段必须是 `time.Time`。
  反射分析结果按模型类型缓存在进程内，`Of` / `New` 只在首次遇到某个模型时反射，之后的构造只是组装结构体。
* `_id` 为零值时在写入前生成：`bson.ObjectID` 生成 ObjectID，string 生成 ObjectID 的 hex，其它类型交给 MongoDB。
* Create 时 `createdAt` 为零值则取当前时间，`updatedAt` 与之相同；其余写操作刷新 `updatedAt`。
* 写操作（Update / UpdateFields / Upsert / Replace / Delete*）必须带条件，确需作用于全部文档时显式传 `store.All()`；
  排序与 Limit 只对 Get / List 生效，传给其它操作直接报错，避免被静默忽略。
* 同一字段出现在多个 Opt 中时（如 `Where("n", {"$gte": 1})` 与 `Where("n", {"$lte": 9})`）改用 `$and` 合并，不会互相覆盖。
* `UpdateFields` 会校验字段路径是否存在于模型中；字段在编码结果中缺失（omitempty 的零值）时 `$unset`。

字段名直接写 bson 字段名字符串，嵌套字段用 `"a.b"`；更新内容仍使用 driver 原生的 `bson.M`。
领域包把常用条件定义为具名 Opt，调用方读起来接近原先的具名方法：

```go
// pkg/bkintegrations/bkrepo/store.go
type RepositoryStore = store.GenericStore[Repository]

var RepositoryMeta = store.Meta[Repository]{Collection: "bkrepo_repositories", NotFound: ErrRepositoryNotFound}

func ByWorkspace(workspaceID string) store.Opt { return store.Where("workspaceID", workspaceID) }

func ByWorkspaceAndType(workspaceID string, repoType RepoType) store.Opt {
	return store.Combine(ByWorkspace(workspaceID), store.Where("type", repoType))
}

// 调用方
repo, err := repoStore.Get(ctx, bkrepo.ByWorkspaceAndType(wsID, bkrepo.RepoTypeHelm))
```

模型包的约定：导出 `XxxMeta`（索引说明注释写在 Meta 上方）、`type XxxStore = store.GenericStore[Xxx]` 类型别名、
具名 Opt；业务哨兵错误保留原名，调用方的 `errors.Is` 无需改动。

## 决策记录

1. **调用方面向什么接口**（已定）：调用方直接使用 `*GenericStore[T]`，不为每个模型定义 Store 接口。
   少数方法语义不平凡的场景（如「仅用于内置流水线」的更新），在模型包内提供具名函数或薄包装，而不是恢复整套领域接口。
2. **字段名写法**（已定）：过滤、排序、更新直接用 `bson.M`，字段名只支持 bson 字段名字符串。
   曾尝试过两种由结构体字段推导字段名的方案，均已放弃：类型化字段描述符（`store.Field[T, V]` + `FieldsOf`）语法偏重、
   需要为每个模型额外声明；基于探针实例取址反查的 `store.F(&store.Of[T]().Field)` 依赖内存地址与反射，机制隐晦、
   误用只能在运行时发现，收益不足以抵消理解成本。
3. **Delete 未命中语义**（已定）：`Delete` 返回 NotFound，`DeleteIfExists` 静默成功，迁移时按调用方现有语义二选一。
4. **Registry 收敛方式**（已定）：被 GenericStore 替换的 Store 直接从 `storereg.Registry` 删除字段，
   `storereg.Registry` 只额外持有 `DB *mongo.Database`，需要时 `store.Of(registry.DB, XxxMeta)`（见决策 7）。
   阶段 2 曾实现类型键控的 `store.Registry` + `store.For[T]`，阶段 3 后删除：它要求每个模型在启动时登记，
   多了一份需要同步维护的清单，而 `store.Of` 加上反射缓存已经足够轻。
5. **钩子拆分**（阶段 2 确定）：草案中单一的 `BeforeWrite` 作用于副本，无法表达「补默认值并让调用方看到」，
   因此拆为 `BeforeSave`（调用方对象）、`Encode`（深拷贝）、`Decode`（读出后）；阶段 3 后改为切片，支持多个钩子。
   深拷贝使用项目已有的 copier（`DeepCopy + IgnoreEmpty`），不用 bson 往返，因为后者会把 `map[string]any` 中的嵌套文档变成 `bson.D`。
   实测 copier v0.4.0：`time.Time`、指针、嵌套 map/切片、bson 文档均正确隔离；不加 `IgnoreEmpty` 时 nil 切片/map 会变成空值
   （编码结果从 null 变为 []），加上后保持 nil；`any` 字段中的指针、数组中的切片仍是浅拷贝，现有模型中不存在这两种形态。

6. **查询条件形式**（阶段 3 确定）：所有查询与写操作的条件统一通过可变参数 `...Opt` 传入，取代位置参数 `filter bson.M`。
   好处是领域包可以把条件封装为具名 Opt（`ByWorkspace` 等）自由组合，排序、Limit 也走同一个入口；
   代价是「写操作必须带条件」只能在运行时校验，所以由 `store.All()` 显式表达全量操作。
7. **Store 的获取方式**（阶段 3 后调整）：领域函数与 Manager 不再接收若干个 Store，而是显式接收 `db *mongo.Database`，
   在用到时 `store.Of(db, XxxMeta)`；handler、taskq 任务传入 `storereg.Registry.DB`。
   没有选择「领域代码内部直接取全局 `database.DB()`」，是为了让依赖在签名上可见，单测也能传入测试库。
   例外：`bkci.NewTriggerPipelineManager(workspaceID)` 是 `build_trigger_contract.md` 中的 W1 契约，签名不变，内部使用 `database.DB()`。
   只操作单个模型的辅助函数（如 `bkci.UpdateBuiltinTemplateVersion(ctx, pipelineStore, p)`）仍接收 Store。
   单测通过 `testutil.NewDB(GinkgoT())` 或 `testutil.NewStore(GinkgoT(), XxxMeta)` 获取依赖（内部都是 fxtest + `fx.Populate`），
   针对测试库读写，不 mock Store。
8. **字段加密方式**（阶段 3 后确定）：加解密由模型在 Encode / Decode 钩子中完成，借助 `crypto.OnDemand` 每个字段一行：

   ```go
   func encryptPassword(c *HelmRepoCredential) (err error) {
   	c.Password, err = crypto.OnDemand(config.G.Encrypt.Secret, crypto.AESEncrypt, c.Password)
   	return errors.Wrap(err, "encrypt password")
   }
   ```

   `OnDemand` 对空字符串直接返回（敏感字段未设置），其余情况调用传入的加解密函数并原样返回错误。
   曾实现过声明式方案（`Meta.Encrypted` 按 bson 路径声明字段 + `Cipher` 接口，`Update` 也能自动加密），
   因按字段路径反射定位、校验路径形态、处理 `Update` 中父字段写入等逻辑过于复杂而撤回；代价是 `Update` 仍会绕过加密（见 API 回顾问题 1）。

## 阶段 2 API 回顾

实现过程中相对草案的调整：钩子拆分（见决策 5）；`List` 的排序改为 `ListOption`，以便加入 `Limit`；
新增 `UpdateFields`、`CreateManyIfAbsent`、`DeleteIfExists`、`Count`；写操作拒绝 nil filter；
`New` 改为接收 `*mongo.Database` 并返回 error（模型不是结构体、时间戳类型不对时尽早失败）。

需要在试点前确认的问题：

1. **`Update` 绕过 `Encode`，会写入明文**：涉及加密字段的更新必须使用 `UpdateFields`；
   目前试点模型中没有用 `Update` 写加密字段的调用方，靠评审与单测保证。
2. ~~`Upsert` 无法区分插入还是更新~~（已解决）：改为返回 `(inserted bool, err error)`，插入时把 `_id`（取自 `UpsertedID`）
   与 `createdAt` 回填到 obj；更新时库中已有的值未知，obj 中的这两个字段保持不变。
3. **`Replace` 原样写入 `obj.CreatedAt`**：传入零值会把创建时间覆盖为零。约定 `obj` 来自 `Get`；
   备选是在 `createdAt` 为零值时拒绝替换。
4. **A 类中仍未覆盖的能力**：`AppDefaultRuleStore` 需要 `FindOneAndDelete`；B 类需要分页（`Skip` + `Count`）与投影。
   计划在 v1.1 加入 `store.Skip(n)` 与 `GetAndDelete`，迁移到对应 Store 时再实现，避免预先设计。
5. **fx 装配约束**：`store.FxModule` 一个应用只能引入一次，各功能模块只写 `store.Provide(meta)`；
   功能模块的单测需要同时引入 `store.FxModule`。非 fx 代码通过 `store.Of(db, meta)` 取用（决策 7）。
6. **字段名拼写错误**：字段名为纯字符串，`UpdateFields` 会按模型校验路径，但 filter、`Update`、排序中的拼写错误不会报错，
   只会表现为查不到或写入多余字段，需要靠单测覆盖。

确认无问题的部分：错误同时匹配通用与业务哨兵、`errors.As` 仍可取到驱动原始错误；每个模型只需一个 `Meta` 变量即可接入。

## 分阶段计划与进度跟踪

| 阶段 | 内容 | 状态 |
|---|---|---|
| 1 | 扫描全部 Store，分类并评估覆盖率 | ✅ 已完成 |
| 2 | 实现 `pkg/store` GenericStore v1、Counter、fx 集成，基于真实 DB 的单测 | ✅ 已完成 |
| 3 | 试点：bkrepo ×2、bkci ×4、HelmRepoCredential（覆盖加解密 hook 路径）；`build/trigger` 暂缓 | ✅ 已完成（见下节） |
| 4 | 迁移剩余 A 类，引用多的（BuildConfig、AppConfigFileDef）放最后 | 未开始 |
| 5 | v1.1：分页、投影、FindOneAndUpdate；迁移 B 类 | 未开始 |
| 6 | Registry 收敛；评估 C 类内嵌 GenericStore 复用基础 CRUD | 未开始 |

## 阶段 3 试点结果

已迁移 7 个 Store：bkci 的 Project / PipelineTemplate / Pipeline / Repository、bkrepo 的 Project / Repository、
`helmcore/credential` 的 HelmRepoCredential。原有的 Store 接口、`*StoreMongo` 实现及 `storereg.Registry` 中的 5 个字段
（`BkCIProjectStore`、`BkCIPipelineStore`、`BkRepoProjectStore`、`BkRepoRepositoryStore`、`HelmRepoCredentialStore`）已删除，
`Registry` 新增 `DB *mongo.Database`。配套新增 `database.DB()`、`testutil.NewStore` 与 `testutil.NewDB`。

`build/trigger` 的 RecordStore 暂缓：它属于 `build_trigger_contract.md` 中的 W1 契约，等该功能稳定后再迁移；
契约只约定 `NewTriggerPipelineManager(workspaceID).Ensure/Cleanup`，bkci 内部改用 GenericStore 不影响它。

行为变化（调用方需要知道的）：

1. **bkrepo Project 的 Create 不再改写调用方对象**：原实现就地加密，Create 之后调用方对象里的密码变成密文；现在只加密落库的副本（即「顺带发现的问题」第 2 条，已修复）。
2. **唯一键冲突**统一返回 `store.ErrDuplicate`（可同时匹配 Meta 中的 `Duplicate`），不再是各 Store 自拼的 "already exists" 文案。
3. **NotFound 错误**带上 collection 与 filter，便于排查；业务哨兵（如 `bkci.ErrPipelineNotFound`）照常匹配。
4. **`bkci.UpdateBuiltinTemplateVersion`** 基于 `UpdateFields`：`templateVersion` 为空时会 `$unset`，原实现写入空字符串；读出后两者都是空字符串。
5. **流水线模板 reload** 由 `ReplaceOne(upsert)` 改为 `Upsert`（`$set` 全部字段）：模板中删掉的字段不会从库中清除。
   目前模板字段固定，无影响；如需整体替换，可改为「`Get` 后 `Replace`，未命中再 `Create`」。

测试改造：上述包的单测全部改为针对测试库，移除了对这些 Store 的 mockey mock。原先 mock `Create` 返回错误来验证
「本地落库失败时回滚远端流水线」的用例，改为预先写入相同流水线 ID 的记录、制造真实的唯一键冲突。
镜像源 `ImageRegistryStore` 不在本次范围，`bkrepo/manager_test.go` 中对它的 mock 保持原样。

遗留：`db/migrations/000001_initial_idx.md` 中仍以 `ProjectStoreMongo` 等旧类型名描述索引来源，它是迁移文件的历史说明，未改动。

## 阶段 3 后续调整

根据试点代码的评审意见做了两项调整，结论已并入上文的 API 与决策 4、5、7、8：

1. **钩子改为切片**：`BeforeSave` / `Encode` / `Decode` 均可配置多个，按顺序执行；
   bkrepo Project、HelmRepoCredential 的加解密钩子改用 `crypto.OnDemand` 实现（决策 8）。
2. **领域函数只接收 db**：`store.Registry` / `store.For` / `store.Register` 已删除，新增 `store.Of`。
   以下函数与类型由接收若干 Store 改为接收 `db *mongo.Database`：
   * `credential.EnsureCredential`、`source.ResolveConfig`、`deploy/helm` 的 `PreviewHelmRelease` / `UpgradeOrInstallHelmChart`
   * `build/chart.NewChartBuildService(db, semverStore, recordStore)`（原先 6 个参数）
   * bkci 的 `NewProjectManager` / `NewPipelineManager` / `NewRepositoryManager`、`EnsureWorkspaceRepositories`、
     `ReloadPipelineTemplates`；bkrepo 的 `NewProjectManager`
   * 因此连带：`workspace.EnsureBkSystems`、`build/image.ExecuteBKCIPipelineBuild`、`build/build.NewService`（新增首个参数 db）、
     `buildpoll` / `chartbuildpoll` 的 poller

   这些调用链的顶端都是持有 `storereg.Registry` 的 handler、taskq 任务或 `cmd/webserver.go`，均传入 `Registry.DB`。
   原先领域代码中 19 处 `store.New(database.DB(), XxxMeta)` 及其错误处理样板已全部消除。

## 扫描中顺带发现的问题

迁移时应一并修复，或单独提交：

1. `depservice/model/service_store.go` 的 `List` 未调用 `serviceFromDBValue`，返回的 plan config 仍是 `{"value": "<json 串>"}`，
   与 `Get` 不一致。目前只有测试调用 `List`，属于潜在 bug。统一的 `Decode` 钩子可以根除这类问题。
2. （bkrepo 部分已在阶段 3 修复）加密在调用方对象上就地进行：`bkrepo` 的 `ProjectStoreMongo.Create`、`build/image` 的 `ConfigStoreMongo.Create/Update`
   会把调用方对象中的密码替换为密文；而 `credential`、`registry` 的实现先复制再加密。`Encode` 作用于深拷贝可统一行为。
3. NotFound 语义不一致：`AppConfigFileDefStoreMongo.GetByID`、`ConfigStoreMongo.Get/Update` 返回不带哨兵的 `errors.Errorf`；
   `networking` 的 `Update` 返回字符串错误；拓扑 `ResourceSnapshotStoreMongo.Get` 未找到时返回 `nil, nil`。迁移时须保持调用方依赖的语义。
4. 不少 Store 直接 `return err` 不加 Wrap；`appdefaults/store.go` 使用 `fmt.Errorf("%w")`，均违反 AGENTS.md 的错误处理规范。
