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
  <div class="host-port-page">
    <!-- 骨架屏 -->
    <template v-if="loading">
      <Layout.shape
        class="mb-[16px] !block w-full rounded-[2px]"
        height="40px"
        type="rect"
        width="100%"
      />
      <div class="overflow-hidden rounded-[2px] shadow-[0_2px_4px_0_#1919290d]">
        <div class="flex h-[32px] items-center bg-[#eaebf0] px-[16px]">
          <Layout.shape
            height="16px"
            type="rect"
            width="64px"
          />
        </div>
        <div class="bg-[#FFF] p-[16px]">
          <div class="host-port-panel">
            <!-- 左侧表格区域：表头 + 2 行 -->
            <div class="min-w-0 flex-1">
              <div class="host-port-row host-port-row--head gap-[24px]">
                <Layout.shape
                  height="14px"
                  type="rect"
                  width="56px"
                />
                <Layout.shape
                  height="14px"
                  type="rect"
                  width="88px"
                />
              </div>
              <div
                v-for="rowIndex in SKELETON_TABLE_ROWS"
                :key="rowIndex"
                class="host-port-row gap-[38px] border-b border-[var(--host-port-border)] last:border-b-0"
              >
                <Layout.shape
                  height="14px"
                  type="rect"
                  width="40px"
                />
                <Layout.shape
                  height="22px"
                  type="rect"
                  width="240px"
                />
              </div>
            </div>
            <!-- 右侧生效范围区域 -->
            <div class="host-port-scope">
              <div class="host-port-row host-port-row--head">
                <Layout.shape
                  height="14px"
                  type="rect"
                  width="56px"
                />
              </div>
              <div class="flex flex-col gap-[8px] px-[16px] py-[12px]">
                <Layout.shape
                  v-for="(width, envIndex) in SKELETON_SCOPE_ENV_WIDTHS"
                  :key="envIndex"
                  height="18px"
                  type="rect"
                  :width="`${width}px`"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 没有联邦集群环境时整个功能用不上，不给表单 -->
    <div
      v-else-if="showEmptyGuide"
      class="host-port-empty-guide bg-[#FFF] rounded-[2px] py-[60px]"
    >
      <Exception
        scene="part"
        type="empty"
      >
        <template #type>
          <img
            class="h-[140px]"
            src="/empty.svg"
          />
        </template>
        <template #description>
          <p class="text-[16px] text-[#313238]">{{ $t('当前空间没有绑定联邦集群的环境') }}</p>
          <p class="mt-[8px] text-[12px] text-[#979BA5]">
            {{ $t('HostPort 目前仅对绑定联邦集群的环境生效') }}
          </p>
        </template>
      </Exception>
    </div>

    <template v-else>
      <Alert
        class="mb-[16px]"
        theme="info"
      >
        <template #title>
          <span>
            {{
              $t(
                '环境绑定联邦集群时，容器需要通过所在节点的 HostPort 对外暴露访问。声明端口后，部署时会为容器端口自动映射节点端口，并将节点端口注入容器环境变量。',
              )
            }}
            <Button
              text
              theme="primary"
              @click="handleHostPortDoc"
            >
              {{ $t('查看详细文档') }}
            </Button>
          </span>
        </template>
      </Alert>

      <BkmsContent
        class="info-title shadow-[0_2px_4px_0_#1919290d]"
        :show-edit-icon="!isEditing"
        :title="$t('端口声明')"
        @edit="handleEdit"
      >
        <div class="bg-[#FFF] p-[16px]">
          <!-- 查看态：无端口且无待部署用空态；清空端口但仍待部署时展示「未配置」行 + 生效范围 -->
          <template v-if="!isEditing">
            <Exception
              v-if="!showPortTable"
              class="normal-exception py-[24px]"
              scene="part"
              type="empty"
            >
              <template #type>
                <img
                  class="h-[100px]"
                  src="/empty.svg"
                />
              </template>
              <template #description>
                <span>{{ $t('尚未配置') }}，</span>
                <Button
                  text
                  theme="primary"
                  @click="handleEdit"
                >
                  {{ $t('立即配置') }}
                </Button>
              </template>
            </Exception>
            <div
              v-else
              class="host-port-panel text-[12px]"
            >
              <table class="min-w-0 flex-1 border-collapse">
                <thead>
                  <tr class="host-port-row--head border-b border-[var(--host-port-border)] text-[#313238]">
                    <th class="host-port-cell w-[120px] text-left font-normal">{{ $t('容器端口') }}</th>
                    <th class="host-port-cell text-left font-normal">{{ $t('节点端口的环境变量') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <!-- 已删除全部端口、与线上不一致待部署：端口列展示「未配置」，环境变量列留空 -->
                  <tr
                    v-if="declaredPorts.length === 0"
                    class="border-b border-[var(--host-port-border)] last:border-b-0"
                  >
                    <td class="host-port-cell text-[#979BA5]">{{ $t('未配置') }}</td>
                    <td class="host-port-cell"></td>
                  </tr>
                  <tr
                    v-for="row in declaredPortRows"
                    :key="row.port"
                    class="border-b border-[var(--host-port-border)] last:border-b-0"
                  >
                    <td class="host-port-cell text-[#313238]">{{ row.port }}</td>
                    <td class="host-port-cell">
                      <div class="flex items-center flex-nowrap">
                        <span class="host-port-env-tag">{{ row.envName }}</span>
                        <Copy
                          class="ml-[4px] shrink-0 cursor-pointer text-[#979BA5] hover:text-[#3A84FF]"
                          @click="handleCopy(row.envName)"
                        />
                      </div>
                    </td>
                  </tr>
                </tbody>
              </table>
              <div class="host-port-scope">
                <div class="host-port-row host-port-row--head text-[#313238]">
                  {{ $t('生效范围')
                  }}<span
                    v-if="pendingEnvCount > 0"
                    class="text-[#979BA5]"
                    >（{{ $t('{count} 个环境待部署生效', { count: pendingEnvCount }) }}）</span
                  >
                </div>
                <div class="px-[16px] py-[12px]">
                  <ScopeEnvList
                    :env-states="envStates"
                    :envs="scopeEnvs"
                    @go-deploy="goDeployEnv"
                  />
                </div>
              </div>
            </div>
          </template>

          <!-- 编辑态：一份应用级端口声明，可增删 -->
          <Form
            v-else
            ref="formRef"
            form-type="vertical"
            :model="formModel"
          >
            <!-- 列标题：所有端口行共用 -->
            <div class="mb-[8px] flex items-center min-w-0 text-[12px] leading-[20px] text-[#63656E]">
              <span class="w-[160px] shrink-0">{{ $t('容器端口') }}</span>
              <span class="ml-[12px]">{{ $t('节点端口的环境变量') }}</span>
            </div>

            <div
              v-for="(portRow, portIndex) in formModel.ports"
              :key="portRow.id"
              class="flex items-center mb-[10px] min-w-0"
            >
              <Form.FormItem
                class="!mb-0 w-[160px] shrink-0"
                error-display-type="tooltips"
                :property="`ports.${portIndex}.port`"
                :rules="portRules"
              >
                <Input
                  v-model.trim="portRow.port"
                  :placeholder="$t('如 8080')"
                />
              </Form.FormItem>
              <div class="ml-[12px] min-w-0 text-[12px]">
                <!-- 合法端口才展示环境变量名；非法 / 空输入显示占位 -->
                <div
                  v-if="hostPortEnvName(portRow.port)"
                  class="flex items-center"
                >
                  <span class="host-port-env-tag host-port-env-tag--edit">
                    {{ hostPortEnvName(portRow.port) }}
                  </span>
                  <Button
                    class="shrink-0 ml-[4px]"
                    text
                    @click="handleCopy(hostPortEnvName(portRow.port))"
                  >
                    <Copy
                      class="text-[#979BA5] hover:text-[#3A84FF]"
                      height="12px"
                      width="12px"
                    />
                  </Button>
                </div>

                <span
                  v-else
                  class="text-[#C4C6CC]"
                >
                  {{ $t('填写容器端口后自动生成') }}
                </span>
              </div>
              <Button
                class="shrink-0 ml-[8px]"
                text
                @click="handleRemovePort(portIndex)"
              >
                <Del
                  class="text-[#979BA5] hover:text-[#4D4F56]"
                  height="14px"
                  width="14px"
                />
              </Button>
            </div>

            <Button
              text
              theme="primary"
              @click="handleAddPort"
            >
              <Plus class="text-[18px]" />
              {{ $t('添加容器端口') }}
            </Button>

            <div class="mt-[16px] pt-[16px] border-t border-[var(--host-port-border)]">
              <div class="flex items-center gap-[8px] mb-[8px]">
                <span class="text-[14px] leading-[22px] text-[#313238]">{{ $t('生效范围') }}</span>
                <span class="text-[12px] leading-[20px] text-[#979BA5]">{{ $t('对所有绑定联邦集群的环境生效') }}</span>
              </div>
              <div class="bg-[#F5F7FA] rounded-[2px] px-[16px] py-[12px]">
                <ScopeEnvList
                  :env-states="envStates"
                  :envs="scopeEnvs"
                  :show-pending="false"
                />
              </div>
              <div class="mt-[16px]">
                <Button
                  :loading="saving"
                  theme="primary"
                  @click="handleSave"
                >
                  {{ $t('保存') }}
                </Button>
                <Button
                  class="ml-[8px]"
                  :disabled="saving"
                  @click="handleCancel"
                >
                  {{ $t('取消') }}
                </Button>
              </div>
            </div>
          </Form>
        </div>
      </BkmsContent>
    </template>
  </div>
</template>

<script lang="ts" setup>
  import { computed, reactive, ref, watch } from 'vue';

  import { Alert, Button, Exception, Form, Input, Message } from 'bkui-vue';
  import { Copy, Del, Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { EnvService, HostportService } from '~/api/modules/v1';
  import { DOC_LINKS } from '~/common/const';
  import BkmsContent from '~/components/bkms-content.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import { isAppModelAppType } from '~/composables/app-type';
  import { useCopy } from '~/composables/use-copy';
  import { useGoDeployEnv } from '~/composables/use-go-deploy-env';
  import { useAppDetail } from '~/stores/app-detail';

  import {
    buildEnvInfoMap,
    compareScopeEnv,
    countPendingEnvs,
    hasPortListChanged,
    hostPortEnvName,
    parseContainerPort,
  } from './host-port-utils';
  import ScopeEnvList from './scope-env-list.vue';

  import type { HostPortEnvStates, ScopeEnv } from './host-port-utils';
  import type { Form as FormType } from 'bkui-vue';
  import type { EnvOutput } from '~/@types/v1/env';
  import type { HostPortsOutput } from '~/@types/v1/hostport';

  interface PortRow {
    id: number;
    port: string;
  }

  /** 骨架屏表格行数（对齐查看态常见展示） */
  const SKELETON_TABLE_ROWS = 2;
  /** 骨架屏生效范围：竖向两长一短 */
  const SKELETON_SCOPE_ENV_WIDTHS = [160, 96, 160];

  const { t } = useI18n();
  const appDetailStore = useAppDetail();
  const { copyText } = useCopy();
  const { goDeployEnv } = useGoDeployEnv();

  const loading = ref(true);
  const saving = ref(false);
  const isEditing = ref(false);
  /** 应用级已声明端口（来自 GET /hostports） */
  const declaredPorts = ref<number[]>([]);
  /** 后端计算的各联邦环境待部署状态；key 即为生效范围环境名 */
  const envStates = ref<HostPortEnvStates>({});
  /** listAppEnvs 缓存，用于按环境名补齐 displayName / type */
  const appEnvList = ref<EnvOutput[]>([]);
  /** 稳定 Form model，避免模板内每次渲染新建对象 */
  const formModel = reactive<{ ports: PortRow[] }>({ ports: [] });
  const formRef = ref<InstanceType<typeof FormType> | null>(null);

  let seq = 0;
  const nextId = () => (seq += 1);

  /**
   * 无联邦环境时展示空态引导（不给端口配置表单）。
   * envStates 的 key 即为后端筛过的联邦环境；无联邦时为 {}。
   */
  const showEmptyGuide = computed(() => Object.keys(envStates.value).length === 0);

  /** 有待部署变更的环境数量，用于表头摘要 */
  const pendingEnvCount = computed(() => countPendingEnvs(envStates.value));

  /**
   * 展示端口表格 + 生效范围：
   * - 有端口声明时正常展示
   * - 已清空端口但仍与线上不一致（待部署）时也要展示，便于提示重新部署
   */
  const showPortTable = computed(() => declaredPorts.value.length > 0 || pendingEnvCount.value > 0);

  /** 查看态端口行：预计算环境变量名 */
  const declaredPortRows = computed(() =>
    declaredPorts.value.map(port => ({
      port,
      envName: hostPortEnvName(String(port)),
    })),
  );

  /**
   * 生效范围列表：
   * - 环境集合以 envStates 的 key 为准（后端已筛联邦环境）
   * - displayName / type 用 listAppEnvs 补齐，缺失时回退环境名（与北极星侧一致）
   */
  const scopeEnvs = computed<ScopeEnv[]>(() => {
    const envInfoMap = buildEnvInfoMap(appEnvList.value);

    return Object.keys(envStates.value)
      .map(envName => {
        const envInfo = envInfoMap.get(envName);
        return {
          name: envName,
          displayName: envInfo?.displayName || envName,
          type: envInfo?.type,
        } satisfies ScopeEnv;
      })
      .sort(compareScopeEnv);
  });

  /** 草稿端口转数字列表（校验通过后调用） */
  function draftPortNumbers() {
    return formModel.ports.map(item => parseContainerPort(item.port)!);
  }

  /** 跳转 HostPort 详细文档 */
  function handleHostPortDoc() {
    window.open(`${import.meta.env.BK_DOC_URL}${DOC_LINKS.HOST_PORT}`, '_blank');
  }

  const portRules = [
    {
      message: t('请输入容器端口'),
      trigger: 'blur',
      validator: (value: unknown) => !!String(value ?? '').trim(),
    },
    {
      message: t('端口需为 1-65535 之间的整数'),
      trigger: 'blur',
      validator: (value: unknown) => {
        const text = String(value ?? '').trim();
        return !text || parseContainerPort(text) != null;
      },
    },
    {
      message: t('端口重复'),
      trigger: 'blur',
      validator: (value: unknown) => {
        const text = String(value ?? '').trim();
        return !text || formModel.ports.filter(row => String(row.port ?? '').trim() === text).length <= 1;
      },
    },
  ];

  function applyHostPortsOutput(data: HostPortsOutput | null | undefined) {
    declaredPorts.value = [...(data?.ports || [])];
    envStates.value = { ...(data?.envStates || {}) };
  }

  function handleAddPort() {
    formModel.ports.push({ id: nextId(), port: '' });
  }

  function handleCancel() {
    isEditing.value = false;
    formModel.ports = [];
  }

  /** 复制环境变量名 */
  function handleCopy(value: string) {
    copyText(value);
  }

  function handleEdit() {
    formModel.ports = declaredPorts.value.map(port => ({ id: nextId(), port: String(port) }));
    // 无已声明端口时默认带一行空输入，避免用户再点一次「添加端口」
    if (formModel.ports.length === 0) {
      handleAddPort();
    }
    isEditing.value = true;
  }

  function handleRemovePort(index: number) {
    formModel.ports.splice(index, 1);
  }

  async function handleSave() {
    if (saving.value) return;

    const valid = await formRef.value?.validate().catch(() => false);
    if (!valid) return;

    const appID = appDetailStore.appID;
    if (!appID) return;

    const ports = draftPortNumbers();
    // 无增删则无需请求；顺序无关，与服务端升序去重语义一致
    if (!hasPortListChanged(declaredPorts.value, ports)) {
      handleCancel();
      return;
    }

    saving.value = true;
    try {
      // 一次 PUT 整表替换；接口直接返回 HostPortsOutput（无 data 包装），需 needRes
      const latest = await HostportService.putHostPorts({ appID, ports }, { needRes: true });
      applyHostPortsOutput(latest);

      Message({
        theme: 'success',
        message: pendingEnvCount.value > 0 ? t('保存成功，需重新部署绑定联邦集群的环境后才会生效') : t('保存成功'),
      });
      handleCancel();
    } catch {
      // 错误提示由全局 interceptor 负责；失败后重新拉取服务端状态
      await loadAll();
    } finally {
      saving.value = false;
    }
  }

  /** 异步返回后若已切走应用 / 类型，丢弃结果 */
  function isStaleLoad(expectedAppID: string) {
    return appDetailStore.appID !== expectedAppID || !isAppModelAppType(appDetailStore.appType);
  }

  async function loadAll() {
    const appID = appDetailStore.appID;
    // HostPort 仅服务应用模型；跨类型切换时 appID 可能先于卸载更新到 Helm
    if (!appID || !isAppModelAppType(appDetailStore.appType)) return;

    loading.value = true;
    handleCancel();
    /** 异步结束后是否已切走；finally 与分支共用，避免多次 isStaleLoad */
    let stale = false;
    try {
      // hostports 决定生效范围集合；listAppEnvs 只负责补齐展示名与类型 Tag
      // HostPorts 响应无 data 包装，需 needRes 取完整 body
      const [hostPorts, envs] = await Promise.all([
        HostportService.listHostPorts({ appID }, { needRes: true }),
        EnvService.listAppEnvs({ appID }).catch(() => [] as EnvOutput[]),
      ]);
      stale = isStaleLoad(appID);
      if (stale) return;
      applyHostPortsOutput(hostPorts);
      appEnvList.value = envs || [];
    } catch {
      stale = isStaleLoad(appID);
      if (stale) return;
      declaredPorts.value = [];
      envStates.value = {};
      appEnvList.value = [];
    } finally {
      if (!stale) {
        loading.value = false;
      }
    }
  }

  // 同时监听 type：仅 appID 变化时 type 可能仍是旧值，避免对 Helm 误调 HostPort 接口
  watch(
    [() => appDetailStore.appID, () => appDetailStore.appType],
    ([appID, appType]) => {
      if (!appID || !isAppModelAppType(appType)) return;
      loadAll();
    },
    { immediate: true },
  );
</script>

<style lang="postcss" scoped>
  .host-port-page {
    /* 查看态 / 骨架屏共用：行高、生效范围列宽、边框色 */
    --host-port-row-height: 42px;
    --host-port-scope-width: 280px;
    --host-port-border: #dcdee5;
  }

  .info-title :deep(.bkms-content-title) {
    background-color: #eaebf0;
  }

  /* 左侧端口表 + 右侧生效范围 */
  .host-port-panel {
    display: flex;
    overflow: hidden;
    border: 1px solid var(--host-port-border);
    border-radius: 2px;
  }

  .host-port-scope {
    width: var(--host-port-scope-width);
    flex-shrink: 0;
    border-left: 1px solid var(--host-port-border);
  }

  .host-port-row {
    display: flex;
    align-items: center;
    height: var(--host-port-row-height);
    padding: 0 16px;
  }

  .host-port-row--head {
    background: #f5f7fa;
    border-bottom: 1px solid var(--host-port-border);
  }

  .host-port-cell {
    height: var(--host-port-row-height);
    padding: 0 16px;
  }

  /* 用 span 模拟 Tag 外观，避免深层覆盖 bk-tag 内部结构 */
  .host-port-env-tag {
    display: inline-flex;
    flex: 0 0 auto;
    align-items: center;
    box-sizing: border-box;
    max-width: none;
    padding: 0 8px;
    overflow: visible;
    color: #63656e;
    font-size: 12px;
    line-height: 22px;
    white-space: nowrap;
    background: #f0f1f5;
    border-radius: 2px;
  }

  .host-port-env-tag--edit {
    height: 32px;
  }

  .host-port-empty-guide {
    :deep(.bk-exception-img) {
      width: auto !important;
      height: auto !important;
    }
  }
</style>
