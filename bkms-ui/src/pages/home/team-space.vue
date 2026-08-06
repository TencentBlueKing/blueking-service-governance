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
    :title="t('新建空间')"
    :width="600"
    @closed="handleClose"
  >
    <div class="px-[24px] py-[18px]">
      <!-- 基础信息 -->
      <h3 class="mb-[16px] font-bold text-[#313238] text-[14px]">{{ $t('基础信息') }}</h3>
      <Form
        ref="formRef"
        form-type="vertical"
        :model="formData"
        :rules="rules"
      >
        <Form.FormItem
          :label="$t('空间 ID')"
          property="id"
          required
        >
          <Input
            v-model.trim="formData.id"
            class="w-[552px]"
            :maxlength="24"
            :minlength="2"
            :placeholder="rules.name[0].message"
          />
        </Form.FormItem>
        <Form.FormItem
          :label="$t('空间展示名')"
          property="displayName"
          required
        >
          <Input
            v-model.trim="formData.displayName"
            class="w-[552px]"
            clearable
            :maxlength="64"
            :minlength="2"
            :placeholder="rules.displayName[0].message"
          />
        </Form.FormItem>
        <Form.FormItem
          :label="$t('空间描述')"
          required
        >
          <Input
            v-model="formData.description"
            class="h-[74px] w-[552px]"
            :maxlength="100"
            :placeholder="$t('请输入')"
            type="textarea"
          />
        </Form.FormItem>
        <Form.FormItem
          :label="$t('管理员')"
          property="managers"
          required
        >
          <UserSelector
            v-model="formData.managers"
            class="w-[552px]"
            multiple
          />
        </Form.FormItem>
        <Form.FormItem
          :label="$t('容器项目来源')"
          property="containerSource"
          required
        >
          <ColRadio
            v-model="containerSource"
            :data="containerProjectSourceList"
            @change.once="handleFetchBkProjectList"
          />
        </Form.FormItem>
        <Form.FormItem
          :label="$t('容器项目')"
          property="bkCIProjectID"
          required
        >
          <Select
            v-model="formData.bkCIProjectID"
            :disabled="isCreateContainer"
            :loading="isBkProjectListLoading"
            :with-validate="false"
          >
            <Select.Option
              v-for="option in bkProjectList"
              :key="option.code"
              :disabled="option.isBoundWorkspace"
              :name="option.name"
              :value="option.code"
            >
              <Popover
                :disabled="!option.isBoundWorkspace"
                placement="bottom"
              >
                <span>{{ option.name }}</span>
                <template #content>
                  {{ $t('项目已被其他空间关联，无法重复关联') }}
                </template>
              </Popover>
            </Select.Option>
          </Select>
        </Form.FormItem>
        <Form.FormItem
          v-if="isCreateContainer"
          :label="$t('关联 CMDB 业务')"
          property="bkCCBizID"
          required
        >
          <Select
            v-model="formData.bkCCBizID"
            class="w-[552px]"
            :loading="isBkBizListLoading"
          >
            <Select.Option
              v-for="option in bkBizList"
              :key="option.bizID"
              :name="option.bizName"
              :value="Number(option.bizID)"
            />
          </Select>
        </Form.FormItem>
        <Form.FormItem
          class="mb-[6px]"
          :label="$t('镜像仓库')"
          required
        >
        </Form.FormItem>
        <ColRadio
          v-model="repositoryType"
          :data="imageGitList"
        >
          <template #current>
            <div
              v-if="repositoryType !== 'system'"
              class="bg-[#F5F7FA] w-full p-[12px] py-[24px] text-[12px] my-[8px]"
            >
              <div class="flex items-center whitespace-nowrap mb-[24px] custom-form-item form-item-has-tips">
                <Form.FormItem
                  class="min-w-[72px] mr-[22px]"
                  :label="$t('镜像命名空间')"
                  label-position="right"
                  label-width="80"
                  property="imageRegistry.registry"
                  required
                >
                </Form.FormItem>
                <div class="w-full">
                  <Input
                    v-model.trim="formData.imageRegistry!.registry"
                    :placeholder="$t('请输入镜像命名空间，如：mirrors.tencent.com/bksaas/')"
                  />
                  <div class="text-[#979BA5] mt-[4px]">
                    {{ $t('仅填写命名空间即可，创建应用的时候再填写具体的仓库名称') }}
                    <Popover trigger="click">
                      <Button
                        class="ml-[8px]"
                        text
                        theme="primary"
                      >
                        {{ $t('填写示例') }}
                      </Button>
                      <template #content>
                        <ul>
                          <li
                            v-for="item in imageGitDisplayInputTips"
                            :key="item.name"
                            class="text-[#fff] flex items-center w-full"
                          >
                            <div class="text-right shrink-0 min-w-[100px]">{{ item.name }}：</div>
                            <div class="text-wrap break-all">{{ item.value }}</div>
                          </li>
                        </ul>
                      </template>
                    </Popover>
                  </div>
                </div>
              </div>
              <div class="flex items-center whitespace-nowrap custom-form-item">
                <Form.FormItem
                  class="mb-0 min-w-[72px] mr-[22px]"
                  :label="$t('镜像账号')"
                  label-position="right"
                  property="imageRegistry.account"
                  required
                >
                </Form.FormItem>
                <div class="grid grid-cols-2 w-full gap-x-[8px]">
                  <Input
                    v-model.trim="formData.imageRegistry!.username"
                    :placeholder="$t('请输入账号')"
                  />
                  <Input
                    v-model.trim="formData.imageRegistry!.password"
                    :placeholder="$t('请输入密码')"
                    type="password"
                  />
                </div>
              </div>
            </div>
          </template>
        </ColRadio>
      </Form>
    </div>
    <template #footer>
      <div class="flex items-center">
        <Button
          class="mr-[8px]"
          :loading="isLoading"
          theme="primary"
          @click="submit"
        >
          {{ t('确定') }}
        </Button>
        <Button @click="handleClose">
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, reactive, ref, watch } from 'vue';

  import { Button, Form, Input, Message, Popover, Select, Sideslider } from 'bkui-vue';
  import { cloneDeep } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { BCSProjectOutput } from '~/@types/v1/bkintegrations-bcs';
  import { BusinessInfoOutput } from '~/@types/v1/bkintegrations-bkcc';
  import { CreateWorkspaceRequest } from '~/@types/v1/workspace';
  import { BkintegrationsBcsService, BkintegrationsBkccService } from '~/api/modules/v1';
  import { BKMS_REGEX } from '~/common/const';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useSpaceStore } from '~/stores/space';
  import { useUserStore } from '~/stores/user';

  import ColRadio from './components/col-radio.vue';

  interface Emits {
    (e: 'confirm'): void;
  }

  const isShow = defineModel('isShow', { type: Boolean });
  const emit = defineEmits<Emits>();
  // 用户store
  const userStore = useUserStore();
  const { t } = useI18n();
  const spaceStore = useSpaceStore();

  const isLoading = ref(false);
  const formRef = ref<InstanceType<typeof Form>>();

  // 容器项目来源 radiosConfig
  const containerProjectSourceList = computed(() => [
    {
      value: 'create',
      name: t('新建'),
      tips: t('平台自动创建容器项目 bkms-{0}', [`${formData.id || t('空间 ID')}`]),
    },
    {
      value: 'bind',
      name: t('绑定已有项目'),
      tips: t('仅能绑定具备运维权限的容器项目，并需同时拥有对应蓝盾项目的管理员权限'),
    },
  ]);
  // 容器项目列表
  const bkProjectList = ref<BCSProjectOutput[]>([]);

  // 镜像仓库 radiosConfig
  const imageGitList = computed(
    (): {
      isHideIcon?: boolean;
      name: string;
      tips?: string;
      value: 'external' | 'system';
    }[] => [
      {
        value: 'system',
        name: t('新建'),
        tips: t('在容器项目{0}的制品库中新建名为 repo 的 docker 仓库', [
          `${formData.bkCIProjectID ? `（${formData.bkCIProjectID}）` : ''}`,
        ]),
        isHideIcon: true,
      },
      {
        value: 'external',
        name: t('绑定已有镜像仓库'),
        isHideIcon: true,
      },
    ],
  );

  // 镜像命名空间填写示例
  const imageGitDisplayInputTips = [
    {
      name: t('仓库地址'),
      value: 'mirros.tencent.com/bkapps/blueking-cmdb/default',
    },
    {
      name: t('镜像命名空间'),
      value: 'mirros.tencent.com/bkapps',
    },
    {
      name: t('仓库名称'),
      value: 'blueking-cmdb/default',
    },
  ];

  const containerSource = ref<'bind' | 'create'>('create');
  // 是否为新建容器项目（新建时需要选择 CMDB 业务，绑定时禁用）
  const isCreateContainer = computed(() => containerSource.value === 'create');

  type FormData = Partial<CreateWorkspaceRequest> & Pick<CreateWorkspaceRequest, 'imageRegistry'>;
  const defaultData: FormData = {
    id: '',
    displayName: '',
    description: '',
    managers: [userStore.userInfo.user_id],
    bkCIProjectID: '',
    bkCCBizID: undefined,
    imageRegistry: {
      registry: '',
      username: '',
      password: '',
    },
  };
  const formData = reactive<FormData>(cloneDeep(defaultData));
  const { confirmBox, forceCleanDirtyTag, withPausedWatch } = useLeaveConfirm(formData);
  const repositoryType = ref<'external' | 'system'>('system');
  const rules = {
    name: [
      {
        validator: () => BKMS_REGEX.spaceNameRegex.test(formData.id || ''),
        message: t('请输入 1-27 字符的空间 ID，由小写字母、数字、中划线组成，以小写字母开头，提交后不可修改'),
        trigger: 'blur',
      },
    ],
    displayName: [
      {
        validator: () => BKMS_REGEX.spaceDisplayNameRegex.test(formData.displayName || ''),
        message: t('请输入 1-32 字符的空间名称'),
        trigger: 'blur',
      },
    ],
    managers: [
      {
        required: true,
        message: t('请输入至少一位管理员'),
        trigger: 'blur',
      },
    ],
    containerSource: [
      {
        required: true,
        validator: () => !!containerSource.value,
        trigger: 'blur',
      },
    ],
    'imageRegistry.registry': [
      {
        validator: () => !!formData.imageRegistry!.registry,
        trigger: 'blur',
        message: t('镜像命名空间不能为空'),
      },
    ],
    'imageRegistry.account': [
      {
        required: true,
        validator: () => {
          if (repositoryType.value === 'system') {
            return !!formData.imageRegistry!.username;
          }
          return !!(formData.imageRegistry!.username && formData.imageRegistry!.password);
        },
        trigger: 'blur',
        message: t('镜像账号不能为空'),
      },
    ],
    bkCCBizID: [
      {
        required: true,
        validator: () => !!formData.bkCCBizID,
        trigger: 'blur',
        message: t('关联的 CMDB 业务不能为空'),
      },
    ],
  };

  const isBkProjectListLoading = ref(false);

  // CMDB 业务列表
  const bkBizList = ref<BusinessInfoOutput[]>([]);
  const isBkBizListLoading = ref(false);

  // 侧边栏关闭前确认
  function handleBeforeClose(): Promise<boolean> {
    return confirmBox();
  }

  async function handleClose() {
    if (await handleBeforeClose()) {
      isShow.value = false;
    }
  }

  // 获取 CMDB 业务列表
  async function handleFetchBkBizList() {
    isBkBizListLoading.value = true;
    bkBizList.value = await BkintegrationsBkccService.listBKCCAuthorizedBusinesses().finally(
      () => (isBkBizListLoading.value = false),
    );
  }

  // 获取管理权限容器项目 仅当容器项目来源切换时触发一次
  async function handleFetchBkProjectList() {
    isBkProjectListLoading.value = true;
    bkProjectList.value = await BkintegrationsBcsService.listBCSAuthorizedProjects().finally(
      () => (isBkProjectListLoading.value = false),
    );
  }

  // 空间编辑或者场景
  async function submit() {
    const valid = await formRef.value?.validate().catch(() => false);
    if (!valid) return;
    try {
      isLoading.value = true;
      const { bkCCBizID, ...restFormData } = formData;
      const params = {
        ...restFormData,
        // 新建容器项目时，bkCIProjectID 为空
        bkCIProjectID: isCreateContainer.value ? '' : formData.bkCIProjectID,
        // 仅新建容器项目时传递 bkCCBizID
        ...(isCreateContainer.value ? { bkCCBizID: formData.bkCCBizID } : {}),
        imageRegistry: repositoryType.value === 'system' ? null : formData.imageRegistry,
      } as CreateWorkspaceRequest;
      const result = await spaceStore.handleCreateWorkspace(params);
      if (result) {
        forceCleanDirtyTag(() => {
          isShow.value = false;
          emit('confirm');
          Message({
            theme: 'success',
            message: t('空间创建成功'),
          });
        });
      } else {
        Message({
          theme: 'error',
          message: t('空间创建失败'),
        });
      }
    } finally {
      isLoading.value = false;
    }
  }

  // 输入空间ID或切换容器项目来源，自动同步容器项目
  watch(
    () => [formData.id, containerSource.value],
    ([newName, newSource]) => {
      if (newSource === 'create' && newName) {
        formData.bkCIProjectID = `bkms-${newName}`;
      }
      if (newSource === 'bind') {
        formData.bkCIProjectID = '';
        formData.bkCCBizID = undefined;
      }
    },
  );

  // 防止userStore数据异步
  watch(
    () => userStore.userInfo.user_id,
    val => {
      if (val) {
        defaultData.managers = [val];
        formData.managers = [val];
      }
    },
  );

  // 重置数据
  watch(isShow, () => {
    if (!isShow.value) {
      withPausedWatch(() => {
        // 重置数据
        Object.assign(formData, cloneDeep(defaultData));
      });
      formRef.value?.clearValidate();
    } else {
      handleFetchBkBizList();
    }
  });
</script>

<style lang="postcss" scoped>
  :deep(.bk-form-label) {
    font-size: 12px;
  }

  :deep(.custom-form-item) {
    .bk-form-label {
      margin-bottom: 0px;
    }
    .bk-form-content {
      .bk-form-error {
        margin-left: 94px;
        margin-top: 6px;
      }
    }
  }

  :deep(.form-item-has-tips) {
    .bk-form-label {
      margin-bottom: 0px;
    }
    .bk-form-content {
      .bk-form-error {
        margin-left: 94px;
        margin-top: 26px;
      }
    }
  }
</style>
