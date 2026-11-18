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
  <div class="flex h-full min-h-0 flex-col overflow-hidden bg-[#f5f7fa]">
    <TabHeader
      v-model:active-tab="activeTab"
      :tabs="tabList"
      :title="$t('业务配置')"
    >
      <template #title-extra>
        <Button
          class="float-right"
          :disabled="loadState.status !== 'ready'"
          text
          theme="primary"
          @click="openDialog"
        >
          <i
            aria-hidden="true"
            class="bkms-icon bkms-icon-setting-line mr-[4px] text-[14px]"
          ></i>
          {{ $t('配置七彩石') }}
        </Button>
      </template>
    </TabHeader>

    <div class="flex min-h-0 flex-1 px-[24px] py-[20px]">
      <div
        v-bkloading="{ loading: loadState.status === 'loading' }"
        class="flex min-h-0 w-full flex-1 bg-white"
      >
        <iframe
          v-if="loadState.status === 'ready' && loadState.url"
          class="min-h-0 w-full flex-1 border-0"
          :src="loadState.url"
          :title="$t('七彩石')"
        ></iframe>
        <Exception
          v-else-if="loadState.status === 'ready'"
          class="large-exception m-auto"
          scene="part"
          type="empty"
        >
          <template #type>
            <img
              alt=""
              src="/empty.svg"
            />
          </template>
          <template #description>
            <div class="mt-[12px] text-[24px] leading-[32px] text-[#313238]">{{ $t('暂未配置七彩石') }}</div>
            <div class="mt-[4px] text-[14px]">{{ $t('配置页面地址后，会在下方嵌入七彩石页面。') }}</div>
          </template>
          <Button
            theme="primary"
            @click="openDialog"
          >
            {{ $t('配置七彩石') }}
          </Button>
        </Exception>
        <Exception
          v-else-if="loadState.status === 'error'"
          class="m-auto large-exception py-[18px]"
          scene="part"
          type="500"
        >
          <template #description>
            <span class="text-[14px]">{{ $t('七彩石配置加载失败') }}</span>
          </template>
          <Button
            theme="primary"
            @click="loadConfig"
          >
            {{ $t('重试') }}
          </Button>
        </Exception>
      </div>
    </div>

    <Dialog
      v-model:is-show="dialogVisible"
      :close-icon="!saving"
      :esc-close="!saving"
      :quick-close="false"
      render-directive="if"
      :title="$t('配置七彩石')"
      :width="640"
    >
      <Form
        form-type="vertical"
        @submit="saveConfig"
      >
        <Form.FormItem
          v-if="hasSavedUrl"
          :label="$t('启用七彩石')"
        >
          <div class="flex items-start gap-[8px]">
            <Switcher
              v-model="draftEnabled"
              :aria-checked="draftEnabled"
              :aria-describedby="draftEnabled ? undefined : 'rainbow-disable-hint'"
              :aria-disabled="saving"
              :aria-label="$t('启用七彩石')"
              class="shrink-0"
              :disabled="saving"
              role="switch"
              theme="primary"
              @keydown.space.prevent="!saving && (draftEnabled = !draftEnabled)"
            />
            <p
              v-if="!draftEnabled"
              id="rainbow-disable-hint"
              class="text-[12px] leading-[20px] text-[#979ba5]"
            >
              {{ $t('关闭后，当前应用将不再嵌入七彩石页面，七彩石上的原有配置不受影响。') }}
            </p>
          </div>
        </Form.FormItem>
        <Form.FormItem
          v-if="!hasSavedUrl || draftEnabled"
          :class="{ 'is-error': validationError }"
          :label="$t('七彩石页面地址')"
          required
        >
          <Input
            v-model="draftUrl"
            aria-describedby="rainbow-url-hint rainbow-url-error"
            :aria-label="$t('七彩石页面地址')"
            :disabled="saving"
            :placeholder="$t('请输入')"
            @input="validationError = ''"
          />
          <p
            id="rainbow-url-hint"
            class="mt-[8px] text-[12px] leading-[20px] text-[#979ba5]"
          >
            {{ $t('只支持 https。建议填写适配 iframe、没有顶部导航的七彩石页面地址。') }}
          </p>
          <p
            v-show="validationError"
            id="rainbow-url-error"
            class="mt-[4px] text-[12px] text-[#ea3636]"
            role="alert"
          >
            {{ validationError }}
          </p>
        </Form.FormItem>
      </Form>
      <div
        v-if="savePermission"
        class="mt-[8px] flex items-center gap-[8px] text-[12px]"
        role="alert"
      >
        <span>{{ $t('没有权限修改七彩石配置') }}</span>
        <a
          v-if="savePermission.applyUrl"
          :href="savePermission.applyUrl"
          rel="noopener noreferrer"
          target="_blank"
        >
          {{ $t('申请成为开发者') }}
        </a>
        <span v-else>{{ $t('请联系空间管理员') }}</span>
      </div>
      <template #footer>
        <Button
          class="mr-[8px]"
          :loading="saving"
          theme="primary"
          @click="saveConfig"
        >
          {{ $t('确定') }}
        </Button>
        <Button
          :disabled="saving"
          @click="dialogVisible = false"
        >
          {{ $t('取消') }}
        </Button>
      </template>
    </Dialog>
  </div>
</template>

<script lang="ts">
  // 以 appID 为键记录进行中的保存请求：页面重新挂载时，仍需等待同一应用此前发出的保存请求。
  const pendingSaves = new Map<string, Promise<string>>();
</script>

<script setup lang="ts">
  import { computed, onBeforeUnmount, ref, watch } from 'vue';

  import { Button, Dialog, Exception, Form, Input, Message, Switcher } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { RainbowtconfService } from '~/api/modules/v1/rainbowtconf';
  import { showApiErrorMessage } from '~/common/util';
  import TabHeader from '~/components/tab-header.vue';
  import { useUrlQuerySync } from '~/composables/use-url-query-sync';
  import { useAppDetail } from '~/stores/app-detail';

  // 后端错误响应结构（仅声明本页用到的字段）
  interface ApiError {
    error?: {
      code?: string;
      details?: Array<{ code?: string; extras?: { developer?: string } }>;
    };
  }

  // 页面加载状态：ready（url 为已配置的页面地址）、loading、error
  type LoadState = { status: 'error' | 'loading' } | { status: 'ready'; url: string };

  // IAM 无权限信息，applyUrl 为申请加入用户组的链接（可能为空）
  interface PermissionDenied {
    applyUrl: string;
  }

  // 应用详情「业务配置」页：加载七彩石配置并以 iframe 嵌入，支持配置 / 停用 / 修改
  const { t } = useI18n();
  const appDetail = useAppDetail();

  // 页面加载状态，及是否已有保存过的地址
  const loadState = ref<LoadState>({ status: 'loading' });
  const hasSavedUrl = computed(() => loadState.value.status === 'ready' && Boolean(loadState.value.url));

  // 弹窗状态：开关与地址草稿、校验错误、保存时的无权限提示
  const dialogVisible = ref(false);
  const draftEnabled = ref(true);
  const draftUrl = ref('');
  const validationError = ref('');
  const savePermission = ref<null | PermissionDenied>(null);
  const saving = ref(false);

  // 请求版本号：递增使旧响应失效，避免过期请求覆盖当前状态
  let requestRevision = 0;

  // 唯一页签「七彩石」，选中态与地址栏 query 同步
  const rainbowTab = 'rainbow';
  const tabList = computed(() => [{ name: rainbowTab, label: t('七彩石') }]);
  const {
    fields: { activeTab },
  } = useUrlQuerySync({ activeTab: { queryKey: 'activeTab', data: { allowed: [rainbowTab], default: rainbowTab } } });

  // 从接口错误解析 IAM 无权限信息并拼接申请链接；非权限错误返回 null
  function getPermissionDenied(cause: unknown): null | PermissionDenied {
    const { error } = (cause || {}) as ApiError;
    const detail = Array.isArray(error?.details)
      ? error.details.find(item => item?.code === 'IAM_NO_PERMISSION')
      : undefined;
    if (error?.code !== 'IAM_NO_PERMISSION' && !detail) return null;

    const group = detail?.extras?.developer;
    const iamUrl = import.meta.env.BK_IAM_URL?.trim().replace(/\/$/, '');
    return { applyUrl: group && iamUrl ? `${iamUrl}/apply-join-user-group?id=${encodeURIComponent(group)}` : '' };
  }

  // 判断响应是否仍属于当前应用的最新一次请求
  function isCurrentRequest(revision: number, appID: string) {
    return revision === requestRevision && appID === appDetail.appID;
  }

  // 拉取七彩石配置；若同应用存在未完成的保存请求，先等其结束后再读
  async function loadConfig() {
    const { appID } = appDetail;
    const revision = ++requestRevision;
    loadState.value = { status: 'loading' };
    if (!appID) return;

    const pendingSave = pendingSaves.get(appID);
    if (pendingSave) {
      // 保存失败也要重新读取，以后端当前值为准。
      await pendingSave.catch(() => {});
      if (!isCurrentRequest(revision, appID)) return;
    }

    try {
      const config = await RainbowtconfService.getRainbowTConf({ appID }, { interceptorErr: false });
      if (!isCurrentRequest(revision, appID)) return;

      // 接口只返回后端已验证的 HTTPS 地址。使用原始地址保留路径、查询和 hash。
      loadState.value = { status: 'ready', url: config.url ?? '' };
    } catch {
      if (!isCurrentRequest(revision, appID)) return;
      loadState.value = { status: 'error' };
    }
  }

  // 打开配置弹窗，以当前配置初始化草稿
  function openDialog() {
    if (loadState.value.status !== 'ready') return;
    draftEnabled.value = true;
    draftUrl.value = loadState.value.url;
    validationError.value = '';
    savePermission.value = null;
    dialogVisible.value = true;
  }

  // 保存配置；已保存且关闭「启用」开关时改为删除配置
  async function saveConfig() {
    const { appID } = appDetail;
    if (!appID || pendingSaves.has(appID) || !dialogVisible.value || loadState.value.status !== 'ready') return;

    // 关闭启用 → 删除；否则保存草稿地址
    const shouldDelete = hasSavedUrl.value && !draftEnabled.value;
    const url = draftUrl.value.trim();
    validationError.value = shouldDelete ? '' : validateUrl(url);
    if (validationError.value) return;

    const revision = requestRevision;
    saving.value = true;
    savePermission.value = null;
    try {
      const pendingSave = shouldDelete
        ? RainbowtconfService.deleteRainbowTConf({ appID }, { interceptorErr: false }).then(() => '')
        : RainbowtconfService.putRainbowTConf({ appID, url }, { interceptorErr: false }).then(
            config => config.url ?? '',
          );
      pendingSaves.set(appID, pendingSave);
      const savedUrl = await pendingSave;
      if (!isCurrentRequest(revision, appID)) return;

      loadState.value = { status: 'ready', url: savedUrl };
      dialogVisible.value = false;
      Message({ message: t('保存成功'), theme: 'success' });
    } catch (cause) {
      if (!isCurrentRequest(revision, appID)) return;

      savePermission.value = getPermissionDenied(cause);
      if (savePermission.value) return;

      // 后端校验不通过时回填为表单校验错误
      if (!shouldDelete && (cause as ApiError | null)?.error?.code === 'INVALID_ARGUMENT') {
        validationError.value = t('七彩石页面地址不合法，请检查后重试');
      } else {
        showApiErrorMessage(cause);
      }
    } finally {
      pendingSaves.delete(appID);
      if (isCurrentRequest(revision, appID)) saving.value = false;
    }
  }

  // 校验地址：必填、不超过 2048 字符、须为显式主机且不含用户名密码的 HTTPS 地址
  function validateUrl(url: string) {
    if (!url) return t('请输入七彩石页面地址');
    if (Array.from(url).length > 2048) return t('七彩石页面地址不能超过 2048 个字符');

    // URL 会补齐省略的主机分隔符，因此先要求输入包含显式主机段。
    const invalidUrl = t('请输入有效的 HTTPS 地址');
    const authority = /^https:\/\/([^/?#]+)/i.exec(url)?.[1];
    if (!authority || /[\s\\]/.test(authority)) return invalidUrl;
    if (authority.includes('@')) return t('七彩石页面地址不能包含用户名或密码');
    try {
      return new URL(url).hostname ? '' : invalidUrl;
    } catch {
      return invalidUrl;
    }
  }

  watch(
    () => appDetail.appID,
    () => {
      dialogVisible.value = false;
      saving.value = false;
      void loadConfig();
    },
    { immediate: true, flush: 'post' },
  );

  onBeforeUnmount(() => {
    requestRevision += 1;
  });
</script>
