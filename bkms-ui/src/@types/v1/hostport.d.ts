/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：hostport

export interface ListHostPortsRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export type PutHostPortsRequest = PutHostPortsInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface HostPortsOutput {
  envStates?: Record<string, HostPortEnvStateOutput>;
  ports?: number[];
}

export interface PutHostPortsInput {
  ports: number[];
}

export interface HostPortEnvStateOutput {
  appliedPorts?: number[];
  pendingAddPorts?: number[];
  pendingRemovePorts?: number[];
}
