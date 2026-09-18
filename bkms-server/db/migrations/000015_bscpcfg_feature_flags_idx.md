# bscpcfg FeatureFlag 唯一索引

- bscpcfg_feature_flags 新建索引：
  - appID_1 (unique): FeatureFlagStoreMongo：一个 app 只有一条 FeatureFlag 记录，配合 Upsert 保证并发安全，防止同 appID 出现多条文档；
