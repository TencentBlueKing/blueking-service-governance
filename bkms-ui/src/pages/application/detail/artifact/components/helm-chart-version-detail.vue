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
    :is-show="isShow"
    :width="960"
    @closed="handleClose"
  >
    <template #header>
      <DividerHeader
        :show-divider="true"
        :title="t('版本详情')"
        :title-size="16"
      >
        <span class="text-[14px] text-[#979BA5]">{{ `${t('版本号')}: ${chartVersion}` }}</span>
      </DividerHeader>
    </template>
    <Loading
      class="h-[calc(100vh_-_52px)]"
      :loading="isLoadingFile"
    >
      <div
        v-if="treeData.length"
        class="flex h-[calc(100vh_-_52px)]"
      >
        <!-- 左侧文件树 -->
        <div class="w-[280px] py-[12px]">
          <Tree
            ref="treeRef"
            children="children"
            class="h-auto"
            :data="treeData"
            expand-all
            label="name"
            :node-content-action="['expand', 'collapse', 'click', 'selected']"
            :node-key="'path'"
            @node-click="handleNodeClick"
          >
            <template #nodeType="node">
              <template v-if="node.isDir">
                <i class="bkms-icon bkms-icon-folder text-[#c4c6cc] text-[16px] mr-[12px]"></i>
              </template>
              <template v-else>
                <i class="bkms-icon bkms-icon-file text-[#c4c6cc] text-[16px] mr-[12px]"></i>
              </template>
            </template>
          </Tree>
        </div>
        <!-- 右侧文件内容 -->
        <div class="flex-1 overflow-hidden">
          <MsEditor
            v-if="currentFileContent !== null"
            lang="yaml"
            :model-value="currentFileContent"
            readonly
            :title="currentFileName"
          />
        </div>
      </div>
      <Exception
        v-else
        :description="$t('暂无数据')"
        scene="part"
        type="empty"
      />
    </Loading>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { nextTick, ref, watch } from 'vue';

  import { Exception, Loading, Sideslider, Tree } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { HelmChartFileNode } from '~/@types/v1/helm-charts';
  import { HelmChartsService } from '~/api/modules/v1';
  import DividerHeader from '~/components/divider-header.vue';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';

  /** bkui-vue Tree 内部节点对象 */
  interface TreeInternalNode {
    hasChild: boolean;
    isOpen: boolean;
  }

  const { t } = useI18n();

  const isShow = defineModel<boolean>('isShow');

  const props = defineProps<{
    appId: string;
    chartVersion: string;
  }>();

  const treeData = ref<HelmChartFileNode[]>([]);
  const treeRef = ref();
  const lastFileNodePath = ref<string>('');
  const currentFileContent = ref<null | string>(null);
  const currentFileName = ref('');
  const isLoadingFile = ref(true);

  /** 获取 Helm Chart 文件树 */
  async function fetchHelmChartFiles() {
    try {
      isLoadingFile.value = true;
      if (!props.appId || !props.chartVersion) return;
      const res = await HelmChartsService.getHelmChartFiles({
        appID: props.appId,
        chartVersion: props.chartVersion,
      });

      if (res.root) {
        treeData.value = [res.root];
        selectFirstFileNode();
      }
    } finally {
      isLoadingFile.value = false;
    }
  }

  /** 递归查找第一个文件节点（非目录节点） */
  function findFirstFileNode(nodes: HelmChartFileNode[]): HelmChartFileNode | null {
    for (const node of nodes) {
      if (!node.isDir) {
        return node;
      }
      if (node.children?.length) {
        const found = findFirstFileNode(node.children);
        if (found) return found;
      }
    }
    return null;
  }

  /** 关闭侧栏 */
  function handleClose() {
    isShow.value = false;
  }

  /** 处理目录节点点击 */
  function handleDirClick(nodeData: HelmChartFileNode, node: TreeInternalNode) {
    if (!node.isOpen) {
      // 当前已展开，点击后收起；组件内部有选中节点时不会自动收起（互斥），无需额外处理
      treeRef.value.setOpen(nodeData, false);
    } else {
      // 当前已收起，点击后展开；组件默认行为，无需手动 setOpen，但需重新选中上一个文件节点，避免选中态跳到目录上
      if (lastFileNodePath.value) {
        treeRef.value?.setSelect([lastFileNodePath.value]);
      }
    }
  }

  /** 处理文件节点点击：缓存路径并显示内容 */
  function handleFileClick(nodeData: HelmChartFileNode) {
    lastFileNodePath.value = nodeData.path!;
    currentFileName.value = nodeData.name!;
    currentFileContent.value = nodeData.content!;
  }

  /** 处理树节点点击，按节点类型分发 */
  function handleNodeClick(nodeData: HelmChartFileNode, node: TreeInternalNode) {
    if (node.hasChild) {
      handleDirClick(nodeData, node);
    } else {
      handleFileClick(nodeData);
    }
  }

  /** 选中第一个文件节点并加载内容 */
  async function selectFirstFileNode() {
    await nextTick();
    const firstFileNode = findFirstFileNode(treeData.value);
    if (firstFileNode) {
      treeRef.value?.setSelect([firstFileNode.path]);
      handleFileClick(firstFileNode);
    }
  }

  /** 监听显示状态和参数变化 */
  watch(
    () => [isShow.value, props.appId, props.chartVersion],
    ([show, appId, chartVersion]) => {
      if (show && appId && chartVersion) {
        fetchHelmChartFiles();
      }
    },
    { immediate: true },
  );

  /** 监听侧栏关闭，重置状态 */
  watch(isShow, show => {
    if (!show) {
      treeData.value = [];
      currentFileContent.value = null;
      currentFileName.value = '';
      lastFileNodePath.value = '';
    }
  });
</script>

<style lang="less" scoped>
  :deep(.bk-node-row) {
    padding: 0 12px;
    font-size: 12px;
    transition: all 0.3s ease;
    &.is-selected {
      color: #3a84ff;
      background-color: #e1ecff;
    }
    &:hover {
      background-color: #e1ecff;
    }

    .bk-node-content {
      .bk-node-text {
        user-select: none;
      }
    }
  }
</style>
