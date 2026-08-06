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
  <div class="flex flex-col border-[1px] border-[#DCDEE5] w-full h-full bg-[#fff]">
    <Tab
      v-model:active="activeTab"
      class="helm-variable-tabs flex flex-col flex-1 min-h-0"
      type="unborder-card"
    >
      <Tab.TabPanel
        name="component"
        render-directive="if"
      >
        <template #label>
          <span class="text-[14px]">{{ $t('镜像变量') }}</span>
        </template>
        <div class="component-variable-panel p-[16px] text-[14px] w-full h-full min-w-0 overflow-auto">
          <Alert
            class="mb-[12px]"
            closable
            theme="info"
            :title="$t('在 Values 文件可通过表达式 {0} 引用{1}。', ['${{ bkms.<var_name> }}', $t('组件变量')])"
          />
          <i18n-t keypath="在 “{0}” 页面构建镜像后，可通过以下变量引用镜像信息。">
            <span class="font-bold">{{ $t('构建管理') }}</span>
          </i18n-t>
          <Table
            v-bkloading="{ loading: isLoading }"
            class="mt-[10px]"
            :data="variablesData"
            row-class-name="group"
            :row-config="{
              isHover: true,
              isCurrent: true,
            }"
          >
            <template #empty>
              <TableException />
            </template>
            <TableColumn
              field="key"
              label="变量名"
              width="240px"
            >
              <template #default="{ row }">
                <div class="flex items-center">
                  <span
                    v-bk-tooltips="row.key"
                    class="ellipsis"
                    >{{ row.key }}</span
                  >
                  <Button
                    class="ml-[6px] shrink-0"
                    text
                    @click="copyText(`\${{ bkms.${row.key} }}`)"
                  >
                    <Copy
                      class="group-hover:opacity-100 opacity-0 transition text-[#3A84FF]"
                      :title="$t('复制')"
                    />
                  </Button>
                </div>
              </template>
            </TableColumn>
            <TableColumn
              field="description"
              label="描述"
              show-overflow="tooltip"
            >
            </TableColumn>
          </Table>
          <Alert
            class="image-secret-tip mt-[24px]"
            theme="warning"
          >
            <template #title>
              <span class="text-[12px]">
                {{ $t('部署时自动将镜像凭证写入 Secret，需在 values 中引用方可拉取镜像') }}
              </span>
            </template>
            <div class="mt-[10px] w-full overflow-hidden bg-[#fff] border border-[#DCDEE5] rounded-[2px]">
              <div
                class="h-[28px] px-[12px] text-[12px] flex items-center justify-between bg-[#F5F7FA] border-b border-[#DCDEE5]"
              >
                <span>{{ $t('使用示例') }}</span>
                <Button
                  class="ml-[6px] shrink-0"
                  text
                  @click="copyText('imagePullSecret:\n  - name:  ${{ bkms.APP_IMAGE_PULL_SECRET }}')"
                >
                  <Copy
                    class="text-[#3A84FF]"
                    :title="$t('复制')"
                  />
                </Button>
              </div>
              <pre
                v-pre
                class="m-0 p-[18px] overflow-x-auto font-mono text-[12px] leading-[22px] whitespace-pre"
              >
imagePullSecret:
  - name:  ${{ bkms.APP_IMAGE_PULL_SECRET }}</pre
              >
            </div>
          </Alert>
        </div>
      </Tab.TabPanel>
      <Tab.TabPanel
        name="env"
        render-directive="if"
      >
        <template #label>
          <span class="text-[14px]">{{ $t('环境变量') }}</span>
        </template>
        <ViewDefaultEnvVars
          :alert-title="$t('在 Values 文件可通过表达式 {0} 引用{1}。', ['${{ env.<Key> }}', $t('环境变量')])"
          class="env-variable-panel h-full"
          :custom-request-fn="handleGetVarEnv"
          :env-list="envList"
          :show-header="false"
        />
      </Tab.TabPanel>
    </Tab>
  </div>
</template>
<script lang="ts" setup>
  import { ref } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Button, Tab } from 'bkui-vue';
  import { Copy } from 'bkui-vue/lib/icon';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import { copyText } from '~/common/util';
  import TableException from '~/components/table-exception.vue';
  import ViewDefaultEnvVars from '~/components/view-default-env-vars/index.vue';
  import useEnvManager from '~/composables/use-env-manager';

  import type { PlaceholderVarOutputObj } from '~/@types/v1/arrangement';

  const activeTab = ref<'component' | 'env'>('component');
  const isLoading = ref(false);
  const variablesData = ref<PlaceholderVarOutputObj[]>([]);
  const { envList, handleGetEnvList } = useEnvManager();

  async function getListPlaceholderVars() {
    isLoading.value = true;
    variablesData.value = await ApiServerService.ListPlaceholderVars({}).catch(() => []);
    isLoading.value = false;
  }

  function handleGetVarEnv(env: string) {
    const envID = envList.value.find(item => item.name === env)?.id;
    if (!envID) return Promise.resolve([]);
    return ApiServerService.ListEnvAvailableEnvVars({ envID });
  }

  getListPlaceholderVars();
  handleGetEnvList();
</script>

<style lang="postcss" scoped>
  :deep(.bk-tab-header) {
    height: 40px;
    padding: 0 24px !important;
    background-color: #f0f1f5;
  }

  :deep(.bk-tab-header-item) {
    padding: 0 !important;
    margin-right: 32px !important;
  }

  :deep(.bk-tab-content) {
    display: flex;
    flex: 1;
    min-height: 0;
    padding: 0 !important;
  }

  :deep(.bk-tab-panel) {
    display: flex;
    flex: 1;
    min-height: 0;
    min-width: 0;
  }

  :deep(.image-secret-tip .bk-alert-wraper) {
    align-items: flex-start;
    width: 100%;
    min-width: 0;
    padding: 10px 16px 16px 10px;
    .bk-alert-icon-info {
      margin-top: 4px;
    }
  }

  :deep(.image-secret-tip .bk-alert-title) {
    font-size: 14px;
    line-height: 22px;
    color: #63656e;
  }
</style>
