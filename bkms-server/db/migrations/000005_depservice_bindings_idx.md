# Mongo 索引：depservice_bindings

- depservice_bindings 新建索引：
  - appID_1_serviceName_1_name_1: ServiceBindingStoreMongo：同一应用、同一服务下绑定名唯一；
  - instanceIDs_1: ServiceBindingStoreMongo：按被引用的实例 ID 反查绑定（删除实例前检查引用）。
