@TC-14 @P0 @readonly
Feature: TC-14 构建管理记录列表
  作为运维人员
  我需要查看、搜索和恢复构建记录列表
  以确认构建管理页的核心只读能力稳定可用

  Background:
    Given AccessToken 认证已配置

  @space:default @app:trpc @appType:trpc
  Scenario: 构建管理页应展示构建记录列表和查询入口
    Given 构建记录接口返回只读测试数据
    Given 我在当前应用的构建管理页
    Then 构建管理页应展示构建记录列表和查询入口
    And 截图 "01-build-management-records"

  @space:default @app:trpc @appType:trpc
  Scenario: 构建记录支持按触发人搜索并清空
    Given 构建记录接口返回只读测试数据
    Given 我在当前应用的构建管理页
    When 我按约定关键字搜索构建记录
    Then 构建记录搜索结果应匹配关键字
    And 截图 "02-build-record-search"
    When 我清空构建记录搜索
    Then 构建记录列表应恢复完整数据

  @space:default @app:trpc @appType:trpc
  Scenario: 构建记录接口异常后应支持刷新恢复
    Given 构建记录接口先失败后恢复
    Given 我在当前应用的构建管理页出现构建记录异常
    Then 构建记录页应展示数据获取异常
    And 截图 "03-build-record-error"
    When 我刷新构建记录异常空态
    Then 构建记录列表应恢复完整数据

  @space:default @app:trpc @appType:trpc
  Scenario: 构建记录支持分页和每页条数切换
    Given 构建记录接口返回分页测试数据
    Given 我在当前应用的构建管理页
    When 我切换构建记录到第二页
    Then 构建记录第二页应展示对应数据
    When 我切换构建记录每页条数为 20
    Then 构建记录每页 20 条应展示完整数据
