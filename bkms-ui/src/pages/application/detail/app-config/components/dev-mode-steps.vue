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
  <div class="rounded-[2px] bg-[#F5F7FA] p-[16px]">
    <p class="mb-[12px]">{{ $t('开启后，仍需执行以下流程，才能使用开发模式') }}：</p>

    <div class="mb-[12px] flex items-center gap-x-[8px] leading-[20px] text-[#4D4F56]">
      <div class="h-[20px] w-[20px] rounded-[50%] bg-[#EAEBF0] text-center text-[14px] leading-[20px]">1</div>
      <span>{{ $t('执行部署') }}</span>
      <Button
        text
        theme="primary"
        @click="goToDeployment"
      >
        {{ $t('去部署') }}
        <AngleRight class="text-[16px]" />
      </Button>
    </div>

    <div class="mb-[12px] flex items-center gap-x-[8px] leading-[20px] text-[#4D4F56]">
      <div class="h-[20px] w-[20px] rounded-[50%] bg-[#EAEBF0] text-center text-[14px] leading-[20px]">2</div>
      <span>{{ $t('登录 bkms-cli') }}</span>
      <Button
        text
        theme="primary"
        @click="goToAccessToken"
      >
        {{ $t('查看 Token') }}
        <Share class="ml-[4px]" />
      </Button>
    </div>

    <div class="mb-[16px] ml-[28px]">
      <div class="mb-[12px] text-[#979BA5]">
        {{ $t('请点击【查看 Token】自行获取 access_token，复制后填入下方命令') }}
      </div>
      <div class="relative overflow-x-auto border border-[#DCDEE5] rounded-[2px] bg-[#FFF] p-[16px] pr-[72px]">
        <pre
          v-bk-xss-html="highlightedLoginCommand"
          class="m-0 whitespace-pre-wrap break-all bg-transparent! leading-[22px] text-[#4D4F56]"
        ></pre>
        <span
          class="absolute right-[10px] top-[10px] h-[24px] w-[24px] flex cursor-pointer items-center justify-center rounded-[2px] hover:bg-[#F0F1F5]"
          role="button"
          tabindex="0"
          :title="$t('复制')"
          @click="copyText(LOGIN_COMMAND)"
          @keydown.enter.space.prevent="copyText(LOGIN_COMMAND)"
        >
          <Copy
            fill="#3a84ff"
            height="16"
            width="16"
          />
        </span>
      </div>
    </div>

    <div class="mb-[12px] flex items-center gap-x-[8px] leading-[20px] text-[#4D4F56]">
      <div class="h-[20px] w-[20px] rounded-[50%] bg-[#EAEBF0] text-center text-[14px] leading-[20px]">3</div>
      <span>{{ $t('使用 bkms-cli 发布二进制') }}</span>
    </div>

    <div class="ml-[28px]">
      <div class="relative overflow-x-auto border border-[#DCDEE5] rounded-[2px] bg-[#FFF] p-[16px] pr-[72px]">
        <pre
          v-bk-xss-html="highlightedCommand"
          class="m-0 whitespace-pre-wrap break-all bg-transparent! leading-[22px] text-[#4D4F56]"
        ></pre>
        <span
          class="absolute right-[10px] top-[10px] h-[24px] w-[24px] flex cursor-pointer items-center justify-center rounded-[2px] hover:bg-[#F0F1F5]"
          role="button"
          tabindex="0"
          :title="$t('复制')"
          @click="copyText(publishCommand)"
          @keydown.enter.space.prevent="copyText(publishCommand)"
        >
          <Copy
            fill="#3a84ff"
            height="16"
            width="16"
          />
        </span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import { Button } from 'bkui-vue';
  import { AngleRight, Copy, Share } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { copyText } from '~/common/util';
  import { useAppDetail } from '~/stores/app-detail';
  import { useSpaceStore } from '~/stores/space';

  // Token 由用户在独立页面手动获取，当前组件不请求、不缓存 Token。
  const ACCESS_TOKEN_URL = `${import.meta.env.BK_API_PREFIX}/user_token/token?redirect_login=true`;
  const ACCESS_TOKEN_PLACEHOLDER = '<access-token>';
  const BINARY_PLACEHOLDER = '<binary-path>';
  const LOGIN_COMMAND = `bkms-cli login --access-token ${ACCESS_TOKEN_PLACEHOLDER}`;

  interface Props {
    envName?: string;
  }

  const props = defineProps<Props>();

  const { t } = useI18n();
  const router = useRouter();
  const appDetailStore = useAppDetail();
  const spaceStore = useSpaceStore();

  // 优先使用当前页面上下文；上下文尚未就绪时保留可识别的参数占位符。
  const currentWorkspace = computed(() => spaceStore.currentSpace || '<workspace-id>');
  const currentAppID = computed(() => appDetailStore.appID || '<app-id>');
  const currentEnvName = computed(() => props.envName || '<env-name>');
  const highlightedLoginCommand = highlightCommand(LOGIN_COMMAND);

  // 保留无样式的原始命令用于复制，保证复制结果可直接在终端中编辑执行。
  const publishCommand = computed(() =>
    [
      `# ${t('设置为当前空间，后续命令无需再带 --workspace')}`,
      `bkms-cli workspace set ${currentWorkspace.value}`,
      '',
      `# ${t('发布二进制到当前环境的所有运行中实例')}`,
      [
        'bkms-cli app publish',
        `  --app ${currentAppID.value}`,
        `  --env ${currentEnvName.value}`,
        `  -f ${BINARY_PLACEHOLDER}`,
        '  --all',
      ].join(' \\\n'),
    ].join('\n'),
  );

  // 展示时仅高亮注释和命令参数，不改变用于复制的原始命令。
  const highlightedCommand = computed(() => highlightCommand(publishCommand.value));

  // 命令通过 v-bk-xss-html 渲染，拼接高亮标签前先转义所有动态文本。
  function escapeHtml(text: string) {
    return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  }

  function goToAccessToken() {
    window.open(ACCESS_TOKEN_URL, '_blank');
  }

  function goToDeployment() {
    // 进入部署管理的实例页，并定位到当前开发环境。
    router.push({
      name: 'detail',
      params: {
        name: appDetailStore.appID,
        menuName: 'deployment',
        type: 'trpc',
      },
      query: {
        activeTab: 'instance',
        ...(props.envName ? { envName: props.envName } : {}),
      },
    });
  }

  function highlightCommand(command: string) {
    return escapeHtml(command)
      .split('\n')
      .map(line =>
        line.startsWith('#')
          ? `<span class="text-[#979BA5]">${line}</span>`
          : line.replace(/(^|\s)(--?[a-z][a-z-]*)/g, '$1<span class="text-[#3A84FF]">$2</span>'),
      )
      .join('\n');
  }
</script>
