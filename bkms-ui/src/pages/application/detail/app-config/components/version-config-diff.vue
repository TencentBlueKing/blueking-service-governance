<!--
 - TencentBlueKing is pleased to support the open source community by making
 - 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 - Copyright (C) Tencent. All rights reserved.
 - Licensed under the MIT License (the "License"); you may not use this file except
 - in compliance with the License. You may obtain a copy of the License at
 -
 -  http://opensource.org/licenses/MIT
 -
 - Unless required by applicable law or agreed to in writing, software distributed under
 - the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 - either express or implied. See the License for the specific language governing permissions and
 - limitations under the License.
 -
 - We undertake not to change the open source license (MIT license) applicable
 - to the current version of the project delivered to anyone in the future.
-->

<template>
  <section class="text-[12px] border border-solid border-[#dcdee5]">
    <div class="px-[16px] py-[10px] bg-[#f5f7fa]">
      <span class="font-bold">{{ $t('配置差异') }}</span>
      <span class="ml-[8px] text-[#979BA5]">{{ $t('仅展示发生变化的配置项') }}</span>
    </div>
    <table
      class="w-full table-fixed border-collapse [&_:is(th,td)]:(px-[16px] py-[10px] text-left [overflow-wrap:anywhere] border-t border-r border-solid border-[#eaebf0]) [&_th]:(font-normal bg-[#fafbfd])"
    >
      <colgroup>
        <col class="w-[15%]" />
        <col class="w-[42.5%]" />
        <col class="w-[42.5%]" />
      </colgroup>
      <thead>
        <tr>
          <th>{{ $t('配置项') }}</th>
          <th>V{{ previousVersion }}</th>
          <th>V{{ currentVersion }}（{{ $t('当前版本') }}）</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="row in rows"
          :key="row.key"
        >
          <td>{{ row.label }}</td>
          <td class="text-[#b34747] bg-[#fff5f5]">{{ row.previous }}</td>
          <td class="text-[#2d8a54] bg-[#f0fff4]">{{ row.current }}</td>
        </tr>
      </tbody>
    </table>
  </section>
</template>

<script setup lang="ts">
  defineProps<{
    currentVersion?: number;
    previousVersion?: number;
    rows: { current: string; key: string; label: string; previous: string }[];
  }>();
</script>
