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

const seedDB = db.getSiblingDB("bkmsapitest");

const seedUser = {
  username: "api-test-platform-admin",
  roleCode: "admin",
  createdAt: new Date("2026-01-01T00:00:00.000Z"),
  creator: "mongo-init",
  updatedAt: new Date("2026-01-01T00:00:00.000Z"),
  updater: "mongo-init",
};

const result = seedDB.plat_admin_role_bindings.updateOne(
  { username: seedUser.username },
  { $setOnInsert: seedUser },
  { upsert: true },
);

if (result.upsertedCount > 0) {
  print(`seeded plat_admin_role_bindings user: ${seedUser.username}`);
} else {
  print(`plat_admin_role_bindings user already present: ${seedUser.username}`);
}

const seedNow = new Date("2026-01-01T00:00:00.000Z");
const platformBuildImages = [
  {
    id: "000000000000000000000101",
    type: "builder",
    name: "golang",
    tag: "1.24",
    repoKey: "d754ed9f64ac293b10268157f283ee23256fb32a4f8dedb25c8446ca5bcb0bb3",
    description: "API test platform build builder image",
  },
  {
    id: "000000000000000000000102",
    type: "runner",
    name: "debian",
    tag: "12",
    repoKey: "81d93757457f988523814ae0009837ae893f38d3fe123f2c37896f118b4c7804",
    description: "API test platform build runner image",
  },
];

for (const image of platformBuildImages) {
  const imageResult = seedDB.runtime_images.updateOne(
    { type: image.type, name: image.name },
    {
      $setOnInsert: {
        _id: image.id,
        type: image.type,
        name: image.name,
        description: image.description,
        createdAt: seedNow,
        updatedAt: seedNow,
      },
    },
    { upsert: true },
  );

  if (imageResult.upsertedCount > 0) {
    print(`seeded runtime image: ${image.type}/${image.name}`);
  } else {
    print(`runtime image already present: ${image.type}/${image.name}`);
  }

  const snapshotResult = seedDB.image_snapshots.updateOne(
    { repoKey: image.repoKey, tag: image.tag },
    {
      $setOnInsert: {
        repoKey: image.repoKey,
        tag: image.tag,
        digest: `sha256:api-test-${image.type}`,
        size: NumberLong(1),
        builtAt: seedNow,
        createdAt: seedNow,
        updatedAt: seedNow,
      },
    },
    { upsert: true },
  );

  if (snapshotResult.upsertedCount > 0) {
    print(`seeded runtime image snapshot: ${image.name}:${image.tag}`);
  } else {
    print(`runtime image snapshot already present: ${image.name}:${image.tag}`);
  }
}
