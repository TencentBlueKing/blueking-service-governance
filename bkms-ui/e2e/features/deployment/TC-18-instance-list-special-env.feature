@TC-18 @P1 @readonly
Feature: TC-18 部署管理实例列表特殊环境
  作为运维人员
  我需要查看联邦环境与特性环境下的实例列表相关入口
  以确认特殊环境分支的展示和保护逻辑稳定可用

  Background:
    Given AccessToken 认证已配置

  @space:default @app:trpc @appType:trpc
  Scenario: 联邦环境实例列表应通过轮询展示数据并禁用灰度
    Given 联邦环境部署实例列表接口返回测试数据
    Given 我在当前应用的部署管理页
    Then 联邦环境实例列表应通过轮询展示数据并禁用灰度
    And 截图 "01-federation-instance-list"

  @space:default @app:trpc @appType:trpc
  Scenario: 应用关联特性环境侧栏应展示部署入口和销毁保护
    Given 部署实例列表接口返回只读测试数据
    Given 我在当前应用的部署管理页
    When 我打开应用关联特性环境侧栏
    Then 特性环境侧栏应展示部署入口和销毁保护
    And 截图 "02-feature-env-sideslider"
