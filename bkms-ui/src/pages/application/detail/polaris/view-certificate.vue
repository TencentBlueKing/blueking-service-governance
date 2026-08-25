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
      <!-- 立即生效模式不提供凭证，需前往北极星控制台查看 Token -->
      <Alert
        v-if="isImmediateMode"
        theme="warning"
      >
        <template #title>
          <div class="leading-[20px]">
            {{
              $t(
                '该配置为立即生效，凭证不会写入应用环境变量。如需查看北极星 Token，请到北极星控制台查看（仅北极星负责人可见）。',
              )
            }}
            <a
              v-if="polarisConsoleUrl"
              class="mt-[4px] flex w-fit items-center text-[#3A84FF]"
              :href="polarisConsoleUrl"
              target="_blank"
            >
              {{ $t('前往北极星') }}
              <Share class="ml-[4px]" />
            </a>
          </div>
        </template>
      </Alert>

      <template v-else>
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
      </template>
    </div>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Sideslider } from 'bkui-vue';
  import { Share } from 'bkui-vue/lib/icon';
  import { PolarisConfigOutputObj, PolarisConfigVarOutput } from '~/@types/v1/polaris-config';
  import { PolarisConfigService } from '~/api/modules/v1';
  import { useAppDetail } from '~/stores/app-detail';

  interface Props {
    configName: string;
    polarisName?: string;
    polarisNamespace?: string;
    registerMode?: PolarisConfigOutputObj['registerMode'];
  }

  const isShow = defineModel<boolean>('isShow');
  const props = defineProps<Props>();
  const appDetailStore = useAppDetail();

  const certificateList = ref<PolarisConfigVarOutput[]>([]);
  const isImmediateMode = computed(() => props.registerMode === 'immediate');
  const polarisConsoleUrl = computed(() => {
    if (!props.polarisNamespace || !props.polarisName) return '';
    return `${import.meta.env.BK_POLARIS_URL}/#/services/info/detail/${encodeURIComponent(props.polarisNamespace)}/${encodeURIComponent(props.polarisName)}`;
  });

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
      if (!newVal) return;
      if (isImmediateMode.value) {
        certificateList.value = [];
        return;
      }
      handleInit();
    },
  );

  function handleClose() {
    isShow.value = false;
  }
</script>
