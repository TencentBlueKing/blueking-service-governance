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
    :before-close="handleBeforeClose"
    :quick-close="false"
    :width="560"
  >
    <template #header>
      <DividerHeader
        :title="deployed ? $t('调整权重') : $t('设置初始权重')"
        :title-size="20"
      >
        <span class="inline-flex min-w-0 items-center">
          <span class="truncate">{{ displayName }}</span>
          <Tag
            v-if="envType && envTypeMap[envType]"
            :class="['ml-[8px] shrink-0', envTypeTagClassMap[envType]]"
            size="small"
          >
            {{ envTypeMap[envType].name }}
          </Tag>
        </span>
      </DividerHeader>
    </template>

    <Alert
      class="mb-[16px] weight-tip-alert"
      theme="warning"
    >
      <template #title>
        <span class="font-bold text-[#313238]">{{ $t('调整说明') }}</span>
      </template>
      <div class="flex flex-col gap-[4px] leading-[20px]">
        <template v-if="deployed">
          <div class="flex items-start">
            <span class="weight-tip-dot"></span>
            <i18n-t
              keypath="数值越大权重越高；保存后按新的单实例权重重算总权重。权重为 {0} 时该环境将不再接收流量。"
              tag="span"
            >
              <span class="font-bold">0</span>
            </i18n-t>
          </div>
          <div class="flex items-start">
            <span class="weight-tip-dot"></span>
            <span>{{ $t('保存后立即同步到该环境下全部实例的北极星注册权重，无需重新部署应用。') }}</span>
          </div>
        </template>
        <template v-else>
          <div class="flex items-start">
            <span class="weight-tip-dot"></span>
            <span>{{ $t('该环境尚未部署，此处设置的是实例首次注册到北极星时的权重。') }}</span>
          </div>
          <div class="flex items-start">
            <span class="weight-tip-dot"></span>
            <i18n-t
              keypath="数值越大权重越高；填 {0} 则该环境部署后不参与分流。"
              tag="span"
            >
              <span class="font-bold">0</span>
            </i18n-t>
          </div>
        </template>
        <div class="flex items-start">
          <span class="weight-tip-dot"></span>
          <span>{{ $t('实例数统计的维度为：健康的实例') }}</span>
        </div>
      </div>
    </Alert>

    <Form form-type="vertical">
      <Form.FormItem
        :label="deployed ? $t('总权重（权重占比）') : $t('单个实例权重')"
        required
      >
        <div class="flex items-center">
          <Input
            v-model.trim="draftWeight"
            :class="['w-[220px]', { 'weight-input-error': showWeightError }]"
            :min="0"
            :precision="0"
            type="number"
          >
            <template #prefix>
              <FormPrefix :label="$t('单个实例权重')" />
            </template>
          </Input>
          <template v-if="deployed">
            <span class="mx-[10px] text-[#63656E]">×</span>
            <Input
              class="w-[160px]"
              disabled
              :model-value="String(healthyCount)"
            >
              <template #prefix>
                <FormPrefix :label="$t('实例数')" />
              </template>
            </Input>
          </template>
        </div>
        <div
          v-if="showWeightError"
          class="mt-[4px] text-[12px] leading-[16px] text-[#EA3636]"
        >
          <i18n-t keypath="单个实例权重需为 {0} ~ {1} 的整数">
            <span class="font-bold">0</span>
            <span class="font-bold">{{ MAX_WEIGHT }}</span>
          </i18n-t>
        </div>
        <!-- 已部署场景，展示总权重和占比 -->
        <div
          v-if="deployed"
          class="mt-[8px] rounded-[2px] bg-[#F5F7FA] px-[12px] py-[8px] text-[12px] leading-[20px]"
        >
          <i18n-t keypath="总权重 = {0}，预计占比 {1}%">
            <span class="font-bold text-[#313238]">{{ draftTotalWeight }}</span>
            <span class="font-bold text-[#313238]">{{ draftPercent }}</span>
          </i18n-t>
          <p
            v-if="weightOverriddenInstanceCount > 0"
            class="mt-[4px] text-[#979BA5]"
          >
            {{ $t('部分实例权重已单独调整，上方为保存后全部统一为该权重后的预计值') }}
          </p>
        </div>
      </Form.FormItem>
      <Form.FormItem>
        <template #label>
          <span>{{ $t('动态权重') }}</span>
        </template>
        <div class="flex items-center gap-[10px]">
          <Switcher
            v-model="draftDynamicWeight"
            class="shrink-0"
            :disabled="!enableWeightFactor"
            theme="primary"
          />
          <span class="text-[12px] leading-[20px] text-[#979BA5]">
            {{
              enableWeightFactor
                ? $t('开启后北极星按机型动态调整该环境实例权重，上方填写的是基准权重。')
                : $t('需先编辑该北极星并开启「权重因子」，保存后再回到此处打开。')
            }}
          </span>
        </div>
      </Form.FormItem>
    </Form>

    <template #footer>
      <div class="flex items-center justify-end">
        <Button
          class="min-w-[88px]"
          :loading="loading"
          theme="primary"
          @click="handleConfirm"
        >
          {{ $t('确定') }}
        </Button>
        <Button
          class="ml-[8px] min-w-[88px]"
          :disabled="loading"
          @click="handleCancel"
        >
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Dialog>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Alert, Button, Dialog, Form, Input, Switcher, Tag } from 'bkui-vue';
  import DividerHeader from '~/components/divider-header.vue';
  import FormPrefix from '~/components/form-prefix.vue';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';

  interface Props {
    deployed: boolean;
    displayName: string;
    dynamicWeight?: boolean;
    enableWeightFactor?: boolean;
    envType?: string;
    healthyCount: number;
    loading?: boolean;
    /** 调整前本环境健康实例的权重总和。 */
    oldEnvTotalWeight: number;
    /** 北极星服务全部健康实例的权重总和。 */
    totalHealthyInstanceWeight: number;
    weight: number;
    /** 当前环境被单独调整权重的实例数。 */
    weightOverriddenInstanceCount: number;
  }

  const MAX_WEIGHT = 10000;

  const isShow = defineModel<boolean>('isShow', { default: false });
  const props = defineProps<Props>();
  const emit = defineEmits<{
    (e: 'confirm', payload: { dynamicWeight?: boolean; weight: number }): void;
  }>();

  const draftWeight = ref('0');
  const draftDynamicWeight = ref(false);
  /** 用户点击过确定后持续展示校验错误，避免输入过程中提前打扰。 */
  const weightTouched = ref(false);

  /** 未部署环境的健康实例数为 0，因此只在已部署场景展示计算结果。 */
  const draftTotalWeight = computed(() => Number(draftWeight.value || 0) * props.healthyCount);
  /** 权重占比 = 新环境权重 /（总健康权重 - 旧环境权重 + 新环境权重）。 */
  const draftPercent = computed(() =>
    toPercent(
      draftTotalWeight.value,
      props.totalHealthyInstanceWeight - props.oldEnvTotalWeight + draftTotalWeight.value,
    ),
  );
  const isWeightInvalid = computed(() => !isValidWeight(draftWeight.value));
  /** 超出上限即时提示；其它非法值在提交或已提示后持续展示。 */
  const showWeightError = computed(() => {
    if (!isWeightInvalid.value) return false;
    if (weightTouched.value) return true;
    const weight = Number(draftWeight.value);
    return draftWeight.value !== '' && Number.isFinite(weight) && weight > MAX_WEIGHT;
  });

  /** 提交过程中阻止 Dialog 被关闭。 */
  function handleBeforeClose() {
    // 提交期间禁止通过关闭按钮或遮罩退出，防止重复操作和状态丢失。
    return !props.loading;
  }

  /** 非提交状态下关闭弹窗。 */
  function handleCancel() {
    if (!props.loading) {
      isShow.value = false;
    }
  }

  /** 校验输入并向父组件提交合法权重。 */
  function handleConfirm() {
    if (props.loading) return;
    if (isWeightInvalid.value) {
      weightTouched.value = true;
      return;
    }

    emit('confirm', {
      weight: Number(draftWeight.value),
      ...(props.enableWeightFactor ? { dynamicWeight: draftDynamicWeight.value } : {}),
    });
  }

  /** 判断输入是否为允许范围内的整数权重。 */
  function isValidWeight(value: string) {
    const weight = Number(value);
    return value !== '' && Number.isInteger(weight) && weight >= 0 && weight <= MAX_WEIGHT;
  }

  /** 将部分权重换算为保留两位小数的百分比。 */
  function toPercent(part: number, total: number) {
    return total ? ((part / total) * 100).toFixed(2) : '0.00';
  }

  watch(isShow, val => {
    if (val) {
      // 每次打开都使用列表中的最新权重，并重置上一次的校验状态。
      draftWeight.value = String(props.weight);
      draftDynamicWeight.value = !!props.dynamicWeight;
      weightTouched.value = false;
    }
  });
</script>

<style lang="postcss" scoped>
  .weight-tip-alert {
    :deep(.bk-alert-title) {
      margin-bottom: 4px;
    }
  }

  .weight-tip-dot {
    display: inline-block;
    width: 4px;
    height: 4px;
    margin-top: 8px;
    margin-right: 8px;
    flex-shrink: 0;
    border-radius: 50%;
    background-color: currentColor;
  }

  :deep(.weight-input-error) {
    border-color: #ea3636;
  }
</style>
