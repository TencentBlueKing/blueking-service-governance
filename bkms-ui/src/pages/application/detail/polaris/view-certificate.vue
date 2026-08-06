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
  <Sideslider
    v-model:is-show="isShow"
    :title="$t('查看凭证')"
    :width="640"
    @closed="handleClose"
  >
    <div class="p-[24px]">
      <!-- 蓝色提示 -->
      <Alert
        class="mb-[16px]"
        theme="info"
      >
        {{ $t('凭证信息会写入到应用环境变量中') }}
      </Alert>

      <!-- 凭证变量表格 -->
      <Table
        auto-resize
        class="w-full"
        :data="certificateList"
        :empty-text="$t('暂无数据')"
        :row-config="{
          isHover: true,
          isCurrent: true,
        }"
        sync-resize
      >
        <template #empty>
          <TableException />
        </template>
        <TableColumn
          field="key"
          :label="$t('变量名')"
          show-overflow="tooltip"
          :width="200"
        >
          <template #default="{ row }">
            <HoverCopy
              :copy-value="row.key"
              :text="row.key"
            />
          </template>
        </TableColumn>
        <TableColumn
          field="value"
          :label="$t('变量值')"
          :min-width="200"
          show-overflow="tooltip"
        >
        </TableColumn>
      </Table>
    </div>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Sideslider } from 'bkui-vue';
  import { PolarisConfigVarOutput } from '~/@types/v1/polaris-config';
  import { PolarisConfigService } from '~/api/modules/v1';
  import { useAppDetail } from '~/stores/app-detail';

  interface Props {
    configName: string;
  }

  const isShow = defineModel<boolean>('isShow');
  const props = defineProps<Props>();
  const appDetailStore = useAppDetail();

  const certificateList = ref<PolarisConfigVarOutput[]>([]);

  async function handleInit() {
    try {
      certificateList.value = await PolarisConfigService.listAppPolarisConfigVars({
        appID: appDetailStore.appID,
        configName: props.configName,
      });
    } catch (err) {
      console.error(err);
    }
  }

  watch(
    () => isShow.value,
    newVal => {
      if (newVal) {
        handleInit();
      }
    },
  );

  function handleClose() {
    isShow.value = false;
  }
</script>
