# 集群插件 assets 说明

每个 YAML 文件定义一个集群插件。服务启动时会自动加载此目录下的所有 YAML 文件
并以 upsert 方式写入 MongoDB (cluster_addon_defs collection)。

字段定义见 bkms-server/pkg/core/env/clusteraddon/model.go ClusterAddonDef