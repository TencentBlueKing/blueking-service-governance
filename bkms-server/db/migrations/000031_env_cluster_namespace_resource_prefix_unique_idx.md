# 环境集群+命名空间+资源前缀唯一索引

- environments 删除索引：
  - cluster.clusterID_1_cluster.namespace_1：由下面的三字段索引取代；
- environments 新建索引：
  - cluster.clusterID_1_cluster.namespace_1_resourcePrefix_1 (unique, partial): EnvMeta：允许复用来源环境命名空间的特性环境（带 resourcePrefix）与来源环境绑定同一集群的同一命名空间（仅当 clusterID 和 namespace 均非空时生效）。resourcePrefix 缺失的文档按 null 参与唯一比较，因此独占命名空间的环境之间仍然互斥；
- 回滚说明：若已存在共用命名空间的特性环境，down 重建两字段唯一索引会因重复键失败，需先删除这些特性环境。
