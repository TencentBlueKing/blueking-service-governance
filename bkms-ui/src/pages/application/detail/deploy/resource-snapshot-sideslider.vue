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
    quick-close
    :title="$t('查看详情')"
    :width="960"
  >
    <Loading
      class="h-[calc(100vh-52px)]"
      :loading="listLoading"
    >
      <div
        v-if="treeData.length"
        class="flex h-full gap-[16px] p-[16px]"
      >
        <Tree
          ref="treeRef"
          class="!w-[280px]"
          :data="treeData"
          expand-all
          label="name"
          node-key="id"
          @node-click="handleNodeClick"
        >
          <template #nodeType="node">
            <i
              class="bkms-icon mr-[12px] text-[16px]"
              :class="node.type === 'group' ? 'bkms-icon-folder text-[#979BA5]' : 'bkms-icon-file text-[#4D4F56]'"
            ></i>
          </template>
        </Tree>
        <Loading
          class="flex-1 min-w-0 h-full"
          :loading="detailLoading"
          :opacity="0.3"
        >
          <MsEditor
            lang="yaml"
            :model-value="manifest"
            readonly
            :title="activeSnapshotName"
          />
        </Loading>
      </div>
      <div
        v-else-if="!listLoading"
        class="flex justify-center items-center h-full"
      >
        <Exception
          class="normal-exception"
          :description="$t('暂无数据')"
          scene="part"
          type="empty"
        >
          <template #type>
            <img src="/empty.svg" />
          </template>
        </Exception>
      </div>
    </Loading>
  </Sideslider>
</template>

<script lang="ts" setup>
  // 部署资源快照侧滑面板：展示该次部署所包含的资源快照列表及对应的 YAML 清单
  // 左侧为按 kind 分组的快照树，右侧为选中快照的 YAML 预览
  import { nextTick, ref, watch } from 'vue';

  import { Exception, Loading, Sideslider, Tree } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import {
    AppModelResourceSnapshot,
    GetAppModelResourceSnapshotOutput,
    GetTafResourceSnapshotRequest,
    GetTrpcResourceSnapshotRequest,
    ListTafResourceSnapshotsRequest,
    ListTrpcResourceSnapshotsRequest,
  } from '~/@types/v1/deploy';
  import { DeployService } from '~/api/modules/v1';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  type GetAppModelResourceSnapshotRequest = GetTafResourceSnapshotRequest | GetTrpcResourceSnapshotRequest;
  type ListAppModelResourceSnapshotsRequest = ListTafResourceSnapshotsRequest | ListTrpcResourceSnapshotsRequest;

  // 资源快照树节点：group 为按 kind 分组的文件夹，snapshot 为叶子节点
  interface ResourceSnapshotTreeNode {
    children: ResourceSnapshotTreeNode[];
    id: string;
    isOpen?: boolean;
    name: string;
    snapshot?: AppModelResourceSnapshot;
    type: 'group' | 'snapshot';
  }

  interface TreeExpose {
    setSelect: (ids: string[]) => void;
  }

  // v-model 控制侧滑显隐，deployId 由父组件传入
  const isShow = defineModel<boolean>('isShow', { default: false });
  const props = defineProps<{
    deployId: string;
  }>();

  const { t } = useI18n();
  const appDetailStore = useAppDetail();
  const trpcDeployStore = useTrpcDeployStore();

  const treeRef = ref<null | TreeExpose>(null);
  // 快照列表（树形结构）与选中快照的详情状态
  const treeData = ref<ResourceSnapshotTreeNode[]>([]);
  const activeSnapshotID = ref('');
  const activeSnapshotName = ref('');
  const manifest = ref<string>('');
  const listLoading = ref(false);
  const detailLoading = ref(false);

  // 递增计数器，用于丢弃过期的异步请求响应，防止竞态
  let listRequestID = 0;
  let detailRequestID = 0;

  // 将扁平快照列表按 kind 分组构建为树结构（group → snapshot）
  function buildResourceSnapshotTree(snapshots: AppModelResourceSnapshot[]) {
    const groups = new Map<string, ResourceSnapshotTreeNode>();

    snapshots.forEach(snapshot => {
      const groupKey = snapshot.kind || '';
      let group = groups.get(groupKey);
      if (!group) {
        group = {
          children: [],
          id: `group:${groupKey}`,
          isOpen: true,
          name: snapshot.kind || t('其他'),
          type: 'group',
        };
        groups.set(groupKey, group);
      }

      group.children.push({
        children: [],
        id: snapshot.id!,
        name: snapshot.name!,
        snapshot,
        type: 'snapshot',
      });
    });

    return [...groups.values()];
  }

  async function fetchAllResourceSnapshots() {
    // 分页拉取全部快照（接口无批量查询，故循环直至收齐）
    const results: AppModelResourceSnapshot[] = [];
    const pageSize = 100;
    let page = 1;

    while (true) {
      const response = await listResourceSnapshots(page, pageSize);
      results.push(...response.results!);

      if (results.length >= Number(response.count) || response.results!.length === 0) {
        return results;
      }
      page += 1;
    }
  }

  async function fetchSnapshotDetail(snapshot: AppModelResourceSnapshot) {
    // 递增 detailRequestID 以标记新请求，后续旧请求的响应会被丢弃
    const requestID = ++detailRequestID;
    activeSnapshotID.value = snapshot.id!;
    activeSnapshotName.value = snapshot.name!;
    manifest.value = '';
    detailLoading.value = true;

    try {
      const res = await getResourceSnapshot(snapshot.id!);
      if (requestID !== detailRequestID) return;
      manifest.value = res?.snapshot!.manifest ?? '';
    } catch {
      manifest.value = '';
    } finally {
      if (requestID === detailRequestID) {
        detailLoading.value = false;
      }
    }
  }

  async function fetchSnapshotList() {
    // 先重置状态并取消旧请求，再发起新请求
    resetState();
    const requestID = listRequestID;
    listLoading.value = true;

    try {
      const snapshots = await fetchAllResourceSnapshots();
      if (requestID !== listRequestID) return;

      treeData.value = buildResourceSnapshotTree(snapshots);
      listLoading.value = false;
      if (snapshots.length > 0) {
        await selectSnapshot(snapshots[0]);
      }
    } catch {
      if (requestID === listRequestID) {
        treeData.value = [];
      }
    } finally {
      if (requestID === listRequestID) {
        listLoading.value = false;
      }
    }
  }

  function getCommonParams() {
    return {
      appID: appDetailStore.appID,
      deployID: props.deployId,
      envName: trpcDeployStore.curEnvItem?.name || '',
    };
  }

  function getResourceSnapshot(snapshotID: string) {
    const params: GetAppModelResourceSnapshotRequest = {
      ...getCommonParams(),
      snapshotID,
    };
    return appDetailStore.appType === 'taf'
      ? DeployService.getTafResourceSnapshot<GetAppModelResourceSnapshotRequest, GetAppModelResourceSnapshotOutput>(
          params,
          { needRes: true },
        )
      : DeployService.getTrpcResourceSnapshot<GetAppModelResourceSnapshotRequest, GetAppModelResourceSnapshotOutput>(
          params,
          { needRes: true },
        );
  }

  async function handleNodeClick(node: ResourceSnapshotTreeNode) {
    // 点击 group 节点不切换选中，只恢复上一次选中的 snapshot 高亮
    if (node.type === 'group') {
      await nextTick();
      if (activeSnapshotID.value) {
        treeRef.value?.setSelect([activeSnapshotID.value]);
      }
      return;
    }
    // 点击 snapshot 节点且非重复选中，切换详情
    if (node.snapshot && node.id !== activeSnapshotID.value) {
      await selectSnapshot(node.snapshot);
    }
  }

  function listResourceSnapshots(page: number, pageSize: number) {
    const params: ListAppModelResourceSnapshotsRequest = {
      ...getCommonParams(),
      page,
      pageSize,
    };
    return appDetailStore.appType === 'taf'
      ? DeployService.listTafResourceSnapshots(params)
      : DeployService.listTrpcResourceSnapshots(params);
  }

  function resetState() {
    // 递增两个 requestID 以废弃所有进行中的请求，恢复为空状态
    listRequestID += 1;
    detailRequestID += 1;
    treeData.value = [];
    activeSnapshotID.value = '';
    activeSnapshotName.value = '';
    manifest.value = '';
    listLoading.value = false;
    detailLoading.value = false;
  }

  async function selectSnapshot(snapshot: AppModelResourceSnapshot) {
    // 先让树组件完成渲染，再高亮节点并加载详情
    await nextTick();
    treeRef.value?.setSelect([snapshot.id!]);
    await fetchSnapshotDetail(snapshot);
  }

  // 依赖：侧滑显隐、部署 ID、应用 ID/类型、环境名称任一变化时，满足条件则刷新列表，否则清空状态
  watch(
    () => [
      isShow.value,
      props.deployId,
      appDetailStore.appID,
      appDetailStore.appType,
      trpcDeployStore.curEnvItem?.name,
    ],
    ([show, deployID, appID, appType, envName]) => {
      if (show && deployID && appID && (appType === 'trpc' || appType === 'taf') && envName) {
        fetchSnapshotList();
      } else {
        resetState();
      }
    },
    { immediate: true },
  );
</script>

<style lang="postcss" scoped>
  :deep(.bk-node-row) {
    font-size: 12px;

    &.is-selected,
    &:hover {
      color: #3a84ff;
      background-color: #e1ecff;
    }
  }
</style>
