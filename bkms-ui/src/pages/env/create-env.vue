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
    :before-close="handleBeforeClose"
    render-directive="if"
    :title="$t('新建环境')"
    :width="600"
    @closed="handleClose"
  >
    <Form
      ref="formRef"
      class="p-[24px]"
      form-type="vertical"
      :model="formData"
      :rules="rules"
    >
      <ToggleCard
        class="mb-[24px]"
        :name="$t('基础信息')"
        type="normal"
      >
        <Form.FormItem
          :label="$t('环境名称')"
          property="name"
          required
        >
          <Input
            v-model="formData.name"
            :placeholder="rules.name[0].message"
          />
        </Form.FormItem>
        <Form.FormItem
          :label="$t('环境展示名')"
          property="displayName"
          required
        >
          <Input
            v-model="formData.displayName"
            :placeholder="rules.displayName[0].message"
          />
        </Form.FormItem>
        <Form.FormItem
          :label="$t('环境分类')"
          property="type"
          required
        >
          <Select
            v-model="formData.type"
            :filterable="false"
          >
            <Select.Option
              v-for="item in envTypeList"
              :key="item.id"
              :name="item.name"
              :value="item.id"
            >
            </Select.Option>
          </Select>
        </Form.FormItem>
        <div class="mb-[16px] font-700 text-[14px] text-[#313238]">
          {{ `APM ${$t('配置')}` }}
        </div>
        <Radio.Group
          v-model="apmMode"
          class="apm-radio-card-group flex flex-col gap-[8px]"
        >
          <div
            v-for="item in apmModeOptions"
            :key="item.value"
            class="apm-radio-card cursor-pointer rounded-[2px] border border-solid border-[#dcdee5] px-[16px] py-[12px] transition-all duration-300"
            :class="{ 'is-active': apmMode === item.value }"
            @click="apmMode = item.value"
          >
            <Radio :label="item.value">
              <div class="ml-[8px]">
                <div class="font-700 text-[14px] text-[#4D4F56]">{{ item.title }}</div>
                <div class="mt-[4px] text-[12px] text-[#979BA5]">{{ item.desc }}</div>
              </div>
            </Radio>
          </div>
        </Radio.Group>
        <div
          v-if="apmMode === 'bindExist'"
          class="bg-[#F5F7FA] px-[16px] py-[12px] mt-[24px]"
        >
          <Form.FormItem
            class="mb-[0]"
            :label="$t('选择 APM 实例')"
            property="apmID"
            required
          >
            <Select
              v-model="formData.apmID"
              filterable
              :loading="apmListLoading"
              :placeholder="$t('请选择 APM 实例')"
            >
              <Select.Group
                v-for="group in groupedApmList"
                :key="group.type"
                :label="group.label"
              >
                <Select.Option
                  v-for="apm in group.items"
                  :key="apm.apmID"
                  :label="getApmLabel(apm)"
                  :value="apm.apmID"
                />
              </Select.Group>
            </Select>
          </Form.FormItem>
        </div>
      </ToggleCard>
      <ToggleCard
        :name="$t('集群资源')"
        type="normal"
      >
        <!-- 容器项目 -->
        <Form.FormItem
          :label="$t('容器项目')"
          property="cluster.projectCode"
          required
        >
          <ProjectSelector
            class="w-full"
            disabled
            :project-code="formData.cluster.projectCode || ''"
            @change="handleProjectCodeChange"
          />
        </Form.FormItem>
        <Form.FormItem
          :label="$t('集群')"
          property="cluster.clusterID"
          required
        >
          <ClusterSelector
            class="w-full"
            :list="clusterData"
            :loading="loading"
            :project-code="formData.cluster.projectCode || ''"
            :value="formData.cluster.clusterID || ''"
            @update:cluster-type="handleClusterTypeChange"
            @update:value="handleClusterIDChange"
          />
        </Form.FormItem>
        <Form.FormItem
          :label="$t('命名空间')"
          property="cluster.namespace"
          required
        >
          <NamespaceSelector
            class="w-full"
            :cluster-id="formData.cluster.clusterID || ''"
            :project-i-d="projectID"
            :value="formData.cluster.namespace || ''"
            @update:value="handleNamespaceChange"
          />
        </Form.FormItem>
        <Form.FormItem>
          <Button
            :loading="isLoading"
            theme="primary"
            @click="handleConfirm"
          >
            {{ $t('确定') }}
          </Button>
          <Button
            class="ml-[8px]"
            @click="handleClose"
          >
            {{ $t('取消') }}
          </Button>
        </Form.FormItem>
      </ToggleCard>
    </Form>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Button, Form, Input, Radio, Select, Sideslider } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { ApmOutput } from '~/@types/v1/bkintegrations-bkmonitor';
  import { CreateEnvInput, CreateEnvRequest } from '~/@types/v1/env';
  import { BkintegrationsBkmonitorService, EnvService, WorkspaceService } from '~/api/modules/v1';
  import { BKMS_REGEX } from '~/common/const';
  import useClusterSelector from '~/components/cluster-selector/use-cluster-selector';
  import { envTypeMap } from '~/composables/use-env-manager';
  import { useErrorHandler } from '~/composables/use-error-handler';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useSpaceStore } from '~/stores/space';

  import ProjectSelector from './components/project-selector/project-selector.vue';

  interface ClusterInfo {
    clusterID: string;
    clusterType: string;
    namespace: string;
    projectCode: string;
  }
  interface Emits {
    (e: 'confirm'): void;
  }
  const isShow = defineModel<boolean>('isShow');
  const emit = defineEmits<Emits>();

  const { t } = useI18n();
  const spaceStore = useSpaceStore();
  const apmMode = ref<'auto' | 'bindExist'>('auto');
  const apmModeOptions = computed(() => [
    {
      value: 'auto' as const,
      title: t('自动创建'),
      desc: t('自动创建独立 APM 实例，适用于大多数场景'),
    },
    {
      value: 'bindExist' as const,
      title: t('关联已有 APM'),
      desc: t('关联到同一环境分类下已有的 APM 实例，适用于多个环境共用 APM 场景'),
    },
  ]);

  const formData = ref<Partial<CreateEnvRequest> & { cluster: ClusterInfo }>({
    name: '',
    displayName: '',
    type: '' as CreateEnvInput['type'],
    apmID: undefined,
    cluster: {
      // 集群
      clusterID: '',
      clusterType: '',
      // 命名空间
      namespace: '',
      // 容器项目
      projectCode: '',
    },
  });
  const { confirmBox, forceCleanDirtyTag } = useLeaveConfirm(formData);

  // 集群选择器相关回调
  const handleProjectCodeChange = (val: string) => {
    formData.value.cluster.projectCode = val;
  };
  const handleClusterTypeChange = (val: string) => {
    formData.value.cluster.clusterType = val;
  };
  const handleClusterIDChange = (val: string) => {
    formData.value.cluster.clusterID = val;
  };
  const handleNamespaceChange = (val: string) => {
    formData.value.cluster.namespace = val;
  };

  // 集群相关变量
  const projectID = ref<string>('');
  const { loading, clusterData, getClusterList } = useClusterSelector(projectID.value || '', 'all');

  const isLoading = ref(false);
  const formRef = ref();
  const envTypeList = Object.entries(envTypeMap).map(([id, config]) => ({ id, name: config.name }));
  const rules = ref({
    name: [
      {
        validator: () => BKMS_REGEX.envNameRegex.test(formData.value.name || ''),
        message: t('请输入 1-20 个字符小写字母、数字、中划线，以小写字母开头，提交后不可修改'),
        trigger: 'blur',
      },
    ],
    displayName: [
      {
        validator: () => BKMS_REGEX.envDisplayNameRegex.test(formData.value.displayName || ''),
        message: t('请输入 1-32 字符的环境展示名'),
        trigger: 'blur',
      },
    ],
    type: [
      {
        required: true,
        message: t('环境分类不能为空'),
        trigger: 'change',
      },
    ],
    'cluster.projectCode': [
      {
        required: true,
        message: t('请选择容器项目'),
        trigger: 'change',
      },
    ],
    'cluster.clusterID': [
      {
        required: true,
        message: t('请选择集群'),
        trigger: 'change',
      },
    ],
    'cluster.namespace': [
      {
        required: true,
        message: t('请选择命名空间'),
        trigger: 'change',
      },
    ],
  });

  // 确认创建环境
  const handleConfirm = async () => {
    const isValid = await formRef.value?.validate().catch(() => false);
    if (!isValid) return;
    isLoading.value = true;

    try {
      const { apmID, ...rest } = formData.value;
      await EnvService.createEnv(
        {
          workspaceID: spaceStore.currentSpace,
          ...rest,
          // 关联已有 APM 时传 apmID，自动创建时不传
          ...(apmMode.value === 'bindExist' && apmID ? { apmID: Number(apmID) } : {}),
        } as unknown as CreateEnvRequest,
        { validateCode: false },
      );
      forceCleanDirtyTag(() => {
        isShow.value = false;
        emit('confirm');
      });
    } catch (err: unknown) {
      const { handleError } = useErrorHandler();
      handleError((err as { error: { code?: string; message?: string } }).error);
    } finally {
      isLoading.value = false;
    }
  };

  // 获取空间详情
  async function getWorkspaceDetail() {
    const ret = await WorkspaceService.getWorkspace(
      {
        workspaceID: spaceStore.currentSpace,
      },
      { validateCode: false },
    );
    formData.value.cluster.projectCode = ret.bkSystems?.bkBCSProjectCode ?? '';
    projectID.value = ret.bkSystems?.bkBCSProjectID ?? '';
  }

  // 侧边栏关闭前确认
  function handleBeforeClose(): Promise<boolean> {
    return confirmBox();
  }

  // 关闭弹窗
  async function handleClose() {
    if (await handleBeforeClose()) {
      isShow.value = false;
    }
  }

  // APM 列表
  const apmList = ref<ApmOutput[]>([]);
  const apmListLoading = ref(false);

  // 环境类型名称映射（基于公共 envTypeMap，追加 custom）
  const localEnvTypeMap = computed<Record<string, string>>(() => ({
    ...Object.fromEntries(Object.entries(envTypeMap).map(([key, val]) => [key, val.name])),
    custom: t('自定义'),
  }));

  // 按环境类型分组的 APM 列表
  const groupedApmList = computed(() => {
    const groups: Record<string, ApmOutput[]> = {};
    for (const apm of apmList.value) {
      const type = apm.type || 'custom';
      if (!groups[type]) {
        groups[type] = [];
      }
      groups[type].push(apm);
    }

    // 转为数组，按固定顺序排列
    const order = ['production', 'development', 'test', 'custom'];
    return order
      .filter(type => groups[type]?.length)
      .map(type => ({
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
    const res = await BkintegrationsBkmonitorService.listApms({
      workspaceID: spaceStore.currentSpace,
    }).catch(() => {
      return { results: [] };
    });
    apmList.value = res.results ?? [];
    apmListLoading.value = false;
  }

  watch(isShow, async val => {
    if (val) {
      await getWorkspaceDetail();
      getApmList();
    } else {
      apmMode.value = 'auto';
      formData.value = {
        name: '',
        displayName: '',
        type: '' as CreateEnvRequest['type'],
        apmID: undefined,
        cluster: {
          projectCode: '',
          clusterID: '',
          namespace: '',
          clusterType: '',
        },
      };
    }
    forceCleanDirtyTag();
  });

  // 监听容器项目变化，获取集群列表
  watch(
    () => projectID.value,
    async newProjectCode => {
      if (newProjectCode) {
        await getClusterList(newProjectCode);
      } else {
        projectID.value = '';
      }
    },
  );
</script>

<style lang="postcss" scoped>
  .apm-radio-card-group {
    :deep(.bk-radio-group) {
      display: flex;
      flex-direction: column;
      gap: 8px;
    }
  }

  .apm-radio-card {
    &:hover {
      border-color: #3a84ff;
    }

    &.is-active {
      border-color: #3a84ff;
      background: #f5f7fa;
    }

    :deep(.bk-radio) {
      align-items: flex-start;
    }

    :deep(.bk-radio-input) {
      margin-top: 2px;
    }

    :deep(.bk-radio-label) {
      font-size: 12px;
    }
  }
</style>
