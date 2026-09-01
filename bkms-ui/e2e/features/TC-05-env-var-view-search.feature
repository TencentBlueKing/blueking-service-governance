@TC-05 @P0 @readonly
Feature: TC-05 环境变量查看与搜索
  作为运维人员
  我需要在应用配置中查看并搜索环境变量
  以验证应用变量与环境级变量的展示与搜索能力

  Background:
    Given AccessToken 认证已配置

  @space:default @app:default
  Scenario: 进入环境变量页应展示应用变量列表与搜索框
    Given 我在当前应用的环境变量页
    Then 应该看到 "新增应用变量"
    And 应该看到 "查看环境级变量"
    And 环境变量搜索框应可见
    And 应该看到 "Key"
    And 应该看到 "Value"
    And 截图 "01-env-variable-list"

  @space:default @app:default
  Scenario: 查看环境级变量侧栏应展示变量与搜索
    Given 我在当前应用的环境变量页
    When 我点击查看环境级变量
    Then 环境级变量侧栏应可见
    And 侧栏环境变量搜索框应可见
    And 侧栏环境变量表格应有数据
    And 截图 "02-env-level-vars-slider"

  @space:default @app:default
  Scenario: 在环境级变量侧栏按关键字搜索应精确匹配相关变量
    Given 我在当前应用的环境变量页
    When 我点击查看环境级变量
    And 我在侧栏搜索环境变量 "BKMS_ENV"
    And 截图 "03-slider-search-by-key"
    Then 应该看到 "BKMS_ENV_TYPE"
    And 不应该看到 "BKMS_BKAPM_API"

  @space:default @app:default
  Scenario: 在环境级变量侧栏搜索不存在的变量应为空，清空后应恢复
    Given 我在当前应用的环境变量页
    When 我点击查看环境级变量
    Then 侧栏环境变量表格应有数据
    When 我在侧栏搜索环境变量 "____NOT_EXIST_VAR____"
    And 截图 "04-slider-search-no-result"
    Then 侧栏环境变量表格应无数据
    When 我清空侧栏环境变量搜索
    And 截图 "05-slider-search-cleared"
    Then 侧栏环境变量表格应有数据
