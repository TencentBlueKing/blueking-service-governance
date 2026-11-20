/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：cfgswitch

export interface GetCfgSwitchesRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export interface CfgSwitchesResponse {
  data?: CfgSwitchesOutput;
}

export interface CfgSwitchesOutput {
  /**
   * 是否开启配置管理（BSCP）
   */
  bscp?: boolean;
  /**
   * 是否展示七彩石
   */
  rainbowTConf?: boolean;
}
