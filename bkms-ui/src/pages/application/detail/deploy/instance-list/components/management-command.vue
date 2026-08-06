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
  <div class="px-[16px] py-[18px]">
    <Alert
      v-if="adminPrecheckFailed"
      class="mb-[12px]"
      theme="error"
    >
      <template #title>
        {{ $t('框架配置文件中 admin.ip 设置无效。若要开启管理命令功能，请将其设置为 127.0.0.1 或 0.0.0.0。') }}
        <Button
          text
          theme="primary"
          @click="handleViewGuide"
        >
          {{ $t('管理命令配置说明') }}
        </Button>
      </template>
    </Alert>
    <Alert
      v-else
      class="mb-[12px]"
      closable
      theme="info"
    >
      <template #title>
        {{
          $t('使用管理命令，必须在 {app} 框架配置的 Server 下增加 admin 配置。', { app: isTrpcApp ? 'tRPC' : 'TAF' })
        }}
        <Button
          v-if="isTrpcApp"
          text
          theme="primary"
          @click="handleViewGuide"
        >
          {{ $t('查看指引') }}
        </Button>
      </template>
    </Alert>
    <ToggleCard
      :name="$t('配置命令')"
      type="normal"
      @change="handleConfigCommandCollapse"
    >
      <div class="flex items-center mb-[2px]">
        <!-- Trpc 应用 -->
        <div
          v-if="isTrpcApp"
          class="flex-1 flex select-group rounded-[2px] overflow-hidden"
        >
          <Select
            ref="commandMethodSelectRef"
            v-model="commandMethod"
            class="w-[180px] rounded-none"
            :clearable="false"
          >
            <Select.Option
              id="GET"
              name="GET"
            />
            <Select.Option
              id="POST"
              name="POST"
            />
            <Select.Option
              id="PUT"
              name="PUT"
            />
          </Select>
          <Select
            ref="commandSelectRef"
            v-model="command"
            class="flex-1 ml-[-1px]"
            filterable
            :loading="loading"
          >
            <Select.Option
              v-for="(item, index) in commandList"
              :id="item"
              :key="`${index}-${item}`"
              :name="item"
            />
          </Select>
        </div>
        <!-- TAF 应用 -->
        <Input
          v-else
          v-model.trim="command"
          class="flex-1"
          clearable
          :placeholder="$t('请输入命令')"
        />
        <Button
          v-bk-tooltips="{
            content: isTrpcApp ? $t('请先选择命令') : $t('请输入命令'),
            disabled: command,
          }"
          class="ml-[8px]"
          :disabled="!command"
          :loading="executeLoading"
          theme="primary"
          @click="handleExecuteCommand"
        >
          {{ $t('执行') }}
        </Button>
      </div>
      <Tab
        v-if="isTrpcApp"
        v-model:active="bodyOrParams"
        type="unborder-card"
      >
        <Tab.TabPanel
          label="body"
          name="body"
        >
          <MsEditor
            v-model="bodyContent"
            class="!h-[400px]"
            lang="json"
            :readonly="false"
            :title="$t('编辑器')"
          >
            <template #tools="{ editor }">
              <IconButton
                class="mr-[8px]"
                :desc="$t('格式化')"
                @click="handleFormat(editor)"
              >
                <template #icon>
                  <span class="bkms-icon bkms-icon-angle-left text-[12px] text-[#979BA5]"></span>
                  <span class="bkms-icon bkms-icon-angle-right text-[12px] text-[#979BA5]"></span>
                </template>
              </IconButton>
            </template>
          </MsEditor>
        </Tab.TabPanel>
        <Tab.TabPanel
          label="params"
          name="params"
        >
          <Radio.Group
            v-model="paramsType"
            class="mb-[8px]"
            type="capsule"
          >
            <Radio.Button label="table">
              <span class="flex items-center">
                <i class="bkms-icon bkms-icon-single-column mr-[2px]"></i>
                <span>{{ $t('表格模式') }}</span>
              </span>
            </Radio.Button>
            <Radio.Button label="text">
              <span class="flex items-center">
                <span class="mr-[4px] h-[12px] leading-[12px] underline underline-offset-1">A</span>
                <span>{{ $t('文本模式') }}</span>
              </span>
            </Radio.Button>
          </Radio.Group>
          <Table
            v-show="paramsType === 'table'"
            border
            :data="tableData"
            :edit-config="{
              trigger: 'click',
              mode: 'cell',
              showIcon: false,
            }"
          >
            <TableColumn
              :edit-render="{}"
              :title="$t('参数名')"
            >
              <template #edit="{ row }">
                <Input
                  v-model.trim="row.key"
                  autofocus
                  clearable
                />
              </template>
              <template #default="{ row }">
                <span v-if="row.key">{{ row.key }}</span>
                <span
                  v-else
                  class="text-[#C4C6CC]"
                  >{{ $t('请输入') }}</span
                >
              </template>
            </TableColumn>
            <TableColumn
              :edit-render="{}"
              :title="$t('参数值')"
            >
              <template #edit="{ row }">
                <Input
                  v-model.trim="row.value"
                  autofocus
                  clearable
                />
              </template>
              <template #default="{ row }">
                <span v-if="row.value">{{ row.value }}</span>
                <span
                  v-else
                  class="text-[#C4C6CC]"
                  >{{ $t('请输入') }}</span
                >
              </template>
            </TableColumn>
            <TableColumn
              :title="$t('操作')"
              width="190"
            >
              <template #default="{ row }">
                <Button
                  size="small"
                  text
                  @click="handleAdd()"
                >
                  <i class="bkms-icon bkms-icon-plus-circle-shape text-[#979BA5] text-[14px] mr-[10px]"></i>
                </Button>
                <Button
                  :disabled="tableData.length === 1"
                  size="small"
                  text
                  @click="handleDelete(row)"
                >
                  <i
                    :class="[
                      'bkms-icon bkms-icon-minus-circle-shape text-[14px]',
                      { 'text-[#979BA5]': tableData.length !== 1 },
                    ]"
                  >
                  </i>
                </Button>
              </template>
            </TableColumn>
          </Table>
          <Input
            v-show="paramsType === 'text'"
            v-model="textContent"
            :placeholder="$t('请输入参数名和参数值，如 {0}，多个参数换行分隔', ['A:1'])"
            :rows="10"
            type="textarea"
          />
        </Tab.TabPanel>
      </Tab>
    </ToggleCard>
    <ToggleCard
      v-if="isExecuted"
      ref="executeResultRef"
      class="mt-[24px]"
      :name="$t('执行结果')"
      type="normal"
    >
      <Tab
        v-if="executeFullList.length > 0"
        v-model:active="responseType"
        type="unborder-card"
        @change="handleTabChange"
      >
        <Tab.TabPanel name="success">
          <template #label>
            {{ $t('执行成功') }}
            <Tag
              class="px-[8px]"
              radius="8px"
              size="small"
              :theme="responseType === 'success' ? 'info' : ''"
            >
              {{ successNum }}
            </Tag>
          </template>
        </Tab.TabPanel>
        <Tab.TabPanel
          :label="$t('执行失败')"
          name="failed"
        >
          <template #label>
            {{ $t('执行失败') }}
            <Tag
              class="px-[8px]"
              radius="8px"
              size="small"
              :theme="responseType === 'failed' ? 'danger' : ''"
            >
              {{ failedNum }}
            </Tag>
          </template>
        </Tab.TabPanel>
        <ResizeLayout
          v-if="executeResult.length > 0"
          :border="false"
          :class="['h-[500px]', { 'is-collapsed': !isCollapse }]"
          collapsible
          :initial-divide="160"
          :is-collapsed="isCollapse"
          :trigger-width="0"
          @collapse-change="handleCollapse"
        >
          <template #aside>
            <div class="h-full bg-[#F5F7FA] py-[8px] flex flex-col w-[160px]">
              <div class="font-bold h-[32px] leading-[32px] shrink-0 px-[12px]">
                {{ `${$t('实例列表')}( ${executeResult.length} )` }}
              </div>
              <div class="flex-1 overflow-auto">
                <div
                  v-for="(item, index) in executeResult"
                  :key="`${index}-${item.instanceID}`"
                  :class="[
                    'h-[32px] leading-[32px] px-[12px]',
                    'hover:bg-[#f0f1f5] cursor-pointer',
                    { '!bg-[#d1e8ff] text-[#3a84ff]': curInstance?.instanceID === item.instanceID },
                  ]"
                  @click="handleClickInstance(item)"
                >
                  <OverflowTitle>{{ item.instanceID }}</OverflowTitle>
                </div>
              </div>
            </div>
          </template>
          <template #main>
            <MsEditor
              class="h-full"
              lang="json"
              :model-value="curInstance?.detail"
              readonly
            >
              <template #tools="{ editor }">
                <IconButton
                  class="mr-[8px]"
                  :desc="$t('格式化')"
                  @click="handleFormat(editor)"
                >
                  <template #icon>
                    <span class="bkms-icon bkms-icon-angle-left text-[12px] text-[#979BA5]"></span>
                    <span class="bkms-icon bkms-icon-angle-right text-[12px] text-[#979BA5]"></span>
                  </template>
                </IconButton>
              </template>
              <template #title>
                <div class="flex items-center">
                  <Success
                    v-if="responseType === 'success'"
                    fill="#2CAF5E"
                    height="16"
                    width="16"
                  />
                  <Close
                    v-else
                    fill="#EA3636"
                    height="16"
                    width="16"
                  />
                  <span class="ml-[6px]">{{ curInstance?.instanceID }}</span>
                </div>
              </template>
            </MsEditor>
          </template>
        </ResizeLayout>
        <Exception
          v-else
          class="mb-[50px]"
          type="empty"
        >
          <template #title>
            <span
              v-if="responseType === 'success'"
              class="text-[#4D4F56] text-[14px]"
              >{{ $t('无执行成功的结果') }}</span
            >
            <span
              v-else
              class="text-[#4D4F56] text-[14px]"
              >{{ $t('无执行失败的结果') }}</span
            >
          </template>
        </Exception>
      </Tab>
      <Exception
        v-else
        class="mb-[50px]"
        type="empty"
      >
        <template #title>
          <span class="text-[#4D4F56] text-[14px]">{{ $t('执行结果为空') }}</span>
        </template>
        <template #description>
          <span class="text-[#979BA5] text-[12px]">{{
            $t('请先选择命令，填写相应的参数，然后点击执行按钮发送请求')
          }}</span>
        </template>
      </Exception>
    </ToggleCard>
    <Alert
      v-if="requestFailed"
      class="mt-[12px]"
      theme="danger"
    >
      <template #title>
        <p class="font-bold text-[#4D4F56] mb-[4px]">{{ $t('API 请求失败') }}:</p>
        <p class="text-[#4D4F56]">{{ `${$t('状态')}: ${requestFailed.status}` }}</p>
        <p>{{ `${JSON.stringify(requestFailed?.error || {}, null, 2)}` }}</p>
      </template>
    </Alert>
  </div>
</template>
<script lang="ts" setup>
  import { computed, nextTick, onMounted, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Button, Exception, Input, OverflowTitle, Radio, ResizeLayout, Select, Tab, Tag } from 'bkui-vue';
  import { Close, Success } from 'bkui-vue/lib/icon';
  import * as monaco from 'monaco-editor';
  import {
    ExecuteTrpcAdminCmdOutput,
    InstanceExecuteTafAdminCmdResultOutputObj,
    InstanceExecuteTrpcAdminCmdResultOutputObj,
    ListTrpcAdminCmdsOutputObjs,
  } from '~/@types/v1/instance';
  import { InstanceService } from '~/api/modules/v1';
  import ToggleCard from '~/components/toggle-card.vue';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import { DeployableAppType, useDeployAPIs } from '../../use-deploy';

  type InstanceExecuteAdminCmdResult =
    | InstanceExecuteTafAdminCmdResultOutputObj
    | InstanceExecuteTrpcAdminCmdResultOutputObj;

  interface IProps {
    data: string[];
  }

  interface RequestError {
    [key: string]: unknown;
    message?: string;
    status?: number;
    statusText?: string;
  }

  interface TableItem {
    _X_ROW_KEY?: string;
    key: string;
    value: string;
  }

  const props = defineProps<IProps>();

  const appDetail = useAppDetail();
  const trpcDeployStore = useTrpcDeployStore();
  const isTrpcApp = computed(() => appDetail?.appType === 'trpc');
  const deployAPIs = useDeployAPIs(appDetail.appType as DeployableAppType);

  const command = ref<string>('');
  const commandMethod = ref<'GET' | 'POST' | 'PUT'>('GET');
  const commandSelectRef = ref<InstanceType<typeof Select> | null>(null);
  const commandMethodSelectRef = ref<InstanceType<typeof Select> | null>(null);
  const bodyOrParams = ref<'body' | 'params'>('body');
  const paramsType = ref<'table' | 'text'>('table');
  const bodyContent = ref<string>('{}'); // 编辑器内容
  const commandList = ref<ListTrpcAdminCmdsOutputObjs['results']>([]);
  const loading = ref<boolean>(false);
  const textContent = ref<string>(''); // 文本内容
  const isExecuted = ref<boolean>(false); // 是否执行成功

  const responseType = ref<'failed' | 'success'>('success');
  const executeResultRef = ref<InstanceType<typeof ToggleCard> | null>(null);
  const requestFailed = ref<null | RequestError>(null);

  type ErrorResponse = {
    error?: {
      details?: Array<{
        code?: string;
      }>;
    };
  };

  const adminPrecheckFailed = ref(false);

  const checkAdminPrecheckFailed = (err: unknown) => {
    // 只有trpc应用才需要判断
    if (!isTrpcApp.value) return false;
    const details = (err as ErrorResponse)?.error?.details;
    return Array.isArray(details) && details.some(d => d.code === 'TRPC_ADMIN_PRECHECK_FAILED');
  };

  async function handleGetCommands() {
    loading.value = true;
    adminPrecheckFailed.value = false;
    try {
      const res = await InstanceService.listTrpcAdminCmds(
        {
          appID: appDetail.appID,
          envName: trpcDeployStore.curEnvItem!.name ?? '',
          instanceIDs: props.data,
        },
        { interceptorErr: false },
      );
      commandList.value = res.results;
    } catch (err: unknown) {
      adminPrecheckFailed.value = checkAdminPrecheckFailed(err);
      commandList.value = [];
    } finally {
      loading.value = false;
    }
  }

  const executeLoading = ref<boolean>(false);
  const executeResult = computed<InstanceExecuteAdminCmdResult[]>(() =>
    executeFullList.value.filter(
      item => (item.success && responseType.value === 'success') || (!item.success && responseType.value === 'failed'),
    ),
  );
  const executeFullList = ref<InstanceExecuteAdminCmdResult[]>([]);
  const successNum = ref<number>(0);
  const failedNum = ref<number>(0);
  /** 执行命令 */
  async function handleExecuteCommand() {
    let params = {} as Record<string, unknown>;
    if (isTrpcApp.value) {
      const data = tableData.value.reduce(
        (pre, cur) => {
          if (!cur.key) return pre;
          pre[cur.key] = cur.value;
          return pre;
        },
        {} as Record<string, boolean | number | string>,
      );

      const trpcParams =
        commandMethod.value !== 'GET'
          ? { body: bodyContent.value }
          : {
              params: paramsType.value === 'table' ? data : parseKeyValueStringAdvanced(textContent.value),
            };

      params = {
        ...trpcParams,
        url: command.value,
        method: commandMethod.value,
      };
    } else {
      params = {
        command: command.value,
      };
    }

    executeLoading.value = true;
    deployAPIs.executeAdminCmd!(
      {
        appID: appDetail.appID,
        envName: trpcDeployStore.curEnvItem!.name,
        instanceIDs: props.data,
        ...params,
      },
      { originalResponse: true, needStatus: true, interceptorErr: false },
    )
      .then(response => {
        const result = response as unknown as Response;
        if (result.status < 200 || result.status >= 300) {
          throw new Error(`HTTP error! status: ${result.status}`);
        }
        return result.json();
      })
      .then(res => {
        // 接口请求成功
        handleSuccess(res);
      })
      .catch(err => {
        if (checkAdminPrecheckFailed(err)) {
          adminPrecheckFailed.value = true;
        }
        // 接口请求失败
        isExecuted.value = false; // 执行失败
        executeFullList.value = [];
        requestFailed.value = err;
      })
      .finally(() => {
        executeLoading.value = false;
      });
  }
  /** 处理成功 */
  function handleSuccess(res: ExecuteTrpcAdminCmdOutput) {
    executeFullList.value = res!.data!.results || [];
    successNum.value = executeFullList.value.filter(item => item.success).length;
    failedNum.value = executeFullList.value.length - successNum.value;
    responseType.value = successNum.value > 0 ? 'success' : 'failed';
    isExecuted.value = true; // 执行成功
    requestFailed.value = null; // 清空错误对象
    adminPrecheckFailed.value = false;
    if (executeFullList.value.length > 0) {
      curInstance.value = executeFullList.value.find(
        item =>
          (item.success && responseType.value === 'success') || (!item.success && responseType.value === 'failed'),
      );
    }
    // 定位执行结果到页面中间
    nextTick(() => {
      executeResultRef.value?.$el?.scrollIntoView({ behavior: 'smooth', block: 'center' });
    });
  }

  /**
   * @description 解析键值对字符串
   * @param str
   * @param options
   */
  function parseKeyValueStringAdvanced(
    str: string,
    options: {
      delimiter?: string;
      lowercaseKeys?: boolean;
      parseBooleans?: boolean;
      parseNumbers?: boolean;
      trimValues?: boolean;
      uppercaseKeys?: boolean;
    } = {},
  ) {
    const {
      delimiter = ':', // 分隔符，默认为冒号
      trimValues = true, // 是否修剪值
      parseNumbers = true, // 是否尝试解析数字
      parseBooleans = true, // 是否尝试解析布尔值
      lowercaseKeys = false, // 是否将键转换为小写
      uppercaseKeys = false, // 是否将键转换为大写
    } = options;

    const result = {} as Record<string, boolean | number | string>;

    // 按换行符分割字符串
    const lines = str.split('\n');

    for (const line of lines) {
      // 跳过空行和注释行（以#开头的行）
      if (!line.trim() || line.trim().startsWith('#')) continue;

      // 查找第一个分隔符的位置
      const delimiterIndex = line.indexOf(delimiter);

      // 如果没有找到分隔符，跳过该行
      if (delimiterIndex === -1) continue;

      // 提取键和值
      const key = line.substring(0, delimiterIndex);
      let value = line.substring(delimiterIndex + 1);

      // 处理键
      let processedKey = key;
      if (trimValues) processedKey = processedKey.trim();
      if (lowercaseKeys) processedKey = processedKey.toLowerCase();
      if (uppercaseKeys) processedKey = processedKey.toUpperCase();

      // 处理值
      let processedValue: boolean | number | string = value;
      if (trimValues) processedValue = processedValue.trim();

      // 尝试解析值的类型
      if (parseNumbers && typeof processedValue === 'number' && !isNaN(processedValue) && processedValue !== '') {
        processedValue = Number(processedValue);
      } else if (parseBooleans) {
        const lowerValue = processedValue.toLowerCase();
        if (lowerValue === 'true') processedValue = true;
        else if (lowerValue === 'false') processedValue = false;
      }

      // 添加到结果对象
      result[processedKey] = processedValue;
    }

    return result;
  }

  const tableData = ref<TableItem[]>([{ key: '', value: '' }]);

  // 格式化代码
  async function handleFormat(
    editor: monaco.editor.IStandaloneCodeEditor | monaco.editor.IStandaloneDiffEditor | null,
  ) {
    if (!editor) return;

    const codeEditor = 'getModifiedEditor' in editor ? editor.getModifiedEditor() : editor;
    const model = codeEditor.getModel();
    if (!model) return;

    try {
      // 先让编辑器获得焦点
      codeEditor.focus();

      const currentValue = model.getValue();
      try {
        // 解析并格式化 JSON
        const parsed = JSON.parse(currentValue);
        const formatted = JSON.stringify(parsed, null, 2);
        model.setValue(formatted);
        return;
      } catch (e) {
        console.error('JSON parse error:', e);
      }
    } catch (error) {
      console.error('Format error:', error);
    }
  }

  // 表格数据转文本
  function tableToText(data: TableItem[]): string {
    return data
      .filter(item => item.key) // 只保留有 key 的行
      .map(item => `${item.key}:${item.value}`)
      .join('\n');
  }

  // 文本转表格数据
  function textToTable(text: string): TableItem[] {
    if (!text.trim()) {
      return [{ key: '', value: '' }];
    }

    const lines = text.split('\n');
    const result: TableItem[] = [];

    for (const line of lines) {
      const trimmedLine = line.trim();

      // 跳过空行
      if (!trimmedLine) continue;

      // 查找冒号分隔符
      const delimiterIndex = trimmedLine.indexOf(':');

      // 如果没有冒号，或者冒号在开头，忽略该行
      if (delimiterIndex <= 0) continue;

      const key = trimmedLine.substring(0, delimiterIndex).trim();
      const value = trimmedLine.substring(delimiterIndex + 1).trim();

      // 只添加有 key 的行（key 不为空）
      if (key) {
        result.push({ key, value });
      }
    }

    // 如果没有有效数据，返回一个空行
    return result.length > 0 ? result : [{ key: '', value: '' }];
  }

  // 监听模式切换，进行数据同步
  watch(paramsType, (newType, oldType) => {
    if (oldType === 'table' && newType === 'text') {
      // 从表格切换到文本：同步表格数据到文本
      textContent.value = tableToText(tableData.value);
    } else if (oldType === 'text' && newType === 'table') {
      // 从文本切换到表格：同步文本数据到表格
      tableData.value = textToTable(textContent.value);
    }
  });

  function handleAdd() {
    tableData.value.push({
      key: '',
      value: '',
    });
  }
  function handleDelete(row: TableItem) {
    const index = tableData.value.findIndex(item => item._X_ROW_KEY === row._X_ROW_KEY);
    tableData.value.splice(index, 1);
  }

  const curInstance = ref<InstanceExecuteAdminCmdResult | undefined>();
  /**
   * @description 点击实例
   */
  function handleClickInstance(row: InstanceExecuteAdminCmdResult) {
    curInstance.value = row;
  }

  const isCollapse = ref<boolean>(false);
  /**
   * @description 折叠
   */
  function handleCollapse(collapse: boolean) {
    isCollapse.value = collapse;
  }
  /**
   * @description 配置命令区域折叠时关闭 Select 下拉框
   */
  function handleConfigCommandCollapse(active: boolean) {
    if (!active) {
      commandSelectRef.value?.hidePopover?.();
      commandMethodSelectRef.value?.hidePopover?.();
    }
  }
  /**
   * @description 切换结果类型
   */
  async function handleTabChange() {
    await nextTick();
    curInstance.value = executeResult.value?.[0];
  }
  /**
   * @description 查看指引
   */
  function handleViewGuide() {
    window.open(`${window.BK_DOC_URL}/p/4016336887`, '_blank');
  }

  onMounted(() => {
    if (isTrpcApp.value) {
      handleGetCommands();
    }
  });
</script>
<style lang="postcss" scoped>
  :deep(.bk-tab-content) {
    padding: 8px 0 !important;
  }
  :deep(.bk-resize-layout-aside) {
    border: 0;
  }
  :deep(.is-collapsed .bk-resize-layout-main) {
    padding-left: 20px;
  }
  :deep(.select-group .bk-input) {
    border-radius: 0;
  }
</style>
