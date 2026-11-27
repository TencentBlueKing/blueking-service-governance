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
 * 回滚：按批次号把已迁移的应用恢复为备份时的值（type、framework、language、frameworkConfigVersion）。
 * 只恢复当前值仍等于迁移结果的应用；迁移后被修改过的应用跳过并报告，不覆盖用户的修改。
 *
 * 用法（默认只打印，加 APPLY=true 写库）：
 *   mongosh "<连接串>" --quiet --eval 'var BATCH_ID="20261110-1"; var APPLY=true' scripts/bkmsapp-migration/restore.js
 */

/* global load, __dirname, print, BKMS */
load(__dirname + "/lib.js");

(function () {
  BKMS.requireBatch();
  BKMS.header("回滚");

  var backups = BKMS.db[BKMS.COLL.backups];
  var restored = [];
  var skipped = [];

  // 字段在备份中为 null 表示原来不存在，回滚时删除
  function setOrUnset(fields) {
    var update = { $set: {}, $unset: {} };
    Object.keys(fields).forEach(function (k) {
      if (fields[k] === null) {
        update.$unset[k] = "";
      } else {
        update.$set[k] = fields[k];
      }
    });
    if (Object.keys(update.$set).length === 0) {
      delete update.$set;
    }
    if (Object.keys(update.$unset).length === 0) {
      delete update.$unset;
    }
    return update;
  }

  backups.find({ batchID: BKMS.cfg.batchID, migratedTo: { $ne: null } }).forEach(function (b) {
    var filter = {
      id: b.appID,
      type: "bkmsApp",
      framework: b.migratedTo.framework,
      language: b.migratedTo.language,
    };
    if (BKMS.db[BKMS.COLL.apps].countDocuments(filter) !== 1) {
      skipped.push({ appID: b.appID, reason: "迁移后已被修改或已删除" });
      return;
    }
    if (BKMS.cfg.apply) {
      var res = BKMS.db[BKMS.COLL.apps].updateOne(filter, setOrUnset({
        type: b.application.type,
        framework: b.application.framework,
        language: b.application.language,
      }));
      if (res.matchedCount !== 1) {
        skipped.push({ appID: b.appID, reason: "写入时应用已被修改" });
        return;
      }
      if (b.appModel.count === 1) {
        BKMS.db[BKMS.COLL.models].updateOne(
          { appID: b.appID },
          setOrUnset({ "workload.frameworkConfigVersion": b.appModel.frameworkConfigVersion })
        );
      }
      backups.updateOne({ _id: b._id }, { $set: { migratedTo: null, restoredAt: new Date() } });
    }
    restored.push({ appID: b.appID, type: b.application.type });
  });

  print("");
  print((BKMS.cfg.apply ? "已回滚: " : "可回滚: ") + restored.length + "    跳过: " + skipped.length);
  BKMS.printRows(BKMS.cfg.apply ? "[已回滚]" : "[可回滚]", restored, function (r) {
    return r.appID + "  bkmsApp -> " + r.type;
  });
  BKMS.printRows("[跳过]", skipped, function (r) {
    return r.appID + "  " + r.reason;
  });
})();
