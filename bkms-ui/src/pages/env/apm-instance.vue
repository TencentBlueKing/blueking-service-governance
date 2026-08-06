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
  <!-- APM 实例信息 -->
  <div class="p-[16px] bg-[#F5F7FA]">
    <div class="flex items-center justify-between mb-[8px]">
      <span class="text-[14px] text-[#979ba5]">{{ $t('APM 实例') }}</span>
      <Button
        class="bg-[#fff]"
        outline
        @click="handleOpenSwitchApm"
      >
        {{ $t('切换 APM 实例') }}
      </Button>
    </div>
    <div class="flex items-center gap-[8px]">
      <span class="text-[14px] font-700 text-[#313238]">{{ currentApm?.name || '--' }}</span>
      <Tag
        v-if="currentApm && apmTagText"
        :theme="apmTagTheme"
      >
        {{ apmTagText }}
      </Tag>
    </div>
    <Alert
      class="mt-[8px]"
      theme="info"
    >
      <template #title>
        {{ apmAlertMessage }}
      </template>
    </Alert>
  </div>
  <!-- 切换 APM 实例弹窗 -->
  <Dialog
    v-model:is-show="switchApmDialogVisible"
    :title="$t('切换 APM 实例')"
    :width="600"
  >
    <Alert
      class="mb-[16px]"
      theme="warning"
    >
      <template #title>
        <p>
          {{ $t('切换 APM 实例将产生以下影响，') }}<strong>{{ $t('请谨慎操作!') }}</strong>
        </p>
        <ul class="mt-[4px] pl-[20px] list-disc line-height-[20px]">
          <li>{{ $t('当前环境的 APM Token 环境变量将被替换为目标实例的值') }}</li>
          <li>{{ $t('切换后新产生的观测数据将上报到新的 APM 实例，历史数据不会迁移') }}</li>
          <li>{{ $t('运行中的服务需要重新部署后新配置才会生效') }}</li>
        </ul>
      </template>
    </Alert>
    <div class="flex p-[16px] bg-[#F5F7FA] text-[12px]">
      <div class="flex flex-col gap-[6px] flex-1">
        <div class="text-[12px] text-[#979ba5]">{{ $t('当前环境') }}</div>
        <div class="flex items-center gap-[8px]">
          <span class="font-bold">{{ data.name }}</span>
          <Tag
            :class="envTagClass"
            type="stroke"
            >{{ envTagText }}</Tag
          >
        </div>
      </div>
      <div class="flex flex-col gap-[6px] flex-1">
        <div class="text-[12px] text-[#979ba5]">{{ $t('当前 APM 实例') }}</div>
        <div class="flex items-center gap-[8px]">
          <span class="font-bold">{{ currentApm?.name || '--' }}</span>
          <Tag
            v-if="currentApm"
            :theme="apmTagTheme"
            type="stroke"
          >
            {{ apmTagText }}
          </Tag>
        </div>
      </div>
    </div>
    <Form
      ref="formRef"
      class="mt-[16px]"
      form-type="vertical"
      :model="formData"
      :rules="formRules"
    >
      <Form.FormItem
        :label="$t('切换到')"
        property="targetApmID"
        required
      >
        <Select
          v-model="formData.targetApmID"
          filterable
          :loading="apmListLoading"
          :placeholder="$t('请选择目标 APM')"
        >
          <Select.Group
            v-for="group in groupedApmList"
            :key="group.type"
            :label="group.label"
          >
            <Select.Option
              v-for="apm in group.items"
              :key="apm.apmID"
              :disabled="apm.apmID === currentApm?.apmID"
              :label="getApmLabel(apm)"
              :value="apm.apmID"
            />
          </Select.Group>
          <Select.Option
            v-if="!envApmExists"
            :key="CREATE_NEW_APM_ID"
            :label="$t('新建 APM')"
            :value="CREATE_NEW_APM_ID"
          />
        </Select>
      </Form.FormItem>
      <Form.FormItem
        property="confirmName"
        required
      >
        <template #label>
          {{ $t('切换不可撤销，请输入当前环境名称') }}
          <strong
            v-bk-tooltips="$t('复制')"
            class="px-[4px] cursor-pointer text-[#ea3636] hover:bg-[#FFEBEB]"
            @click="handleCopyEnvName"
          >
            {{ data.name }}
          </strong>
          {{ $t('进行确认') }}
        </template>
        <Input
          v-model.trim="formData.confirmName"
          :placeholder="$t('请输入环境名称以确认')"
        />
      </Form.FormItem>
    </Form>
    <template #footer>
      <Button
        class="mr-[8px]"
        :loading="switchApmLoading"
        theme="primary"
        @click="handleConfirmSwitchApm"
      >
        {{ $t('确认') }}
      </Button>
      <Button @click="switchApmDialogVisible = false">
        {{ $t('取消') }}
      </Button>
    </template>
  </Dialog>
</template>
<script lang="ts" setup>
  import { computed, ref } from 'vue';

  import { Alert, Button, Dialog, Form, Input, Message, Select, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { ApmOutput, GetEnvApmOutput } from '~/@types/v1/bkintegrations-bkmonitor';
  import { EnvOutput } from '~/@types/v1/env';
  import { BkintegrationsBkmonitorService } from '~/api/modules/v1';
  import { useCopy } from '~/composables/use-copy';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';
  import { useSpaceStore } from '~/stores/space';

  // 特殊值：表示"新建 APM"
  const CREATE_NEW_APM_ID = -1;
  // APM 分组排序
  const APM_GROUP_ORDER = ['development', 'test', 'staging', 'production', 'custom'] as const;

  interface IProps {
    currentApm: GetEnvApmOutput | null;
    data: EnvOutput;
  }
  const props = defineProps<IProps>();
  const emit = defineEmits<{
    'update:currentApm': [];
  }>();

  const { t } = useI18n();
  const { copyText } = useCopy();
  const spaceStore = useSpaceStore();

  const apmList = ref<ApmOutput[]>([]);
  const apmListLoading = ref(false);
  const switchApmDialogVisible = ref(false);
  const switchApmLoading = ref(false);

  const formRef = ref<InstanceType<typeof Form>>();
  const formData = ref({
    confirmName: '',
    targetApmID: undefined as number | undefined,
  });
  const formRules = computed(() => ({
    targetApmID: [{ required: true, message: t('请选择目标 APM'), trigger: 'change' }],
    confirmName: [
      { required: true, message: t('请输入环境名称'), trigger: 'blur' },
      {
        validator: (value: string) => value === props.data.name,
        message: t('请输入正确的环境名称'),
        trigger: 'blur',
      },
    ],
  }));

  // 环境类型映射（基于公共 envTypeMap，追加 custom）
  const localEnvTypeMap = computed<Record<string, string>>(() => ({
    ...Object.fromEntries(Object.entries(envTypeMap).map(([key, val]) => [key, val.name])),
    custom: t('自定义'),
  }));

  const envTagClass = computed(() => (props.data?.type ? envTypeTagClassMap[props.data.type] : ''));

  const envTagText = computed(() => {
    if (props.data?.type && envTypeMap[props.data.type]) {
      return envTypeMap[props.data.type]?.name;
    }
    return props.data.type;
  });

  // 从 apmList 中查找当前 APM 的完整信息（包含关联环境等）
  const currentApmDetail = computed(() => {
    if (!props.currentApm) return null;
    return apmList.value.find(apm => apm.apmID === props.currentApm!.apmID) ?? null;
  });

  // 当前 APM 是否为共享实例（关联多个环境）
  const isSharedApm = computed(() => (currentApmDetail.value?.associatedEnvs?.length ?? 0) > 1);

  // APM 标签
  const apmTagTheme = computed(() => (isSharedApm.value ? 'success' : 'info'));

  const apmTagText = computed(() => {
    if (!currentApmDetail.value) return '';
    if (!currentApmDetail.value.associatedEnvs?.length) return '';
    return isSharedApm.value ? t('共享') : t('独占');
  });

  // APM 提示信息
  const apmAlertMessage = computed(() => {
    if (!props.currentApm) return t('当前环境暂无关联的 APM 实例。');
    if (!isSharedApm.value) return t('该 APM 实例仅当前环境使用。');
    const envNames = currentApmDetail.value!.associatedEnvs?.map(env => env.envName).join('、') ?? '';
    return t('该实例被以下环境共享：{names}。', { names: envNames });
  });

  // 当前环境名在 APM 列表中是否已存在同名 APM
  const envApmExists = computed(() => apmList.value.some(apm => apm.name === props.data.name));

  // 按环境类型分组的 APM 列表
  const groupedApmList = computed(() => {
    const groups: Record<string, ApmOutput[]> = {};
    for (const apm of apmList.value) {
      const type = apm.type || 'custom';
      (groups[type] ??= []).push(apm);
    }
    return APM_GROUP_ORDER.filter(type => groups[type]?.length).map(type => ({
      type,
      label: localEnvTypeMap.value[type] || type,
      items: groups[type],
    }));
  });

  // 获取 APM 显示标签
  function getApmLabel(apm: ApmOutput) {
    const count = apm.associatedEnvs?.length ?? 0;
    if (count === 0) return apm.name;
    if (count === 1) return `${apm.name} (${t('独占')})`;
    return `${apm.name} (${t('共享中，关联{n}个环境', { n: count })})`;
  }

  // 获取 APM 列表
  async function getApmList() {
    apmListLoading.value = true;
    try {
      const res = await BkintegrationsBkmonitorService.listApms({
        workspaceID: spaceStore.currentSpace,
      }).catch(() => ({ results: [] }));
      apmList.value = res.results ?? [];
    } finally {
      apmListLoading.value = false;
    }
  }

  // 确认切换 APM
  async function handleConfirmSwitchApm() {
    const isValid = await formRef.value?.validate().catch(() => false);
    if (!isValid) return;

    switchApmLoading.value = true;
    try {
      if (formData.value.targetApmID === CREATE_NEW_APM_ID) {
        await BkintegrationsBkmonitorService.createEnvApm({ envID: props.data?.id || '' });
      } else {
        await BkintegrationsBkmonitorService.bindApmToEnv({
          apmID: String(formData.value.targetApmID!),
          envID: props.data?.id || '',
        });
      }
      Message({ theme: 'success', message: t('切换 APM 实例成功') });
      switchApmDialogVisible.value = false;
      emit('update:currentApm');
      await getApmList();
    } finally {
      switchApmLoading.value = false;
    }
  }

  // 复制环境名称
  function handleCopyEnvName() {
    copyText(props.data?.name || '');
  }

  // 打开切换 APM 弹窗
  function handleOpenSwitchApm() {
    formData.value.targetApmID = undefined;
    formData.value.confirmName = '';
    switchApmDialogVisible.value = true;
    getApmList();
  }

  // 加载数据
  getApmList();
</script>
