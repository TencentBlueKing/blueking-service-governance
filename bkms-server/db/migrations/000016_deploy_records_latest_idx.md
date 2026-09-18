# 部署记录「取最新」查询索引

空间总览（`ListWorkspacesOverview`）、空间应用列表（`ListApps`）与部署总览都会经
`DeployStatusService` 聚合应用在各环境的部署状态，热路径落在三张部署记录表的
「取最新一条」查询上。其中 `app_model_deploy_records` 与 `helm_deploy_records`
此前只有默认 `_id` 索引，按 `appID + envName + trafficLaneName` 过滤再按 `createdAt`
倒序取首条只能走全表扫描 + 内存排序；两张表又都是只增不减的部署流水，耗时随部署
次数持续劣化。

- app_model_deploy_records 新建索引：
  - appID_1_envName_1_trafficLaneName_1_createdAt_-1：支撑 `RecordStoreMongo.GetLatest` /
    `GetLatestByStatuses` / `List`，按 appID + envName + trafficLaneName 过滤并按 createdAt 倒序；
  - appID_1_trafficLaneName_1_createdAt_-1：支撑 `RecordStoreMongo.ListLatestByApp`
    跨环境聚合各环境最新记录时的 $match + $sort 阶段。
- helm_deploy_records 新建索引：
  - appID_1_envName_1_trafficLaneName_1_createdAt_-1：支撑 `helm.RecordStoreMongo.GetLatest` /
    `GetLatestByStatuses` / `List`；
  - appID_1_trafficLaneName_1_createdAt_-1：为按应用跨环境聚合最新部署记录预留，
    与另两张部署记录表保持一致的索引形态。
- build_auto_deploy_records 新建索引：
  - appID_1_trafficLaneName_1_createdAt_-1：补齐 `ListLatestByApp` 的跨环境聚合；
    该表已有的 appID_1_envName_1_trafficLaneName_1_createdAt_-1 因 envName 位于
    trafficLaneName 之前，无法支撑不带 envName 的过滤。
