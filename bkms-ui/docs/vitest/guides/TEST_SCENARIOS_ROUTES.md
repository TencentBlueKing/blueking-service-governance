# 路由级测试评估与场景台账

> 定位：唯一的**完备性枚举层 + 场景执行台账**——以 `src/modules/router.ts` 为基准评估「做/不做」，并维护 S 编号场景卡（含验收摘要）。
>
> 评估方法见 `TEST_GUIDELINE.md`。打法见 `TEST_PLAYBOOK.md`。已知问题见 `TEST_PILOT_LOG.md`。
>
> **状态仅两值**：`✅ 入选` / `❌ 落选`。新场景**零新增 md**：只更新本文件场景卡 + 测试文件头。

## 1. 路由全量清单与评分（权威枚举）

> 枚举来源：`src/modules/router.ts`。评分：频率 / 影响 / 复杂度（各 1~3）。

| 路由 | 名称 | 页面 | 频率 | 影响 | 复杂度 | 总分 | 状态 |
|---|---|---|---|---|---|---|---|
| `:space/app/:envName?` | app | 应用列表页（Application） | 3 | 3 | 2 | **8** | ✅ 入选（S14） |
| `:space/app/create`（+ 默认/trpc/helm/taf/agones 模板子路由） | createApplication 等 | 创建应用向导 | 3 | 3 | 3 | **9** | ✅ 入选（S1） |
| `:space/app/:name` + `:type/:menuName` | appNavigation/detail | 应用详情（动态菜单，子页明细见下） | 3 | 3 | — | — | 逐子页评估 ↓ |
| ↳ `detail/deploy/` | — | 提交部署 | 3 | 3 | 3 | **9** | ✅ 入选（S2） |
| ↳ `detail/app-config/` | — | 配置编辑保存 | 3 | 3 | 3 | **9** | ✅ 入选（S3） |
| ↳ `detail/artifact/` | — | 制品管理 | 2 | 3 | 2 | **7** | ✅ 入选（S11） |
| ↳ `detail/helm-deploy/`（deploy-application/deploy-history/preview-rollback/remove-infoxBox） | — | Helm 部署、历史与预览回滚 | 2 | 3 | 3 | **8** | ✅ 入选（S15） |
| ↳ `detail/app-build/build-management.vue` | — | 构建管理 | 2 | 3 | 2 | **7** | ✅ 入选（S18） |
| ↳ `detail/` 其余（overview/observation/polaris/network-access/alert-record/operation-history/base-info） | — | 查看/监控展示型子页 | 2 | 1 | 1 | 4 | ❌ 落选（只读展示，无判定节点） |
| ↳ `application/components/`（topo、image-select、pipeline-params-form、fields、repo-selector 等构建/创建子件） | — | 创建/构建流程子组件 | — | — | — | — | ❌ 落选（不单独建场景，随 S1/S2/S18 连带覆盖） |
| ↳ `application/components/delete-app-dialog.vue` | — | 删除应用确认弹窗 | 2 | 3 | 1 | 6 | ❌ 落选（不单独建场景，随 S6 连带覆盖） |
| ↳ `application/global-view.vue`、`result.vue` | — | 全局视图 / 创建结果页 | — | — | — | — | ❌ 落选（result 随 S1；global-view 评分 6 落选） |
| `:space/component` | component | 组件市场列表（浏览部分） | 2 | 1 | 1 | **4** | ❌ 落选（浏览型，无判定节点） |
| ↳ `component-management.vue` 弹层 + `marketplace/components/` | — | 组件新建/编辑向导及其子件 | 2 | 3 | 3 | **8** | ✅ 入选（S12） |
| `:space/env` | env | 环境管理列表（浏览 + 创建/删除入口） | 2 | 3 | 2 | **7** | ✅ 入选（S9） |
| ↳ `env/public-env-vars/` | — | 公共环境变量增删改 | 2 | 3 | 2 | **7** | ✅ 入选（S16） |
| ↳ `env/cluster-components/` | — | 集群组件安装与配置 | 1 | 3 | 3 | **7** | ✅ 入选（S17） |
| ↳ `env/` 其余子页 | — | 环境详情/设置/泳道等 | 1~2 | 2 | 2 | 5~6 | ❌ 落选（低频可复评） |
| `:space/env/:envId/health` | clusterHealthDiagnosis | 集群健康诊断 | 1 | 2 | 2 | **5** | ❌ 落选 |
| `:space/env/:envId/port-pool` | clusterPortPool | 端口池管理 | 1 | 2 | 2 | **5** | ❌ 落选 |
| `:space/basic`（app-default-config/deploy-config） | basicItem | 应用默认配置编辑 | 2 | 2 | 2 | **6** | ❌ 落选 |
| `:space/basic`（base-info/operation-history） | basicItem | 基础信息/操作历史 | 2 | 1 | 1 | **4** | ❌ 落选 |
| `:space/plugin/:menuName` | componentList | 插件配置 | 1 | 2 | 2 | **5** | ❌ 落选 |
| `/platform` | platform 等 | 平台管理 | 1 | 2 | 2 | **5** | ❌ 落选 |
| `/home`、`/space-list` | home/spaceList | 首页 / 空间列表 | 1 | 1 | 2 | **4** | ❌ 落选 |
| `/403`、`/:pathMatch(.*)*` | 403/404 | 静态页 | — | — | — | — | ❌ 落选（跳转逻辑入选 S13） |
| — | — | `smartGoBack` | 3 | 2 | 3 | **8** | ✅ 入选（S13） |
| — | — | `beforeEach` 空间权限守卫 | 3 | 3 | 2 | **8** | ✅ 入选（并入 S13） |
| — | — | `src/components/` 各选择器（git/user/env-group 等） | 2 | 2 | 2 | 6 | ❌ 落选（随使用处连带） |
| — | — | `src/components/repo-ref-select/` | 2 | 3 | 3 | **8** | ✅ 入选（S19） |
| — | — | `src/components/` 其余交互组件 | 2 | 2 | 2~3 | 6~7 | ❌ 落选（候选池） |
| — | — | `src/components/` 纯展示组件 | — | — | — | — | ❌ 落选 |

### 覆盖口径

- S1~S19 是评分 ≥7 的第一梯队，不等于全项目覆盖；S 编号是场景族。
- 全项目覆盖 = §1 每个枚举对象均有入选/落选结论。
- 升级：落选对象复评总分 ≥7 可升入选并新增 S 编号。

## 2. 场景台账

### 2.1 索引总表

| 编号 | 场景 | 评分 | 状态 |
|---|---|---|---|
| S1 | 创建应用向导 | 9 | `✅ 入选` |
| S2 | 提交部署 | 9 | `✅ 入选` |
| S3 | 配置编辑保存（多态切换） | 9 | `✅ 入选` |
| S4 | 环境变量可编辑表格 | 8 | `✅ 入选` |
| S5 | 动态 / 重复输入项 | 7 | `✅ 入选` |
| S6 | 删除二次确认 | 7 | `✅ 入选` |
| S7 | 异步异常与空态（横切） | 7 | `✅ 入选`（附带覆盖，不独立验收） |
| S9 | 环境管理（列表 + 创建/删除） | 7 | `✅ 入选` |
| S10 | 应用模板 | — | `❌ 落选`（随 S1） |
| S11 | 制品管理 | 7 | `✅ 入选` |
| S12 | 组件新建/编辑向导 | 8 | `✅ 入选` |
| S13 | 路由智能返回 + 空间权限守卫 | 8 | `✅ 入选` |
| S14 | 应用列表页 | 8 | `✅ 入选` |
| S15 | Helm 部署与预览回滚 | 8 | `✅ 入选` |
| S16 | 公共环境变量管理 | 7 | `✅ 入选` |
| S17 | 集群组件安装与配置 | 7 | `✅ 入选` |
| S18 | 构建管理 | 7 | `✅ 入选` |
| S19 | RepoRefSelect（Input 路径） | 8 | `✅ 入选` |

> S8 编号预留未使用（首版评审即保留空位，避免与既有讨论引用冲突）。场景卡模板：状态/评分/来源/目标/V → 用例 → 验收 → backlog → 打法。

### 2.2 场景卡

#### S1 创建应用向导

- 状态：`✅ 入选` ｜ 评分：9 ｜ 来源：`src/pages/application/create.vue` + 模板子路由
- 用户目标：选模板 → 填信息 → 提交成功
- V（校准）= 8：步骤条配置（tRPC 三步 / Helm 两步计入口差异）/ 搜索空态 / 模板跳转 / 校验拦截 / 步骤前进 / 取消回退 / 创建成功 / 创建失败
- 用例：`test/scenarios/create-application.test.ts`（9 全绿；V=8 + 1 条入口差异 it）
- 验收：连跑 3 稳定；变异 4/4；评审 88/100
- backlog：其余模板向导与子件表单规则
- 打法：PLAYBOOK「全局注册组件打桩」

#### S2 提交部署

- 状态：`✅ 入选` ｜ 评分：9 ｜ 来源：`src/pages/application/detail/deploy/`
- 用户目标：选环境 → 提交 → 结果反馈
- V（当前）= 2（仅入口权限分发；台账原估 4）
- 用例：`test/scenarios/deploy-management.test.ts`（2 全绿）
- 验收：连跑稳定；变异 2/2；评审 80/100
- backlog：P1 部署提交主路径；真实请求噪音待补 mock
- 打法：PLAYBOOK「Proxy service / pinia / vxe」

#### S3 配置编辑保存（多态切换）

- 状态：`✅ 入选` ｜ 评分：9 ｜ 来源：`src/pages/application/detail/app-config/`
- 用户目标：编辑态改配置并保存/取消
- V（校准）= 6：进入编辑 / 取消回滚 / 保存失败 / 默认环境写入 / 普通环境写入 / 恢复默认
- 用例：`test/scenarios/app-config-resources.test.ts`（6 全绿，资源规格为代表）
- 验收：连跑 3 稳定；变异 4/4；评审 87/100
- backlog：其余配置子模块（探针/生命周期/更新策略/元数据/网络访问/程序配置——复核确认与资源规格**非同构**，需按子件分别设计）
- 打法：PLAYBOOK「多态配置子模块」

#### S4 环境变量可编辑表格

- 状态：`✅ 入选` ｜ 评分：8 ｜ 来源：`src/components/editable-variable-table/`
- 用户目标：变量表格增删改
- V（当前）= 2：只读渲染 + 进入编辑态
- 用例：`test/scenarios/editable-variable-table.test.ts`（2 全绿）
- 验收：连跑 3 稳定；变异 2/2
- backlog：P1 新增空行 / 改值保存 / key 重复；PopConfirm 浮层 jsdom 不稳已裁剪
- 打法：PLAYBOOK「vxe 垫片 + 表格 stub」

#### S5 动态 / 重复输入项

- 状态：`✅ 入选` ｜ 评分：7 ｜ 来源：`dynamic-input` / `repeatable-input` / `key-value`
- 用户目标：动态增删输入项
- V（校准）= 5（dynamic-input；含数字/文本域/禁用）
- 用例：`test/scenarios/dynamic-input.test.ts`（5 全绿）；另有 `repeatable-input` / `key-value` 文件
- 验收：变异 3/3；评审 92/100
- backlog：emoji/注入样例；repeatable/key-value 补齐口径
- 打法：PLAYBOOK「Harness / 消极断言」中的 Harness 包装（勿套用「弹窗表单」）；类型分发细节见用例文件头

#### S6 删除二次确认

- 状态：`✅ 入选` ｜ 评分：7 ｜ 来源：`src/components/delete-comfirm.vue`
- 用户目标：删除二次确认
- V（校准）= 4：展示 / 确认 / 取消 / loading 禁取消
- 用例：`test/scenarios/delete-confirm.test.ts`（4 全绿）
- 验收：变异 4/4；评审 93/100
- backlog：业务级删除入口随列表场景连带
- 打法：PLAYBOOK「弹窗表单直接测本体」

#### S7 异步异常与空态（横切）

- 状态：`✅ 入选`（附带覆盖，**不独立验收**）｜ 评分：7 ｜ 来源：随 S1~S3/S14 附带
- 用户目标：加载失败/空数据有清晰反馈
- V ≈ 3：加载中 / 失败 / 空数据
- 用例：无独立文件；随各场景附带（不要求独立连跑/变异）
- 验收：不单独过门；以宿主场景失败/空态分支为准
- backlog：专项横切用例可复评单独立项

#### S9 环境管理（列表 + 创建/删除）

- 状态：`✅ 入选` ｜ 评分：7 ｜ 来源：`env.vue`（列表）+ 删除域 `basic-info.vue` → `components/delete-env-action.vue`
- 用户目标：环境列表管理
- V = 9（列表 2 + 删除 7；2026-09-18 随 #208「去宽窄表」重构校准，删除域入口从列表行移至详情页，新增「获取详情失败提示」「删除中防重」两条可感知路径）
- 用例：`test/scenarios/env-management.test.ts`（9 全绿；删除域以 harness 复刻 `basic-info.vue:457-482` 接线，DeleteEnvAction/Dialog/Warning 真实渲染）
- 验收（重构适配轮）：连跑 3 稳定；变异 4/4 捕获（无部署不弹确认 / 有部署不弹告警 / 不 emit deleted / 移除 deleting 守卫）；tests 段 1.3s
- backlog：P2 新建环境弹窗（CreateEnv，不判 N/A）；P3 搜索筛选 / 排序（写前先校准台账 V）
- 打法：PLAYBOOK「vxe 垫片 + 表格 stub」+ harness 驱动（defineExpose 命令式入口）

#### S10 应用模板

- 状态：`❌ 落选` ｜ 摸底：即 S1 向导组成部分，不单独建场景

#### S11 制品管理

- 状态：`✅ 入选` ｜ 评分：7 ｜ 来源：`detail/artifact/`
- 用户目标：按类型分发制品页
- V = 3：类型分发 2 + 页签切换 1
- 用例：`test/scenarios/artifact-management.test.ts`（3 全绿）
- 验收：连跑 3 稳定；变异 2/2；评审 85/100
- backlog：两个子页上传交互

#### S12 组件新建/编辑向导

- 状态：`✅ 入选` ｜ 评分：8 ｜ 来源：`component-management.vue`
- 用户目标：新建/编辑组件并保存
- V ≈ 15（21 条 it，含 it.each）
- 用例：`test/scenarios/component-management.test.ts`（21 全绿）
- 验收：连跑稳定；变异 4/4；评审 92/100
- backlog：M3 loading 重复提交（推断/需确认）、M4 编辑×脏检查组合态、P3 空态/注入样例
- 打法：PLAYBOOK「Harness / 消极断言」「写 stub 前读契约」

#### S13 路由智能返回 + 空间权限守卫

- 状态：`✅ 入选` ｜ 评分：8 ｜ 来源：`src/modules/router.ts`
- 用户目标：正确返回；无权限正确反馈
- V ≈ 9（smartGoBack + 守卫）
- 用例：`test/scenarios/router-guards.test.ts`（10 全绿）
- 验收：变异 4/4；评审 92/100
- 打法：PLAYBOOK「纯路由逻辑」

#### S14 应用列表页

- 状态：`✅ 入选` ｜ 评分：8 ｜ 来源：`application.vue`
- 用户目标：浏览列表并进入入口
- V = 4
- 用例：`test/scenarios/application-list.test.ts`（4 全绿）
- 验收：连跑稳定；变异 2/2；评审 88/100
- 打法：PLAYBOOK「vxe 垫片 + 表格 stub」

#### S15 Helm 部署与预览回滚

- 状态：`✅ 入选` ｜ 评分：8 ｜ 来源：`detail/helm-deploy/`
- 用户目标：部署 / 历史 / 回滚
- V = 7：更新禁用 / 校验拦截 / 部署主路径 / 预检仍部署 / 预检取消 / 回滚入口 / 回滚确认
- 用例：`test/scenarios/helm-deploy.test.ts`（7 全绿）
- 验收：连跑 3 稳定；变异 4/4；评审 88/100
- backlog：移除部署确认（InfoBox + deleteHelmDeploy）/ 查看 Values 侧滑 / 生产环境仅晋级 Tag 分支 / 脏离开确认（useLeaveConfirm）/ 部署历史搜索与分页
- 打法：PLAYBOOK「stub 与真实并存」「bkui 弹层」

#### S16 公共环境变量管理

- 状态：`✅ 入选` ｜ 评分：7 ｜ 来源：`env/public-env-vars/`
- 用户目标：公共变量增删改
- V（当前）= 4（表单弹窗）
- 用例：`test/scenarios/public-env-var-form.test.ts`（4 全绿）
- 验收：连跑 3 稳定；变异 3/3；评审 91/100
- backlog：列表与删除
- 打法：PLAYBOOK「弹窗表单直接测本体」

#### S17 集群组件安装与配置

- 状态：`✅ 入选` ｜ 评分：7 ｜ 来源：`env/cluster-components/`
- 用户目标：安装组件并维护配置
- V（当前）= 3：空态 / 分组展开 / 安装侧滑
- 用例：`test/scenarios/cluster-components.test.ts`（3 全绿）
- 验收：连跑 3 稳定；变异 3/3；评审 88/100
- backlog：侧滑内表单提交

#### S18 构建管理

- 状态：`✅ 入选` ｜ 评分：7 ｜ 来源：`build-management.vue`
- 用户目标：按镜像来源控制构建入口并弹出执行配置
- V = 3：镜像仓库禁用 / 代码仓库可用 / 点击弹出执行配置
- 用例：`test/scenarios/build-management.test.ts`（3 全绿）
- 验收：连跑 3 稳定；变异 2/2；评审 88/100
- backlog：P2 提交成功/失败；P3 构建历史列表交互；流水线参数配置
- 打法：PLAYBOOK「Proxy service / pinia / vxe」

#### S19 RepoRefSelect（Input 路径）

- 状态：`✅ 入选` ｜ 评分：8 ｜ 来源：`src/components/repo-ref-select/`
- 用户目标：流水线手动输入分支；trim + 防抖确认
- V = 4：trim 同步 / 无变化不确认 / 变化确认 / 空串不确认
- 用例：`test/scenarios/repo-ref-select.test.ts`（4 全绿）
- 验收：连跑 3 稳定；变异 2/2；评审 90/100
- backlog：P2 Select 下拉路径
- 打法：PLAYBOOK「防抖 + VTU 契约」

## 3. 覆盖模型检查

| 维度 | 覆盖状态 | 缺口 |
|---|---|---|
| 正向流程 | 部分 | 各场景 backlog |
| 反向流程 | 完整 | — |
| 边界 | 部分 | 表单字符规则边界待统一校准 |
| 权限 | 部分 | 仅 S13；页面内 apply-perm 未立项 |
| 异常 | 完整 | S7 横切 + 各场景失败分支 |
| 状态 | 部分 | S2 部署状态机待补 |
| 数据 | 部分 | 分页等随列表附带 |
| 恶意输入 | 部分 | S5 等未含 emoji/注入 |
| 业务合理外推 | 部分 | 重复提交等标推断/需确认 |

## 4. 备忘

**外部方案评审（2026-09-11）中未执行的建议**（优先级对齐台账 backlog 体系）：

1. 【P2】Harness 包装模式标准化（提取 `createModelHarness` 等通用工厂到 `test/helpers/`）
2. 【P2】测试风格约定：场景测试统一 Testing Library，纯逻辑/契约测试可用 test-utils（写入指南）
3. 【P2】jsdom 降级严重的场景（vxe 渲染、浮层定位）标注为 E2E 候选，由 Playwright 冒烟覆盖（`e2e/` 基础设施已存在）
4. 【P2】`anyService` Proxy 兜底：在 `afterEach` 检查非预期调用（warning 追踪已有，fail-fast 检查未做）
5. 【P0】文档性验收：把用例标题清单给不熟悉模块的同事复述验证（全部场景待执行）。**跟踪**：合入主仓库后一周内执行，结论回写本行

**人工确认项**：应用列表加载失败时，真实浏览器里异常提示是否可见（jsdom 无渲染引擎，工具边界外）；确认可见则结案，不可见则提体验缺陷。
