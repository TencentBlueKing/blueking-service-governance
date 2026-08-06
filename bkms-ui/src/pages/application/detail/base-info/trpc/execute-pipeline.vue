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
  <Sideslider
    v-model:is-show="isShow"
    :width="960"
    @closed="isShow = false"
  >
    <template #header>
      <div class="flex items-center">
        <span class="text-[#313238] text-[16px]">
          {{ $t('填写入参') }}
          <div class="inline-block ml-[16px] text-[12px]">
            <Button
              :disabled="curPipelineUseParams === 'last'"
              text
              theme="primary"
              @click="handleToggleUseParams('last')"
            >
              {{ $t('使用上一次的参数') }}
            </Button>
            <Divider
              class="mx-[8px]"
              direction="vertical"
            ></Divider>
            <Button
              :disabled="curPipelineUseParams === 'default'"
              text
              theme="primary"
              @click="handleToggleUseParams('default')"
            >
              {{ $t('恢复默认') }}
            </Button>
          </div>
        </span>
      </div>
    </template>
    <PipelineParamsForm
      ref="pipelineParamsRef"
      v-bkloading="{ loading: paramsLoading }"
      class="px-[24px]"
      :params="params"
    />
    <template #footer>
      <div class="flex items-center">
        <Button
          class="min-w-[88px]"
          theme="primary"
          @click="handleSave"
        >
          {{ $t('执行') }}
        </Button>
        <Button
          class="min-w-[88px] ml-[8px]"
          @click="isShow = false"
        >
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { ref, watch } from 'vue';

  import { Button, Divider, Sideslider } from 'bkui-vue';
  import { BkCIPipelineVariableOutput } from '~/@types/v1/bkintegrations-bkci';
  import { BkintegrationsBkciService } from '~/api/modules/v1';
  import { useSpaceStore } from '~/stores/space';

  import PipelineParamsForm from '../components/pipeline-params-form.vue';
  interface IProps {
    lastParams: Record<string, string>;
    pipelineId: string;
  }
  type PipelineUseParams = 'default' | 'last';
  const isShow = defineModel<boolean>('isShow');
  const props = defineProps<IProps>();
  const emit = defineEmits(['confirm']);

  const spaceStore = useSpaceStore();
  const params = ref<BkCIPipelineVariableOutput[]>([]);
  const paramsLoading = ref(false);
  const getParamsData = async () => {
    try {
      paramsLoading.value = true;
      const res = await BkintegrationsBkciService.getBkCIPipelineVariables({
        workspaceID: spaceStore.currentSpace,
        pipelineID: props.pipelineId,
      });
      params.value = res || [];
    } catch (err) {
      console.error(err);
    } finally {
      paramsLoading.value = false;
    }
  };
  /**
   * @description 当前选中的流水线配置，默认default
   * @param default 恢复默认
   * @param last 使用上一次的参数
   */
  const curPipelineUseParams = ref<PipelineUseParams>('default');

  /** 使用上一次的参数，作为动态表单的props传入 */
  const lastParamsValue = ref<Record<string, string | undefined>>({});
  /** 切换 使用上一次的参数/默认参数 */
  const handleToggleUseParams = (targetType: PipelineUseParams) => {
    /** 获取pipelineId缓存的数据（上一次的参数） */
    if (targetType === 'last') {
      // 触发使用上一次的参数
      for (const item of params.value) {
        lastParamsValue.value[item.id!] = props.lastParams[item.id!];
      }
    } else {
      // 使用默认参数，则填入defaultValue
      for (const item of params.value) {
        lastParamsValue.value[item.id!] = item.defaultValue;
      }
    }
    curPipelineUseParams.value = targetType;
    // 更新动态表单内的input value
    pipelineParamsRef.value?.setData(lastParamsValue.value);
  };

  const pipelineParamsRef = ref();
  const handleSave = async () => {
    // 如果流水线没有参数，直接允许执行
    if (!pipelineParamsRef.value) {
      emit('confirm', {});
      isShow.value = false;
      return;
    }

    const { valid, data } = await pipelineParamsRef.value.save();
    if (valid) {
      emit('confirm', data);
      isShow.value = false;
    }
  };

  const init = () => {
    getParamsData();
    handleToggleUseParams('default');
  };

  watch(isShow, val => {
    if (val) init();
  });
</script>
