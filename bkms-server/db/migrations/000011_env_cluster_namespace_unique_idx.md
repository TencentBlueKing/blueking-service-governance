# 环境集群+命名空间唯一索引

- environments 新建索引：
  - cluster.clusterID_1_cluster.namespace_1 (unique, partial): EnvironmentStoreMongo：同一集群下的命名空间全局唯一（仅当 clusterID 和 namespace 均非空时生效），防止多个环境绑定到同一集群的同一命名空间；
