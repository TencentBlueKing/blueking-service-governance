运行项目单测需要的 **数据库类型** 依赖服务，比如 mongo、redis、etcd 等。本目录中的配置主要针对
CI 服务使用（快起快销），并没有启用数据持久化等配置，开发者的本地开发环境不建议使用。

### 说明

- 使用 NETWORK_NAME 环境变量配置网络名（需要提前使用 docker network create 创建）；
- 服务启动后，通过环境变量 NETWORK_NAME 网络访问；
