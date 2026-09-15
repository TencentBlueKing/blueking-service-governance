# devmode_publish_records 索引

- devmode_publish_records 新建索引：
  - appID_1_envName_1_createdAt_-1: DevMode PublishRecordStoreMongo.List：按 appID + envName 列表查询时支持 createdAt 倒序；
  - appID_1_envName_1_instance_1_createdAt_-1: DevMode PublishRecordStoreMongo.ListLatestByInstance：按 appID + envName + instance 取各实例最新发布记录（同实例多条记录时按 createdAt 倒序取第一条）。
