# BKMS E2E 测试框架

基于 **Playwright + playwright-bdd（Gherkin）** 的端到端测试，配合 `bkms-bdd-gen` skill 由 AI 自动生成与维护用例。

## 目录结构

```
e2e/
├── features/        # Gherkin .feature 用例（每个 TC 一份）
├── steps/           # Step 定义（common.steps.ts + <TC-ID>.steps.ts）
├── actions/         # 业务流程封装（按业务模块划分，可被多个 TC 复用）
├── pages/           # Page Object（BasePage + 子页面，原子 UI 操作）
├── data/            # 表单 schema 与数据
├── utils/           # 工具与常量（NAVIGATION_ROUTE_MAP、Schema 类型等）
├── fixtures/        # playwright-bdd fixtures（pages / testConfig / userData）
├── tests/           # 普通 Playwright spec（不参与 BDD 生成）
├── scripts/         # 报告生成、BKRepo 上传、profile runner
├── .env.test        # 测试环境变量（本地运行配置）
├── Dockerfile       # 测试容器镜像（node:22 + Chromium）
├── run.sh           # 一键执行脚本（bddgen → 测试 → HTML 报告 → 可选上传）
├── playwright.config.ts
└── playwright.spec.config.ts
```

## 架构分层

```
.feature  (语义层，描述行为)
   ↓
*.steps.ts  (桥接层，参数透传，禁止写 selector)
   ↓
*.action.ts  (业务层，组合多个 Page Object 完成业务流程)
   ↓
*.page.ts  (UI 抽象层，原子操作，继承 BasePage)
   ↓
data/*.ts + utils/form.ts  (表单 Schema / Form Engine)
```

**核心原则**：上层只能调下层；step 文件不允许出现 `locator()` / `.bk-xxx` 选择器。

## 快速开始

```bash
# 安装依赖
cd apps/ui/e2e && pnpm install

# 配置环境变量（按需修改）
vim .env.test

# pnpm 运行全部 BDD 用例
pnpm test

# 按 profile 运行
pnpm test:smoke
pnpm test:deploy
pnpm test:config
pnpm test:readonly
pnpm test:spec

# pnpm 运行指定用例或仅列出用例
pnpm test -- --grep "TC-XX"
pnpm test:smoke -- --list

# ui模型运行
pnpm test:ui

# debug模式运行
pnpm test:debug -- --grep "TC-xx"

# 本地运行全部用例
chmod +x ./run.sh
set -a && source .env.test && set +a && ./run.sh --profile all

# 按 profile 运行
set -a && source .env.test && set +a && ./run.sh --profile smoke
set -a && source .env.test && set +a && ./run.sh --profile deploy
set -a && source .env.test && set +a && ./run.sh --profile config
set -a && source .env.test && set +a && ./run.sh --profile readonly

# 只跑某个 TC
chmod +x ./run.sh
set -a && source .env.test && set +a && ./run.sh --tags '@TC-04'

# 容器化运行（推荐 CI 使用）
docker build -t bkms-e2e-runner:latest -f e2e/Dockerfile .
docker run --rm \
  --env-file e2e/.env.test \
  -v "$(pwd):/workspace/apps/ui" \
  -w /workspace/apps/ui \
  bkms-e2e-runner:latest \
  bash -c "pnpm install --frozen-lockfile && ./e2e/run.sh --tags '@TC-04'"
```

执行结果：

- `test-reports/<dir>/result.json` — BDD 执行结果
- `test-reports/<dir>/report.html` — 可视化 HTML 报告
- `test-reports/<dir>/*.png` — `截图 "NN-xxx"` 截图
- `test-results/` — 失败时自动生成 trace / error-context

## 运行 Profile

| profile / script | 标签或配置 | 说明 |
|------------------|------------|------|
| `pnpm test` / `pnpm test:all` | 全部 BDD | 默认串行，适合 stateful 用例 |
| `pnpm test:smoke` | `@smoke` | 最小可用性检查 |
| `pnpm test:deploy` | `@deploy-flow` | 部署、扩缩容、移除等状态流转 |
| `pnpm test:config` | `@config-flow` | 应用配置类用例 |
| `pnpm test:readonly` | `@readonly` | 只读用例，默认 `E2E_PARALLEL=true` 且 `E2E_WORKERS=2` |
| `pnpm test:spec` | `playwright.spec.config.ts` | 普通 Playwright spec，不参与 BDD 生成 |

`./run.sh --tags '<expr>'` 优先级高于 `--profile`，适合临时组合标签。

## 环境变量

`.env.test` 中维护，必填变量缺一即报错。

| 变量 | 说明 |
|------|------|
| `BKMS_TEST_ACCESS_TOKEN` | 测试环境 AccessToken（敏感，勿入库） |
| `BKMS_TEST_SITE` | 目标站点 URL |
| `BKMS_TEST_DEFAULT_SPACE` | 默认空间（对应 `@space:default`） |
| `BKMS_TEST_DEFAULT_ENV` | 默认环境（对应 `@env:default`，可空） |
| `BKMS_TEST_DEFAULT_APP` | 默认应用（对应 `@app:default`，可空；未配置时可自动发现 `e2e-` 前缀 tRPC 应用） |
| `BKMS_TEST_TRPC_APP` | tRPC 测试应用覆盖值（对应 `@app:trpc`，推荐 `e2e-trpc`；未配置时回退 `BKMS_TEST_DEFAULT_APP`，再自动发现） |
| `BKMS_TEST_HELM_APP` | Helm 测试应用覆盖值（对应 `@app:helm`，推荐 `e2e-helm`；未配置时自动发现） |
| `BKMS_TEST_REPORT_DIR` | 报告输出目录（相对路径按仓库根目录解析） |
| `BK_CI` | CI 模式：retries=1、headless |
| `BKMS_TEST_USE_MOCK` | 启用 `mocks.json` 路由拦截 |
| `BKREPO_ADDR` / `BKREPO_PROJECT` / `BKREPO_REPO` / `BKREPO_TOKEN` | 全部齐全时自动上传 HTML 报告（地址勿硬编码） |
| `BKREPO_USERNAME` | BKRepo 用户名（可选） |
| `E2E_REPORT_FONT_BASE_URL` | HTML 报告字体 CDN 根地址；未配置则使用系统字体 |
| `E2E_PARALLEL` | 设置为 `true` 时允许并行执行；默认串行 |
| `E2E_WORKERS` | 并行 worker 数，仅 `E2E_PARALLEL=true` 时生效 |
| `E2E_SKIP_PREFLIGHT` | 设置为 `true` 时跳过 `run.sh` 目标站点预检 |
| `E2E_PREFLIGHT_TIMEOUT` | `run.sh` 目标站点预检超时时间，默认 10 秒 |

`BKMS_TEST_REPORT_DIR` 使用相对路径时按仓库根目录解析。`run.sh` 每次执行前会清理该目录下的 `result.json`、`report.html`、顶层 `*.png` 和 `playwright-html/`。

### 测试应用与数据命名

- 优先使用 `e2e-` 前缀应用承载测试，例如 `e2e-trpc`、`e2e-helm`。
- `@app:trpc` / `@app:helm` 表示应用类型语义，不在 feature 中写死具体环境应用名。
- 应用解析优先级：显式环境变量覆盖（如 `BKMS_TEST_HELM_APP`）→ 当前空间内自动查找类型匹配且名称以 `e2e-` 开头的应用；若存在 `e2e-${appType}` 会优先使用。
- 长期 stateful 测试数据使用 `e2e-${timestamp}-${caseId}` 命名，便于识别、清理和后续并行隔离。

## 新增一个测试用例（TC-XX）

> 推荐：直接调用 `bkms-bdd-gen` skill 让 AI 生成；以下为手工流程参考。

1. **写 `.feature`**：在 `features/` 新建 `TC-XX-<desc>.feature`
   - 首行 `@TC-XX @P0`
   - `Background: Given AccessToken 认证已配置`
   - 按需打 `@space:default` / `@env:xxx` / `@app:xxx` / `@appType:xxx`
   - 步骤优先复用 `steps/common.steps.ts` 中已有 step
   - 关键节点用 `And 截图 "NN-desc"`

2. **（可选）写 `<TC-XX>.steps.ts`**：仅当 common 中无匹配 step 时
   - 从 `../fixtures/fixtures` 导入 `Given/When/Then`
   - 通过 `{ pages, testConfig, userData, page, request }` 解构
   - 复杂流程委托给 `actions/*.action.ts`，禁止写 selector

3. **（可选）扩 Page Object**：`pages/<name>.page.ts` 继承 `BasePage`，只写原子操作；新建后**必须**在 `fixtures/fixtures.ts` 的 `pages` fixture 注册实例

4. **（可选）写 Action**：按业务模块命名（如 `deploy.action.ts`），组合 Page Object 方法形成完整业务流程

5. **（可选）新增表单**：
   - 在 `data/<name>-data.ts` 定义 `XxxFormData` 类型与 `Schema<XxxFormData>`
   - 在 `actions/form.action.ts` 扩展 `FormType` 联合类型与 `schemaMap`
   - step 中调用 `fillFormByType(basePage, 'Xxx', data)`

6. **（可选）新增导航**：在 `utils/config.ts` 的 `NAVIGATION_ROUTE_MAP` 补充 `中文名: '路由路径'`

7. **运行验证**：`pnpm bddgen test --tags '@TC-XX' && pnpm playwright test --list`

## 注意事项

### 必须遵守

- ❌ **严禁修改前端/后端源码** — 测试只在 `e2e/` 目录下变更
- ❌ **step 文件禁止出现 selector** — `locator()` / `.bk-xxx` 必须封装到 Page Object
- ❌ **不要直接拼深层 URL** — 应用详情路由 `/:space/app/:name/:type/:menuName`，跳过会 404，必须 UI 点击导航
- ❌ **断言失败不要改值掩盖** — 怀疑是平台 bug 时保留失败并向用户告知

### 推荐实践

- ✅ **优先使用 `data-testid`/`v-test` 定位** — Page Object 用 `this.testId('xxx')`，CSS selector 仅作兜底
- ✅ **Scenario 步骤数 ≤ 15** — 超过则拆分
- ✅ **环境标签按需打** — 仅依赖 space 就只写 `@space:default`，避免凑齐 env/app
- ✅ **测试应用优先使用 `e2e-` 前缀** — feature 中使用 `@app:trpc` / `@app:helm` 等语义标签，具体应用名放到环境变量
- ✅ **action 按业务模块划分** — 不要按 TC-ID 1:1 命名，跨 TC 复用
- ✅ **新增 Page Object 后立即去 `fixtures.ts` 注册**，否则运行时拿不到

### 升级 Playwright

版本必须三处一致，缺一即镜像失效：

- `e2e/package.json` — `@playwright/test` 精确版本
- `e2e/Dockerfile` — 预装 Chromium 版本
- 重建镜像 `docker build -t bkms-e2e-runner:latest -f e2e/Dockerfile .`

## 相关资源

- AI 生成 skill：`.claude/skills/bkms-bdd-gen/SKILL.md`
- 模板规范：`.claude/skills/bkms-bdd-gen/templates/`
- playwright-bdd 文档：<https://vitalets.github.io/playwright-bdd/>
