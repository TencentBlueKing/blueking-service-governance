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

export interface AppendServiceToTrafficLaneRequest {
  service: ServiceData[];
}

export interface AppendServiceToTrafficLaneResponse {
  code: number;
  msg: string;
}

export interface AppGateway {
  createdAt: string;
  // creator & createdAt 创建人和创建时间
  creator: string;
  // gatewayAnnotations 网关注解, 扩展字段 或者 存储网关配置信息
  gatewayAnnotations: Record<string, string>;
  // gatewayDesc 网关描述
  gatewayDesc: string;
  // gatewayId 应用网关ID 且 以gateway开头
  gatewayId: string;
  // gatewayName 应用网关名称, 全局唯一
  gatewayName: string;
  // gatewayProvider 应用网关的底层提供者(Istio、蓝鲸网关、IngressConroller等)
  gatewayProvider: string;
  // gatewaySelector 网关选择器, 下发网关时默认配置, 可以自定义，若不填写则默认设置
  gatewaySelector: Record<string, string>;
  // gatewayServers 网关服务配置
  gatewayServers: GatewayServer[];
  // gatewaySpace 应用网关所在工作空间
  gatewaySpace: string;
  // gatewayStatus 网关状态, 当为enable时会部署应用网关，当为disable时只存储应用网关配置不会部署实际网关
  gatewayStatus: string;
  updatedAt: string;
  // updater & updatedAt
  updater: string;
}

export interface CorsPolicy {}

export interface CreateAppGatewayRequest {
  // annotations 网关注解
  annotations: Record<string, string>;
  // creator 创建人
  creator: string;
  // desc 网关描述
  desc: string;
  // gatewaySelector 网关选择器, 如果为空则设置网关默认值
  gatewaySelector: Record<string, string>;
  // name 网关名称
  name: string;
  // namespace 网关空间
  namespace: string;
  // provider 网关提供者
  provider: string;
  // servers 网关服务列表
  servers: GatewayServer[];
  // status 网关状态，若为空则默认关闭
  status: string;
}

export interface CreateAppGatewayResponse {
  code: number;
  data: AppGateway;
  msg: string;
}

export interface CreateServiceRequest {
  config: EastWestTrafficConfig;
  creator: string;
  // 是否只创建服务数据，不下发规则
  onlyData: boolean;
  service: ServiceData;
  serviceLane: string;
}

export interface CreateServiceResponse {
  code: number;
  data: TrafficLaneService;
  msg: string;
}

export interface CreateTrafficLaneRequest {
  // laneAnnotations 泳道自定义注解
  annotations: MapStruct;
  // clusters BCS集群列表
  clusters: string[];
  // creator 创建者
  creator: string;
  desc: string;
  // laneApp 泳道所属的微服务,每个微服务包含多个泳道，base泳道 特性泳道; 属于泳道组标签
  laneApp: string;
  laneEnv: string;
  // laneProvider 泳道南北向流量的provider, istio or 其他
  laneProvider: string;
  // laneServiceLabels 泳道服务不同版本所标识的标签, 可自定义或者使用平台公共Key SERVICE_VERSION_TAG
  // 泳道所属的标签实例值, 泳道配置则使用泳道配置值；若泳道不配置则使用平台默认值SERVICE_VERSION_TAG 且 自定义version
  laneServiceLabels: Record<string, string>;
  // laneServiceProvider 泳道东西向服务流量的provider，若为空则和南北向相同
  laneServiceProvider: string;
  laneSpace: string;
  laneTenantId: string;
  laneType: string;
  // 对于特性泳道来说，永远都是继承基线泳道的服务，基线泳道存在完成的服务列表；当服务存在基线泳道的时候才能加入到特性泳道
  mode: string;
  name: string;
  // serviceConfig 泳道服务配置, 支持在custom模式下直接创建服务
  serviceConfig: ServiceTrafficConfig;
}

export interface CreateTrafficLaneResponse {
  code: number;
  data: TrafficLane;
  msg: string;
}

export interface DeleteAppGatewayRequest {
  gatewayId: string;
}

export interface DeleteAppGatewayResponse {
  code: number;
  msg: string;
}

export interface DeleteServiceFromTrafficLaneRequest {
  service: ServiceData;
}

export interface DeleteServiceFromTrafficLaneResponse {
  code: number;
  msg: string;
}

export interface DeleteServiceRequest {
  retainData: boolean;
  serviceId: string;
}

export interface DeleteServiceResponse {
  code: number;
  msg: string;
}

export interface DeleteTrafficLaneRequest {
  laneId: string;
}

export interface DeleteTrafficLaneResponse {
  code: number;
  msg: string;
}

export interface Destination {
  host: string;
  port: number;
  subset: string;
}

export interface EastWestTrafficConfig {
  // global 默认情况下为nil，仅基线设置时表示全局的流量策略规则
  global: TrafficPolicy;
  // 服务级别的路由规则设置
  http: HttpRoute;
  tcp: TcpRoute;
  tls: TlsRoute;
  // trafficPolicy 服务级别的
  trafficPolicy: TrafficPolicy;
}

export interface GatewayServer {
  hosts: string[];
  port: Port;
  tls: ServerTLSSettings;
}

export interface GetAppGatewayRequest {
  gatewayId: string;
}

export interface GetAppGatewayResponse {
  code: number;
  data: AppGateway;
  msg: string;
}

export interface GetServiceRequest {
  serviceId: string;
}

export interface GetServiceResponse {
  code: number;
  data: TrafficLaneService;
  msg: string;
}

export interface GetTrafficLaneRequest {
  laneId: string;
}

export interface GetTrafficLaneResponse {
  code: number;
  data: TrafficLaneData;
  msg: string;
}

export interface HeaderOperations {
  // add Append the given values to the headers specified by keys
  add: Record<string, string>;
  // remove Remove the specified headers
  remove: string[];
  // set Overwrite the headers specified by key with the given values
  set: Record<string, string>;
}

export interface Headers {
  request: HeaderOperations;
  response: HeaderOperations;
}

/**
 * @deprecated 请改用 `~/@types/v1` 下对应 type（本属性已绑定 v1 实现）。
 * 事实上 `~/@types/v1` 并未提供对应 type，这里仍保留deprecated，待后续补充
 */
export interface HealthRequest {}

export interface HealthResponse {
  code: number;
  msg: string;
}

export interface HelloReply {
  value: string;
}

export interface HelloRequest {
  key: string;
}

export interface HTTPFaultInjection {}

export interface HttpMatchRequestRule {
  authority: MatchStringRule;
  // header 用于匹配 HTTP 请求头,键值对形式
  header: MatchStringRule[];
  // ignoreUriCase 忽略 URI 大小写
  ignoreUriCase: boolean;
  method: MatchStringRule;
  // name 匹配规则名称用于日志记录, assigned to the route for debugging purposes.
  name: string;
  // port 用于匹配 HTTP 请求的端口
  port: number;
  // queryParams 用于匹配 HTTP 请求的查询参数
  queryParams: MatchStringRule[];
  scheme: MatchStringRule;
  // sourceLabels 限制规则适用的源工作负载标签
  sourceLabels: Record<string, string>;
  // Uri, Scheme, Method, Authority: 用于匹配 HTTP 请求的 URI、协议、方法和权限。这些字段使用 StringMatch 类型,支持精确匹配、前缀匹配和正则匹配。
  uri: MatchStringRule;
}

export interface HttpRedirect {
  // authority 用于在重定向时替换 URL 的 Authority/Host 部分
  authority: string;
  // redirectCode 用于指定重定向响应中使用的 HTTP 状态码。默认的状态码是 301（永久移动）
  redirectCode: number;
  // redirectPort 用于指定重定向时使用的端口。它可以是具体的端口号（HTTPRedirect_Port）或者从其他设置中派生（HTTPRedirect_DerivePort）。
  redirectPort: number;
  // scheme 用于在重定向时替换 URL 的协议部分，例如 http 或 https。如果没有设置，将使用原始的协议。如果设置了 derivePort 为 FROM_PROTOCOL_DEFAULT，这也会影响到使用的端口。
  scheme: string;
  // uri 用于在重定向时替换 URL 的路径部分。无论请求的 URI 是精确匹配还是前缀匹配，整个路径都会被替换。
  uri: string;
}

export interface HttpRetry {
  attempts: number;
  // perTryTimeout format: 1h/1m/1s/1ms. MUST be >=1ms.
  perTryTimeout: string;
  // Specifies the conditions under which retry takes place.
  // One or more policies can be specified using a ',' delimited list.
  // If not specified, this defaults to `connect-failure,refused-stream,unavailable,cancelled,503`.
  retryOn: string;
  retryRemoteLocalities: google.protobuf.BoolValue;
}

export interface HttpRewrite {
  authority: string;
  uri: string;
  uriRegexRewrite: UriRegexRewrite;
}

export interface HttpRoute {
  // corsPolicy 用于配置 CORS(Cross-Origin Resource Sharing) 跨域资源共享规则
  corsPolicy: CorsPolicy;
  // fault injection policy to apply on HTTP traffic at the client side.
  // Note that timeouts or retries will not be enabled when faults are enabled on the client side.
  fault: HTTPFaultInjection;
  // headers 用于配置 HTTP 头部
  headers: Headers;
  // match 用于匹配 HTTP 请求的 URI、协议、方法和权限。这些字段使用 StringMatch 类型,支持精确匹配、前缀匹配和正则匹配。
  match: HttpMatchRequestRule[];
  name: string;
  // redirect 重定向配置
  redirect: HttpRedirect;
  // retries retry policy for HTTP requests.
  retries: HttpRetry;
  // rewrite 重写配置
  rewrite: HttpRewrite;
  // route  HTTP 流量的转发目标。可以指定多个服务版本作为目标，并通过权重来控制每个版本接收流量的比例
  route: HttpRouteDestination[];
  // timeout for HTTP requests, default is disabled when is 0 & seconds
  timeout: number;
}

export interface HttpRouteDestination {
  destination: Destination;
  headers: Headers;
  weight: number;
}

export interface L4MatchAttributes {
  // desSubnets IPv4 or IPv6 ip addresses of destination with optional subnet.
  desSubnets: string[];
  // gateways Names of gateways where the rule should be applied.
  gateways: string[];
  // port Specifies the port on the host that is being addressed.
  port: number;
  // sourceLabels One or more labels that constrain the applicability of a rule to workloads with the given labels.
  sourceLabels: Record<string, string>;
  sourceNamespace: string;
  sourceSubnet: string;
}

export interface LaneServiceRules {
  headers: Record<string, string>;
}

export interface ListAppGatewayRequest {
  env: string;
  gatewayId: string;
  provider: string;
  workspace: string;
}

export interface ListAppGatewayResponse {
  code: number;
  data: AppGateway[];
  msg: string;
}

export interface ListClusterServicesRequest {
  // 集群ID
  clusterId: string;
  // 环境ID，暂不使用
  envID: string;
}

export interface ListClusterServicesResponse {
  code: number;
  data: serviceInfo[];
  msg: string;
}

export interface ListServiceRequest {
  serviceHost: string;
  serviceId: string;
  serviceLane: string;
  serviceName: string;
  serviceSpace: string;
  serviceVersion: string;
}

export interface ListServiceResponse {
  code: number;
  data: TrafficLaneService[];
  msg: string;
}

export interface ListTrafficLaneData {
  count: number;
  data: TrafficLaneData[];
}

export interface ListTrafficLaneRequest {
  laneApp: string;
  laneEnv: string;
  laneId: string;
  laneName: string;
  laneServiceStatus: string;
  laneSpace: string;
  laneStatus: string;
  laneType: string;
  limit: number;
  page: number;
}

export interface ListTrafficLaneResponse {
  code: number;
  data: ListTrafficLaneData;
  msg: string;
}

export interface MapStruct {
  values: Record<string, string>;
}

export interface MatchStringRule {
  // matchKey 匹配键
  matchKey: string;
  // matchType 支持精确匹配、前缀匹配和正则匹配
  matchType: string;
  // matchValue 匹配值
  matchValue: string;
}

export interface NorthSouthTrafficConfig {
  // gatewayName 入口服务使用的网关
  gatewayName: string;
  hosts: string[];
  http: HttpRoute;
  // laneNameSpace 泳道所处的命名空间
  laneNameSpace: string;
  tcp: TcpRoute;
  tls: TlsRoute;
}

export interface Port {
  // name label assigned to the port.
  name: string;
  // port a valid non-negative integer port number.
  port: number;
  // protocol the protocol exposed on the port. MUST BE one of HTTP|HTTPS|GRPC|GRPC-WEB|HTTP2|MONGO|TCP|TLS.
  // TLS can be either used to terminate non-HTTP based connections on a specific port
  // or to route traffic based on SNI header to the destination without terminating the TLS connection.
  protocol: string;
}

export interface RouteDestination {
  destination: Destination;
  weight: number;
}

export interface ServerTLSSettings {
  // caCertificates REQUIRED if mode is `MUTUAL` or `OPTIONAL_MUTUAL`. The path to a file containing certificate
  // authority certificates to use in verifying a presented client side certificate.
  caCertificates: string;
  // credentialName For gateways running on Kubernetes, the name of the secret that holds the TLS certs including
  // the CA certificates. Applicable only on Kubernetes.
  credentialName: string;
  // mode 0: "PASSTHROUGH", 1: "SIMPLE", 2: "MUTUAL",
  mode: number;
  // privateKey REQUIRED if mode is `SIMPLE` or `MUTUAL`. The path to the file holding the server's private key
  privateKey: string;
  // serverCertificate holding the server-side TLS certificate to use
  serverCertificate: string;
}

export interface ServiceData {
  serviceHost: string;
  serviceKey: string;
  serviceName: string;
  serviceSpace: string;
  serviceVersion: string;
}

export interface serviceInfo {
  namespace: string;
  serviceName: string;
}

export interface ServiceTrafficConfig {
  headers: MapStruct;
  services: ServiceData[];
}

export interface TcpRoute {
  match: L4MatchAttributes[];
  route: RouteDestination[];
}

export interface TLSMatchAttributes {
  // desSubnets IPv4 or IPv6 ip addresses of destination with optional subnet.
  desSubnets: string[];
  // gateways Names of gateways where the rule should be applied.
  gateways: string[];
  // port Specifies the port on the host that is being addressed.
  port: number;
  // sniHosts SNI (server name indicator) to match on.
  sniHosts: string[];
  // sourceLabels One or more labels that constrain the applicability of a rule to workloads with the given labels.
  sourceLabels: Record<string, string>;
  sourceNamespace: string;
}

export interface TlsRoute {
  match: TLSMatchAttributes[];
  route: RouteDestination[];
}

export interface TrafficLane {
  // clusters 泳道规则下发的集群，默认通过BCS集群通道下发
  clusters: string[];
  createdAt: string;
  creator: string;
  // laneAnnotations 泳道扩展字段, 针对不同产品通过不同的kv扩展
  laneAnnotations: Record<string, string>;
  laneDesc: string;
  // laneEnv 泳道所属环境
  laneEnv: string;
  laneId: string;
  // laneLabels 泳道labels (包含泳道所属的微服务信息，由平台注入或者产品化时平台提供用户选择 (key: Lane-Service-Attribute)，类似于泳道组的概念)
  laneLabels: Record<string, string>;
  laneName: string;
  // laneProvider 泳道提供者，目前仅支持istio
  laneProvider: string;
  // laneProvider 泳道提供者，目前仅支持istio
  laneServiceProvider: string;
  // laneServiceVersionLabels 泳道服务版本标签, 平台注入或者用户自定义(key: SERVICE_VERSION_TAG), 由平台主动注入和用户版本解耦
  // 若为空，则表示使用平台默认注入
  laneServiceVersionLabels: Record<string, string>;
  // laneSpace 泳道空间，顶层所属层级
  laneSpace: string;
  // laneStatus 泳道状态, 当为enable时会部署泳道南北向流程，当为disable时关闭泳道的南北向流量
  laneStatus: string;
  // laneTenantId 泳道所属租户Id
  laneTenantId: string;
  laneType: string;
  // 基线泳道生效
  mode: string;
  // serviceRules 泳道内服务内统一流量规则，若不统一配置可由各个服务单独配置
  serviceRules: LaneServiceRules;
  // serviceStatus 泳道服务状态, 当为enable时会部署泳道内服务的东西向流量，当为disable时关闭泳道的东西向流量; 保留泳道服务数据
  serviceStatus: string;
  // trafficConfig 泳道南北向流量配置
  trafficConfig: NorthSouthTrafficConfig;
  updatedAt: string;
  updater: string;
}

export interface TrafficLaneData {
  lane: TrafficLane;
  services: TrafficLaneService[];
}

export interface TrafficLaneService {
  createdAt: string;
  creator: string;
  // serviceHost 服务域名，可访问的通信域名
  serviceHost: string;
  // serviceId 服务ID
  serviceId: string;
  // serviceKey 服务label的Key，可由用户进行自定义, 若为空则使用平台默认的Key. 平台默认Key: SERVICE_VERSION_TAG
  serviceKey: string;
  // serviceLane 服务所属泳道, 服务可变更泳道信息
  serviceLane: string;
  // serviceName 服务名称
  serviceName: string;
  // serviceSpace 服务所属空间(命令空间或者其他东西向流量设置空间标识)
  serviceSpace: string;
  // serviceVersion 服务版本, 相同服务名称拥有不同的服务版本
  serviceVersion: string;
  // trafficConfig 服务东西向流量配置
  trafficConfig: EastWestTrafficConfig;
  updatedAt: string;
  updater: string;
}

export interface TrafficPolicy {}

export interface UpdateAppGatewayRequest {
  annotations: Record<string, string>;
  desc: string;
  gatewayId: string;
  servers: GatewayServer[];
  updater: string;
}

export interface UpdateAppGatewayResponse {
  code: number;
  msg: string;
}

export interface UpdateAppGatewayStatusRequest {
  // enable 为true开启网关，为false关闭网关
  enable: boolean;
  gatewayId: string;
}

export interface UpdateAppGatewayStatusResponse {
  code: number;
  msg: string;
}

export interface UpdateServiceLaneRequest {
  config: EastWestTrafficConfig;
  lane: string;
  serviceId: string;
}

export interface UpdateServiceLaneResponse {
  code: number;
  msg: string;
}

export interface UpdateServiceRequest {
  serviceId: string;
}

export interface UpdateServiceResponse {
  code: number;
  msg: string;
}

export interface UpdateServiceTrafficEntryRequest {
  config: EastWestTrafficConfig;
  serviceId: string;
}

export interface UpdateServiceTrafficEntryResponse {
  code: number;
  msg: string;
}

export interface UpdateTrafficLaneEntryRequest {
  config: NorthSouthTrafficConfig;
  laneId: string;
}

export interface UpdateTrafficLaneEntryResponse {
  code: number;
  msg: string;
}

export interface UpdateTrafficLaneRequest {
  annotations: MapStruct;
  laneDesc: string;
  laneId: string;
  laneName: string;
  // 泳道生效的全部services列表，前端传入全量的services列表，后端进行实际匹配处理
  // 基线泳道 选择的是集群所有service; 特性泳道 选择的是基线泳道的service列表
  services: ServiceData[];
  updater: string;
}

export interface UpdateTrafficLaneResponse {
  code: number;
  msg: string;
}

export interface UpdateTrafficLaneServicesStatusRequest {
  // enable 为true开启泳道内所有services流量规则，为false关闭泳道内所有services的流量规则
  enable: boolean;
  laneId: string;
}

export interface UpdateTrafficLaneServicesStatusResponse {
  code: number;
  msg: string;
}

export interface UpdateTrafficLaneStatusRequest {
  // enable 为true开启泳道南北向流量，为false关闭泳道南北向流量
  enable: boolean;
  laneId: string;
}

export interface UpdateTrafficLaneStatusResponse {
  code: number;
  msg: string;
}

export interface UriRegexRewrite {
  match: string;
  rewrite: string;
}
