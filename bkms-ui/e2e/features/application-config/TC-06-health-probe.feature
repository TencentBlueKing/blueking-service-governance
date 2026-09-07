@TC-06 @P0 @config-flow @stateful
Feature: TC-06 健康探针配置
  作为运维人员
  我需要在应用配置中查看、编辑并保存健康探针配置
  以验证健康探针核心交互与异常输入校验能力

  Background:
    Given AccessToken 认证已配置

  @space:default @app:default
  Scenario: 查看编辑校验并保存测试环境健康探针
    Given 我在当前应用的测试环境部署配置页
    Then 健康探针区域应展示三个探针卡片
    And 截图 "01-health-probe-view"
    When 我编辑 "存活探针" 后取消
    And 截图 "02-health-probe-liveness-cancelled"
    Then "存活探针" 不应包含取消测试路径
    When 我提交无效 "存活探针" 配置
    And 截图 "03-health-probe-liveness-invalid"
    Then "存活探针" 应展示校验提示并限制异常端口
    When 我保存有效 "存活探针" 配置
    And 截图 "04-health-probe-liveness-saved"
    Then "存活探针" 应处于查看态
    And "存活探针" 应展示已保存配置
    When 我编辑 "就绪探针" 后取消
    And 截图 "05-health-probe-readiness-cancelled"
    Then "就绪探针" 不应包含取消测试路径
    When 我提交无效 "就绪探针" 配置
    And 截图 "06-health-probe-readiness-invalid"
    Then "就绪探针" 应展示校验提示并限制异常端口
    When 我保存有效 "就绪探针" 配置
    And 截图 "07-health-probe-readiness-saved"
    Then "就绪探针" 应处于查看态
    And "就绪探针" 应展示已保存配置
    When 我编辑 "启动探针" 后取消
    And 截图 "08-health-probe-startup-cancelled"
    Then "启动探针" 不应包含取消测试路径
    When 我提交无效 "启动探针" 配置
    And 截图 "09-health-probe-startup-invalid"
    Then "启动探针" 应展示校验提示并限制异常端口
    When 我保存有效 "启动探针" 配置
    And 截图 "10-health-probe-startup-saved"
    Then "启动探针" 应处于查看态
    And "启动探针" 应展示已保存配置
