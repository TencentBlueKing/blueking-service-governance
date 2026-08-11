<template>
  <Sideslider
    v-model:is-show="visible"
    :before-close="handleBeforeClose"
    :width="640"
    @closed="handleClosed"
  >
    <template #header>
      <span class="text-[16px] font-[600]">{{ headerTitle }}</span>
    </template>
    <Form
      ref="formRef"
      v-bkloading="{ loading: detailLoading }"
      class="px-[24px] pt-[16px]"
      form-type="vertical"
      :model="formModel"
      :rules="formRules"
    >
      <!-- 告警组名称 -->
      <Form.FormItem
        :label="$t('告警组名称')"
        property="name"
        required
      >
        <Input
          v-model.trim="formModel.name"
          :maxlength="32"
          :placeholder="$t('请输入')"
        />
      </Form.FormItem>

      <!-- 通知对象 -->
      <Form.FormItem
        :label="$t('通知对象')"
        property="users"
        required
      >
        <UserSelector
          v-model="formModel.users"
          multiple
        />
        <div class="text-[#979BA5] text-[12px] mt-[4px]">
          {{ $t('处理套餐中使用了电话语音通知，拨打的顺序是按通知对象顺序依次拨打，用户组内无法保证顺序') }}
        </div>
      </Form.FormItem>

      <!-- 通知方式 -->
      <Form.FormItem
        :label="$t('通知方式')"
        property="noticeWays"
        required
      >
        <Table
          bordered
          :data="noticeMatrix"
          :resizable="false"
        >
          <TableColumn
            :label="$t('告警级别')"
            :width="120"
          >
            <template #default="{ row }">
              <SeverityLabel :severity="row.level" />
            </template>
          </TableColumn>
          <TableColumn
            v-for="col in noticeWayColumns"
            :key="col.name"
            align="center"
            :min-width="100"
          >
            <template #header>
              <div class="flex items-center justify-center gap-[8px]">
                <i
                  class="text-[#979BA5] text-[16px]"
                  :class="col.iconClass"
                />
                <span>{{ $t(col.labelKey) }}</span>
              </div>
            </template>
            <template #default="{ row }">
              <Checkbox
                :model-value="row.checked.includes(col.name)"
                @change="(val: boolean) => handleToggleWay(row.level, col.name, val)"
              />
            </template>
          </TableColumn>
        </Table>
        <div class="text-[#979BA5] text-[12px] mt-[8px]">
          {{ $t('每个告警级别至少配置一种通知渠道；勾选的通知类型至少在某级别有配置。') }}
        </div>
      </Form.FormItem>
    </Form>
    <template #footer>
      <div class="flex items-center gap-[8px]">
        <Button
          :loading="submitting"
          theme="primary"
          @click="handleSubmit"
        >
          {{ $t('保存') }}
        </Button>
        <Button @click="handleCancel">
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, reactive, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Checkbox, Form, Input, Message, Sideslider } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { BkintegrationsBkmonitorService } from '~/api/modules/v1/bkintegrations-bkmonitor';
  import UserSelector from '~/components/user-selector.vue';
  import { useFocusOnErrorField } from '~/composables/use-focus-on-error-field';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useSpaceStore } from '~/stores/space';

  import SeverityLabel from './severity-label.vue';

  import type { UserGroupUser } from '~/@types/v1/bkintegrations-bkmonitor';

  interface Emits {
    (e: 'editSuccess'): void;
    (e: 'success'): void;
    (e: 'update:isShow', value: boolean): void;
  }

  /** 侧滑表单模式：create-新建 / edit-编辑 */
  type FormMode = 'create' | 'edit';

  interface NoticeCol {
    iconClass: string;
    labelKey: string;
    name: NoticeWayName;
  }

  /** 告警级别（与后端 AlertNoticeConfig.level 对齐） */
  type NoticeLevel = 1 | 2 | 3;

  interface NoticeRow {
    checked: NoticeWayName[];
    level: NoticeLevel;
  }

  /** 通知方式名称（与后端 NoticeWay.name 对齐） */
  type NoticeWayName = 'mail' | 'sms' | 'voice' | 'weixin';

  const noticeWayNameSet = new Set<string>(['mail', 'sms', 'voice', 'weixin']);

  interface Props {
    groupID?: number;
    isShow: boolean;
    mode?: FormMode;
  }

  const props = withDefaults(defineProps<Props>(), { mode: 'create' });
  const emit = defineEmits<Emits>();

  const { t } = useI18n();
  const spaceStore = useSpaceStore();
  const { focusOnErrorField } = useFocusOnErrorField();

  const visible = computed({
    get: () => props.isShow,
    set: (val: boolean) => emit('update:isShow', val),
  });

  /** 是否为编辑模式 */
  const isEdit = computed(() => props.mode === 'edit' && typeof props.groupID === 'number' && props.groupID > 0);

  /** 侧滑标题 */
  const headerTitle = computed(() => {
    if (isEdit.value) return t('编辑告警组');
    return t('新建告警组');
  });

  /** 加载详情时的 loading 状态 */
  const detailLoading = ref(false);

  /** 通知方式列定义（顺序与表头保持一致） */
  const noticeWayColumns: NoticeCol[] = [
    { name: 'weixin', labelKey: t('企业微信'), iconClass: 'bkms-icon bkms-icon-qiyeweixin' },
    { name: 'mail', labelKey: t('邮箱'), iconClass: 'bkms-icon bkms-icon-youjian' },
    { name: 'sms', labelKey: t('短信'), iconClass: 'bkms-icon bkms-icon-duanxin' },
    { name: 'voice', labelKey: t('语音'), iconClass: 'bkms-icon bkms-icon-yuyin' },
  ];

  /** 告警级别行配置 */
  const levelRowConfig: Array<Omit<NoticeRow, 'checked'>> = [{ level: 1 }, { level: 2 }, { level: 3 }];

  interface FormModel {
    name: string;
    noticeWays: Record<NoticeLevel, NoticeWayName[]>;
    users: string[];
  }

  function createDefaultForm(): FormModel {
    return {
      name: '',
      users: [],
      noticeWays: {
        1: [],
        2: [],
        3: [],
      },
    };
  }

  const formModel = reactive<FormModel>(createDefaultForm());
  const formRef = ref<InstanceType<typeof Form>>();
  const submitting = ref(false);

  /** 表格行：方便在模板中循环渲染 */
  const noticeMatrix = computed<NoticeRow[]>(() =>
    levelRowConfig.map(cfg => ({
      ...cfg,
      checked: formModel.noticeWays[cfg.level] ?? [],
    })),
  );

  /** 切换某级别某个通知方式的勾选状态 */
  function handleToggleWay(level: NoticeLevel, name: NoticeWayName, checked: boolean) {
    const list = formModel.noticeWays[level] ?? [];
    const idx = list.indexOf(name);
    if (checked && idx === -1) {
      formModel.noticeWays[level] = [...list, name];
    } else if (!checked && idx > -1) {
      formModel.noticeWays[level] = list.filter(n => n !== name);
    }
  }

  /** 自定义校验：每个告警级别至少勾选一种通知方式 */
  const validateNoticeWays = () => {
    const allEmpty = (Object.values(formModel.noticeWays) as NoticeWayName[][]).every(arr => arr.length === 0);
    if (allEmpty) {
      return false;
    }
    return (Object.values(formModel.noticeWays) as NoticeWayName[][]).every(arr => arr.length > 0);
  };

  /** 表单校验规则 */
  const formRules = computed(() => ({
    name: [{ required: true, message: t('请输入告警组名称'), trigger: 'blur' }],
    users: [
      {
        required: true,
        validator: () => (formModel.users?.length ?? 0) > 0,
        message: t('请选择通知对象'),
        trigger: 'change',
      },
    ],
    noticeWays: [
      {
        required: true,
        validator: () => validateNoticeWays(),
        message: t('每个告警级别至少配置一种通知渠道'),
        trigger: 'change',
      },
    ],
  }));

  const { confirmBox, forceCleanDirtyTag, withPausedWatch } = useLeaveConfirm(formModel);

  function buildActionConfig(phase: 1 | 2 | 3) {
    // 处理通知与告警通知共用同一份勾选结果
    const allWays = Array.from(new Set((Object.values(formModel.noticeWays) as NoticeWayName[][]).flat()));
    if (allWays.length === 0) return null;
    return {
      phase,
      type: [] as string[],
      notice_ways: allWays.map(name => ({ name })),
    };
  }

  /** 构建 alertNotice 请求体（API 约定 time_range 固定为全天） */
  function buildNoticeConfig(level: NoticeLevel) {
    const ways = formModel.noticeWays[level] ?? [];
    if (ways.length === 0) return null;
    return {
      level,
      type: [] as string[],
      notice_ways: ways.map(name => ({ name })),
    };
  }

  /** 获取告警组详情并填充表单 */
  async function fetchDetail() {
    if (!spaceStore.currentSpace || !props.groupID) return;
    detailLoading.value = true;
    try {
      const detail = await BkintegrationsBkmonitorService.getUserGroup({
        workspaceID: spaceStore.currentSpace,
        groupID: props.groupID,
      });
      // 回填表单时暂停脏标记监听，避免服务端数据被误判为用户修改
      withPausedWatch(() => {
        // 填充名称
        formModel.name = detail?.name ?? '';
        // 填充通知对象（以 duty_arranges 中的值班用户为准）
        formModel.users = (detail?.duty_arranges ?? [])
          .flatMap(arrange => arrange.users ?? [])
          .map(u => u.id ?? '')
          .filter(Boolean);
        // 填充通知方式（从 alert_notice[0].notify_config 还原）
        const resetWays: Record<NoticeLevel, NoticeWayName[]> = { 1: [], 2: [], 3: [] };
        const notifyConfig = detail?.alert_notice?.[0]?.notify_config ?? [];
        for (const cfg of notifyConfig) {
          const level = cfg.level;
          if (level === 1 || level === 2 || level === 3) {
            resetWays[level] = (cfg.notice_ways ?? [])
              .map(w => w.name)
              .filter((n): n is NoticeWayName => (noticeWayNameSet as Set<string>).has(n as string));
          }
        }
        formModel.noticeWays = resetWays;
      });
      forceCleanDirtyTag();
    } catch {
      Message({ theme: 'error', message: t('获取告警组详情失败') });
      visible.value = false;
    } finally {
      detailLoading.value = false;
    }
  }

  /** 关闭前询问 */
  async function handleBeforeClose() {
    return confirmBox();
  }

  /** 取消按钮 */
  async function handleCancel() {
    if (await handleBeforeClose()) {
      visible.value = false;
    }
  }

  /** 关闭动画结束后重置表单 */
  function handleClosed() {
    withPausedWatch(() => {
      Object.assign(formModel, createDefaultForm());
    });
    forceCleanDirtyTag();
    formRef.value?.clearValidate();
  }

  /** 提交保存 */
  async function handleSubmit() {
    const isValid = await formRef.value?.validate().catch(() => false);
    if (!isValid) {
      focusOnErrorField();
      return;
    }
    if (!spaceStore.currentSpace) return;

    const alertNoticeConfig = levelRowConfig
      .map(row => buildNoticeConfig(row.level))
      .filter((v): v is NonNullable<typeof v> => v != null);
    if (alertNoticeConfig.length === 0) return;

    const actionNoticeConfig = ([1, 2, 3] as const)
      .map(phase => buildActionConfig(phase))
      .filter((v): v is NonNullable<typeof v> => v != null);

    const usersPayload: UserGroupUser[] = formModel.users.map(id => ({
      id,
      type: 'user' as const,
      display_name: id,
    }));

    const body = {
      workspaceID: spaceStore.currentSpace,
      name: formModel.name,
      channels: ['user'] as string[],
      alertNotice: [
        {
          time_range: '00:00--23:59',
          notify_config: alertNoticeConfig,
        },
      ],
      actionNotice: [
        {
          time_range: '00:00--23:59',
          notify_config: actionNoticeConfig,
        },
      ],
      users: usersPayload,
    };

    submitting.value = true;
    try {
      if (isEdit.value && props.groupID) {
        await BkintegrationsBkmonitorService.updateUserGroup({
          ...body,
          groupID: props.groupID,
        } as Parameters<typeof BkintegrationsBkmonitorService.updateUserGroup>[0] & { users: UserGroupUser[] });
        Message({ theme: 'success', message: t('更新成功') });
      } else {
        await BkintegrationsBkmonitorService.createUserGroup(
          body as Parameters<typeof BkintegrationsBkmonitorService.createUserGroup>[0] & { users: UserGroupUser[] },
        );
        Message({ theme: 'success', message: t('创建成功') });
      }
      forceCleanDirtyTag(() => {
        if (isEdit.value) {
          emit('editSuccess');
        } else {
          emit('success');
        }
        visible.value = false;
      });
    } catch {
      // 错误由拦截器统一处理
    } finally {
      submitting.value = false;
    }
  }

  // 侧滑打开时清空校验 / 编辑模式拉取详情
  watch(
    () => props.isShow,
    async val => {
      if (val) {
        formRef.value?.clearValidate();
        if (isEdit.value) {
          await fetchDetail();
        }
      }
    },
  );
</script>

<style lang="postcss" scoped></style>
