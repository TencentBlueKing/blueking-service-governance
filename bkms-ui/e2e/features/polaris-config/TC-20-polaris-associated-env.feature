@TC-20 @P1 @readonly
Feature: TC-20 北极星关联环境
  作为运维人员
  我需要查看北极星配置关联的环境与权重统计
  以确认关联环境侧栏展示和统计接口稳定可用

  Background:
    Given AccessToken 认证已配置

  @space:default @app:trpc @appType:trpc
  Scenario: 北极星关联环境侧栏应展示环境统计
    Given 北极星配置接口返回 CRUD 测试数据
    Given 我在当前应用的北极星配置页
    When 我打开首条北极星配置的关联环境侧栏
    Then 北极星关联环境侧栏应展示环境统计
    And 截图 "01-polaris-associated-envs"
