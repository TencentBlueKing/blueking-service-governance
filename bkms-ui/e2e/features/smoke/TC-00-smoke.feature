@TC-00 @smoke @readonly
Feature: TC-00 冒烟测试
  作为测试工程师
  我需要验证 BDD 测试框架是否正常工作
  以确保 playwright-bdd 集成正确

  Background:
    Given AccessToken 认证已配置

  Scenario: 应用可以正常加载
    When 进入应用
    And 截图 "01-app-list"
    Then 应该看到 "蓝鲸服务治理"
