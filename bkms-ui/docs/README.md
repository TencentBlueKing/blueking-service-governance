# bkms-ui 文档目录

本目录存放 bkms-ui 前端项目的开发文档与参考资料。

## 文档索引

| 文档                                                     | 说明                                                                   |
| -------------------------------------------------------- | ---------------------------------------------------------------------- |
| [ARCHITECTURE.md](./ARCHITECTURE.md)                     | 前端架构 — 启动链路、目录职责、关键机制、依赖方向与功能放置指引        |
| [architecture-diagram.html](./architecture-diagram.html) | 文件结构可视化快照；目录和数量可能随代码演进变化，以代码及架构文档为准 |
| [DEVELOPMENT.md](./DEVELOPMENT.md)                       | 本地开发指南 — 环境搭建、命令、调试技巧                                |
| [API_GUIDE.md](./API_GUIDE.md)                           | API 请求层指南 — ConsoleFetch、拦截器、新增接口步骤                    |
| [CODING_STANDARDS.md](./CODING_STANDARDS.md)             | 编码规范 — 命名、组件、License header、样式                            |
| [ENV_VARIABLES.md](./ENV_VARIABLES.md)                   | 环境变量说明 — 构建时/运行时变量完整清单                               |

## 快速入口

- **新成员入门** → [DEVELOPMENT.md](./DEVELOPMENT.md) + [ARCHITECTURE.md](./ARCHITECTURE.md)
- **了解目录全貌** → [ARCHITECTURE.md](./ARCHITECTURE.md) + [architecture-diagram.html](./architecture-diagram.html)
- **新增 API 接口** → [API_GUIDE.md](./API_GUIDE.md)
- **新增页面/组件** → [CODING_STANDARDS.md](./CODING_STANDARDS.md)
- **部署配置** → [ENV_VARIABLES.md](./ENV_VARIABLES.md) + 项目根 [DEPLOY.md](../DEPLOY.md)

## 相关文档

- [项目根 README](../README.md) — 快速开始、项目结构
- [DEPLOY.md](../DEPLOY.md) — Docker 生产部署完整说明
- [AGENTS.md](../../AGENTS.md) — 仓库全局规范（License header 等）
