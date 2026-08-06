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
  <div class="w-full mt-[16px] pb-[16px]">
    <div
      v-bkloading="{ loading: isLoading, opacity: 1, size: 'small' }"
      class="min-h-[100px]"
    >
      <Exception
        v-if="!isLoading && appTypeGroups.length === 0"
        :description="$t('暂无组件')"
        scene="part"
        type="empty"
      >
      </Exception>

      <!-- 按应用类型分组 -->
      <div
        v-for="group in appTypeGroups"
        :key="group.appType"
        class="rounded-[2px] border border-solid mb-[8px] last:mb-0"
        :style="groupStatusMap[group.appType].style"
      >
        <!-- 应用类型头部行 -->
        <div class="flex items-center justify-between h-[40px] px-[16px]">
          <div
            class="flex items-center flex-1 min-w-0 cursor-pointer"
            @click="toggleGroupExpand(group.appType)"
          >
            <!-- 展开/收起图标 -->
            <i
              :class="[
                'bkms-icon comp-expand-icon mr-[8px] cursor-pointer text-[14px] text-[#979BA5]',
                expandedGroups.has(group.appType) ? 'bkms-icon-down-shape' : 'bkms-icon-right-shape',
              ]"
            ></i>
            <!-- 应用类型名称 -->
            <span class="text-[12px] text-[#313238] font-bold mr-[8px] flex-shrink-0">
              {{ getAppTypeLabel(group.appType) }}
            </span>
            <!-- 应用类型整体状态 -->
            <Tag
              class="mr-[8px] flex-shrink-0"
              :theme="groupStatusMap[group.appType].theme"
              type="stroke"
            >
              <div class="flex items-center">
                <component
                  :is="groupStatusMap[group.appType].icon"
                  class="text-[14px]"
                />
                <span class="ml-[5px]">{{ groupStatusMap[group.appType].label }}</span>
              </div>
            </Tag>
            （
            <div class="inline-flex items-center text-[#4D4F56] font-d flex-shrink-0 text-[12px]">
              {{ $t('必选组件') }}：
              <div class="font-bold text-[#4d4f56]">
                <span :class="{ 'text-[#2DCB56]': group.requiredInstalledCount > 0 }">{{
                  group.requiredInstalledCount
                }}</span
                >/{{ group.requiredAddons.length }}
              </div>
              ，{{ $t('可选组件') }}：
              <div class="font-bold text-[#4d4f56]">
                <span :class="{ 'text-[#2DCB56]': group.optionalInstalledCount > 0 }">{{
                  group.optionalInstalledCount
                }}</span
                >/{{ group.optionalAddons.length }}
              </div>
            </div>
            ）
          </div>
        </div>

        <!-- 展开后的组件列表 -->
        <div
          v-if="expandedGroups.has(group.appType)"
          class="px-[16px] pb-[16px]"
        >
          <div
            v-for="item in group.allAddons"
            :key="item.addon.name"
            class="rounded-[2px] border border-solid border-[#EAEBF0] mb-[8px] last:mb-0 px-[16px] h-[40px] flex items-center justify-between bg-[#FFF]"
          >
            <div class="flex items-center flex-1 min-w-0">
              <span class="text-[12px] font-700 text-[#313238] mr-[4px] flex-shrink-0">
                {{ item.addon.displayName || item.addon.name }}
              </span>
              <Tag
                v-if="item.isRequired"
                size="small"
                >{{ $t('必选') }}</Tag
              >
              <!-- 安装状态 -->
              <span :class="['inline-flex items-center text-[12px] flex-shrink-0', getAddonStatusClass(item.addon)]">
                （{{ getAddonStatusLabel(item.addon) }}）
              </span>
              <span
                v-if="item.addon.description"
                class="text-[12px] text-[#979BA5] ml-[4px] truncate"
              >
                {{ item.addon.description }}
              </span>
            </div>

            <!-- 右侧：操作按钮 -->
            <div class="flex items-center gap-[8px] flex-shrink-0 ml-[16px]">
              <!-- Agones 应用 & bcs-ingress-controller组件，支持端口池 -->
              <Button
                v-if="group.appType === 'agones' && item.addon.name === 'bcs-ingress-controller'"
                outline
                size="small"
                theme="primary"
                @click="handleGoPortPool"
              >
                {{ $t('配置端口池') }}
                <span v-if="portPoolCount > 0">: {{ $t('共 {0} 个', [portPoolCount]) }}</span>
              </Button>
              <Button
                v-if="supportsAction(item.addon, 'install')"
                outline
                size="small"
                theme="primary"
                @click="handleInstall(item.addon)"
              >
                {{ $t('安装') }}
              </Button>
              <Button
                v-if="supportsAction(item.addon, 'upgrade')"
                outline
                size="small"
                theme="primary"
                @click="handleUpgrade(item.addon)"
              >
                {{ $t('更新') }}
              </Button>
              <Button
                v-if="supportsAction(item.addon, 'uninstall')"
                outline
                size="small"
                theme="danger"
                @click="handleUninstall(item.addon)"
              >
                {{ $t('卸载') }}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <InstallSideslider
    v-model:visible="isShowInstallSideslider"
    :addon-info="currentAddon"
    :components-name="currentComponentName"
    :is-update="isUpdateMode"
    :loading="submittingComponentName === currentComponentName"
    @cancel="isShowInstallSideslider = false"
    @confirm="handleConfirmInstall"
  />
</template>

<script setup lang="ts">
  import { type Component, type Ref, computed, inject, nextTick, ref, watch } from 'vue';

  import { Button, Exception, InfoBox, Message, Tag } from 'bkui-vue';
  import { Info, Success, Warn } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { ClusterAddonInfoOutput, ListClusterAddonsOutput } from '~/@types/v1/cluster-addon';
  import { ClusterAddonService, PortPoolService } from '~/api/modules/v1';

  import InstallSideslider from './install-sideslider.vue';

  // 应用类型分组中的组件项
  interface AddonItem {
    addon: ClusterAddonInfoOutput;
    isRequired: boolean;
  }

  // 应用类型分组
  interface AppTypeGroup {
    allAddons: AddonItem[];
    appType: string;
    optionalAddons: AddonItem[];
    optionalInstalledCount: number;
    requiredAddons: AddonItem[];
    requiredInstalledCount: number;
  }

  interface Props {
    autoExpandAppType?: string;
    envId: string;
    hasClusterConfig?: boolean;
  }

  const props = defineProps<Props>();
  const router = useRouter();
  const injectedEnvName = inject<Ref<string>>('envName', ref(''));
  const { t } = useI18n();

  // 安装状态映射
  const statusLabelMap: Record<string, string> = {
    deployed: t('已安装'),
    failed: t('安装失败'),
    uninstalled: t('已卸载'),
    uninstalling: t('卸载中'),
    unknown: t('未知'),
  };

  // 应用类型展示名称映射
  const appTypeLabelMap: Record<string, string> = {
    helm: t('Helm 应用'),
    taf: t('TAF 应用'),
    trpc: t('tRPC 应用'),
    agones: t('Agones 应用'),
  };

  // 分组状态枚举
  enum GroupStatus {
    Failed = 'failed',
    NotInstalled = 'notInstalled',
    PartialInstalled = 'partialInstalled',
    Ready = 'ready',
  }

  // 分组状态配置
  interface GroupStatusConfig {
    icon: Component;
    label: string;
    style: { background: string; borderColor: string };
    theme: 'danger' | 'default' | 'success' | 'warning';
  }

  const defaultStatusStyle = { borderColor: '#DCDEE5', background: '#FAFBFD' };

  // 统一状态映射配置
  const groupStatusConfigMap: Record<GroupStatus, GroupStatusConfig> = {
    [GroupStatus.Failed]: {
      icon: Warn,
      label: t('安装失败'),
      style: { borderColor: '#F8B4B4', background: '#fff0f0b3' },
      theme: 'danger',
    },
    [GroupStatus.NotInstalled]: {
      icon: Info,
      label: t('未安装'),
      style: defaultStatusStyle,
      theme: 'default',
    },
    [GroupStatus.PartialInstalled]: {
      icon: Info,
      label: t('部分安装'),
      style: { borderColor: '#F9D090', background: '#fdf4e8b3' },
      theme: 'warning',
    },
    [GroupStatus.Ready]: {
      icon: Success,
      label: t('已就绪'),
      style: { borderColor: '#A1E3BA', background: '#ebfaf0b3' },
      theme: 'success',
    },
  };

  const isLoading = ref(false);
  const submittingComponentName = ref('');
  const addonList = ref<ClusterAddonInfoOutput[]>([]);
  const portPoolCount = ref(0);
  const isShowInstallSideslider = ref(false);
  const currentComponentName = ref('');
  const currentAddon = ref<ClusterAddonInfoOutput | null>(null);
  const isUpdateMode = ref(false);
  const autoExpandedAppType = ref('');

  // 展开的应用类型分组集合
  const expandedGroups = ref<Set<string>>(new Set());

  // 按应用类型分组的计算属性
  const appTypeGroups = computed<AppTypeGroup[]>(() => {
    // 收集所有出现过的应用类型
    const appTypeSet = new Set<string>();
    for (const addon of addonList.value) {
      for (const at of addon.requiredForAppTypes || []) {
        appTypeSet.add(at);
      }
      for (const at of addon.optionalForAppTypes || []) {
        appTypeSet.add(at);
      }
    }

    // 为每个应用类型构建分组
    const groups: AppTypeGroup[] = [];
    for (const appType of Array.from(appTypeSet).sort()) {
      const requiredAddons: AddonItem[] = [];
      const optionalAddons: AddonItem[] = [];

      for (const addon of addonList.value) {
        const isRequired = addon.requiredForAppTypes?.includes(appType);
        const isOptional = addon.optionalForAppTypes?.includes(appType);
        if (isRequired) {
          requiredAddons.push({ addon, isRequired: true });
        } else if (isOptional) {
          optionalAddons.push({ addon, isRequired: false });
        }
      }

      const requiredInstalledCount = requiredAddons.filter(i => i.addon.installInfo?.status === 'deployed').length;
      const optionalInstalledCount = optionalAddons.filter(i => i.addon.installInfo?.status === 'deployed').length;

      groups.push({
        allAddons: [...requiredAddons, ...optionalAddons],
        appType,
        optionalAddons,
        optionalInstalledCount,
        requiredAddons,
        requiredInstalledCount,
      });
    }

    return groups;
  });

  // 分组状态映射（appType → 对应状态配置）
  const groupStatusMap = computed<Record<string, GroupStatusConfig>>(() => {
    const map: Record<string, GroupStatusConfig> = {};
    for (const group of appTypeGroups.value) {
      map[group.appType] = groupStatusConfigMap[getGroupStatus(group)];
    }
    return map;
  });

  function autoExpandTargetGroup() {
    const appType = props.autoExpandAppType;
    if (!appType || autoExpandedAppType.value === appType) {
      return;
    }
    if (!appTypeGroups.value.some(group => group.appType === appType)) {
      return;
    }
    expandedGroups.value.add(appType);
    autoExpandedAppType.value = appType;
  }

  // 获取集群组件列表
  async function fetchClusterAddons() {
    if (!props.envId) return;
    isLoading.value = true;
    try {
      const data: ListClusterAddonsOutput = await ClusterAddonService.listClusterAddons(
        { envID: props.envId },
        { needRes: true },
      );
      addonList.value = data?.addons || [];
      // 当 bcs-ingress-controller 已安装时，获取端口池数量
      const ingressController = addonList.value.find(a => a.name === 'bcs-ingress-controller');
      if (ingressController && ingressController.installInfo?.status === 'deployed') {
        fetchPortPoolCount();
      }
    } catch {
      addonList.value = [];
    } finally {
      isLoading.value = false;
    }
  }

  // 获取端口池数量
  async function fetchPortPoolCount() {
    if (!props.envId) return;
    try {
      const data = await PortPoolService.listPortPools({ envID: props.envId });
      portPoolCount.value = data?.length ?? 0;
    } catch {
      portPoolCount.value = 0;
    }
  }

  // 获取组件状态颜色 class
  function getAddonStatusClass(addon: ClusterAddonInfoOutput): string {
    const status = addon.installInfo?.status;
    if (status === 'deployed') return 'text-[#2DCB56]'; // 已部署
    if (status === 'failed') return 'text-[#EA3636]'; // 部署失败
    if (status === 'uninstalling') return 'text-[#3A84FF]'; // 卸载中
    if (status === 'unknown') return 'text-[#FF9C01]'; // 未知
    return 'text-[#DCDEE5]'; // 已卸载/未安装
  }

  // 获取组件的安装状态标签
  function getAddonStatusLabel(addon: ClusterAddonInfoOutput): string {
    if (addon.installInfo?.status && statusLabelMap[addon.installInfo?.status]) {
      return statusLabelMap[addon.installInfo?.status];
    }
    return t('未安装');
  }

  // 获取应用类型展示名称
  function getAppTypeLabel(appType: string): string {
    return appTypeLabelMap[appType] || `${appType} ${t('应用')}`;
  }

  // 获取应用类型分组的整体状态
  function getGroupStatus(group: AppTypeGroup): GroupStatus {
    // 任何组件失败都算分组失败
    if (group.allAddons.some(i => i.addon.installInfo?.status === 'failed')) {
      return GroupStatus.Failed;
    }
    // 就绪判断
    const hasRequired = group.requiredAddons.length > 0;
    const hasOptional = group.optionalAddons.length > 0;
    const allRequiredDeployed = group.requiredAddons.every(i => i.addon.installInfo?.status === 'deployed');
    const allOptionalDeployed = group.optionalAddons.every(i => i.addon.installInfo?.status === 'deployed');

    // 情况1：有必选组件，必选组件全部已部署 -> 已就绪、情况2：没有必选组件，只有可选组件，且可选组件全部已部署 -> 已就绪
    if (hasRequired && allRequiredDeployed) {
      return GroupStatus.Ready;
    }
    if (!hasRequired && hasOptional && allOptionalDeployed) {
      return GroupStatus.Ready;
    }
    // 部分安装：有任意组件已部署
    if (group.allAddons.some(i => i.addon.installInfo?.status === 'deployed')) {
      return GroupStatus.PartialInstalled;
    }
    return GroupStatus.NotInstalled;
  }

  // 确认安装/更新组件
  async function handleConfirmInstall(data: Record<string, unknown>) {
    if (!props.envId || !currentComponentName.value) return;

    submittingComponentName.value = currentComponentName.value;
    try {
      await ClusterAddonService.upsertClusterAddon({
        envID: props.envId,
        addonName: currentComponentName.value,
        namespace: data.namespace as string,
        chartVersion: data.chartVersion as string,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        values: (data.values || {}) as Record<string, any>,
      });
      // 关闭侧栏并刷新列表
      isShowInstallSideslider.value = false;

      Message({
        message: t('操作成功'),
        theme: 'success',
      });
      await fetchClusterAddons();
    } finally {
      submittingComponentName.value = '';
    }
  }

  // 跳转到端口池配置页面
  function handleGoPortPool() {
    router.push({
      name: 'clusterPortPool',
      params: { envId: props.envId },
      query: { envName: injectedEnvName.value },
    });
  }

  // 安装组件
  function handleInstall(addon: ClusterAddonInfoOutput) {
    currentComponentName.value = addon?.name || '';
    currentAddon.value = addon;
    isUpdateMode.value = false;
    isShowInstallSideslider.value = true;
  }

  // 卸载组件
  function handleUninstall(addon: ClusterAddonInfoOutput) {
    InfoBox({
      type: 'warning',
      title: t('确认卸载'),
      subTitle: t('确定卸载组件 {name} ？', { name: addon.displayName }),
      cancelText: t('取消'),
      headerAlign: 'center',
      contentAlign: 'center',
      footerAlign: 'center',
      onConfirm: async () => {
        const result = await ClusterAddonService.deleteClusterAddon({
          envID: props.envId,
          addonName: addon?.name || '',
        })
          .then(() => true)
          .catch(() => false);
        if (result) {
          Message({
            message: t('组件卸载成功'),
            theme: 'success',
          });
          nextTick(() => {
            fetchClusterAddons();
          });
        }
      },
    });
  }

  // 更新组件
  function handleUpgrade(addon: ClusterAddonInfoOutput) {
    currentComponentName.value = addon?.name || '';
    currentAddon.value = addon;
    isUpdateMode.value = true;
    isShowInstallSideslider.value = true;
  }

  // 判断插件是否支持某个操作
  function supportsAction(addon: ClusterAddonInfoOutput, action: string): boolean {
    return addon.supportedActions?.includes(action) ?? false;
  }

  // 切换应用类型分组展开/收起状态
  function toggleGroupExpand(appType: string) {
    if (expandedGroups.value.has(appType)) {
      expandedGroups.value.delete(appType);
    } else {
      expandedGroups.value.add(appType);
    }
  }

  watch(
    [() => props.autoExpandAppType, appTypeGroups],
    ([appType]) => {
      if (!appType) {
        autoExpandedAppType.value = '';
        return;
      }
      autoExpandTargetGroup();
    },
    { immediate: true },
  );

  // 监听 envId 和 hasClusterConfig 变化，重新获取数据
  watch(
    [() => props.envId, () => props.hasClusterConfig],
    ([envId], [oldEnvId]) => {
      if (envId !== oldEnvId) {
        autoExpandedAppType.value = '';
      }
      if (props.hasClusterConfig) {
        fetchClusterAddons();
      } else {
        addonList.value = [];
        isLoading.value = false;
      }
    },
    { immediate: true },
  );
</script>

<style lang="postcss" scoped>
  .comp-expand-icon {
    transition: transform 0.3s ease;
  }
  :deep(.bk-exception-img) {
    width: 220px;
    height: 120px;
  }
</style>
