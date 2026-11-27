# `useTableSettings` 表格列设置 Hook

> 源码：`src/composables/use-table-settings.ts`

## 这个 Hook 做什么

表格右上角有"列设置"，用户可以自己勾选显示哪些列、调整行高。这个 Hook 就是管这件事的：

1. 用户调完的设置会**存下来**，下次打开页面自动恢复；
2. 告诉表格"当前该显示哪些列"（`settings`），并在用户改动时去保存（`handleSettingChange`）。

## 数据怎么存

**不存完整勾选，只存"和默认不一样的地方"**，避免存一堆重复数据。

比如默认勾选 A、B、C 三列，用户把 B 取消、勾上了 D，那就只记：

- `hidden: ['B']` —— 默认勾选的，但被用户取消的
- `shown: ['D']` —— 默认没勾的，但被用户勾上的
- `size` —— 用户选的行高

最后显示哪些列 = 默认勾选 − hidden + shown + 不可操作列（disabled），去重后得出。

存的位置是 `uiStore.preferences`，key 是 `table:${tableId}`（比如 `table:deploy-overview`），刷新页面不会丢。

## 几个设计细节

- **没改动就不留记录**：用户什么也没调，这条记录会被自动清掉，不存垃圾（记录里含不可操作列的差异时例外，那部分会保留）。
- **格式不对就按默认来**：如果本地存的数据结构不合法，直接当作没改过，显示默认列。
- **行高 `small` 不存**：`small` 就是表格默认行高，等于没个性化。否则用户只是点开一下设置面板、啥都没动，也会被存进去，导致"没改动就清除"失效。
- **不可操作列（disabled）**：既包括"必须展示、用户不许隐藏"的列（如"环境"），也包括"当前环境下根本不渲染、用户看不到也操作不了"的列（如未开启开发模式时的热更新列）。它们会被强制并入显示列表、不写入差异；记录里已存在的这些列的差异会原样保留，避免在别的上下文保存时被算掉。

## 怎么用

```ts
const { settings, handleSettingChange } = useTableSettings(tableId, {
  defaultChecked: ['id', 'image', 'ip'], // 默认勾选哪些列（可传 ref/computed）
  disabled: ['id'],                       // 用户在当前上下文无法操作的列（可选，可传 ref/computed）
});

// 列是否渲染随上下文变化时，把当前不渲染的列也一并传进来
// disabled: () => (isDevModeEnabled.value ? ['id'] : ['id', 'latestPublish']),
```

模板里绑给表格：

```vue
<Table
  :settings="settings"
  :show-settings="true"
  @setting-change="handleSettingChange"
/>
```

两个参数：

| 参数 | 说明 |
|------|------|
| `tableId` | 给这张表起个唯一的名字（支持 `ref`/`computed`）。**同一个 id 就是同一份设置**；页面里 v-for 渲染多张表时，每张要传不同的 id，否则设置会互相串。 |
| `defaultChecked` | 默认勾选列，支持 `ref`/`computed`，但**建议保持稳定**。差异始终相对当前基线计算，基线一变，每次保存都会按新基线重算全部差异，会把"只在另一种基线下有意义"的勾选/取消静默算掉。列是否存在这类动态差异，请改用 `disabled` + 渲染条件表达。 |
| `disabled` | 用户在当前上下文无法操作的列：不可隐藏的列，或当前环境不渲染的列。支持 `ref`/`computed`；会强制并入显示列表、不写入差异，记录中已存在的差异原样保留。 |

## 现在谁在用

| 页面 | tableId | 默认勾选 | 不可操作列（disabled） |
|------|---------|----------|----------|
| `application.vue` | `'application-list'`（固定） | 应用名称、应用类型、语言、已部署环境、最近操作人、最近操作时间 6 列 | 应用名称列 |
| `deploy-overview.vue` | `'deploy-overview'`（固定） | 环境、类型、状态等 6 列 | 环境列 |
| `instance-table.vue` | `'instance-table-v2'`（固定） | 基线固定为全部 9 列：实例、状态、就绪、镜像 Tag、热更新、网络、Restart、Age、资源规格 | id 列 + 非开发模式环境下的热更新列 |

`application-list` 是应用列表页唯一一张表，tableId 固定。`instance-table` 曾按环境名拼 tableId，列合并后换用固定新 id `'instance-table-v2'`——所有环境共用一份列设置。
