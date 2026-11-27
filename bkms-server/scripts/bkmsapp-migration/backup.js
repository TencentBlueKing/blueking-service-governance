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
 * 第 1 步：备份。把待迁移应用（type 为 trpc / taf）的原值写入 bkmsapp_migration_backups，按批次号记录。
 * 同一批次重复执行只补充新出现的应用，不覆盖已有备份。备份不改动业务数据。
 *
 * 用法：
 *   mongosh "<连接串>" --quiet --eval 'var BATCH_ID="20261110-1"' scripts/bkmsapp-migration/backup.js
 *
 * 执行前请另外用 mongodump 导出 applications、app_models、app_config_file_defs。
 */

/* global load, __dirname, print, BKMS */
load(__dirname + "/lib.js");

(function () {
  BKMS.requireBatch();
  BKMS.header("备份", "写入备份集合，不改动业务数据");

  var backups = BKMS.db[BKMS.COLL.backups];
  backups.createIndex({ batchID: 1, appID: 1 }, { unique: true });

  var created = 0;
  var existed = 0;
  BKMS.legacyApps().forEach(function (app) {
    var models = BKMS.modelsOf(app.id);
    var workload = models.length === 1 ? models[0].workload || {} : {};
    var doc = {
      batchID: BKMS.cfg.batchID,
      appID: app.id,
      tenantID: app.tenantID,
      createdAt: new Date(),
      application: {
        type: app.type,
        framework: BKMS.nonEmpty(app.framework) ? app.framework : null,
        language: BKMS.nonEmpty(app.language) ? app.language : null,
        trpcSpec: app.trpcSpec || null,
      },
      appModel: {
        count: models.length,
        frameworkConfigVersion: BKMS.nonEmpty(workload.frameworkConfigVersion) ? workload.frameworkConfigVersion : null,
      },
      // migrate.js 写库后填写迁移结果，restore.js 据此判断能否回滚
      migratedTo: null,
    };
    var res = backups.updateOne(
      { batchID: doc.batchID, appID: doc.appID },
      { $setOnInsert: doc },
      { upsert: true }
    );
    if (res.upsertedCount === 1) {
      created += 1;
    } else {
      existed += 1;
    }
  });

  print("新增备份: " + created + "    已存在: " + existed +
    "    本批次合计: " + backups.countDocuments({ batchID: BKMS.cfg.batchID }));
})();
