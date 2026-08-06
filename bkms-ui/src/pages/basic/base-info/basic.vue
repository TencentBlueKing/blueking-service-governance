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
  <Skeleton
    class="bg-[#fff] p-[24px]"
    :loading="isLoading"
  >
    <template #loading>
      <div class="flex">
        <Layout.shape
          :height="80"
          :width="80"
        />
        <div class="ml-[16px]">
          <Layout.shape
            class="mt-[10px] !block"
            :height="18"
            :width="200"
          />
          <Layout.shape
            class="mt-[24px] mb-[24px] !block"
            :height="18"
            :width="400"
          />
          <Layout.shape class="mr-[8px]" />
          <Layout.shape class="mr-[8px]" />
        </div>
      </div>
      <Layout.shape
        class="mt-[60px]"
        :height="28"
        width="100%"
      />
      <div class="grid grid-cols-2 gap-4 text-[12px] mt-[16px] pl-[80px]">
        <Layout.formItem
          :item-height="18"
          :label-height="18"
        />
        <Layout.formItem
          :item-height="18"
          :label-height="18"
        />
        <Layout.formItem
          :item-height="18"
          :label-height="18"
        />
      </div>
      <Layout.shape
        class="mt-[24px]"
        :height="28"
        width="100%"
      />
      <div class="grid grid-cols-2 gap-4 text-[12px] mt-[16px] pl-[80px]">
        <Layout.formItem
          :item-height="18"
          :label-height="18"
        />
        <Layout.formItem
          :item-height="18"
          :label-height="18"
        />
      </div>
      <Layout.shape
        class="mt-[24px]"
        :height="28"
        width="100%"
      />
      <FlexRow
        average
        class="mt-[16px] px-[40px]"
        lclass="flex items-center"
        rclass="flex justify-end ml-[8px]"
      >
        <template #left>
          <Layout.shape
            :height="32"
            :width="100"
          />
          <Layout.shape
            class="ml-[16px] flex-1 max-w-[490px]"
            :height="32"
          />
          <Layout.shape
            class="ml-[8px]"
            :height="32"
            :width="72"
          />
        </template>
        <template #right>
          <Layout.shape
            class="flex-1 max-w-[400px]"
            :height="32"
          />
        </template>
      </FlexRow>
      <Layout.table class="mt-[16px] px-[40px]" />
    </template>
    <!-- 基本信息 -->
    <div class="w-[clac(100% - 80px)] min-h-[172px] bg-[#fff] p-[24px] flex shadow-[0_2px_4px_0_#1919290d]">
      <div class="w-[80px]">
        <div class="w-[100%] h-[80px] bg-[#f0f5ff] rounded-lg flex items-center justify-center">
          <span class="bkms-icon bkms-icon-space-basic text-[38px] text-[#3a84ff]"></span>
        </div>
      </div>
      <div class="w-full">
        <div class="mx-[16px] mt-[10px]">
          <EditBlock
            class="h-[30px]"
            :disabled="curSpace?.state !== spaceStore.spaceState.Ready"
            :loading="isLoading"
            @cancel="() => handleCancel('displayName')"
            @confirm="handleConfirm"
          >
            <template #text>
              <span class="font-bold mr-[5px]">{{ curSpace?.displayName || '--' }}</span>
              <span class="ml-[5px] mr-[5px] text-[12px] text-[#979BA5]">
                {{ curSpace?.id ? `( ${curSpace?.id} )` : '' }}
              </span>
            </template>
            <template #edit="{ focus }">
              <Input
                :ref="(el: InputType) => focus(el, 'input')"
                v-model="formData.displayName"
                class="mr-[8px] w-[300px]"
                :maxlength="24"
                :minlength="2"
                :placeholder="$t('请输入')"
              />
            </template>
          </EditBlock>
          <div class="flex mt-[10px]">
            <span class="text-[12px]">{{ $t('空间描述') }}：</span>
            <EditBlock
              :disabled="curSpace?.state !== spaceStore.spaceState.Ready"
              :loading="isLoading"
              @cancel="() => handleCancel('description')"
              @confirm="handleConfirm"
            >
              <template #text>
                <span class="text-[12px] mr-[5px]">{{ curSpace?.description || '--' }}</span>
              </template>
              <template #edit="{ focus }">
                <Input
                  :ref="(el: InputType) => focus(el, 'input')"
                  v-model="formData.description"
                  class="mr-[8px] w-[300px]"
                  :placeholder="$t('请输入')"
                  type="textarea"
                />
              </template>
            </EditBlock>
          </div>
        </div>
        <div class="ml-[16px] mt-[24px]">
          <PopConfirm
            :content="$t('停用后将无法使用，请谨慎操作！')"
            :disabled="curSpace?.state !== spaceStore.spaceState.Ready"
            placement="top"
            :title="$t('确定停用该空间？')"
            trigger="click"
            width="280"
            @confirm="handleChangeStatus"
          >
            <Button class="w-[88px]">
              {{ $t('停用') }}
            </Button>
          </PopConfirm>
        </div>
      </div>
    </div>

    <div class="w-[clac(100% - 80px)] bg-[#fff] p-[24px] mt-[16px] shadow-[0_2px_4px_0_#1919290d]">
      <!-- 空间仓库 -->
      <BkmsContent :title="$t('镜像仓库')">
        <div class="grid grid-cols-4 gap-4 mt-[16px] text-[12px] w-full pl-[80px] place-items-center">
          <FieldItem
            :field-value="$t('类型')"
            :value="spaceStore.getRepositoryTypeName(curSpace?.imageRegistryType)"
          />
          <FieldItem
            class="col-span-3"
            :field-value="$t('仓库地址')"
            :value="curSpace?.imageRegistry?.registry"
            value-max-width="100%"
          />
          <FieldItem
            class="col-span-4"
            :field-value="$t('仓库凭证')"
          >
            <template #value>
              <SecretToggle
                v-if="imageRegistryInfo.display"
                :value="imageRegistryInfo.display"
              />
              <Copy
                v-if="imageRegistryInfo.copy"
                v-bk-tooltips="{
                  content: $t('复制凭据信息'),
                }"
                class="ml-[6px] cursor-pointer hover:text-[#3A84FF]"
                :title="$t('复制')"
                @click="copyText(imageRegistryInfo.copy)"
              >
              </Copy>
            </template>
          </FieldItem>
        </div>
      </BkmsContent>
      <!-- 关联项目 -->
      <BkmsContent
        class="mt-[24px]"
        :title="$t('关联项目')"
      >
        <div class="grid grid-cols-4 gap-4 mt-[16px] text-[12px] w-full pl-[80px] place-items-center">
          <FieldItem
            :field-value="$t('项目 ID')"
            :value="curSpace?.bkSystems?.bkCIProjectID"
          />
          <FieldItem
            class="col-span-3"
            :field-value="$t('涉及的平台项目')"
          >
            <template #value>
              <div class="flex">
                <div>{{ $t('蓝盾') }}</div>
                <Share
                  class="text-[#3A84FF] mx-[5px] cursor-pointer"
                  @click="() => handleToLink('devops', curSpace?.bkSystems?.bkCIProjectID)"
                >
                </Share>
                <span>，</span>
              </div>
              <div class="flex">
                <div>{{ $t('容器项目') }}</div>
                <Share
                  class="text-[#3A84FF] mx-[5px] cursor-pointer"
                  @click="() => handleToLink('bcs', curSpace?.bkSystems?.bkBCSProjectCode)"
                >
                </Share>
                <span>，</span>
              </div>
              <div class="flex">
                <div>{{ $t('监控') }}</div>
                <Share
                  class="text-[#3A84FF] mx-[5px] cursor-pointer"
                  @click="() => handleToLink('monitor', curSpace?.bkSystems?.bkMonitorProjectID)"
                >
                </Share>
              </div>
            </template>
          </FieldItem>
        </div>
      </BkmsContent>
    </div>

    <div class="w-[clac(100% - 80px)] bg-[#fff] p-[24px] mt-[16px] shadow-[0_2px_4px_0_#1919290d]">
      <BkmsContent :title="$t('成员管理')">
        <div class="pt-[16px] px-[40px]">
          <FlexRow
            average
            lclass="flex items-center"
            rclass="flex justify-end ml-[8px]"
          >
            <template #left>
              <Button
                theme="primary"
                @click="isShowAddMember = true"
              >
                <Plus
                  :height="24"
                  :width="24"
                />
                <span class="h-[24px] leading-[24px]">{{ $t('新增成员') }}</span>
              </Button>
              <Radio.Group
                v-model="memberType"
                class="ml-[16px]"
                type="capsule"
              >
                <Radio.Button label="all">
                  {{ `${$t('全部成员')} ( ${memberData.length} )` }}
                </Radio.Button>
                <Radio.Button label="admin">
                  {{ `${$t('管理员')} ( ${memberRoleMap['admin']?.length} )` }}
                </Radio.Button>
                <Radio.Button label="developer">
                  {{ `${$t('开发者')} ( ${memberRoleMap['developer']?.length} )` }}
                </Radio.Button>
                <Radio.Button label="sre">
                  {{ `SRE (${memberRoleMap['sre']?.length})` }}
                </Radio.Button>
                <Radio.Button label="operator">
                  {{ `${$t('运营者')} ( ${memberRoleMap['operator']?.length} )` }}
                </Radio.Button>
              </Radio.Group>
              <Button
                class="ml-[8px]"
                text
                theme="primary"
                @click="isShow = true"
                >{{ $t('查看权限模型') }}</Button
              >
            </template>
            <template #right>
              <Input
                v-model.trim="searchValue"
                class="max-w-[400px]"
                clearable
                :placeholder="$t('搜索用户名')"
                type="search"
              />
            </template>
          </FlexRow>
          <Table
            class="mt-[16px]"
            :data="tableData"
            :pagination="pagination"
            :row-config="{
              isHover: true,
              isCurrent: true,
            }"
          >
            <template #empty>
              <TableException
                :type="curExceptionType"
                @clear="searchValue = ''"
                @refresh="getListWorkspaceRoleMemberGroups"
              />
            </template>
            <TableColumn
              field="username"
              :label="$t('用户名')"
              :width="260"
            />
            <TableColumn
              :label="$t('角色')"
              :width="260"
            >
              <template #default="{ row }: { row: Row }">
                <Tag
                  v-for="(item, index) in row.role"
                  :key="`${index}-${item}`"
                  class="mr-[4px]"
                  :class="roleMap?.[item]?.style"
                  size="small"
                >
                  {{ roleMap?.[item]?.label }}
                </Tag>
              </template>
            </TableColumn>
            <TableColumn :label="$t('权限描述')">
              <template #default="{ row }">
                {{ getMemberRoleDescription(row.role) }}
              </template>
            </TableColumn>
            <TableColumn
              :label="$t('操作')"
              :width="200"
            >
              <template #default="{ row }: { row: Row }">
                <Button
                  text
                  theme="primary"
                  @click="handleDeleteMember(row)"
                  >{{ $t('删除成员') }}</Button
                >
              </template>
            </TableColumn>
          </Table>
        </div>
      </BkmsContent>
    </div>
  </Skeleton>
  <Sideslider
    v-model:is-show="isShow"
    :title="$t('权限模型')"
    :width="960"
  >
    <permissionModel class="px-[24px] pt-[18px]" />
  </Sideslider>
  <Sideslider
    v-model:is-show="isShowAddMember"
    :before-close="handleBeforeClose"
    :title="$t('新增成员')"
    :width="960"
    @hidden="handleHidden"
  >
    <div class="px-[24px] pt-[18px]">
      <Form
        ref="formRef"
        form-type="vertical"
        :model="formModel"
      >
        <Form.FormItem
          :label="$t('用户名')"
          property="username"
          required
        >
          <UserSelector
            v-model="formModel.username"
            multiple
          />
        </Form.FormItem>
        <Form.FormItem
          class="!mb-0"
          :label="$t('角色')"
          property="role"
          required
        >
          <Radio.Group v-model="formModel.role">
            <Radio label="admin">{{ $t('管理员') }}</Radio>
            <Radio label="developer">{{ $t('开发者') }}</Radio>
            <Radio label="sre">SRE</Radio>
            <Radio label="operator">{{ $t('运营者') }}</Radio>
          </Radio.Group>
        </Form.FormItem>
      </Form>
      <PermissionContent
        class="mb-[24px]"
        :value="formModel.role"
      />
      <permissionModel
        :active="formModel.role"
        @change="handleRoleChange"
      />
    </div>
    <template #footer>
      <Button
        :loading="isLoading"
        theme="primary"
        @click="handleSubmit"
        >{{ $t('确定') }}</Button
      >
      <Button
        class="ml-[8px]"
        :loading="isLoading"
        @click="handleClose"
        >{{ $t('取消') }}</Button
      >
    </template>
  </Sideslider>
</template>

<script setup lang="ts">
  import type { Ref } from 'vue';
  import { computed, h, nextTick, onMounted, reactive, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Form, InfoBox, Input, Message, PopConfirm, Radio, Sideslider, Tag } from 'bkui-vue';
  import { Copy, Plus, Share } from 'bkui-vue/lib/icon';
  import { cloneDeep } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { UpdateWorkspaceInfoRequest, WorkspaceDetailOutputObj } from '~/@types/v1/workspace';
  import { WorkspaceService } from '~/api/modules/v1';
  import BkmsContent from '~/components/bkms-content.vue';
  import EditBlock from '~/components/edit-block.vue';
  import FieldItem from '~/components/field-item.vue';
  import FlexRow from '~/components/flex-row.vue';
  import SecretToggle from '~/components/secret-toggle.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import UserSelector from '~/components/user-selector.vue';
  import { useCopy } from '~/composables/use-copy';
  import useDebouncedRef from '~/composables/use-debounce';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import useTableEmpty from '~/composables/use-table-empty';
  import useToLink from '~/composables/use-to-link';
  import { useSpaceStore } from '~/stores/space';

  import PermissionContent from './permission-content.vue';
  import permissionModel from './permissionModel.vue';

  import type { InputType } from 'bkui-vue/lib/input/input';

  type RoleCode = 'admin' | 'developer' | 'operator' | 'sre';

  interface Row {
    role: RoleCode[];
    username: string;
  }

  const router = useRouter();
  const spaceStore = useSpaceStore();
  const { handleToLink } = useToLink();
  const { copyText } = useCopy();
  const { t } = useI18n();

  const defaultData: Partial<WorkspaceDetailOutputObj> = {
    id: '',
    displayName: '',
    description: '',
  };
  const formData = reactive(cloneDeep(defaultData));

  // 取消修改
  async function handleCancel(properties: keyof Pick<WorkspaceDetailOutputObj, 'description' | 'displayName'>) {
    if (curSpace.value) {
      formData[properties] = curSpace.value[properties];
    }
  }

  // 停用
  async function handleChangeStatus() {
    await WorkspaceService.setWorkspaceState({
      workspaceID: spaceStore.currentSpace,
      state: 'Disabled',
    });
    // 返回类型为EmptyResponse，即无返回值，停用直接跳转至首页
    router.replace({ name: 'home' });
  }

  // 基本信息修改
  async function handleConfirm() {
    try {
      isLoading.value = true;
      const params: UpdateWorkspaceInfoRequest = {
        workspaceID: spaceStore.currentSpace,
        displayName: formData.displayName || '',
        description: formData.description || '',
      };
      await spaceStore.handleUpdateWorkspace(params);
      await spaceStore.handleGetWorkspaceList();
      getWorkspaceDetail();
    } catch (err) {
      console.error(err);
    } finally {
      isLoading.value = false;
    }
  }

  const isLoading = ref(false);
  const curSpace = ref<WorkspaceDetailOutputObj>();

  // 镜像仓库凭据信息（用于显示和复制）
  const imageRegistryInfo = computed(() => {
    if (!curSpace.value?.imageRegistry) {
      return { display: '', copy: '' };
    }
    const { username, password } = curSpace.value.imageRegistry;
    return {
      display: `${username}：${password}`,
      copy: `${username} ${password}`,
    };
  });

  // 获取工作空间下角色成员组列表
  async function getListWorkspaceRoleMemberGroups() {
    try {
      const res = await WorkspaceService.listWorkspaceRoleMemberGroups(
        {
          workspaceID: spaceStore.currentSpace,
        },
        { validateCode: false },
      );

      // 初始化角色映射表
      memberRoleMap.value = { admin: [], developer: [], sre: [], operator: [] };

      const tempData: Record<string, Row> = {};

      // 处理角色成员组数据
      res.forEach(group => {
        const roleCode = group.roleCode as RoleCode;
        memberRoleMap.value[roleCode] = group.members || [];

        group.members?.forEach(username => {
          if (tempData[username]) {
            tempData[username].role.push(roleCode);
          } else {
            tempData[username] = { username, role: [roleCode] };
          }
        });
      });

      memberData.value = Object.values(tempData);
      pagination.value.count = memberData.value.length;
      clearErrorType();
    } catch (error) {
      console.error(error);
      setTypeToError();
    }
  }

  // 获取空间详情
  async function getWorkspaceDetail() {
    try {
      isLoading.value = true;
      pagination.value.current = 1;
      const curRowData = await WorkspaceService.getWorkspace(
        {
          workspaceID: spaceStore.currentSpace,
        },
        { validateCode: false },
      );
      curSpace.value = curRowData;
      await getListWorkspaceRoleMemberGroups();

      Object.assign(formData, curRowData);
    } catch (err) {
      console.error(err);
    } finally {
      isLoading.value = false;
    }
  }

  // 空间成员相关
  const memberType = ref<'all' | RoleCode>('all');
  const roleMap = {
    admin: {
      label: t('管理员'),
      style: 'bg-[#DAF6E5] text-[#299E56] text-[10px]',
    },
    developer: {
      label: t('开发者'),
      style: 'bg-[#FDEED8] text-[#E38B02] text-[10px]',
    },
    sre: {
      label: 'SRE',
      style: 'bg-[#E1ECFF] text-[#1768EF] text-[10px]',
    },
    operator: {
      label: t('运营者'),
      style: 'bg-[#D6F0F9] text-[#428499] text-[10px]',
    },
  };
  const memberRoleMap = ref<Partial<Record<RoleCode, string[]>>>({});
  const memberData = ref<Row[]>([]);
  const tableData = computed(() =>
    memberData.value.filter(item => {
      const result1 = memberType.value === 'all' || item.role.includes(memberType.value);
      if (searchValue.value) {
        const result2 = item.username.toLowerCase().includes(searchValue.value.toLowerCase());
        return result1 && result2;
      }
      return result1;
    }),
  );
  const pagination = ref({ count: 0, limit: 10, current: 1 });
  const searchValue = useDebouncedRef('', 300) as Ref<string>; // 搜索值
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchValue,
  });
  /**
   * 角色描述
   * @param role 角色
   */
  function getMemberRoleDescription(role: RoleCode[]) {
    if (role.includes('admin')) {
      return t('拥有全部权限');
    }
    if (role.length === 1) {
      if (role[0] === 'developer') {
        return t('专注于应用开发和部署，可以创建和管理应用，部署环境，但无法管理空间和环境的配置');
      }
      if (role[0] === 'sre') {
        return t('负责环境运维管理，可以创建、编辑、删除环境，创建和管理应用，但无法删除应用和管理空间');
      }
      return t('仅具有只读权限，可以查看所有空间、应用和环境信息，但无法进行任何修改操作');
    }
    if (role.length === 2) {
      if (role.includes('developer') && role.includes('sre')) {
        return t('可以创建和管理应用，部署和管理环境，但无法管理空间');
      }
      if (role.includes('developer') && role.includes('operator')) {
        return t('可以创建和管理应用，部署环境，同时具有全局查看权限，但无法管理空间和环境配置');
      }
      return t('可以创建、编辑、删除环境，创建和管理应用，同时具有全局查看权限，但无法删除应用和管理空间');
    }
    if (role.length === 3) {
      return t('可以创建和管理应用，管理环境，具有全局查看权限，但无法管理空间');
    }
    return '--';
  }

  // 删除成员
  function handleDeleteMember(row: Row) {
    InfoBox({
      title: t('确定删除该成员？'),
      contentAlign: 'left',
      content: h('div', [
        h('div', t('成员：{0}', [row.username])),
        h(
          'p',
          { class: 'mt-[16px] px-[16px] py-[12px] bg-[#F5F7FA]' },
          t('删除后该用户将失去此空间的所有权限，请谨慎操作'),
        ),
      ]),
      confirmText: t('删除'),
      cancelText: t('取消'),
      confirmButtonTheme: 'danger',
      onConfirm: async () => {
        const result = await WorkspaceService.removeWorkspaceUser(
          {
            workspaceID: spaceStore.currentSpace,
            userID: row.username,
          },
          { validateCode: false },
        )
          .then(() => true)
          .catch(() => false);
        if (result) {
          getListWorkspaceRoleMemberGroups();
          Message({
            theme: 'success',
            message: t('操作成功'),
          });
        }
      },
    });
  }

  // 权限模型
  const isShow = ref<boolean>(false);
  // 新增成员
  const isShowAddMember = ref<boolean>(false);
  const formRef = ref<InstanceType<typeof Form>>();
  const formModel = ref<{
    role: RoleCode;
    username: string[];
  }>({
    username: [],
    role: 'developer',
  });
  const { confirmBox, forceCleanDirtyTag, withPausedWatch } = useLeaveConfirm(formModel);

  // 侧边栏关闭前确认
  function handleBeforeClose(): Promise<boolean> {
    return confirmBox();
  }

  async function handleClose() {
    if (await handleBeforeClose()) {
      isShowAddMember.value = false;
    }
  }

  // 侧边栏隐藏时重置表单
  function handleHidden() {
    withPausedWatch(() => {
      handleReset();
    });
  }

  /**
   * 重置表单
   */
  async function handleReset() {
    formModel.value = {
      username: [],
      role: 'developer',
    };
    await nextTick();
    formRef.value?.clearValidate();
  }
  function handleRoleChange(value: RoleCode) {
    formModel.value.role = value;
  }
  /**
   * 新增成员
   */
  async function handleSubmit() {
    const result = await formRef.value
      ?.validate()
      .then(() => true)
      .catch(() => false);
    if (!result) return;
    await WorkspaceService.addWorkspaceUser({
      workspaceID: spaceStore.currentSpace,
      roleCode: formModel.value.role,
      userIDs: formModel.value.username,
    });
    getListWorkspaceRoleMemberGroups();
    forceCleanDirtyTag(() => {
      Message({
        theme: 'success',
        message: t('添加成功'),
      });
      isShowAddMember.value = false;
    });
  }

  watch(searchValue, () => {
    pagination.value.current = 1;
  });
  // 应用列表变化时，更新总数
  watch(tableData, newValue => {
    pagination.value.count = newValue.length;
  });

  onMounted(() => {
    getWorkspaceDetail();
  });
</script>
