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

/**
 * 清理操作记录（operation_records）中历史写入的敏感明文。
 *
 * 背景：审计层脱敏（字段上的 audit tag，见 pkg/misc/audit/sanitize.go）上线前，操作记录的
 * data.before / data.after / data.extras 直接序列化了解密后的对象，其中可能包含敏感环境变量、BSCP token、
 * Helm 仓库密码、构建镜像仓库密码、北极星 token 以及组件属性中的 token 等明文。
 * 本脚本按与 Go 侧 audit tag 一致的规则将其替换为 "******"。
 *
 * 打码规则（递归作用于 JSON 中的每个对象）：
 *   1. 对象的 IsSensitive / isSensitive 为 true 时，Value / value 打码（对应环境变量）；
 *   2. key 为 BscpToken、Password、password、PolarisToken 的非空字符串打码；
 *   3. key 为 Properties / properties 的对象内，key 以 token / password / passwd / secret 结尾
 *      （不区分大小写）的非空字符串打码，嵌套对象与数组递归处理。
 *
 * 行为：
 *   - 默认 dry-run，只统计，不写库；设置环境变量 APPLY=1 才写回。
 *   - 只写回内容确实发生变化的记录，可重复执行。
 *   - JSON 解析失败、或含超出 JS 安全整数范围的整数（重新序列化会丢精度）的记录跳过并打印 _id。
 *
 * 执行前务必备份（备份文件本身含明文，验证无误后删除）：
 *   mongodump --uri "<mongo-uri>" --collection operation_records --out ./operation-records-backup
 *
 * 用法：
 *   mongosh "<mongo-uri>" --file scripts/mask-sensitive-operation-records.js            # dry-run
 *   APPLY=1 mongosh "<mongo-uri>" --file scripts/mask-sensitive-operation-records.js    # 写回
 */

const MASK = '******';
const APPLY = process.env.APPLY === '1';
const BATCH_SIZE = 500;
const SAMPLE_LIMIT = 10;
const DATA_FIELDS = ['before', 'after', 'extras'];
const SECRET_KEYS = new Set(['BscpToken', 'Password', 'password', 'PolarisToken']);
const PROPERTY_KEYS = new Set(['Properties', 'properties']);
const SENSITIVE_KEY_SUFFIXES = ['token', 'password', 'passwd', 'secret'];

const isPlainObject = (v) => v !== null && typeof v === 'object' && !Array.isArray(v);

const isSensitiveKey = (key) => {
  const lower = key.toLowerCase();
  return SENSITIVE_KEY_SUFFIXES.some((suffix) => lower.endsWith(suffix));
};

// maskString 对非空字符串打码，返回是否发生了变化
const maskString = (obj, key) => {
  if (typeof obj[key] !== 'string' || obj[key] === '' || obj[key] === MASK) {
    return false;
  }
  obj[key] = MASK;
  return true;
};

// maskProperties 按 key 后缀规则原地打码组件属性
const maskProperties = (value) => {
  let changed = false;
  if (Array.isArray(value)) {
    value.forEach((item) => {
      changed = maskProperties(item) || changed;
    });
    return changed;
  }
  if (!isPlainObject(value)) {
    return false;
  }
  Object.keys(value).forEach((key) => {
    if (typeof value[key] === 'string' && isSensitiveKey(key)) {
      changed = maskString(value, key) || changed;
    } else {
      changed = maskProperties(value[key]) || changed;
    }
  });
  return changed;
};

// maskNode 原地打码，返回是否发生了变化
const maskNode = (node) => {
  let changed = false;
  if (Array.isArray(node)) {
    node.forEach((item) => {
      changed = maskNode(item) || changed;
    });
    return changed;
  }
  if (!isPlainObject(node)) {
    return false;
  }
  if (node.IsSensitive === true || node.isSensitive === true) {
    ['Value', 'value'].forEach((key) => {
      if (typeof node[key] === 'string' && node[key] !== MASK) {
        node[key] = MASK;
        changed = true;
      }
    });
  }
  Object.keys(node).forEach((key) => {
    if (SECRET_KEYS.has(key)) {
      changed = maskString(node, key) || changed;
    } else if (PROPERTY_KEYS.has(key) && isPlainObject(node[key])) {
      changed = maskProperties(node[key]) || changed;
    } else {
      changed = maskNode(node[key]) || changed;
    }
  });
  return changed;
};

// hasUnsafeInteger 检查是否存在超出安全整数范围的整数，这类数据重新序列化会丢失精度
const hasUnsafeInteger = (node) => {
  if (typeof node === 'number') {
    return Number.isInteger(node) && !Number.isSafeInteger(node);
  }
  if (Array.isArray(node)) {
    return node.some(hasUnsafeInteger);
  }
  if (isPlainObject(node)) {
    return Object.values(node).some(hasUnsafeInteger);
  }
  return false;
};

const decode = (bin) => Buffer.from(bin.buffer).toString('utf8');
const encode = (str) => BinData(0, Buffer.from(str, 'utf8').toString('base64'));

const coll = db.getCollection('operation_records');
const filter = { $or: DATA_FIELDS.map((f) => ({ [`data.${f}`]: { $exists: true } })) };
const projection = { attribute: 1, data: 1 };

const stats = { scanned: 0, matched: 0, updated: 0, skipped: 0, byAttribute: {} };
const samples = [];
let ops = [];

const flush = () => {
  if (ops.length === 0) {
    return;
  }
  if (APPLY) {
    const res = coll.bulkWrite(ops, { ordered: false });
    stats.updated += res.modifiedCount;
  }
  ops = [];
};

coll.find(filter, projection).forEach((doc) => {
  stats.scanned += 1;
  const set = {};
  let skip = false;

  DATA_FIELDS.forEach((field) => {
    const bin = doc.data && doc.data[field];
    if (skip || !bin || !bin.buffer) {
      return;
    }
    let parsed;
    try {
      parsed = JSON.parse(decode(bin));
    } catch (e) {
      print(`[skip] ${doc._id} data.${field}: invalid JSON (${e.message})`);
      skip = true;
      return;
    }
    if (!maskNode(parsed)) {
      return;
    }
    if (hasUnsafeInteger(parsed)) {
      print(`[skip] ${doc._id} data.${field}: contains unsafe integer, mask manually`);
      skip = true;
      return;
    }
    set[`data.${field}`] = encode(JSON.stringify(parsed));
  });

  if (skip) {
    stats.skipped += 1;
    return;
  }
  if (Object.keys(set).length === 0) {
    return;
  }

  stats.matched += 1;
  const attr = doc.attribute || '(empty)';
  stats.byAttribute[attr] = (stats.byAttribute[attr] || 0) + 1;
  if (samples.length < SAMPLE_LIMIT) {
    samples.push(`${doc._id} (${attr}): ${Object.keys(set).join(', ')}`);
  }

  ops.push({ updateOne: { filter: { _id: doc._id }, update: { $set: set } } });
  if (ops.length >= BATCH_SIZE) {
    flush();
  }
});
flush();

print(`mode: ${APPLY ? 'APPLY' : 'DRY-RUN (set APPLY=1 to write)'}`);
print(`scanned: ${stats.scanned}, need mask: ${stats.matched}, updated: ${stats.updated}, skipped: ${stats.skipped}`);
print(`need mask by attribute: ${JSON.stringify(stats.byAttribute)}`);
if (samples.length > 0) {
  print('samples:');
  samples.forEach((s) => print(`  ${s}`));
}
