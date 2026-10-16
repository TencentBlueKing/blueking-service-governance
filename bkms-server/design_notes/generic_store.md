# GenericStore：基于泛型的通用 Store

> 状态：阶段 5 已完成，A、B 两类 Store 全部迁移（见「阶段 4 迁移结果」与「阶段 5 计划」下的 5a～5e 结果）；
> v1.1 的 `ListPage` / `WithoutTotal` / `OmitFields` 已实现；Update / Upsert 已统一为同一套字段语义（决策 13）。
> Registry 剩 17 个 C 类 Store，留待阶段 6。
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
阶段 5 精读后重新分组，其中 4 个只需 v1 能力，见「阶段 5 计划」。

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

## API（v1 + v1.1，已实现于 `pkg/store`）

```go
type Meta[T any] struct {
	Collection string // 必填
	NotFound   error  // 返回的错误同时满足 errors.Is(err, NotFound) 与 errors.Is(err, store.ErrNotFound)
	Duplicate  error  // 同上，对应 store.ErrDuplicate

	// 钩子均可配置多个，按顺序执行
	BeforeSave []func(*T) error // 作用于调用方对象：补默认值、校验；仅完整文档写入时执行（不含 Update、传入 map 的 Upsert）
	Encode     []func(*T) error // 作用于深拷贝：加密、序列化等；所有写入都执行，Update 传入的 map 也会先转换为 T
	Decode     []func(*T) error // 作用于每条读出的记录：解密、反序列化等（Get / List）
}

func Of[T any](db *mongo.Database, meta Meta[T]) *GenericStore[T]           // Meta 配置错误时 panic，日常使用
func New[T any](db *mongo.Database, meta Meta[T]) (*GenericStore[T], error) // 以 error 返回配置错误

// 选项（Opt 接口，按用途分为不同类型，见决策 9）：多个过滤条件按 AND 合并
func Where(field string, value any) CondOpt // 单字段条件，value 可以是 bson.M{"$gte": 1} 等操作符文档
func Filter(m bson.M) CondOpt               // 一次传入多个条件，按 key 排序以保证生成的 filter 稳定
func ByID(id any) CondOpt                   // 按 _id 匹配，id 类型须与模型 _id 字段一致（ObjectID 与 hex 串不能混用）
func ContainsIgnoreCase(keyword, field string, moreFields ...string) CondOpt // 忽略大小写模糊匹配，多字段为 OR；
                                            // keyword 去首尾空白、转义正则元字符，空白时不产生条件
func All() CondOpt                          // 写操作显式声明作用于全部文档
func Asc(field string) SortOpt / Desc(field string) SortOpt // 仅 Get / List / ListPage / GetAndDelete 支持
func Limit(n int64) LimitOpt                // 仅 List 支持
func OmitFields(fields ...string) ProjectOpt // 读取时不取回指定字段（被排除的字段为零值），仅 Get / List / ListPage 支持；
                                            // 字段必须存在于模型中，互为父子的字段直接报错
func WithoutTotal() PageOpt                 // 仅 ListPage 支持，不统计总数
// 以下仅 Update / Upsert 传入 *T 时支持，决定写入哪些字段（见决策 13）
func UpdateSelected(fields ...string) UpdateOpt // 指定字段，"*" 表示全部顶层字段
func UpdateOmit(fields ...string) UpdateOpt     // 从候选字段中去掉指定字段
func UpdateNonZero() UpdateOpt                  // 只保留对象中值非零的字段
func Combine(opts ...Opt) Opt               // 组合多个 Opt，供领域包定义 ByWorkspaceAndType 之类的复合条件

// 读（不传条件即匹配全部）
Get(ctx, opts ...Opt) (*T, error)                           // 未命中返回 NotFound；可配合 Desc 取最新一条
GetByID(ctx, id any) (*T, error)                            // 等价于 Get(ctx, ByID(id))
List(ctx, opts ...Opt) ([]T, error)                         // 无结果返回空切片
ListPage(ctx, page, size int64, opts ...Opt) (*Page[T], error) // page 从 1 开始；Page{Items, Total, HasNext}，见「v1.1 能力」
Exists(ctx, opts ...Opt) (bool, error)
Count(ctx, opts ...Opt) (int64, error)
// 写
Create(ctx, obj *T) error                                   // 回填 _id 与时间戳
CreateMany(ctx, objs []*T) error
CreateManyIfAbsent(ctx, objs []*T) error                    // 跳过唯一键冲突的文档
Update(ctx, target any, opts ...Opt) error                  // target 为 map 或 *T（须带 UpdateOpt），经 Encode 后 $set / $unset
Upsert(ctx, target any, opts ...Opt) (inserted bool, err error) // 同 Update，*T 默认全部顶层字段；未选中的字段放入 $setOnInsert
var ErrNoFieldsToUpdate                                     // 没有任何待写入的字段（空 map、UpdateNonZero 时全为零值等）
Replace(ctx, obj *T, opts ...Opt) error                     // 整体替换，保留原 _id；未命中返回 NotFound
// 删
Delete(ctx, opts ...Opt) error                              // 未命中返回 NotFound
DeleteIfExists(ctx, opts ...Opt) error                      // 未命中静默成功
DeleteMany(ctx, opts ...Opt) (int64, error)
GetAndDelete(ctx, opts ...Opt) (*T, error)                  // 原子删除并返回被删文档（经过 Decode），可配合 Desc 指定删除哪条
// 逃生口
Collection() *mongo.Collection                              // 聚合、$inc 等未覆盖的操作

// 计数器
func NewCounter(db *mongo.Database, collection string) *Counter // Next(ctx, key) (int64, error)

// fx（见决策 10）
database.PrivateFxModule // 功能模块内私有地提供 *mongo.Database（以及 client、库名），各模块可各自引入
store.FxModule           // 向整个应用公开提供 *mongo.Database，一个 fx 应用只引入一次，主要供测试 Populate
store.Provide(meta)      // 提供 *GenericStore[T]，启动时即构造，Meta 配置错误在启动阶段暴露
```

自动维护的约定：

* 按 bson tag 识别 `_id`、`createdAt`、`updatedAt`（含 `,inline` 嵌入的结构体）；时间戳字段必须是 `time.Time`。
  反射分析结果按模型类型缓存在进程内，`Of` / `New` 只在首次遇到某个模型时反射，之后的构造只是组装结构体。
* `_id` 为零值时在写入前生成：`bson.ObjectID` 生成 ObjectID，string 生成 ObjectID 的 hex，其它类型交给 MongoDB。
* Create 时 `createdAt` 为零值则取当前时间，`updatedAt` 与之相同；其余写操作刷新 `updatedAt`。
* 写操作（Update / Replace / Delete*）必须带条件，确需作用于全部文档时显式传 `store.All()`；
  `Upsert` 在 obj 已有 `_id` 时会自动把它追加到过滤条件，因此此时可以不额外传条件；obj 的 `_id` 为零值时仍须传条件；
  每个操作只接受部分类型的 Opt（如 Limit 只用于 List），传入不支持的 Opt 直接报错，避免被静默忽略。
* 同一字段出现在多个 Opt 中时（如 `Where("n", {"$gte": 1})` 与 `Where("n", {"$lte": 9})`）改用 `$and` 合并，不会互相覆盖。
* Update / Upsert 的字段必须存在于模型中，`_id` 与 `createdAt` 不能更新；字段在编码结果中缺失（omitempty 的零值）时 `$unset`。
字段名可以写 bson 字段名或 Go 字段名（如 `"spec.color"` 与 `"Spec.Color"`），嵌套字段用 `"a.b"`；代码中优先写 bson 字段名。
过滤、排序中模型不存在的字段原样保留，只有歧义（某一段既是一个字段的 bson key、又是另一个字段的 Go 字段名）时报错。
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
具名 Opt；业务哨兵错误保留原名，调用方的 `errors.Is` 无需改动。阶段 4 中沉淀的补充约定：

* 按主键查询直接用 `GetByID`，其它操作用 `store.ByID`，不在模型包中定义只包装 `_id` 的具名 Opt（原 `appcfg.DefByID` 已删除）。
  主键需要附加范围约束时（如 `appdefaults.ByID` 要求规则属于指定工作空间与配置类型，防止越权访问），
  仍定义具名 Opt，并用 `store.Combine(..., store.ByID(id))` 组合。

* 结构体校验（`validate.Struct`）、补默认名称等写入前处理放进 `BeforeSave`；只在「新建」时成立的默认值
  （如 GPA 新建即启用）不放进钩子，而是由模型包的具名函数（`gpa.CreateConfig`）或调用方处理，避免 `Upsert` / `Replace` 时被误用。
* 语义不平凡的操作提供接收单个 Store 的具名函数，例如「读出、合并、整体校验后再写回」的 `gpa.UpdateConfig`、
  按自然键幂等 upsert 的 `promotion.Promote`、全量替换可变字段的 `appcfg.UpdateDef`（`UpdateSelected`）/ `build.UpdateConfig`（`UpdateOmit`）。
  可变字段直接写在调用处的 UpdateOpt 中，不再单独定义字段列表变量。
* 局部更新（PATCH 语义）直接用 `map[string]any` 表达待更新的字段，由 serializer 生成、经 service 传给 `Update`，
  key 与值类型由 store 在写入前校验（见 bscpcfg 的 `PatchMetadataInput.ToUpdateFields`）。
  networking、helm 组件、dashboard、gpa 中仍有历史遗留的 `XxxUpdateData.ToBSON()`，后续按同样方式收敛。
* 可选过滤条件在入参为空时让具名 Opt 返回 nil，`parseOpts` 会跳过 nil；写操作仍会因缺少条件而报错，不会误作用于全量。
  关键字模糊匹配直接用 `store.ContainsIgnoreCase`（空白关键字同样不产生条件），不再各自拼 `$regex` + `QuoteMeta`，
  如 `runtime.KeywordMatches`、`admin.UsernameContains`。
* 列表接口原先返回 `[]*T` 的，调用方需要时用 `lo.ToSlicePtr` 转换，不为此扩展 GenericStore。

## 决策记录

1. **调用方面向什么接口**（已定）：调用方直接使用 `*GenericStore[T]`，不为每个模型定义 Store 接口。
   少数方法语义不平凡的场景（如「仅用于内置流水线」的更新），在模型包内提供具名函数或薄包装，而不是恢复整套领域接口。
2. **字段名写法**（决策 13 调整）：字段名仍是字符串，但除 bson 字段名外也接受 Go 字段名，由 `resolvePath` 按模型
   逐段转换（bson key 优先、两者指向不同字段时报歧义），过滤、排序与更新共用；约定代码中优先写 bson 字段名。
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
   代价是「写操作必须带条件」只能在运行时校验，所以由 `store.All()` 显式表达全量操作；
   `Upsert` 中非零的 obj `_id` 本身也是过滤条件。
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
   因按字段路径反射定位、校验路径形态、处理 `Update` 中父字段写入等逻辑过于复杂而撤回。
   原先的代价是 `Update` 会绕过加密，决策 13 中 `Update` 传入的 map 也会先转换为 T 再经过 Encode，该问题已不存在。
9. **选项类型**（评审 Upsert 语义后确定）：`Opt` 是只能由 `pkg/store` 实现的接口，按用途分为 `CondOpt`（Where / Filter / All）、
   `SortOpt`（Asc / Desc）、`LimitOpt`（Limit）、`UpdateOpt`（UpdateSelected / UpdateOmit / UpdateNonZero）。每个操作声明自己接受的类别，
   遇到其它类别（含 `Combine` 组合出的）时返回错误，取代原先散落在各处的「排序与 Limit 仅 Get / List 可用」判断；
   新增行为开关时只需新增类型并登记到对应的操作，其它操作不会被动接受它。
   仍保留统一的 `...Opt` 参数而非为每个操作定义不同的参数接口，好处是领域包的具名 Opt 可以返回 `store.Opt` 并自由组合，
   代价是不支持的选项只能在运行时发现（与决策 6 相同的取舍）。随之 `Get` 不再静默忽略 `Limit`。
   原先的 `UpsertOpt`（`UnsetOmitted`）已由决策 13 的 `UpdateOpt` 取代。
10. **fx 装配方式**（阶段 4 确定，取代阶段 2 的「`store.FxModule` 一个应用只引入一次、功能模块单测另外引入」）：
    `database.PrivateFxModule` 额外以 `fx.Private` 提供 `*mongo.Database`，功能模块（如 `gpa.FxModule`）因此可以自包含地
    构造接收 db 的 Service；`store.FxModule` 改为直接公开提供 db，只在需要 `fx.Populate` db 或在根上 `fx.Provide`
    接收 db 的构造函数时（如 `deploy_precheck_test`）引入。已验证模块内私有的 db 与根上公开的 db 共存、嵌套模块各自私有均不冲突。
    迁移后的功能模块不再提供被迁移的 Store（只剩 Store 的模块直接删除，如 promotion、networking、appdefaults），
    测试统一用 `testutil.NewStore` / `testutil.NewDB` 获取。
11. **类型化的 NotFound 错误**（阶段 4 确定）：depservice 以 `*NotFoundError` 类型 + `errors.As`（`AsNotFoundError`）判断未找到。
    没有让 `storeError` 支持 `errors.As` 委托给业务错误，而是为迁移的模型新增哨兵 `ErrServiceNotFound` / `ErrBindingNotFound`
    放进 Meta，并让 `AsNotFoundError` 同时识别这两个哨兵；仍保留自定义 Store 的 `ServiceInstance` 继续返回 `*NotFoundError`。
12. **`GetAndDelete`**（阶段 4 新增，原计划在 v1.1）：`AppDefaultRule` 删除后需要被删文档写审计，
    「先 `Get` 再 `Delete`」不是原子的，因此提供基于 `FindOneAndDelete` 的 `GetAndDelete`，支持 CondOpt 与 SortOpt。
13. **Update / Upsert 统一语义**（阶段 4 后确定）：原先有三种更新入口——不经过钩子的 `Update(bson.M)`、
    按字段列表取值的 `UpdateFields(obj, fields)`、只针对 Upsert 的 `UnsetOmitted()`，各自语义不同，`Update` 还会绕过加密。
    现统一为「一个 T 对象 + 一组字段」：
    * `Update` / `Upsert` 的 target 为 `map[string]any` 或 `*T`。map 的 key 即字段，值按字段类型赋给一个新的 T
      （可直接赋值的直接赋值，否则借助 bson 转换，不整体做 bson 往返，以免 `map[string]any` 中的嵌套文档变成 `bson.D`）；
      `*T` 的字段由 UpdateOpt 计算：候选为 `UpdateSelected` 指定的字段（未指定或 `"*"` 时为全部顶层字段），
      去掉 `UpdateOmit` 的字段，再按 `UpdateNonZero` 过滤零值。
    * 对象经过 Encode 后，字段存在的 `$set`、缺失的 `$unset`。`UnsetOmitted` 因此不再需要：`UpdateSelected("*")`
      下 omitempty 的零值本来就会被 `$unset`。没有任何待写入字段时返回 `ErrNoFieldsToUpdate`。
    * `Update` 传入 `*T` 时必须显式给出 UpdateOpt。没有采用「默认只更新非零值」：迁移前的 `*T` 调用方全都需要写入
      false / 0 / 空值，非零默认会静默丢失这类修改；对 Upsert 还会导致 `ClusterAddonDef.UnsupportedOnFederation`
      由 true 改为 false 时被跳过。
    * `Upsert` 传入 `*T` 时默认 `"*"`，现有调用方的模型除 `_id` 外没有 omitempty 字段，行为不变。未被选中的字段放入
      `$setOnInsert`，从而支持「不存在时插入完整文档，已存在时只更新部分字段」；过滤条件中的等值字段不放入
      `$setOnInsert`（插入时由 MongoDB 取过滤条件中的值，避免被零值覆盖）；字段只能是顶层字段，避免与 `$setOnInsert` 冲突。
    * BeforeSave 只在 `Upsert` 传入 `*T` 时执行：`Update` 的对象可能只填写了部分字段，传入 map 的 `Upsert`
      事先无法知道是插入还是更新，校验类钩子都可能误判。

## 阶段 2 API 回顾

实现过程中相对草案的调整：钩子拆分（见决策 5）；`List` 的排序改为 `ListOption`，以便加入 `Limit`；
新增 `UpdateFields`、`CreateManyIfAbsent`、`DeleteIfExists`、`Count`；写操作拒绝 nil filter；
`New` 改为接收 `*mongo.Database` 并返回 error（模型不是结构体、时间戳类型不对时尽早失败）。

需要在试点前确认的问题：

1. ~~**`Update` 绕过 `Encode`，会写入明文**~~（已解决，决策 13）：传入的 map 会先转换为 T 再经过 Encode。
2. ~~`Upsert` 无法区分插入还是更新~~（已解决）：改为返回 `(inserted bool, err error)`，插入时把 `_id`（取自 `UpsertedID`）
   与 `createdAt` 回填到 obj；更新时库中已有的值未知，obj 中的这两个字段保持不变。
3. **`Replace` 原样写入 `obj.CreatedAt`**：传入零值会把创建时间覆盖为零。约定 `obj` 来自 `Get`；
   备选是在 `createdAt` 为零值时拒绝替换。
4. **A 类中仍未覆盖的能力**：~~`AppDefaultRuleStore` 需要 `FindOneAndDelete`~~（阶段 4 已加入 `GetAndDelete`，决策 12）；
   B 类需要分页与投影，v1.1 以 `ListPage` 与 `store.OmitFields` 提供（见「阶段 5 计划」），已随阶段 5b 实现。
   `Promotion` 的「只取 tag 列表」原先用投影，A 类迁移中改为 `List` 后取字段，记录很小，暂不需要投影。
5. ~~**fx 装配约束**~~：已由决策 10 取代。
6. **字段名拼写错误**：字段名为纯字符串，`Update` / `Upsert` 会按模型校验字段，但 filter、排序中的拼写错误不会报错
   （需要兼容模型中已删除的旧字段），只会表现为查不到，需要靠单测覆盖。

确认无问题的部分：错误同时匹配通用与业务哨兵、`errors.As` 仍可取到驱动原始错误；每个模型只需一个 `Meta` 变量即可接入。

## 分阶段计划与进度跟踪

| 阶段 | 内容 | 状态 |
|---|---|---|
| 1 | 扫描全部 Store，分类并评估覆盖率 | ✅ 已完成 |
| 2 | 实现 `pkg/store` GenericStore v1、Counter、fx 集成，基于真实 DB 的单测 | ✅ 已完成 |
| 3 | 试点：bkrepo ×2、bkci ×4、HelmRepoCredential（覆盖加解密 hook 路径）；`build/trigger` 暂缓 | ✅ 已完成（见下节） |
| 4 | 迁移剩余 A 类，引用多的（BuildConfig、AppConfigFileDef）放最后 | ✅ 已完成（见下节） |
| 5 | v1.1：分页、投影；迁移 B 类（分 5a～5e 五批，见「阶段 5 计划」） | ✅ 已完成 |
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
5. **流水线模板 reload** 由 `ReplaceOne(upsert)` 改为 `Upsert(..., store.UnsetOmitted())`（决策 13 后 Upsert 默认即此行为，
   已去掉该选项）：模型中定义的字段与模板文件保持一致，但模型中删掉的字段不会从库中清除。目前模板字段固定，无影响；如需整体替换，可改为「`Get` 后 `Replace`，未命中再 `Create`」。

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

## 阶段 4 迁移结果

A 类剩余的 18 个 Store 全部迁移，`storereg.Registry` 中对应字段已删除，原 Store 接口、`*StoreMongo` 实现与只提供 Store 的 FxModule 一并删除：

| 原 Store（Registry 字段） | Meta | 要点 |
|---|---|---|
| ImageRegistryStore | `registry.ImageRegistryMeta` | 密码经 Encode / Decode 加解密 |
| ClusterAddonDefStore | `clusteraddon.ClusterAddonDefMeta` | `SaveDef` 按 name upsert；`ListDefs` 按名称排序并返回 `[]*T` |
| BscpCfgStore | `bscpcfg/model.MetadataMeta`、`EnvBindingMeta` | 门面 Store 删除，聚合改为 `model.GetSnapshot(ctx, db, appID, envName)` |
| AppDashboardStore | `dashboard.Meta` | 校验进 BeforeSave；按 uid 局部更新的逻辑移到 Service |
| HelmAppComponentStore | `helmcomponent.Meta` | `EnsureName` 进 BeforeSave；`InExecutionOrder()` 用 `Combine` 组合两个排序 |
| GPAConfigStore | `gpa.ConfigMeta` | `Validate` 进 BeforeSave；`CreateConfig`（补默认值）、`UpdateConfig`（合并后整体校验） |
| AppServiceStore | `networking.ServiceMeta` | `GroupByAppID`；`SvcUpdateData.ToBSON` |
| RuntimeImageStore | `runtime.Meta` | 校验与名称 trim 进 BeforeSave；`UpdateDescription` 先校验描述再更新 |
| AppDefaultRuleStore | `appdefaults.RuleMeta` | 更新用 `Replace`；删除用 `GetAndDelete` |
| DepSvcStore | `depservice/model.ServiceMeta` | plan config 的 JSON 编解码进 Encode / Decode |
| DepSvcBindingStore | `depservice/model.ServiceBindingMeta` | `SyncInstanceIDs` 进 BeforeSave；`UpdateBinding`；`ByBindingQuery` |
| PromotionStore | `promotion.Meta` | `Promote`（按 appID + repoKey + tag upsert）、`ListPromotedTags` |
| TempAdminRecordStore | `platmgt/workspace/admin.Meta` | 「整体更新但排除不可变字段」改为对可变字段 `UpdateFields` |
| PlatAdminStore | `platmgt/admin.RoleBindingMeta` | `CreateManyIfAbsent`；默认更新人进 BeforeSave |
| PortForwardWhitelistStore | `portforward.Meta` | `CreateManyIfAbsent`；文档 `_id` 即 envID |
| FeatureEnvCounterStore | — | 改用 `store.Counter`（`envmodel.NewFeatureEnvCounter(db)`） |
| BuildConfigStore | `build.ConfigMeta` | 镜像凭证密码加解密；`UpdateConfig` 全量替换可变字段 |
| AppConfigFileDefStore | `appcfg.AppConfigFileDefMeta` | `ListDefsByApp`、`UpdateDef`；`DefFilterConfigKind` 改为具名 Opt，调用写法不变 |

按决策 7，原先接收上述 Store 的构造函数与领域函数改为接收 `db *mongo.Database`（参数位置不变），主要有：
appmodel 部署的 `ServiceDeps`（新增 `DB` 字段）、`workload.NewBuilderService`、`appmodeldeploy.NewDeployer`、`snapshot.NewService`、
`tagdeletion.NewService`、`deploy.NewPreDeployChecker` / `NewDeployPreChecker`、`overview.NewService`、`depservice.New`、
`polaris.NewPolarisPlatformManager`、`depenvvars.NewReader`、`initdata.Do`、`bscpcfg/service.NewManager`、
`trpc` / `taf` 的 `NewService`、appcfg 的 `NewAppConfigFileService` / `NewBaseAppCfgFileService` / `NewMountableFileProvider` /
`NewAppConfigFileEditor` / `NewBaseContentProvider`、`appcfghooks.RegisterDeleteHooks`、`appdefaultshooks.RegisterPreDeleteHooks`、
`workspace.GetWorkspaceImageRegistry` / `CreateExternalImageRegistry`、`build.ResolveImageRepoInfo`、`secret.NewImagePullSecretSyncer`、
`customruntime.NewExistenceChecker` / `NewPersistManager` / `NewTagQueryManager`、`postrenderer.Build`、`wlbscpcfg.InjectFromStore` /
`BuildFromStore`、`clusteraddon.ReloadBuiltinClusterAddons`，以及平台管理员、port-forward 白名单、临时管理员、仪表盘、
GPA、特性环境等 Service。原先在函数内部用全局 `database.Client()` 构造 Store 的（如 `GetWorkspaceImageRegistry`、
helm 部署的 values 渲染、post renderer）也随之改为使用传入的 db。

行为变化（调用方需要知道的）：

1. **唯一键冲突**统一返回 `store.ErrDuplicate`，原先只返回自拼文案的模型新增了业务哨兵：
   `networking.ErrServiceAlreadyExists`、`appcfg.ErrAppConfigFileDefAlreadyExists`（depservice 服务、构建配置只匹配 `store.ErrDuplicate`）。
2. **NotFound** 新增业务哨兵：`build.ErrConfigNotFound`、`appcfg.ErrAppConfigFileDefNotFound`、
   `depsvcmodel.ErrServiceNotFound` / `ErrBindingNotFound`（`AsNotFoundError` 可识别，决策 11）。原先返回 `errors.Errorf` 文案的不再依赖字符串。
3. **未命中由静默改为 NotFound**：networking 的 Service 更新（原先返回字符串错误）、`appcfg.UpdateDef`（原先返回 `ModifiedCount` 0）；
   networking 更新内容为空时直接返回，不再刷新 `updatedAt`。
4. **加密只作用于落库副本**：构建配置的 Create / Update 不再把调用方对象中的镜像密码改成密文（「顺带发现的问题」第 2 条已全部修复）。
5. **depservice 服务的 `List` 也会解码 plan config**（「顺带发现的问题」第 1 条已修复）；`Upsert` 不再跳过空的 displayName / description / category，
   内置初始化数据字段齐全，无影响。
6. **`UpdateFields` 对 omitempty 的空值 `$unset`**：构建配置清空 codeRepo / image / pipeline、配置文件 def 清空 mountDir 时，
   原先写入 null / 空串，现在删除字段，读出结果一致；mountDir 的条件唯一索引对两种形态都不生效。
7. **depservice.ServiceManager 总是检查绑定引用**：原先北极星路径传入 nil binding store 以跳过检查，现在删除北极星实例时也会查询绑定；
   北极星实例不会被应用绑定，结果不变，只多一次查询。
8. **运行时镜像的查询不再预校验入参**：`GetByID` 不再对非法 hex 提前返回，自然未命中返回 NotFound；按类型 + 名称查询不再校验 type / name，
   新建时由 BeforeSave 校验。

测试改造：移除了对这些 Store 的 mock 与内存实现，改为写入测试库：`bkrepo/manager_test`（镜像源）、`workspace/facade_test`
（原先 mock 工作空间与镜像源 Store 的构造函数与方法）、`deploy/helm/deploy_test`（构建配置）、`deploy/pre_checker_test`（晋级记录）、
`runtime/validator_test`、depservice `envvars/reader_test` 的内存 binding store（为此给该套件补上了测试库初始化）。
dbfactory 的 `TrpcApplicationStores` / `TafApplicationStores` 不再需要 `BuildConfigStore` / `AppConfigFileDefStore` 字段。

遗留：

* ~~`workspace.GetWorkspaceImageRegistry` 内部仍用传入的 db 构造 `WorkspaceStore`（C 类）~~（阶段 5e 已收敛为 `store.Of(db, workspace.Meta)`）。
* depservice `envvars/reader_test` 中 ServiceInstance 的内存实现仍保留（C 类，不在本次范围）。
* ~~`bkiam/role.RoleStore`、`appcfg.AppValuesConfigStore` 未精读，未纳入本次迁移~~（阶段 5a 已迁移 / 删除）。

## 阶段 5 计划

### 重新分组

精读 B 类与阶段 1 未精读的 Store 后，按迁移实际需要的能力重新分组：

| 分组 | Store | 要点 |
|---|---|---|
| 只需 v1 | `customruntime.Store` | 原实现用 FindOneAndUpdate 返回文档，但唯一调用方 `persist.upsertRecord` 不使用返回值；「命中时只刷新 updatedAt」用 `Upsert` + `UpdateSelected("updatedAt")` 表达，模型函数先设置 `UpdatedAt`（显式选中时取对象中的值）；`Validate` 与 trim 进 BeforeSave |
| 只需 v1 | `alertstrategy.Store` | 查询均为 Where / `$or` / `$elemMatch` + 排序；`Create` 返回的 ObjectID 改由 `obj.ID` 回填；校验与 nil 切片默认值进 BeforeSave；`Delete` 用 `DeleteIfExists` |
| 只需 v1 | `topology.ResourceSnapshotStore` | `UpsertWithVersion` 改为一次带 `dataVersion $lte` 条件的 `Upsert`，唯一索引冲突即版本冲突；`Get` 有 5 处调用方依赖「未找到返回 `nil, nil`」，由具名函数 `FindSnapshot` 保持该语义；`UpdateStatus`、`Delete` 未命中保持静默 |
| 只需 v1 | `bkiam/role.RoleStore` | 两个集合各一个 Meta；`Get` 未命中由包装后的 `ErrNoDocuments` 改为 `store.ErrNotFound`（无调用方依赖）；`bkiam/service_test` 的 `fakeRoleStore` 改为测试库 |
| 删除 | `appcfg.AppValuesConfigStore` | 只有接口、没有实现与调用方 |
| 需要分页 | `build/image.RecordStore`、`build/chart.RecordStore` | 计数器集合结构即 `{_id: appID, seq}`，直接用 `store.Counter`；chart 记录没有时间戳字段，`StartedAt` 在具名函数中赋值 |
| 需要分页 + 投影 | `appmodeldeploy.ResourceSnapshotStore` | 元数据列表排除 `manifest`（单条可达 5MB）；非法 hex 视为未找到的约定保留在具名函数中 |
| 需要分页 | `appcfg.AppConfigFileVersionStore` | 接口注入面广（29 个文件）；`page <= 0` 时不分页的分支留在具名函数中；`GetByFileAndVersion` 新增哨兵 |
| 需要分页 | `workspace.WorkspaceStore` | 引用最多；`CountByState` 聚合用 `Collection()`；`Delete` 内的 `runPreDeleteHooks` 移到具名函数；`Update` 未命中改为 NotFound |
| 需要 `$inc` | `component.ComponentDefStore` | 除 `UpdateInstanceCount` 外均为 v1：Upsert 用 `UpdateOmit` 排除两个计数字段，nil 的 patchers / specs 在 Encode 中转为 `[]`；关键字改用 `ContainsIgnoreCase`（原实现未转义正则元字符） |
| 需要 `$inc` | `semver.CounterStore` | 改为模型包具名函数：`Next`（`$inc` + `$set` + FindOneAndUpdate）用 `Collection()`，`Get` 用 GenericStore，从 Registry 移除 |

`build/trigger` 的两个 Store 仍属 W1 契约，继续暂缓。

### v1.1 能力

1. **分页 `ListPage`**：分页参数是位置参数，结果是结构体，取代曾考虑的 `Skip` / `Page` 选项 + `ListAndCount`
   （后者统计总数时静默忽略分页选项，两者的关系从签名上看不出来）。现有 10 处 `SetSkip` 均按 `(page - 1) * pageSize` 计算，不需要原始偏移量。

   ```go
   type Page[T any] struct {
       Items   []T   // 无结果时为空切片
       Total   int64 // 匹配的总数；传入 WithoutTotal 时不统计，恒为 0
       HasNext bool  // 是否还有下一页，两种模式下均有效
   }

   // page 从 1 开始、size >= 1，否则返回错误。支持的选项：CondOpt、SortOpt、ProjectOpt、PageOpt（不接受 LimitOpt）
   func (s *GenericStore[T]) ListPage(ctx context.Context, page, size int64, opts ...Opt) (*Page[T], error)

   func WithoutTotal() PageOpt // 仅 ListPage 支持
   ```

   * 默认先 `Count` 再 `Find`，总数为 0 时跳过 `Find`，`HasNext = page*size < Total`：现有分页调用点全部需要返回总数。
   * `WithoutTotal()` 不执行 `Count`，多取一条（limit = size + 1）判断 `HasNext`。`Total` 保持 `int64` 而不是指针，
     避免所有统计总数的调用方解引用，代价是此时 0 不代表没有数据，需看注释。
   * 不自动追加 `_id` 作为二级排序：现有索引多为 `appID + createdAt` 形态，追加后可能无法走索引排序；
     由调用方保证排序最终落在唯一字段上（如 workspace 的 `id`、version 的 `_id`）。
2. **投影 `store.OmitFields(fields...)`**：新增 `ProjectOpt`，Get / List / ListPage 支持；只提供排除不提供包含，
   被排除的字段为零值，Decode 钩子照常执行（钩子须能处理被排除的字段）。目前只有 appmodel 资源快照使用。
   命名上不用 `Exclude`，避免被理解为过滤条件（「排除某些文档」）；`Omit` 与写操作的 `UpdateOmit` 呼应，都表示「不碰这些字段」。
   字段按模型校验，拼错直接报错：否则会静默地读回本想跳过的大字段（如 5MB 的 manifest）。
3. **`$inc` 不进 GenericStore**：两处写法差异大，在模型包内基于 `Collection()` 实现具名函数。
4. **FindOneAndUpdate 从 v1.1 中删除**：唯一候选 customruntime 并不需要返回的文档。

### 分批

| 批次 | 内容 | 状态 |
|---|---|---|
| 5a | 只需 v1 的 4 个 Store；删除 `AppValuesConfigStore` | 已完成，见「5a 结果」 |
| 5b | 实现 `ListPage`、`OmitFields`；迁移两个构建记录 Store 与 appmodel 资源快照 | 已完成，见「5b 结果」 |
| 5c | `ComponentDefStore`（含组件引用计数钩子装配）、semver | 已完成，见「5c 结果」 |
| 5d | `AppConfigFileVersionStore` | 已完成，见「5d 结果」 |
| 5e | `WorkspaceStore`，并收敛阶段 4 遗留的 `GetWorkspaceImageRegistry` | 已完成，见「5e 结果」 |

每批沿用阶段 4 的做法：删除 Registry 字段、原接口与 `*StoreMongo`，构造函数改为接收 db，测试改为针对测试库并移除 fake / stub，
完成后把结果与行为变化写入本节。全部完成后 Registry 剩 17 个 C 类 Store（已核实）。

### 5a 结果

Registry 中的 `CustomRuntimeImageStore`、`AlertStrategyStore`、`ResourceSnapshotStore` 已删除（role 不在 Registry 中），
原接口、`*StoreMongo` 与只提供 Store 的 FxModule（customruntime）一并删除；`appcfg.AppValuesConfigStore` 及其模型直接删除。

| 原 Store | Meta | 要点 |
|---|---|---|
| `customruntime.Store` | `customruntime.Meta` | `Register` 按 workspaceID + type + name 做 `Upsert` + `UpdateSelected("updatedAt")`；`Validate` 与 trim 进 BeforeSave；具名 Opt `ByWorkspace` / `ByType` / `ByKey` / `NameContains` / `InListOrder` |
| `alertstrategy.Store` | `strategy.Meta` | `Duplicate` 设为 `ErrDisplayNameAlreadyExists`（集合唯一的唯一索引就是策略名称）；校验与 nil 切片默认值进 BeforeSave；具名 Opt `ByApp` / `ByDisplayName` / `ByRemoteEnv` / `EnabledMatchingEnv` / `NewestFirst` |
| `topology.ResourceSnapshotStore` | `topology.ResourceSnapshotMeta` | `FindSnapshot`（未找到返回 `nil, nil`）、`SaveWithVersion`、`UpdateRefreshStatus`、`ByScope` |
| `bkiam/role.RoleStore` | `role.RoleMeta`、`role.GradeManagerMeta` | 模型没有 `_id` 字段；`RoleQueryParams` 改为具名 Opt `ByWorkspace` / `ByGradeManager` / `ByScope` / `ByRoleID` / `ByUserGroupIDs` |

按决策 7 改为接收 `db *mongo.Database` 的有：`customruntime.NewPersistManager` / `NewTagQueryManager`、`topology.NewService` /
`NewRefresher`、`strategy.NewService`（不再接收 snapshot store，由 db 构造）、`strategyhooks.RegisterUpdateHooks`、
`bkiam.NewIAMService`（`ListRoles` 等仍返回 `[]*role.Role`）。部署 handler 删除拓扑快照改为 `DeleteIfExists(topology.ByScope(...))`；
helm / appmodel 部署轮询任务中触发拓扑刷新改用 `storereg.G().DB`，不再自行构造 Store。

行为变化（调用方需要知道的）：

1. **拓扑快照的乐观锁改为一次原子 upsert**：过滤条件带 `dataVersion <= expected`，版本已被推进时匹配不到文档，MongoDB 转而插入，
   因作用域唯一索引冲突而失败，转为 `ErrVersionConflict`；回写读出的旧快照（带 `_id`）同理。原实现分两步（先确保文档存在、再带版本更新），
   同一作用域首次刷新并发时两者都能通过版本检查、后写覆盖先写；现在后到者得到冲突并被 refresher 丢弃，两次结果的版本号相同，无实际影响。
2. **告警策略名称冲突**：`Create` / `Update` 的唯一键冲突返回包装后的错误，`errors.Is(err, ErrDisplayNameAlreadyExists)` 仍成立
   （handler 本就用 `errors.Is`），不再是裸哨兵。`Update` 未命中改为 NotFound（原实现同样返回 `ErrNotFound`）。
3. **告警策略的应用范围校验放进查询条件**：handler 的 `getAlertStrategyInApp` 用 `ByID` + `ByApp` 查询，跨应用的策略与不存在的策略一样返回 `ErrNotFound`。
4. **默认告警策略初始化失败不再在内部打日志**：改为带模板 code 包装后返回，由调用方记录一次。
5. **customruntime**：`Register` 命中已有记录时不再把 ID / createdAt 回填到调用方对象（无调用方使用）；`Exists` / `List` 不再拒绝空 workspaceID，
   自然匹配不到；查询不再预校验 type（handler 已用 `oneof=builder runner` 绑定）。
6. **role**：`Get` 未命中返回 `store.ErrNotFound`；`ByUserGroupIDs` 把 nil 切片转为空数组，避免 `$in: null` 报错。

测试改造：`bkiam/service_test` 的 `fakeRoleStore` 删除，改为测试库（为此给 bkiam 套件补上了测试库初始化）；告警策略 `service_test`
中包装 Store 让 `Create` 失败的 `failingCreateStore` 删除，改为在远端同步回调中插入同名策略，模拟并发创建导致本地持久化失败；
告警策略测试原先清理的集合名 `resource_snapshots` 有误，改为 `topology.ResourceSnapshotMeta.Collection`。

### 5b 结果

`pkg/store` 新增 `ListPage` / `Page[T]`、`WithoutTotal`（`PageOpt`）与 `OmitFields`（`ProjectOpt`），设计见「v1.1 能力」；
`ListPage` 额外拒绝 `page * size` 溢出的参数。Registry 中的 `BuildRecordStore`、`HelmChartBuildRecordStore`、
`AppModelDeployResourceSnapshotStore` 已删除；semver 因 `NewChartBuildService` 依赖它，提前在本批一并迁移（见 5c）。

| 原 Store | Meta | 要点 |
|---|---|---|
| `build/image.RecordStore` | `build.RecordMeta` | `CreateRecord(ctx, db, record)` 用 `store.Counter`（集合 `build_counters`，文档结构不变）分配序号后创建；`UpdateRecord` 用 `UpdateSelected("num", "status", "extras", "endedAt")`；具名 Opt `ByApp`（与构建配置共用）/ `ByBuild` / `RecordKeywordMatches` |
| `build/chart.RecordStore` | `chart.RecordMeta` | 同上，计数器集合 `helm_chart_build_counters`；模型没有时间戳字段，`CreateRecord` 设置 `StartedAt`；具名 Opt `ByApp` / `ByBuild` / `KeywordMatches` / `NewestFirst` |
| `appmodeldeploy.ResourceSnapshotStore` | `appmodeldeploy.ResourceSnapshotMeta` | `ListSnapshotMeta` 用 `ListPage` + `OmitFields("manifest")`；`GetSnapshot` 保留「非法 hex 视为未找到」；写入用 `CreateMany`，具名 Opt `ByDeployRecord` / `InKindNameOrder` |

按决策 7 改为接收 db 的有：`build/build.NewService`（去掉 record store 参数）、`chart.NewChartBuildService(db)`（原先 3 个参数）、
`buildpoll` / `chartbuildpoll` 的 `newPoller`。

行为变化（调用方需要知道的）：

1. **构建记录唯一键冲突**返回 `store.ErrDuplicate`，不再是自拼的 "already exists" 文案；未命中照常匹配 `ErrRecordNotFound`。
2. **列表关键字去首尾空白**：原实现已转义正则元字符，现在额外 trim，空白关键字不再过滤。
3. **资源快照分页追加 `_id` 作为末级排序**：原实现只按 kind + name 排序，同名资源跨页时顺序不稳定；非法 deployID 返回空页
   （原先返回 `nil, 0, nil`，handler 的输出一致）。
4. 无调用方的 `ListByDeployRecord` 与 `ErrResourceSnapshotNotFound` 已删除。

### 5c 结果

Registry 中的 `ComponentDefStore`、`HelmChartSemverCounterStore` 已删除，`component.FxModule`（只提供 Store）一并删除。

| 原 Store | Meta / 函数 | 要点 |
|---|---|---|
| `component.ComponentDefStore` | `component.ComponentDefMeta` | 校验进 BeforeSave，nil 的 patchers / specs 在 Encode 中转为 `[]`；`UpsertDef` 用 `UpdateOmit` 保留两个实例计数；`UpdateInstanceCount` 基于 `Collection()` 执行 `$inc`；`ListOptions` 改为具名 Opt `ByNameVersion` / `ExcludeInvisible` / `UsableInWorkspace` / `ManagedByWorkspace` / `KeywordMatches`，serializer 的 `ToModel` 改为 `ToListOpt` |
| `semver.CounterStore` | `semver.CounterMeta`、`semver.Next` / `semver.Get` | `Next` 的「按段 `$inc` 并归零其余段」基于 `Collection()` 的 FindOneAndUpdate；`Get` 未命中仍返回零值 counter |

按决策 7 改为接收 db 的有：`workspace.NewComponentRefCountHooks(db)`、`appmodel.NewComponentRefCountHooks(db)`、
`component.NewAppPropertiesBuilder(db, envSvc)`、`NewAppComponentApplier(db, envSvc)`、appspec 的 `migrate.New`（去掉 component def store）；
`LoadBuiltinFromFolder` 与 dbfactory 的 `CompDef` 改为接收 `*component.ComponentDefStore`。

行为变化（调用方需要知道的）：

1. **关键字按字面匹配**：原实现把 keyword 直接作为 `$regex`，未转义元字符；现在用 `ContainsIgnoreCase`，并去首尾空白。
2. **重复创建**返回包装后的错误，`errors.Is(err, ErrComponentDefAlreadyExists)` 仍成立，不再是裸哨兵；未命中同理匹配 `ErrComponentDefNotFound`。
3. `List` 返回 `[]ComponentDef`（原先 `[]*ComponentDef`）；删除改用 `DeleteIfExists`，handler 原本就不使用删除数量。

### 5d 结果

Registry 中的 `AppConfigFileVersionStore` 已删除，`appcfg.FxModule` 不再提供它；`trpc` / `taf` 的 `NewService`、
appmodel 部署的 `ServiceDeps`、`buildpoll`、dbfactory 的 `TrpcApplicationStores` 等不再需要该 Store。

| 原 Store | Meta | 要点 |
|---|---|---|
| `appcfg.AppConfigFileVersionStore` | `appcfg.AppConfigFileVersionMeta` | 具名 Opt `VersionsByApp` / `VersionsByFile` / `VersionNotDeleted` / `VersionsNewestFirst`（createdAt、_id 倒序）；`BaseAppCfgFileService.ListVersions` 用 `ListPage` 并固定追加应用范围、未删除与排序条件；软删除用 `Update(map)`；按文件 / 应用清理用 `DeleteMany` |

构造函数变化：`NewBaseAppCfgFileService(db, fileStore)`、`NewAppConfigFileService(fileStore, db)`、`NewMountableFileProvider(fileStore, db)`、
`appcfghooks.RegisterDeleteHooks(appStore, fileStore, db)`；serializer 的 `ToVersionListOptions` 改为 `ToVersionListOpt() (store.Opt, error)`
（文件 ID 非法 hex 时返回错误）。删除应用时由 handler 直接 `DeleteMany(appcfg.VersionsByApp(appID))`。

与计划的出入：

* `GetByFileAndVersion` 没有调用方，直接删除，未新增哨兵。
* `page <= 0` 时不分页的分支只有测试使用，删除；handler 的分页参数本就要求 `page >= 1`。

行为变化（调用方需要知道的）：

1. **描述关键字按字面匹配并去首尾空白**。
2. **软删除时版本已不存在**返回 `ErrAppCfgFileVersionNotFound`（原先静默成功）；`DeleteVersion` 会先读出目标版本，只有并发物理删除时才会遇到。
3. **唯一键冲突**新增业务哨兵 `ErrAppCfgFileVersionAlreadyExists`。
4. 只有 `ListVersions` 排除已软删除的版本；`GetVersionByAppAndID` 与 `CompareVersions` 仍能读到它们（与原实现一致）。

### 5e 结果

Registry 中的 `WorkspaceStore` 已删除，原接口、`WorkspaceStoreMongo`、`ListOptions` / `ListPageOptions` 一并删除，
`workspace.FxModule` 只剩 `WorkspaceCompsStore`。

| 原 Store | Meta | 要点 |
|---|---|---|
| `workspace.WorkspaceStore` | `workspace.Meta` | 模型没有 `_id`，按 ID 查询须用 `ByWorkspaceID`（不能用 `GetByID`）；具名 Opt `InState` / `KeywordMatches` / `DefaultOrder`（updatedAt 倒序、id 正序）；具名函数 `Update`（`UpdateSelected` 可变字段）、`Delete`（先执行 pre-delete 钩子再 `DeleteIfExists`）、`CountByState` |

调用方统一写作 `store.Of(db, workspace.Meta).Get(ctx, workspace.ByWorkspaceID(id))`，不再定义 `workspace.Get`（决策 7）。
按决策 7 改为接收 db 的有：`workload.NewBuilderService`（去掉 workspace store 参数）、`strategyhooks.RegisterUpdateHooks(db, envStore, appStore)`、
`platmgt/workspace.NewService(db, appStore, envStore)`（其 FxModule 为此引入 `database.PrivateFxModule`）、
`platmgt/workspace/admin.NewService(db, permMgr)`；`ListSortByOpTime` 与 dbfactory 的 `Workspace` 改为接收 `*workspace.WorkspaceStore`。
平台管理的工作空间列表用 `ListPage`，筛选条件由 `WorkspaceListOptions.ToWorkspaceStoreListOpt()` / `ToWorkspaceStoreStatisticsOpt()` 生成。

与计划的出入：

* `CountByState` 没有改用 `Collection()` 聚合，而是对三个已知状态各执行一次 `Count`：工作空间数量很少，代码更直观；
  结果固定包含这三个状态（数量可为 0），库中的其它状态值不再出现（原聚合只返回出现过的状态值，调用方只读取这三个）。

行为变化（调用方需要知道的）：

1. **`Update` 未命中**返回 `ErrWorkspaceNotFound`（原先静默成功）；`Get` 未命中返回包装后的错误，调用方均已使用 `errors.Is`。
2. **`Create` 时 `updatedAt` 与 `createdAt` 相同**：原实现中的兼容性 FIXME 随旧代码删除。
3. **关键字去首尾空白**（原实现已转义正则元字符）。
4. `Delete` 行为不变：工作空间不存在时静默成功，pre-delete 钩子失败时不删除。

测试改造：`workspace/store_test` 按具名 Opt 与具名函数重写，并补充 `Update` 未命中、删除钩子顺序与失败、分页排序的用例；
其余约 10 个测试文件改用 `testutil.NewStore(GinkgoT(), workspace.Meta)`，不再经 fx 注入工作空间 Store。

## 扫描中顺带发现的问题

迁移时应一并修复，或单独提交：

1. ~~`depservice/model/service_store.go` 的 `List` 未调用 `serviceFromDBValue`~~（阶段 4 已修复：plan config 编解码改为 Decode 钩子）。
2. ~~加密在调用方对象上就地进行~~（bkrepo 于阶段 3、build/image 于阶段 4 修复）。
3. NotFound 语义不一致：appcfg def、构建配置、networking 已在阶段 4 统一为哨兵错误；
   拓扑快照未找到时返回 `nil, nil` 是调用方依赖的语义，阶段 5a 由具名函数 `topology.FindSnapshot` 显式保留。
4. 不少 Store 直接 `return err` 不加 Wrap；`appdefaults` 中 Store 及其调用方的 `fmt.Errorf("%w")` 已在阶段 4 随迁移改为 `errors.Wrap`，
   其余未迁移的 Store 仍需处理。
