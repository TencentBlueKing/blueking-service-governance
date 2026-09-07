@TC-15 @P0 @readonly
Feature: TC-15 构建日志与状态展示
  作为运维人员
  我需要从构建记录查看日志和失败状态
  以快速定位构建过程与异常原因

  Background:
    Given AccessToken 认证已配置

  @space:default @app:trpc @appType:trpc
  Scenario: 历史构建记录应支持打开构建日志
    Given 构建记录接口返回只读测试数据
    And 构建日志接口返回流式日志
    Given 我在当前应用的构建管理页
    When 我打开首行构建日志
    Then 构建日志侧栏应展示成功状态和日志内容
    And 截图 "01-build-log-success"

  @space:default @app:trpc @appType:trpc
  Scenario: 失败构建记录应展示失败状态
    Given 构建记录接口返回失败状态测试数据
    And 构建日志接口返回流式日志
    Given 我在当前应用的构建管理页
    Then 构建失败记录应可见
    When 我打开首行构建日志
    Then 构建日志侧栏应展示失败状态和日志内容
    And 截图 "02-build-log-failed"
