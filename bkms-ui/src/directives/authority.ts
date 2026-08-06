/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

// import { Message } from 'bkui-vue';
import type { Directive, DirectiveBinding, VNode } from 'vue';

import { userPerms, userPermsByAction } from '~/api/modules/custom';
import { deepEqual } from '~/common/util';
import { useEventBus } from '~/composables/use-event-bus';
interface AuthResponse {
  data?: {
    perms?: Record<string, boolean | string | undefined> & {
      apply_url?: string;
    };
  };
}
interface IElement extends HTMLElement {
  cloneEl?: IElement;
  element?: HTMLDivElement | null;
  originEl?: IElement;
  clickHandler?: (e: Event) => void;
  mouseEnterHandler?: () => void;
  mouseLeaveHandler?: () => void;
  mouseMoveHandler?: (event: MouseEvent) => void;
}
interface IOptions {
  actionId?: string | string[];
  clickable: boolean;
  cls: string;
  disablePerms?: boolean; // 是否禁用自动权限请求（完全交个外部控制clickable的值决定状态）
  offset: number[];
  resourceName?: string;
  // key?: string; // 防止指令替换DOM后，Vue diff Vnode时进行Vnode替换找不到节点报错问题
  permCtx?: {
    cluster_id: string; // 集群权限 如果实例无关，可不传cluster_id
    name: string; // 命名空间相关权限 如果实例无关，可不传name
    project_id: string; // 项目权限 如果实例无关，可不传
    resource_type?: string; // 资源类型
    template_id: string; // 模板集相关权限  果实例无关，可不传template_id
  };
}
interface MutableVNode extends VNode {
  elm?: Element | null;
}

const DEFAULT_OPTIONS: IOptions = {
  clickable: false,
  offset: [12, 0],
  cls: 'bkms-cursor-element',
  disablePerms: false,
};

const { emit } = useEventBus();

function destroy(cloneEl: IElement, vNode: VNode) {
  const el = cloneEl.originEl;
  if (!el?.cloneEl) return;

  // 还原原始节点
  const parent = cloneEl.parentNode;
  parent?.replaceChild(el, el.cloneEl);
  (vNode as MutableVNode).elm = el;

  // bkTooltips.unmounted(cloneEl);
  if (cloneEl.mouseEnterHandler) cloneEl.removeEventListener('mouseenter', cloneEl.mouseEnterHandler);
  if (cloneEl.mouseMoveHandler) cloneEl.removeEventListener('mousemove', cloneEl.mouseMoveHandler);
  if (cloneEl.mouseLeaveHandler) cloneEl.removeEventListener('mouseleave', cloneEl.mouseLeaveHandler);
  if (cloneEl.clickHandler) cloneEl.removeEventListener('click', cloneEl.clickHandler);
  cloneEl.element?.remove();
  cloneEl.element = null;
  delete el.cloneEl;
  return el;
}

function init(el: IElement, binding: DirectiveBinding, vNode: VNode) {
  // const { t } = useI18n();
  // 节点被替换过时需要还原回来
  if (el.originEl) {
    const restoredEl = destroy(el, vNode);
    if (!restoredEl) return;
    el = restoredEl;
  }
  const parent = el.parentNode;
  const options: IOptions = Object.assign({}, DEFAULT_OPTIONS, binding.value);
  if (options.clickable || el.dataset.clickable || !parent) return;

  if (!el.cloneEl) {
    el.cloneEl = el.cloneNode(true) as IElement;
  }
  const { cloneEl } = el;
  // 保留原始节点
  cloneEl.originEl = el;
  // 替换当前节点（为了移除节点的所有事件）
  parent?.replaceChild(cloneEl, el);
  (vNode as MutableVNode).elm = cloneEl;

  cloneEl.style.filter = 'grayscale(100%)';
  cloneEl.style.cursor = 'not-allowed';
  // bkTooltips.update(cloneEl, binding);
  cloneEl.mouseEnterHandler = function () {
    const element = document.createElement('div');
    element.id = 'directive-ele';
    element.style.position = 'absolute';
    element.style.zIndex = '9999';
    cloneEl.element = element;
    cloneEl.element.style.left = '0px';
    cloneEl.element.style.top = '0px';
    document.body.appendChild(element);

    element.classList.add(options.cls || DEFAULT_OPTIONS.cls);
    if (cloneEl.mouseMoveHandler) cloneEl.addEventListener('mousemove', cloneEl.mouseMoveHandler);
  };
  cloneEl.mouseMoveHandler = function (event: MouseEvent) {
    if (!cloneEl.element) return;
    const { pageX, pageY } = event;
    const elLeft = pageX + DEFAULT_OPTIONS.offset[0];
    const elTop = pageY + DEFAULT_OPTIONS.offset[1];
    cloneEl.element.style.left = `${elLeft}px`;
    cloneEl.element.style.top = `${elTop}px`;
  };
  cloneEl.mouseLeaveHandler = function () {
    cloneEl.element?.remove();
    cloneEl.element = null;
    if (cloneEl.mouseMoveHandler) cloneEl.removeEventListener('mousemove', cloneEl.mouseMoveHandler);
  };
  cloneEl.clickHandler = (e: Event) => {
    e.stopPropagation();
    const { actionId, permCtx, resourceName } = options;
    if (!actionId || actionId.length === 0) return;

    delete permCtx?.resource_type;
    const $actionId = Array.isArray(actionId) ? actionId[0] : actionId;

    emit('show-apply-perm-modal', async () => {
      const res = normalizeAuthResponse(
        await userPermsByAction(
          {
            actionId: $actionId,
            perm_ctx: permCtx,
          },
          { needRes: true },
        ).catch(() => ({})),
      );

      // if (res?.data?.perms?.[$actionId]) {
      //   Message('generic.msg.info.refreshAuth');
      // }
      return {
        perms: {
          apply_url: res?.data?.perms?.apply_url,
          action_list: [
            {
              action_id: $actionId,
              resource_name: resourceName,
            },
          ],
        },
      };
    });
  };

  cloneEl.addEventListener('mouseenter', cloneEl.mouseEnterHandler);
  cloneEl.addEventListener('mouseleave', cloneEl.mouseLeaveHandler);
  cloneEl.addEventListener('click', cloneEl.clickHandler);
}

function normalizeAuthResponse(res: unknown): AuthResponse {
  return res && typeof res === 'object' ? (res as AuthResponse) : {};
}

async function updatePerms(el: IElement, binding: DirectiveBinding, vNode: VNode) {
  const { actionId = '', permCtx } = binding.value as IOptions;
  const {
    cluster_id: clusterId,
    name,
    project_id: projectId,
    resource_type: resourceType,
    template_id: templateId,
  } = permCtx || {};
  // 校验数据完整性
  if (
    !actionId ||
    (!resourceType && actionId !== 'project_create') ||
    (resourceType === 'cluster' && !clusterId) ||
    (resourceType === 'project' && !projectId) ||
    (resourceType === 'templateset' && !templateId) ||
    (resourceType === 'namespace' && (!clusterId || !name))
  )
    return;

  const actionIds = Array.isArray(actionId) ? actionId : [actionId];
  const res = normalizeAuthResponse(
    await userPerms(
      {
        action_ids: actionIds,
        perm_ctx: permCtx,
      },
      { needRes: true },
    ).catch(() => ({})),
  );
  const clickable = actionIds.every(actionId => res?.data?.perms?.[actionId]);
  el.dataset.clickable = clickable ? 'true' : '';

  const cloneBinding = JSON.parse(JSON.stringify(binding)) as DirectiveBinding<IOptions>;
  cloneBinding.value.clickable = clickable;
  init(el, cloneBinding, vNode);
}

const AuthorityDirective: Directive = {
  created(el: IElement, _binding: DirectiveBinding, vNode: VNode) {
    // 父节点不存在时直接返回
    if (!el.parentNode) return;
    if (!vNode.key) {
      vNode.key = new Date().getTime();
    }
  },
  mounted(el: IElement, binding: DirectiveBinding, vNode: VNode) {
    if (!el.parentNode) return;
    // 和资源无关时自动发送鉴权逻辑
    const { disablePerms } = binding.value as IOptions;
    if (!disablePerms) {
      updatePerms(el, binding, vNode);
    } else {
      init(el, binding, vNode);
    }
  },
  beforeUpdate(el: IElement, binding: DirectiveBinding, vNode: VNode) {
    if (!el.parentNode) return;
    const { value, oldValue } = binding;
    if (deepEqual(value, oldValue)) return;
    init(el, binding, vNode);
  },
  beforeUnmount(el: IElement, _binding: DirectiveBinding, vNode: VNode) {
    let element = document.getElementById('directive-ele');
    element?.remove();
    element = null;
    destroy(el, vNode);
  },
};

export default AuthorityDirective;
