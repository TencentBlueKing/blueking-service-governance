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
  <Dialog
    :is-show="dialogConf.isShow"
    :quick-close="false"
    :width="dialogConf.width"
    @closed="hide"
  >
    <div class="permission-modal">
      <div class="permission-header">
        <Exception
          scene="part"
          type="403"
        />
        <h3>{{ $t('需要申请以下权限') }}</h3>
      </div>
      <div v-bkloading="{ isLoading }">
        <Table
          :data="actionList"
          :row-config="{
            isHover: true,
            isCurrent: true,
          }"
        >
          <TableColumn
            field="system"
            :label="$t('系统')"
            min-width="150"
          >
            <template #default>
              <span class="text-[12px]">{{ siteName }}</span>
            </template>
          </TableColumn>
          <TableColumn
            field="auth"
            :label="$t('需要申请的权限')"
            min-width="150"
          >
            <template #default="{ row }">
              <span class="text-[12px]">{{ actions[row.action_id] || '--' }}</span>
            </template>
          </TableColumn>
          <TableColumn
            field="resource"
            :label="$t('关联的资源实例')"
            min-width="150"
          >
            <template #default="{ row }">
              <span class="text-[12px]">{{ row?.resource_name || '--' }}</span>
            </template>
          </TableColumn>
        </Table>
      </div>
    </div>
    <template #footer>
      <div class="permission-footer">
        <div class="button-group">
          <div
            v-bk-tooltips="{
              content: $t('暂无权限申请地址'),
              disabled: !!applyUrl,
            }"
          >
            <Button
              :disabled="!applyUrl"
              theme="primary"
              @click="goApplyUrl"
              >{{ $t('去申请') }}</Button
            >
          </div>
          <Button
            class="ml-[10px]"
            @click="hide"
            >{{ $t('取消') }}</Button
          >
        </div>
      </div>
    </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { type UnwrapRef, computed, onBeforeMount, onUnmounted, ref } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Dialog, Exception } from 'bkui-vue';
  import actionsMap from '~/composables/use-action-map';
  import usePlatform from '~/composables/use-platform';
  import { usePlatformConfigStore } from '~/stores/platform-config';

  interface ActionItem {
    action_id: string;
    resource_name?: string;
    system?: string;
  }

  interface PermData {
    action_list: ActionItem[];
    apply_url: string;
  }

  type PlatformConfig = UnwrapRef<ReturnType<typeof usePlatformConfigStore>['$state']>;

  interface ShowCallbackData {
    perms?: PermData;
  }

  const dialogConf = ref({
    isShow: false,
    width: 640,
  });
  const applyUrl = ref('');
  const actionList = ref<ActionItem[]>([]);
  const isLoading = ref(false);
  const config = ref<Partial<PlatformConfig>>({});

  const actions = actionsMap();

  const siteName = computed(() => config.value?.i18n?.name);

  function goApplyUrl() {
    window.open(applyUrl.value);
    hide();
  }
  function hide() {
    isLoading.value = false;
    dialogConf.value.isShow = false;
    applyUrl.value = '';
    actionList.value = [];
  }
  async function show(callbackData: (() => Promise<ShowCallbackData>) | ShowCallbackData = {}) {
    dialogConf.value.isShow = true;
    let data: ShowCallbackData = {};
    if (typeof callbackData === 'function') {
      isLoading.value = true;
      data = await callbackData();
      isLoading.value = false;
    } else {
      data = callbackData;
    }
    const { apply_url, action_list = [] } = data.perms ?? {};
    applyUrl.value = apply_url ?? '';
    actionList.value = action_list;
  }

  onBeforeMount(() => {
    const { platformConfig } = usePlatform();
    config.value = platformConfig;
  });

  onUnmounted(() => {
    applyUrl.value = '';
  });

  defineExpose({
    show,
  });
</script>
<style lang="postcss" scoped>
  :deep(.bk-modal-mask),
  :deep(.bk-modal-wrapper) {
    z-index: 9999 !important;
  }
  .permission-modal {
    .permission-header {
      text-align: center;
      .title-icon {
        display: inline-block;
      }
      .lock-img {
        width: 120px;
      }
      h3 {
        margin: 6px 0 24px;
        color: #63656e;
        font-size: 20px;
        font-weight: normal;
        line-height: 1;
      }
    }
  }
  .button-group {
    display: flex;
    justify-content: flex-end;
    .Button {
      margin-left: 7px;
    }
  }
</style>
