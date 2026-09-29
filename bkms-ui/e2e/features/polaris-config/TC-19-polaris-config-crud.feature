@TC-19 @P1 @readonly
Feature: TC-19 北极星配置增删改
  作为运维人员
  我需要新增、编辑并删除北极星配置
  以确认北极星配置页的核心变更请求参数正确

  Background:
    Given AccessToken 认证已配置

  @space:default @app:trpc @appType:trpc
  Scenario: 北极星配置支持新增编辑删除
    Given 北极星配置接口返回 CRUD 测试数据
    Given 我在当前应用的北极星配置页
    When 我添加从现有引入北极星配置
    Then 北极星新增请求参数应正确
    And 截图 "01-polaris-created"
    When 我编辑首条北极星配置
    Then 北极星编辑请求参数应正确
    And 截图 "02-polaris-edited"
    When 我删除首条北极星配置
    Then 北极星删除请求应命中目标配置
    And 截图 "03-polaris-deleted"
