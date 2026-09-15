@TC-21 @P1 @readonly
Feature: TC-21 北极星配置类型侧边栏
  作为运维人员
  我需要在新增和编辑侧边栏中确认不同北极星配置类型的字段和请求参数
  以避免错误修改北极星服务来源或提交不适用字段

  Background:
    Given AccessToken 认证已配置

  @space:default @app:trpc @appType:trpc
  Scenario: 北极星配置类型在新增和编辑侧边栏正确展示并提交匹配参数
    Given 北极星配置接口返回 CRUD 测试数据
    Given 我在当前应用的北极星配置页
    When 我打开北极星新增侧边栏并选择平台自动生成
    Then 北极星新增侧边栏应展示平台自动生成字段
    When 我提交平台自动生成北极星配置
    Then 平台自动生成北极星新增请求不应传递从现有引入字段
    And 截图 "01-polaris-auto-generated-create"
    When 我打开北极星新增侧边栏并选择从现有引入
    Then 北极星新增侧边栏应展示从现有引入字段
    When 我提交从现有引入北极星配置
    Then 从现有引入北极星新增请求不应传递平台自动生成字段
    And 截图 "02-polaris-import-create"
    When 我打开首条从现有引入配置的编辑侧边栏
    Then 北极星编辑侧边栏应保留从现有引入并限制不可编辑字段
    And 截图 "03-polaris-import-edit"

  @space:default @app:trpc @appType:trpc
  Scenario: 从现有引入编辑请求不传平台自动生成字段
    Given 北极星配置接口返回 CRUD 测试数据
    Given 我在当前应用的北极星配置页
    When 我保存首条从现有引入配置
    Then 从现有引入编辑请求不应传递平台自动生成字段
    And 截图 "04-polaris-import-edit-request"