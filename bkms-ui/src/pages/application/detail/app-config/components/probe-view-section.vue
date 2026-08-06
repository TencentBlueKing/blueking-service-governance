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
  <div class="probe-view-section border border-[#EAEBF0] rounded-[8px] flex-1 min-w-0 overflow-hidden">
    <!-- 卡片标题行 -->
    <div class="card-header flex items-center gap-[10px] h-[32px] px-[12px] border-b border-[#EAEBF0]">
      <div class="flex items-center gap-[6px]">
        <i
          v-if="modified"
          class="w-[3px] h-[18px] bg-[#ff9c01] flex-shrink-0"
        ></i>
        <span class="probe-title text-[12px] font-bold">{{ label }}</span>
      </div>
      <Popover
        v-if="showEditIcon && !disabled && editingTip"
        :content="editingTip"
        placement="top"
        theme="dark"
      >
        <Button
          class="!hover:text-[#3A84FF]"
          disabled
          text
        >
          <EditLine />
        </Button>
      </Popover>
      <Button
        v-else-if="showEditIcon && !disabled"
        class="!hover:text-[#3A84FF]"
        text
        @click="$emit('edit')"
      >
        <EditLine />
      </Button>
    </div>

    <!-- 卡片内容 -->
    <div :class="['px-[12px] py-[8px]', isProbeConfigured ? '' : 'h-[240px]']">
      <!-- 已配置：展示详细字段 -->
      <template v-if="isProbeConfigured">
        <div class="flex flex-col">
          <ProbeDetailItem
            :label="$t('探测方法')"
            :value="probe.probeHandler?.type"
          />
          <ProbeDetailItem
            v-if="probe.probeHandler?.type === ProbeType.EXEC"
            :label="$t('执行命令')"
            :value="commandDisplay"
          />
          <ProbeDetailItem
            v-if="probe.probeHandler?.type === ProbeType.HTTP"
            :label="$t('检查路径')"
            :value="probe.probeHandler?.url || '--'"
          />
          <ProbeDetailItem
            v-if="probe.probeHandler?.type === ProbeType.HTTP || probe.probeHandler?.type === ProbeType.TCP"
            :label="$t('检查端口')"
            :value="probe.probeHandler?.port || '--'"
          />
          <ProbeDetailItem
            :label="$t('延迟探测时间')"
            :value="`${probe.initialDelaySeconds ?? 0}s`"
          />
          <ProbeDetailItem
            :label="$t('探测超时时间')"
            :value="`${probe.timeoutSeconds ?? 1}s`"
          />
          <ProbeDetailItem
            :label="$t('探测频率')"
            :value="`${probe.periodSeconds ?? 10}s`"
          />
          <ProbeDetailItem
            :label="$t('连续探测成功次数')"
            :value="`${probe.successThreshold ?? 1} ${$t('次')}`"
          />
          <ProbeDetailItem
            :label="$t('连续探测失败次数')"
            :value="`${probe.failureThreshold ?? 3} ${$t('次')}`"
          />
        </div>
      </template>

      <!-- 未配置 -->
      <template v-else>
        <Exception
          class="large-exception"
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
            <div class="empty-description">
              <span> {{ $t('尚未配置') }}</span>
              <template v-if="disabled && disabledTip">，{{ disabledTip }}</template>
              <template v-else>
                <span>，</span>
                <Popover
                  v-if="editingTip"
                  :content="editingTip"
                  placement="top"
                  theme="dark"
                >
                  <Button
                    disabled
                    text
                    theme="primary"
                  >
                    {{ $t('立即配置') }}
                  </Button>
                </Popover>
                <Button
                  v-else
                  text
                  theme="primary"
                  @click="$emit('edit')"
                >
                  {{ $t('立即配置') }}
                </Button>
              </template>
              <!-- 探针描述 -->
              <div
                v-if="description"
                class="description p-[12px] mt-[8px] bg-[#F5F7FA] text-align-left"
              >
                {{ description }}
              </div>
            </div>
          </template>
        </Exception>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { Button, Exception, Popover } from 'bkui-vue';
  import { EditLine } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { ProbeOutput } from '~/@types/v1/app-spec';

  import ProbeDetailItem from './probe-detail-item.vue';
  import { ProbeType } from './types';

  interface Props {
    /** 探针描述文案 */
    description?: string;
    /** 是否禁用编辑（如启动探针依赖就绪探针） */
    disabled?: boolean;
    /** 禁用时的提示文案 */
    disabledTip?: string;
    /** 有其他探针正在编辑时的提示文案 */
    editingTip?: string;
    label: string;
    modified?: boolean;
    probe: ProbeOutput;
    showEditIcon?: boolean;
  }

  const props = withDefaults(defineProps<Props>(), {
    description: '',
    disabled: false,
    disabledTip: '',
    editingTip: '',
    modified: false,
    showEditIcon: true,
  });

  defineEmits<{
    edit: [];
  }>();

  useI18n();

  /** 是否已配置探针（有探针类型即视为已配置） */
  const isProbeConfigured = computed(() => !!props.probe?.probeHandler?.type);

  /** EXEC 命令展示文本 */
  const commandDisplay = computed(() => {
    const handler = props.probe?.probeHandler;
    if (!handler) return '--';
    // shell 模式展示 shCommand
    if (handler.shCommand) return handler.shCommand;
    const cmd = handler.command;
    return cmd?.length ? cmd.join(',') : '--';
  });
</script>

<style lang="postcss" scoped>
  :deep(.bk-exception-part .bk-exception-img) {
    width: 220px !important;
    height: 100px !important;
  }
  :deep(.bk-exception-part .bk-exception-description) {
    width: 100% !important;
  }

  .probe-view-section {
    --probe-title-color: #4d4f56;
    --probe-description-color: #313238;
    --probe-header-bg: #f5f7fa;
    --probe-detail-label-color: #979ba5;
    --probe-detail-value-color: #313238;
  }

  .card-header {
    color: var(--probe-title-color);
    background-color: var(--probe-header-bg);
  }

  .probe-title {
    color: var(--probe-title-color);
  }

  .empty-description,
  .description {
    color: var(--probe-description-color);
  }

  /* 启动探针在主探针未配置时的弱化样式 */
  .startup-probe-deemphasized {
    opacity: 0.8;

    --probe-title-color: #c4c6cc;
    --probe-description-color: #c4c6cc;
    --probe-header-bg: #fafbfd;
    --probe-detail-label-color: #c4c6cc;
    --probe-detail-value-color: #c4c6cc;
  }
</style>
