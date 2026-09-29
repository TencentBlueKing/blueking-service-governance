@TC-17 @P1 @readonly
Feature: TC-17 部署管理实例列表只读能力
  作为运维人员
  我需要查看部署实例并打开只读操作入口
  以确认实例列表核心展示与入口能力稳定可用

  Background:
    Given AccessToken 认证已配置

  @space:default @app:trpc @appType:trpc
  Scenario: 实例列表应展示关键字段并建立 SSE 订阅
    Given 部署实例列表接口返回只读测试数据
    Given 我在当前应用的部署管理页
    Then 实例列表应展示关键字段、行操作入口并建立 SSE 订阅
    And 截图 "01-instance-list-readonly"

  @space:default @app:trpc @appType:trpc
  Scenario: 实例行操作支持打开日志灰度和调整权重弹窗
    Given 部署实例列表接口返回只读测试数据
    Given 我在当前应用的部署管理页
    When 我打开首行实例日志
    Then 实例日志侧栏应展示日志内容
    And 截图 "02-instance-log-sideslider"
    When 我关闭实例日志侧栏
    When 我打开首行实例灰度弹窗
    Then 灰度实例弹窗应展示镜像选择
    And 截图 "03-instance-gray-dialog"
    When 我关闭实例操作弹窗
    When 我打开首行实例权重弹窗
    Then 调整权重弹窗应展示权重表单
    And 截图 "04-instance-weight-dialog"

  @space:default @app:trpc @appType:trpc
  Scenario: 实例批量操作栏与管理命令入口可用
    Given 部署实例列表接口返回只读测试数据
    Given 我在当前应用的部署管理页
    When 我选中首个部署实例
    Then 实例批量操作栏应展示操作入口
    And 截图 "05-instance-batch-toolbar"
    When 我打开部署实例管理命令侧栏
    Then 管理命令侧栏应展示已选实例和命令选项
    And 截图 "06-admin-command-sideslider"

  @space:default @app:trpc @appType:trpc
  Scenario: 批量删除实例应展示确认信息
    Given 部署实例列表接口返回只读测试数据
    Given 我在当前应用的部署管理页
    When 我选中首个部署实例
    When 我打开选中部署实例的删除确认弹窗
    Then 删除实例确认弹窗应展示选中的实例信息
    And 截图 "07-instance-delete-confirm"
    When 我关闭实例删除确认弹窗
