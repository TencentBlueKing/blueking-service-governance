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
    v-model:is-show="isShow"
    :quick-close="false"
    :width="640"
    @closed="emit('cancel')"
  >
    <template #header>
      <DividerHeader
        :title="mode === 'conflict' ? $t('权重因子参数') : $t('确认修改北极星上的权重因子？')"
        :title-size="20"
      >
        <span class="truncate text-[#979BA5]">{{ namespace }} / {{ serviceName }}</span>
      </DividerHeader>
    </template>

    <Alert
      class="mb-[16px]"
      theme="warning"
      :title="
        mode === 'conflict'
          ? $t('权重因子与北极星线上配置不一致，请选择处理方式')
          : $t('修改将在侧栏保存时写入北极星，影响该服务的线上流量分配。')
      "
    />

    <template v-if="mode === 'conflict'">
      <div class="mb-[12px] text-[14px] text-[#313238]">{{ $t('请选择处理方式：') }}</div>
      <Radio.Group
        v-model="action"
        class="weight-factor-options mb-[20px] flex flex-col gap-[12px]"
      >
        <div
          v-for="item in options"
          :key="item.value"
          class="cursor-pointer rounded-[4px] border border-solid px-[16px] py-[14px]"
          :class="action === item.value ? 'border-[#3A84FF] bg-[#F0F5FF]' : 'border-[#DCDEE5]'"
          @click="action = item.value"
        >
          <Radio :label="item.value">
            <div class="ml-[8px] whitespace-normal">
              <div class="text-[12px] font-bold text-[#313238]">{{ item.title }}</div>
              <div class="mt-[8px] text-[12px] leading-[20px] text-[#63656E]">{{ item.description }}</div>
            </div>
          </Radio>
        </div>
      </Radio.Group>
    </template>

    <div class="rounded-[2px] bg-[#F5F7FA] px-[16px] py-[12px]">
      <div class="flex min-w-0 items-center text-[12px] leading-[20px]">
        <span class="shrink-0">{{ $t('北极星服务名') }}：</span>
        <span class="truncate">{{ namespace }} / {{ serviceName }}</span>
        <a
          class="ml-[4px] shrink-0 text-[#3A84FF]"
          :href="serviceUrl"
          rel="noopener noreferrer"
          target="_blank"
        >
          <Share :title="$t('前往北极星')" />
        </a>
      </div>
      <table class="weight-factor-table mt-[12px] w-full table-fixed bg-white text-[12px] leading-[20px]">
        <colgroup>
          <col class="w-[28%]" />
          <col class="w-[36%]" />
          <col class="w-[36%]" />
        </colgroup>
        <thead>
          <tr>
            <th>{{ $t('配置项') }}</th>
            <th :class="{ selected: mode === 'conflict' && action === 'keep' }">{{ $t('北极星线上') }}</th>
            <th :class="{ selected: mode === 'change' || action === 'overwrite' }">
              {{ mode === 'change' ? $t('修改后') : $t('平台设置') }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in rows"
            :key="row.label"
          >
            <td class="text-[#979BA5]">{{ row.label }}</td>
            <td
              :class="{ selected: mode === 'conflict' && action === 'keep', 'font-bold': row.remote !== row.platform }"
            >
              {{ row.remote }}
            </td>
            <td
              :class="{
                selected: mode === 'change' || action === 'overwrite',
                'font-bold': row.remote !== row.platform,
              }"
            >
              {{ row.platform }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <template #footer>
      <div class="flex justify-end gap-[8px]">
        <Button
          theme="primary"
          @click="emit('confirm', mode === 'change' ? 'overwrite' : action)"
        >
          {{ $t('确定') }}
        </Button>
        <Button @click="emit('cancel')">{{ $t('取消') }}</Button>
      </div>
    </template>
  </Dialog>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Alert, Button, Dialog, Radio } from 'bkui-vue';
  import { Share } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { PLATFORM_WEIGHT_FACTOR_CONFIG } from '~/common/const';
  import DividerHeader from '~/components/divider-header.vue';

  import { weightFactorValue } from './use-imported-polaris';

  import type { WeightFactorResolution, WeightFactorState } from './use-imported-polaris';

  const props = defineProps<{
    enabled: boolean;
    // conflict 选择保留或覆盖；change 确认开关修改。
    mode: 'change' | 'conflict';
    namespace?: string;
    remote: WeightFactorState;
    resolution: null | WeightFactorResolution;
    serviceName?: string;
  }>();
  const emit = defineEmits<{
    cancel: [];
    confirm: [action: WeightFactorResolution];
  }>();
  const isShow = defineModel<boolean>('isShow', { default: false });
  const { t } = useI18n();
  // 选择暂存在弹窗内，确认后才通知侧栏。
  const action = ref<WeightFactorResolution>('keep');
  const options = computed(() => [
    {
      value: 'keep' as const,
      title: t('使用北极星配置'),
      description: t('保留北极星上的开关和自定义公式，平台以线上配置为准。'),
    },
    {
      value: 'overwrite' as const,
      title: t('使用平台配置，并覆盖北极星'),
      description: t('保存时开启北极星权重因子，并使用平台默认公式覆盖线上配置。'),
    },
  ]);
  const serviceUrl = computed(
    () =>
      `${import.meta.env.BK_POLARIS_URL}/#/services/info/detail/${encodeURIComponent(props.namespace || '')}/${encodeURIComponent(props.serviceName || '')}`,
  );
  const rows = computed(() => {
    // 差异处理提供平台开启配置；修改确认展示本次保存的实际开关。
    const enabled = props.mode === 'conflict' || props.enabled;
    const platform = { enabled, config: enabled ? PLATFORM_WEIGHT_FACTOR_CONFIG : undefined };
    return [
      {
        label: t('权重因子'),
        remote: props.remote.enabled ? t('已开启') : t('未开启'),
        platform: enabled ? t('已开启') : t('未开启'),
      },
      ...(['func', 'a', 'b', 'min', 'max'] as const).map(key => ({
        label: key === 'func' ? t('计算函数') : key,
        remote: weightFactorValue(props.remote, key),
        platform: weightFactorValue(platform, key),
      })),
    ];
  });
  // 重新打开恢复已确认选择，取消的临时选择不保留。
  watch(
    isShow,
    show => {
      if (show) action.value = props.resolution ?? 'keep';
    },
    { immediate: true },
  );
</script>

<style lang="postcss" scoped>
  .weight-factor-options :deep(.bk-radio) {
    height: auto;
    white-space: normal;
  }

  .weight-factor-table {
    border-collapse: collapse;

    th,
    td {
      padding: 8px 12px;
      text-align: left;
      overflow-wrap: anywhere;
      border-bottom: 1px solid #f0f1f5;
    }

    th {
      font-weight: normal;
      color: #979ba5;
      background-color: #fafbfd;
    }

    .selected {
      background-color: #f0f5ff;
      box-shadow:
        inset 1px 0 #3a84ff,
        inset -1px 0 #3a84ff;
    }

    th.selected {
      color: #3a84ff;
      box-shadow:
        inset 1px 0 #3a84ff,
        inset -1px 0 #3a84ff,
        inset 0 1px #3a84ff;
    }

    tr:last-child td.selected {
      box-shadow:
        inset 1px 0 #3a84ff,
        inset -1px 0 #3a84ff,
        inset 0 -1px #3a84ff;
    }
  }
</style>
