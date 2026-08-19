# bkms-ui

## 项目介绍

`bkms-ui` 是蓝鲸服务治理平台的 Web 前端，基于 Vue 3、TypeScript、Vite、Vue Router 和 Pinia 构建。

## 文档入口

- [前端架构](./docs/ARCHITECTURE.md)：启动链路、目录职责、关键机制和依赖约定
- [本地开发指南](./docs/DEVELOPMENT.md)：环境准备、常用命令与调试方式
- [API 请求层指南](./docs/API_GUIDE.md)：请求封装、拦截器和接口新增流程
- [编码规范](./docs/CODING_STANDARDS.md)：命名、组件、样式和 License header
- [环境变量说明](./docs/ENV_VARIABLES.md)：构建时与运行时变量
- [生产部署说明](./DEPLOY.md)：Docker 镜像与运行时配置
- [完整文档索引](./docs/README.md)

## 快速开始

1. 克隆项目仓库
2. 安装依赖：`pnpm install`
3. 启动开发服务器：`pnpm dev`
4. 在浏览器中打开 `http://localhost:5008`（或控制台输出的 URL）

## 项目结构

- `src/pages/`：业务页面和页面内私有组件
- `src/components/`：跨页面复用组件，会自动注册
- `src/composables/`：可复用组合式逻辑
- `src/layouts/`：通用页面布局
- `src/modules/`：应用级模块，实现 `install` 方法并在启动时自动安装
- `src/stores/`：Pinia 状态管理
- `src/api/`：API 客户端、请求基础设施和领域接口
- `src/config/`：导航等静态配置
- `src/styles/`：全局样式
- `src/main.ts`：应用入口
- `src/App.vue`：根组件

更完整的职责边界和依赖方向见[前端架构](./docs/ARCHITECTURE.md)。

## 常用命令

| 命令              | 说明                                      |
| ----------------- | ----------------------------------------- |
| `pnpm dev`        | 启动本地开发服务器                        |
| `pnpm build`      | 构建生产产物到 `dist/`                    |
| `pnpm typecheck`  | 执行 Vue 与 TypeScript 类型检查           |
| `pnpm lint`       | 检查 JavaScript、TypeScript 和 Vue 文件   |
| `pnpm stylelint`  | 检查 CSS、Less 和 Vue 样式                |
| `pnpm test:unit`  | 运行 Vitest 单元测试                      |
| `pnpm gen:api:v1` | 根据服务端 Swagger 定义生成 v1 API 与类型 |

具体开发约定统一维护在[编码规范](./docs/CODING_STANDARDS.md)中，API 路径参数使用 `{var}` 格式；自动生成的 `src/api/modules/v1/` 与 `src/@types/v1/` 文件不要手工修改。

## 构建和部署

执行 `pnpm build` 后，生产产物生成在 `dist/`。生产镜像构建、环境变量注入和部署方式请参阅[生产部署说明](./DEPLOY.md)。

## 常见问题

如果开发过程中遇到问题，请先查看[文档索引](./docs/README.md)及相关指南；仍无法解决时，可在项目仓库提交 Issue。

## 贡献指南

我们欢迎并感谢任何形式的贡献。提交变更前，请阅读仓库根目录的[贡献指南](../docs/CONTRIBUTING.md)。
