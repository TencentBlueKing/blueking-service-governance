# bkms-ui 开发指南

## 1. 环境要求

| 工具     | 版本        | 说明                                                |
| -------- | ----------- | --------------------------------------------------- |
| Node.js  | ≥ 18        | 推荐 20+                                            |
| pnpm     | 10.33.4     | 包管理器（`package.json` 中 `packageManager` 指定） |
| 后端服务 | bkms-server | 本地开发需启动或连接后端 API                        |

## 2. 快速开始

```bash
# 1. 安装依赖
pnpm install

# 2. 配置环境变量
#    仓库已包含 .env.development；个人覆盖项建议写入不提交的 .env.development.local
#    变量清单和示例见 docs/ENV_VARIABLES.md

# 3. 启动开发服务器（默认端口 5008，自动打开浏览器）
pnpm dev
```

浏览器访问控制台输出的 URL。实际主机和端口由 `BK_APP_HOST`、`BK_APP_PORT` 决定；仓库当前开发配置不一定使用 `localhost`。

## 3. 常用命令

```bash
# 开发
pnpm dev                  # 启动 Vite dev server（mode: development）
pnpm preview              # 预览构建产物
pnpm server               # http-server 托管 dist/（端口 5000）

# 构建
pnpm build                # Vite 生产构建 → dist/

# 代码质量
pnpm lint                 # ESLint 检查 (.js/.ts/.vue)
pnpm lint:fix             # ESLint 自动修复（含 License header 插入）
pnpm format               # Prettier 格式化
pnpm stylelint            # Stylelint 样式检查 (.css/.scss/.vue)
pnpm stylelint:fix        # Stylelint 自动修复
pnpm biome:check          # Biome 检查
pnpm biome:format         # Biome 格式化
pnpm typecheck            # vue-tsc 类型检查（增量）
pnpm typecheck:staged     # 仅检查 git staged 文件（配合 lint-staged）

# 测试
pnpm test:unit            # Vitest 单元测试 (test/**/*.test.ts)

# API 代码生成
pnpm gen:api:v1           # 从 bkms-server swagger.json 生成 API 模块和类型

# 依赖管理
pnpm up                   # taze 检查依赖更新
pnpm sizecheck            # vite-bundle-visualizer 产物体积分析
```

## 4. 开发服务器代理

`vite.config.mts` 中配置了以下 dev proxy，将前端请求转发到后端服务：

| 路径                    | 目标                  | 说明                 |
| ----------------------- | --------------------- | -------------------- |
| `/bkms`                 | `BK_API_BASE_URL`     | bkms-server 核心 API |
| `/bcsapi`               | `BK_BCS_API_BASE_URL` | 蓝鲸容器服务 BCS     |
| `/ms`                   | `BK_REPO_URL`         | 制品库               |
| `/api-bk-user-selector` | bk-user-web           | 用户选择器           |
| `/simple_account`       | `BK_API_BASE_URL`     | 账号服务             |
| `/generic`              | `BK_STATIC_URL`       | 静态资源             |

> 所有 `BK_` 前缀的环境变量通过 Vite 注入 `import.meta.env`，配置在 `.env.development` 中。

## 5. 路径别名

```ts
// tsconfig.json + vite.config.mts
'~/'  → src/    // 统一使用
```

`vite.config.mts` 还保留了 `@/` 运行时别名，但 `tsconfig.json` 未声明对应的 `paths`。为保证编辑器与 `vue-tsc` 一致，在补齐 TypeScript 配置前不要新增 `@/` 导入。

导入示例：

```ts
import { useUserStore } from '~/stores/user';
import { getUser } from '~/api/modules/user';
```

## 6. 自动导入机制

### 6.1 组件自动导入

`unplugin-vue-components` 扫描 `src/components/` 下所有 `.vue` 文件，自动注册。

```vue
<!-- 无需 import，直接使用 -->
<template>
  <FlexRow>
    <MonacoEditor />
  </FlexRow>
</template>
```

类型声明自动生成到 `src/components.d.ts`（**勿手动编辑**）。

### 6.2 API 模块自动生成

```bash
pnpm gen:api:v1
```

从 `bkms-server/docs/apis/swagger.json` 生成：

- API 文件 → `src/api/modules/v1/*.ts`
- 类型文件 → `src/@types/v1/*.ts`

> 这些文件为自动生成，**勿手动编辑**。后端接口变更后重新执行生成命令。

## 7. 调试技巧

### 7.1 Vue DevTools

开发环境自动加载 `vite-plugin-vue-devtools`，浏览器中按 `Alt+Shift+D` 打开。

### 7.2 UnoCSS Inspector

开发环境访问 UnoCSS 检查器查看原子类使用情况：

```
http://localhost:5008/__unocss
```

### 7.3 产物体积分析

```bash
pnpm sizecheck
```

生成可视化产物分析图，定位体积瓶颈。

### 7.4 路由调试

路由使用 Hash 模式（`createWebHashHistory`），URL 格式 `/#/path`。

- `router.back()` 已覆写为智能返回（优先使用浏览器历史，无历史时自动推导父路由）
- `router.originalBack()` 保留原始 back 行为
- 空间路由（`:space/...`）有全局守卫校验空间是否存在且状态为 Ready

## 8. Git Hooks

仓库包含 Husky、lint-staged 和 staged typecheck 配置，但当前 `package.json#prepare` 与 `.husky/pre-commit` 仍引用迁移前的 `apps/ui` 路径，不能视为已生效的提交保护。

在 Hook 路径修复前，提交前至少手工执行：

```bash
pnpm lint
pnpm stylelint
pnpm typecheck
```

修复 Hook 时应同时确认：

- `prepare` 能从当前仓库结构找到根目录 `.git` 与 `bkms-ui/.husky`
- `pre-commit` 在 `bkms-ui` 工作目录执行命令
- `lint-staged` 被实际调用，而不只是存在配置文件

## 9. 国际化

- 语言文件：`locales/zh-CN.yml`、`locales/en-US.yml`
- 默认语言：`zh-CN`
- 使用方式：
  ```ts
  // 模板中
  {
    {
      $t('key');
    }
  }

  // script 中
  window.i18n.t('请求异常');
  ```
- 语言文件通过 `@intlify/unplugin-vue-i18n` 预编译，`import.meta.glob` 异步加载

## 10. 常见问题

| 问题                       | 解决方案                                                                               |
| -------------------------- | -------------------------------------------------------------------------------------- |
| `pnpm install` 失败        | 检查 Node/pnpm 版本、镜像源与错误日志；不要为排障直接删除受版本控制的 `pnpm-lock.yaml` |
| 页面白屏                   | 检查 `.env.development` 中 `BK_API_BASE_URL` 是否正确                                  |
| API 404                    | 确认 dev proxy 配置正确，后端服务已启动                                                |
| 类型报错                   | 运行 `pnpm typecheck` 查看完整错误；确认 `pnpm gen:api:v1` 已执行                      |
| ESLint License header 报错 | 运行 `pnpm lint:fix` 自动插入 header                                                   |
| `frozen-lockfile` 构建失败 | 本地改过依赖后需更新 `pnpm-lock.yaml`                                                  |

## 11. 相关文档

- [API 请求层指南](./API_GUIDE.md)
- [编码规范](./CODING_STANDARDS.md)
- [环境变量说明](./ENV_VARIABLES.md)
- [架构图](./architecture-diagram.html)
- [Docker 部署](../DEPLOY.md)
