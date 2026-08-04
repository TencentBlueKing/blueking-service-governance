# 环境变量导入解析与预览设计说明

本文档主要讨论**解析与预览能力**，并约束 env file 的文本格式约定；不覆盖正式导入执行流程、页面交互细节、应用页按范围导出等后续能力。

## 1. 需求范围

本设计要沉淀的是一套共享的 `.env` 文本解析与预览能力，供下面三个入口复用：

- 公共环境变量页面
- 单环境变量页面
- 应用环境变量页面

- 解析 `.env` 文本中的 `KEY=VALUE`
- 解析固定注释行语法 `# desc:`
- 在公共环境变量导入场景下解析 `# scopeType:`、`# scopeValue:`
- 结合页面上下文计算逐条预览结果
- 输出新增/覆盖等逐条结果与汇总信息
- 保证预览语义可以作为后续正式导入的前置基础

## 2. 当前实现与社区约定的关系

当前实现并不是完整支持某个 dotenv 形态，而是实现了一套**面向 BKMS 预览需求的 `.env` 子集 + 元数据扩展**。

### 2.1 与社区约定一致的部分

- 使用 `KEY=VALUE` 作为主体格式
- 支持空行
- 支持完整注释行
- key 命名规则与社区常见约定一致：`[A-Za-z_][A-Za-z0-9_]*`

### 2.2 当前额外扩展的部分

为了服务 BKMS 的预览需求，当前实现引入了三类完整注释行元数据：

- `# desc: <描述>`
- `# scopeType: <workspace|envType>`
- `# scopeValue: <作用域值>`

这不是通用 dotenv 规范的一部分，而是 BKMS 自己在 env file 边界上增加的约定，用来承载描述与目标作用域信息。其中：

- `# desc:` 可用于三类页面
- `# scopeType:` / `# scopeValue:` 只用于公共环境变量页面

### 2.3 当前没有支持的部分

当前 parser 参考社区成熟 `.env` parser 的行级解析语义，已支持：

- quoted value（单引号 / 双引号）
- inline comment

但仍然**不支持**下面这些能力：

- `export KEY=value`
- multiline value
- 变量插值

## 3. 当前文件格式设计

本期解析器支持的文本格式如下：

```dotenv
# desc: 数据库地址
# scopeType: envType
# scopeValue: production
DB_LABEL="primary # main"
```

其规则为：

- 主体记录必须是 `KEY=VALUE`
- 赋值行支持 quoted value 和 inline comment
- 引号内的 `#` 仍然属于 value 本身，不视为注释起始符
- 仍不支持 `export KEY=value`
- `# desc:`、`# scopeType:`、`# scopeValue:` 都是完整注释行
- 元数据只作用于紧随其后的下一条变量记录
- `# desc:` 可用于所有导入页面
- `# scopeType:` / `# scopeValue:` 仅用于公共环境变量页面
- 普通注释行和空行忽略
- 若注释行匹配 `# field: value` 形式，但字段名不是 `desc` / `scopeType` / `scopeValue`，则直接报错
- 若声明了 `scopeValue` 但未声明 `scopeType`，则直接报错

## 4. 三类页面上下文下的预览语义

虽然三类页面都复用同一个 parser，但 preview 阶段对 `scopeType` / `scopeValue` 的使用方式不同，`overwrite` 的判断口径保持一致。

### 4.1 公共环境变量页面

公共环境变量页面要求**显式**声明 scope 元数据，不支持省略。

合法形式只有两种：

```dotenv
# scopeType: workspace
KEY=value
```

```dotenv
# scopeType: envType
# scopeValue: development
KEY=value
```

规则如下：

- `scopeType=workspace`：`scopeValue` 必须省略
- `scopeType=envType`：`scopeValue` 必须存在，且取值只能是 `development` / `test` / `production`
- 未声明 `scopeType`：直接报错

### 4.2 单环境变量页面

单环境页面改为和应用环境变量页面一致，由**当前页面上下文**决定目标导入范围，文件中不再声明目标 env scope：

```dotenv
KEY=value
```

规则如下：

- 不要求声明 `scopeType`
- 不要求声明 `scopeValue`
- 导入目标环境由当前页面的环境上下文唯一确定
- 若文件中出现 `# scopeType:` 或 `# scopeValue:`，直接报错

### 4.3 应用环境变量页面

应用环境变量页面不使用 scoped env var 模型，因此文件中**不允许出现任何 scope 元数据**：

- 不允许 `# scopeType:`
- 不允许 `# scopeValue:`

若出现上述字段，直接报错。

### 4.4 三类页面的职责分界

三类页面在 env file 上的职责划分调整为：

- 公共环境变量页面：文件内显式声明 scope，文件本身携带目标范围
- 单环境变量页面：目标范围完全由页面上下文决定，文件只描述 key/value/desc
- 应用环境变量页面：目标范围完全由页面上下文决定，文件只描述 key/value/desc

## 5. 预览中的覆盖判断

子需求要求输出新增/覆盖等逐 key 处理结果。
这里的 `overwrite` 不是和"最终所有生效环境变量集合"做比较，而是和**目标写入范围内直接存在的数据**比较。

这样做的目的是避免把继承值误判成覆盖。例如某个 key 当前只是从 workspace 公共变量继承到 env 页面，若用户在 env 页面导入同名 key，这次动作本质上是在 env scope 新建 override，预览更合理的结果应当是 `new`，不是 `overwrite`。

## 6. 校验规则

当前 parser 采用 fail-fast 模式，遇到第一个错误立即返回。当前规则包括：

- key 必须匹配 `[A-Za-z_][A-Za-z0-9_]*`
- key 长度不超过 `256`
- value 长度不超过 `8192`
- 同一份导入内容不允许重复 key
- 不支持未知元数据字段
- `# desc:` / `# scopeType:` / `# scopeValue:` 不能悬空出现在文件结尾
- `scopeValue` 不能脱离 `scopeType` 单独出现
- 公共环境变量页面：要求显式声明合法 scope 元数据
- 单环境变量页面：不要求 scope 元数据，若出现则报错
- 应用环境变量页面：不要求 scope 元数据，若出现则报错

其中 key 的正则和 key/value 的长度约束已收敛到共享定义中，目的是保证：

- 单条 CRUD 校验
- 批量解析预览
- 后续正式导入

遵循同一套基础约束。

## 7. 预览返回结构

子需求要求"输出逐 key 处理结果 + 输出总数/新增/覆盖/忽略/报错等汇总统计"，因此当前 preview 结果主要由两部分组成：

- `items`：逐条记录的预览结果
- `summary`：汇总统计

其中部分字段使用 `omitempty`，是因为它们只在特定语义下有意义，例如：

- `originalValue`：仅覆盖场景返回
- `declaredScopeType` / `declaredScopeValue`：仅输入文本显式声明了 scope 元数据时返回
- `messages`：仅需要补充提示时返回

这里的重点不是压缩响应体大小，而是让预览结果尽量只暴露"当前条目真正有意义的信息"。
