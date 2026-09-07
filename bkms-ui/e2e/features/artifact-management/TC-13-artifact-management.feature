@TC-13 @P1 @readonly
Feature: TC-13 制品管理查看
  作为运维人员
  我需要查看应用的镜像制品列表
  以确认可用制品及其部署环境信息可被查询

  Background:
    Given AccessToken 认证已配置

  @space:default @app:default
  Scenario: 进入制品管理页应展示镜像列表和查询入口
    Given 我在当前应用的制品管理页
    Then 制品管理页应展示镜像列表和查询入口
    And 截图 "01-artifact-management"

  @space:default @app:default
  Scenario: 制品列表首行应支持展开和收起
    Given 我在当前应用的制品管理页
    When 我展开制品列表首行
    Then 制品列表首行详情应展示
    When 我收起制品列表首行
    Then 制品列表首行详情应隐藏
    And 截图 "02-artifact-row-collapsed"

  @space:default @app:default
  Scenario: 制品列表支持按首行 Tag 查询
    Given 我在当前应用的制品管理页
    When 我按制品列表首行 Tag 查询
    Then 制品列表应仅展示匹配的 Tag
    And 截图 "03-artifact-search"

  @space:default @app:default
  Scenario: 展开的制品行应展示详情字段和部署记录状态
    Given 我在当前应用的制品管理页
    When 我展开制品列表首行
    Then 制品列表首行详情应展示完整制品字段和部署记录状态
