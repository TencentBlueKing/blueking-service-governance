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
  <Popover
    :boundary="'parent'"
    placement="top-start"
    theme="light"
    trigger="click"
  >
    <slot />
    <template #content>
      <div class="push-guide-content w-[520px] p-[4px]">
        <div class="text-[14px] font-bold text-[#111214] leading-[22px] mb-[16px]">
          {{ $t('推送指引') }}
        </div>
        <div class="text-[12px] text-[#4A5973] leading-[20px] mb-[4px]">
          {{ $t('请将镜像推送至当前空间绑定仓库后使用。凭证可在「空间设置 → 基本信息」查看。') }}
        </div>
        <div class="bg-[#F6F7FB] px-[16px] py-[12px]">
          <FieldItem
            class="leading-[20px]"
            :container-height="20"
            :field-value="$t('仓库地址')"
            :field-width="60"
          >
            <template #value>
              <span class="push-guide-command-string push-guide-mono text-[14px] font-mono break-all !text-[#111214]">{{
                repositoryAddress
              }}</span>
              <Copy
                v-bk-tooltips="{
                  content: $t('复制仓库地址'),
                }"
                class="ml-[6px] cursor-pointer hover:text-[#3B6EF5] shrink-0"
                fill="#8F9FBD"
                :title="$t('复制')"
                @click="copyText(repositoryAddress)"
              >
              </Copy>
            </template>
          </FieldItem>
          <FieldItem
            class="mt-[12px] leading-[20px]"
            :container-height="20"
            :field-value="$t('仓库凭证')"
            :field-width="60"
          >
            <template #value>
              <SecretToggle
                :empty-value-placeholder="$t('暂无凭证')"
                icon-fill="#8F9FBD"
                :placeholder="'********'"
                :value="password"
                @click.stop
                @toggle="handlePasswordToggle"
              />
              <Copy
                v-if="password"
                v-bk-tooltips="{
                  content: $t('复制凭证'),
                }"
                class="ml-[6px] cursor-pointer hover:text-[#3B6EF5] shrink-0"
                fill="#8F9FBD"
                :title="$t('复制')"
                @click="copyText(password)"
              >
              </Copy>
            </template>
          </FieldItem>
        </div>

        <!-- 步骤① 登录镜像仓库 -->
        <div class="mt-[18px] flex items-center justify-between">
          <div class="flex items-center">
            <span class="push-guide-step-circle">1</span>
            <span class="ml-[8px] text-[12px] font-bold text-[#4A5973]">{{ $t('登录镜像仓库') }}</span>
          </div>
          <Button
            text
            theme="primary"
            @click="copyText(loginCommand)"
          >
            <Copy class="mr-[4px]" />
            {{ $t('复制') }}
          </Button>
        </div>
        <div class="bg-[#F0F4FA] mt-[6px] px-[12px] ml-[28px] py-[16px]">
          <pre
            v-bk-xss-html="highlightedLoginCommand"
            class="push-guide-mono push-guide-pre m-0 whitespace-pre-wrap break-all"
          ></pre>
        </div>

        <!-- 步骤② 推送到空间仓库 -->
        <div class="mt-[6px] flex items-center justify-between">
          <div class="flex items-center">
            <span class="push-guide-step-circle">2</span>
            <span class="ml-[8px] text-[12px] font-bold text-[#4A5973]">{{ $t('推送到空间仓库') }}</span>
          </div>
          <Button
            text
            theme="primary"
            @click="copyText(pushCommand)"
          >
            <Copy class="mr-[4px]" />
            {{ $t('复制') }}
          </Button>
        </div>
        <div class="bg-[#F0F4FA] mt-[6px] px-[12px] ml-[28px] py-[16px]">
          <pre
            v-bk-xss-html="highlightedPushCommand"
            class="push-guide-mono push-guide-pre m-0 whitespace-pre-wrap break-all"
          ></pre>
        </div>
      </div>
    </template>
  </Popover>
</template>

<script setup lang="ts">
  import { computed, ref } from 'vue';

  import { Button, Popover } from 'bkui-vue';
  import { Copy } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import FieldItem from '~/components/field-item.vue';
  import SecretToggle from '~/components/secret-toggle.vue';
  import { useCopy } from '~/composables/use-copy';

  interface IProps {
    /** 镜像名称（用于拼接 docker push 命令） */
    imageName?: string;
    /** 镜像 Tag（用于拼接 docker push 命令） */
    imageTag?: string;
    /** 仓库凭证/密码 */
    password?: string;
    repositoryAddress: string;
    /** 用户名 */
    username?: string;
  }

  const props = withDefaults(defineProps<IProps>(), {
    imageName: '',
    imageTag: '',
    password: '',
    username: '',
  });

  const { copyText } = useCopy();
  const { t } = useI18n();

  // ========== 密码显隐联动 ==========
  // SecretToggle 切换时同步登录命令里 -p 的展示：显示明文 / 隐藏掩码；复制始终用真实密码，不受影响
  const passwordVisible = ref(false);
  function handlePasswordToggle(visible: boolean) {
    passwordVisible.value = visible;
  }

  // ========== 纯文本命令（用于复制）==========
  // 命令占位符（如 <镜像名称>）为「替换成实际值」的语义占位，不走 i18n，zh/en 保持一致
  const loginCommand = computed(() => {
    const username = props.username || '<用户名>';
    const password = props.password || '<仓库凭证>';
    return `docker login ${props.repositoryAddress} -u ${username} -p ${password}`;
  });

  const pushCommand = computed(() => {
    const imageName = props.imageName || `${props.repositoryAddress}/<镜像名称>`;
    const imageTag = props.imageTag || '<镜像Tag>';
    return `docker push ${imageName}:${imageTag}`;
  });

  // ========== 高亮 HTML（用于渲染）==========
  // 颜色约定（与设计稿对齐）：
  //   push-guide-command-keyword  → 关键字 docker、命令名 login/push（蓝 #3B6EF5）
  //   push-guide-command-string   → 仓库地址、用户名、镜像路径等字符串（绿 #2EAD98）
  //   基础文本（容器）                    → 灰 #4A5973
  // 命令通过 v-bk-xss-html 渲染，拼接高亮标签前先转义所有动态文本（参照 dev-mode-steps.vue 先例）
  function escapeHtml(text: string) {
    return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  }

  const highlightedLoginCommand = computed(() => {
    const username = escapeHtml(props.username || '<用户名>');
    // 密码展示跟随 SecretToggle 显隐：显示明文 / 隐藏掩码；无密码时展示占位符
    const password = props.password
      ? escapeHtml(passwordVisible.value ? props.password : '********')
      : escapeHtml('<仓库凭证>');
    return [
      '<span class="push-guide-command-keyword">docker login</span>',
      // 仓库地址弱化显示为灰色（继承 .push-guide-mono 容器色 #4A5973），与用户名/密码的绿色高亮区分
      `<span>${escapeHtml(props.repositoryAddress || '')}</span>`,
      '<span class="push-guide-command-keyword">-u</span>',
      `<span class="push-guide-command-string">${username}</span>`,
      '<span class="push-guide-command-keyword">-p</span>',
      `<span class="push-guide-command-string">${password}</span>`,
    ].join(' ');
  });

  const highlightedPushCommand = computed(() => {
    // 推送指引为通用教程，展示固定占位符，不跟随当前已选镜像名/tag；
    // 仓库地址展示真实值（空间绑定仓库，非「已选镜像」），为空时降级为 <镜像仓库> 占位
    // 占位符含尖括号，会被 v-bk-xss-html 当未知标签过滤，必须先 escapeHtml 再拼 HTML（参照 login 命令先例）
    const repositoryAddress = escapeHtml(props.repositoryAddress || `<${t('镜像仓库')}>`);
    const imageName = escapeHtml(`<${t('镜像名称')}>`);
    const imageTag = escapeHtml(`<${t('镜像 Tag')}>`);
    const fullImagePath = `${repositoryAddress}/${imageName}:${imageTag}`;
    return [
      '<span class="push-guide-command-keyword">docker push</span>',
      `<span class="push-guide-command-string">${fullImagePath}</span>`,
    ].join(' ');
  });
</script>

<style lang="postcss" scoped>
  .push-guide-mono {
    color: #4A5973;
    font-family: Consolas, 'Courier New', monospace;
  }

  .push-guide-pre {
    margin: 0;
    background: transparent !important;
    white-space: pre-wrap;
    word-break: break-all;
  }

  .push-guide-step-circle {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 22px;
    border-radius: 50%;
    background-color: #e9edf5;
    color: #4A5973;
    font-size: 14px;
    font-weight: 700;
    line-height: 22px;
  }

  .push-guide-content :deep(.push-guide-command-keyword) {
    color: #3B6EF5;
  }

  .push-guide-content :deep(.push-guide-command-string) {
    color: #14a38b;
  }
</style>
