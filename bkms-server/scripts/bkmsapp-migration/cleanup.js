/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

/*
 * 清理：删除 bkmsApp 应用的旧字段 applications.trpcSpec、app_models 的 workload.type / trpcConfig / tafConfig。
 * 只在库内没有 trpc / taf 应用时执行；执行后 restore.js 不再可用，回滚只能依赖 mongodump。
 * 前提：迁移冲突已清零，观察一个发布周期确认不再回滚。不在迁移当次执行。
 *
 * 用法（默认只打印，加 APPLY=true 写库）：
 *   mongosh "<连接串>" --quiet --eval 'var APPLY=true' scripts/bkmsapp-migration/cleanup.js
 */

/* global load, __dirname, print, BKMS */
load(__dirname + "/lib.js");

(function () {
  BKMS.header("清理旧字段");

  var remaining = BKMS.legacyApps().length;
  if (remaining > 0) {
    print("仍有 " + remaining + " 个 trpc / taf 应用，先处理冲突并完成迁移，未执行清理。");
    return;
  }

  var appIDs = BKMS.db[BKMS.COLL.apps].distinct("id", { type: "bkmsApp" });
  var appFilter = { type: "bkmsApp", trpcSpec: { $exists: true } };
  var modelFilter = {
    appID: { $in: appIDs },
    $or: [
      { "workload.type": { $exists: true } },
      { "workload.trpcConfig": { $exists: true } },
      { "workload.tafConfig": { $exists: true } },
    ],
  };
  var appCount = BKMS.db[BKMS.COLL.apps].countDocuments(appFilter);
  var modelCount = BKMS.db[BKMS.COLL.models].countDocuments(modelFilter);

  if (BKMS.cfg.apply) {
    BKMS.db[BKMS.COLL.apps].updateMany(appFilter, { $unset: { trpcSpec: "" } });
    BKMS.db[BKMS.COLL.models].updateMany(modelFilter, {
      $unset: { "workload.type": "", "workload.trpcConfig": "", "workload.tafConfig": "" },
    });
  }

  print("");
  print((BKMS.cfg.apply ? "已清理" : "待清理") + "    Application.trpcSpec: " + appCount +
    "    AppModel 旧字段: " + modelCount);
})();
