#!/usr/bin/env python3
# -*- coding: utf-8 -*-

# TencentBlueKing is pleased to support the open source community by making
# 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
# Copyright (C) Tencent. All rights reserved.
# Licensed under the MIT License (the "License"); you may not use this file except
# in compliance with the License. You may obtain a copy of the License at
#
#  http://opensource.org/licenses/MIT
#
# Unless required by applicable law or agreed to in writing, software distributed under
# the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
# either express or implied. See the License for the specific language governing permissions and
# limitations under the License.
#

"""
generate_e2e_report.py - 将 E2E 测试 JSON 结果渲染为自包含 HTML 报告

用法:
    python generate_e2e_report.py --input result.json --output report.html
    python generate_e2e_report.py --input result.json --output report.html \
        --screenshots-dir ./test-reports/20260306-0952-xxx/

输入:
    JSON 文件，格式见 SKILL.md 中的 result.json 说明

输出:
    自包含的 HTML 报告文件（截图以 base64 内联）
"""

import argparse
import base64
import json
import mimetypes
import os
import sys
from datetime import datetime


def esc(text):
    """HTML 转义"""
    return (
        str(text)
        .replace("&", "&amp;")
        .replace("<", "&lt;")
        .replace(">", "&gt;")
        .replace('"', "&quot;")
    )


def build_font_face_css(font_base_url=""):
    """根据字体 CDN 根地址生成 @font-face；未配置时回退系统字体。"""
    font_base_url = (font_base_url or os.environ.get("E2E_REPORT_FONT_BASE_URL", "")).rstrip("/")
    if not font_base_url:
        return ""
    return (
        "@font-face {\n"
        "            font-family: 'TencentSans';\n"
        f"            src: url('{esc(font_base_url)}/TencentSans-W3-CN.woff2') format('woff2'),\n"
        f"                 url('{esc(font_base_url)}/TencentSans-W3.woff') format('woff');\n"
        "            font-weight: normal; font-style: normal; font-display: swap;\n"
        "        }\n"
        "        @font-face {\n"
        "            font-family: 'TencentSans';\n"
        f"            src: url('{esc(font_base_url)}/TencentSans-W7-CN.woff2') format('woff2'),\n"
        f"                 url('{esc(font_base_url)}/TencentSans-W7.woff') format('woff');\n"
        "            font-weight: bold; font-style: normal; font-display: swap;\n"
        "        }"
    )


def load_screenshot_base64(screenshots_dir, filename):
    """加载截图并转为 base64 data URI"""
    if not filename:
        return None
    filepath = os.path.join(screenshots_dir, filename)
    if not os.path.exists(filepath):
        return None
    mime_type, _ = mimetypes.guess_type(filepath)
    if not mime_type:
        mime_type = "image/png"
    try:
        with open(filepath, "rb") as f:
            data = f.read()
        b64 = base64.b64encode(data).decode("ascii")
        return f"data:{mime_type};base64,{b64}"
    except Exception:
        return None


def build_summary_rows(summary):
    """构建结果摘要表格行"""
    rows = []
    total_steps = 0
    total_passed = 0
    total_failed = 0
    for item in summary:
        name = esc(item.get("name", "-"))
        steps = item.get("total_steps", 0)
        passed = item.get("passed", 0)
        failed = item.get("failed", 0)
        status = item.get("status", "pass")
        total_steps += steps
        total_passed += passed
        total_failed += failed
        if status == "pass":
            status_html = '<span class="status-tag status-pass">PASS</span>'
        else:
            status_html = '<span class="status-tag status-fail">FAIL</span>'
        rows.append(
            f"<tr><td>{name}</td><td>{steps}</td>"
            f"<td>{passed}</td><td>{failed}</td><td>{status_html}</td></tr>"
        )
    # 合计行
    if total_failed == 0:
        overall_html = '<span class="status-tag status-pass">ALL PASS</span>'
    elif total_passed == 0:
        overall_html = '<span class="status-tag status-fail">ALL FAIL</span>'
    else:
        overall_html = '<span class="status-tag status-warn">PARTIAL</span>'
    rows.append(
        f'<tr class="summary-total"><td><b>合计</b></td><td><b>{total_steps}</b></td>'
        f"<td><b>{total_passed}</b></td><td><b>{total_failed}</b></td>"
        f"<td>{overall_html}</td></tr>"
    )
    return "\n".join(rows)


def build_step_details(steps, screenshots_dir):
    """构建详细步骤部分"""
    if not steps:
        return '<div class="card"><p style="color:var(--text-hint);text-align:center;">暂无步骤数据</p></div>'

    # 按 case_id 分组
    cases = {}
    for step in steps:
        cid = step.get("case_id", "default")
        cname = step.get("case_name", cid)
        if cid not in cases:
            cases[cid] = {"name": cname, "steps": []}
        cases[cid]["steps"].append(step)

    parts = []
    for cid, case_data in cases.items():
        case_name = esc(case_data["name"])
        step_cards = []
        for step in case_data["steps"]:
            step_num = step.get("step_num", "?")
            title = esc(step.get("title", "-"))
            command = esc(step.get("command", ""))
            result = esc(step.get("result", ""))
            assertion = esc(step.get("assertion", ""))
            status = step.get("status", "pass")
            screenshot = step.get("screenshot", "")

            icon_cls = "pass" if status == "pass" else "fail"
            icon_text = "V" if status == "pass" else "X"
            status_prefix = "PASS" if status == "pass" else "FAIL"

            screenshot_html = ""
            if screenshot:
                # 支持逗号分隔的多个截图文件名
                screenshot_files = [s.strip() for s in screenshot.split(",") if s.strip()]
                img_tags = []
                for scr_file in screenshot_files:
                    img_src = load_screenshot_base64(screenshots_dir, scr_file)
                    if img_src:
                        img_tags.append(
                            f'<img src="{img_src}" class="screenshot" '
                            f'alt="{esc(scr_file)}" loading="lazy" '
                            f'style="margin-bottom:8px;">'
                        )
                if img_tags:
                    screenshot_html = (
                        f'<div class="step-screenshot">'
                        f'{"".join(img_tags)}'
                        f"</div>"
                    )

            step_cards.append(
                f'<div class="step-card">'
                f'<div class="step-header">'
                f'<span class="step-icon {icon_cls}">{icon_text}</span>'
                f'<span class="step-num">步骤 {step_num}</span>'
                f'<span class="step-title">{title}</span>'
                f'<span class="step-status step-status-{icon_cls}">{status_prefix}</span>'
                f"</div>"
                f'<div class="step-body">'
                f'<div class="step-detail"><span class="detail-label">结果</span>'
                f"<span>{result}</span></div>"
                f'<div class="step-detail"><span class="detail-label">断言</span>'
                f"<span>{assertion}</span></div>"
                f"{screenshot_html}"
                f"</div></div>"
            )

        parts.append(
            f'<div class="card">'
            f'<h2>[{esc(cid)}] {case_name}</h2>'
            f'{"".join(step_cards)}'
            f"</div>"
        )

    return "\n".join(parts)


def build_issues_section(issues):
    """构建问题汇总部分"""
    if not issues:
        return ""
    rows = []
    for iss in issues:
        rows.append(
            f'<tr><td>{esc(iss.get("case_id", "-"))}</td>'
            f'<td>{esc(iss.get("step", "-"))}</td>'
            f'<td>{esc(iss.get("description", "-"))}</td>'
            f'<td>{esc(iss.get("suggestion", "-"))}</td></tr>'
        )
    return (
        '<div class="card">'
        '<h2>问题汇总</h2>'
        '<div class="table-wrap">'
        '<table class="issue-table">'
        "<thead><tr><th>用例</th><th>步骤</th><th>问题描述</th><th>建议</th></tr></thead>"
        f'<tbody>{"".join(rows)}</tbody>'
        "</table></div></div>"
    )


# ──────────────────────────────────────────────
# HTML 模板
# ──────────────────────────────────────────────
HTML_TEMPLATE = r"""<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>BKMS E2E 测试报告 - {test_cases}</title>
    <style>
        {font_face_css}

        :root {{
            --primary: #0052d9;
            --primary-hover: #366ef4;
            --primary-light: #f2f3ff;
            --primary-light-hover: #d9e1ff;
            --success: #2ba471;
            --warning: #e37318;
            --danger: #d54941;
            --text: rgba(0, 0, 0, 0.9);
            --text-secondary: rgba(0, 0, 0, 0.6);
            --text-hint: rgba(0, 0, 0, 0.4);
            --border: #e8e8e8;
            --bg: #f5f6f7;
            --card-bg: #ffffff;
            --code-bg: #f7f8fa;
            --shadow: 0 1px 10px rgba(0,0,0,0.05), 0 4px 5px rgba(0,0,0,0.08), 0 2px 4px -1px rgba(0,0,0,0.12);
            --font-main: 'TencentSans', system-ui, -apple-system, 'Segoe UI', sans-serif;
            --font-code: 'Menlo', 'Monaco', 'Consolas', monospace;
            --cover-gradient: linear-gradient(135deg, #0a1628 0%, #0d2b5e 40%, #0052d9 100%);
        }}

        * {{ margin: 0; padding: 0; box-sizing: border-box; }}
        body {{
            font-family: var(--font-main);
            background: var(--bg);
            color: var(--text);
            -webkit-font-smoothing: antialiased;
            line-height: 1.6;
        }}

        /* ── 封面 ── */
        .report-cover {{
            background: var(--cover-gradient);
            color: white;
            padding: 60px 48px;
            position: relative;
            overflow: hidden;
            min-height: 280px;
            display: flex;
            flex-direction: column;
            justify-content: center;
        }}
        .report-cover::before {{
            content: '';
            position: absolute;
            width: 500px; height: 500px;
            border-radius: 50%;
            background: radial-gradient(circle, rgba(0,82,217,0.15) 0%, transparent 70%);
            right: -80px; top: -80px;
        }}
        .report-cover::after {{
            content: '';
            position: absolute;
            width: 350px; height: 350px;
            border-radius: 50%;
            background: radial-gradient(circle, rgba(43,164,113,0.1) 0%, transparent 70%);
            left: 10%; bottom: -60px;
        }}
        .cover-logo {{
            position: absolute;
            top: 24px; left: 36px;
            background: rgba(255,255,255,0.12);
            backdrop-filter: blur(10px);
            border: 1px solid rgba(255,255,255,0.2);
            border-radius: 10px;
            padding: 6px 16px;
            font-weight: bold;
            font-size: 13px;
            color: white;
            letter-spacing: 2px;
        }}
        .cover-badge {{
            display: inline-block;
            background: rgba(43,164,113,0.2);
            border: 1px solid rgba(43,164,113,0.4);
            color: #6ee7b7;
            padding: 4px 14px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: bold;
            letter-spacing: 1px;
            margin-bottom: 20px;
            width: fit-content;
        }}
        .report-cover h1 {{
            font-size: 36px;
            font-weight: bold;
            line-height: 1.3;
            margin-bottom: 14px;
            letter-spacing: -0.02em;
            position: relative;
            z-index: 1;
        }}
        .report-cover h1 span {{ color: #60a5fa; }}
        .report-cover .subtitle {{
            font-size: 14px;
            line-height: 1.8;
            color: rgba(255,255,255,0.7);
            max-width: 700px;
            position: relative;
            z-index: 1;
        }}
        .cover-tags {{
            display: flex;
            gap: 10px;
            margin-top: 24px;
            flex-wrap: wrap;
            position: relative;
            z-index: 1;
        }}
        .cover-tag {{
            background: rgba(255,255,255,0.08);
            border: 1px solid rgba(255,255,255,0.15);
            color: rgba(255,255,255,0.85);
            padding: 5px 12px;
            border-radius: 8px;
            font-size: 12px;
            font-weight: 500;
        }}

        /* ── 内容区域 ── */
        .report-body {{
            max-width: 1200px;
            margin: 0 auto;
            padding: 36px 24px 48px;
        }}

        /* ── 卡片 ── */
        .card {{
            background: var(--card-bg);
            border: 1px solid var(--border);
            border-radius: 16px;
            padding: 24px;
            box-shadow: var(--shadow);
            margin-bottom: 20px;
        }}
        .card h2 {{
            font-size: 18px;
            color: var(--primary);
            margin-bottom: 20px;
            padding-bottom: 12px;
            border-bottom: 3px solid var(--primary);
            display: flex;
            align-items: center;
            gap: 8px;
        }}

        /* ── 摘要指标 ── */
        .summary-metrics {{
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
            gap: 14px;
            margin-bottom: 24px;
        }}
        .metric-card {{
            background: var(--primary-light);
            border-radius: 14px;
            padding: 18px 14px;
            text-align: center;
        }}
        .metric-card .metric-value {{
            font-size: 28px;
            font-weight: bold;
            color: var(--primary);
            margin-bottom: 2px;
        }}
        .metric-card .metric-label {{
            font-size: 12px;
            color: var(--text-secondary);
        }}
        .metric-card.metric-pass .metric-value {{ color: var(--success); }}
        .metric-card.metric-pass {{ background: rgba(43,164,113,0.08); }}
        .metric-card.metric-fail .metric-value {{ color: var(--danger); }}
        .metric-card.metric-fail {{ background: rgba(213,73,65,0.08); }}

        /* ── 摘要表格 ── */
        .table-wrap {{ overflow-x: auto; }}
        .summary-table {{
            width: 100%;
            border-collapse: separate;
            border-spacing: 0;
            font-size: 14px;
        }}
        .summary-table th {{
            background: var(--primary-light);
            color: var(--primary);
            font-weight: 700;
            padding: 10px 14px;
            text-align: left;
            border-bottom: 2px solid var(--primary);
            white-space: nowrap;
        }}
        .summary-table td {{
            padding: 10px 14px;
            border-bottom: 1px solid var(--border);
            color: var(--text-secondary);
        }}
        .summary-table tr:hover td {{ background: var(--primary-light); }}
        .summary-total td {{ background: rgba(0,82,217,0.03); }}

        /* ── 状态标签 ── */
        .status-tag {{
            display: inline-block;
            padding: 3px 12px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: 700;
        }}
        .status-pass {{ background: rgba(43,164,113,0.1); color: var(--success); border: 1px solid rgba(43,164,113,0.3); }}
        .status-fail {{ background: rgba(213,73,65,0.1); color: var(--danger); border: 1px solid rgba(213,73,65,0.3); }}
        .status-warn {{ background: rgba(227,115,24,0.1); color: var(--warning); border: 1px solid rgba(227,115,24,0.3); }}

        /* ── 步骤卡片 ── */
        .step-card {{
            margin-bottom: 14px;
            padding: 16px 18px;
            border: 1px solid var(--border);
            border-radius: 10px;
            transition: box-shadow 0.2s;
        }}
        .step-card:hover {{ box-shadow: 0 2px 8px rgba(0,0,0,0.06); }}
        .step-card:last-child {{ margin-bottom: 0; }}
        .step-header {{
            display: flex;
            align-items: center;
            gap: 10px;
            margin-bottom: 12px;
        }}
        .step-icon {{
            width: 22px; height: 22px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 11px;
            font-weight: bold;
            color: white;
            flex-shrink: 0;
        }}
        .step-icon.pass {{ background: var(--success); }}
        .step-icon.fail {{ background: var(--danger); }}
        .step-num {{
            font-size: 12px;
            color: var(--text-hint);
            font-weight: 600;
        }}
        .step-title {{
            font-weight: 600;
            font-size: 14px;
            flex: 1;
        }}
        .step-status {{
            font-size: 11px;
            font-weight: 700;
            padding: 2px 10px;
            border-radius: 10px;
        }}
        .step-status-pass {{ background: rgba(43,164,113,0.1); color: var(--success); }}
        .step-status-fail {{ background: rgba(213,73,65,0.1); color: var(--danger); }}
        .step-body {{
            padding-left: 32px;
        }}
        .step-detail {{
            display: flex;
            gap: 8px;
            margin-bottom: 6px;
            font-size: 13px;
            color: var(--text-secondary);
            line-height: 1.5;
        }}
        .step-detail:last-child {{ margin-bottom: 0; }}
        .detail-label {{
            flex-shrink: 0;
            font-weight: 700;
            color: var(--text-hint);
            width: 36px;
            text-align: right;
        }}
        .step-detail code {{
            background: var(--code-bg);
            padding: 2px 8px;
            border-radius: 4px;
            font-size: 12px;
            font-family: var(--font-code);
            word-break: break-all;
            border: 1px solid var(--border);
        }}
        .step-screenshot {{
            margin-top: 12px;
        }}
        .screenshot {{
            max-width: 100%;
            border-radius: 8px;
            border: 1px solid var(--border);
            box-shadow: 0 1px 4px rgba(0,0,0,0.08);
        }}

        /* ── 问题汇总表 ── */
        .issue-table {{
            width: 100%;
            border-collapse: separate;
            border-spacing: 0;
            font-size: 13px;
        }}
        .issue-table th {{
            background: rgba(227,115,24,0.08);
            color: var(--warning);
            font-weight: 700;
            padding: 10px 12px;
            text-align: left;
            border-bottom: 2px solid var(--warning);
        }}
        .issue-table td {{
            padding: 10px 12px;
            border-bottom: 1px solid var(--border);
            color: var(--text-secondary);
        }}
        .issue-table tr:hover td {{ background: rgba(227,115,24,0.04); }}

        /* ── 页脚 ── */
        .report-footer {{
            text-align: center;
            padding: 28px 24px;
            color: var(--text-hint);
            font-size: 12px;
            border-top: 1px solid var(--border);
        }}

        /* ── 响应式 ── */
        @media (max-width: 768px) {{
            .report-cover {{ padding: 48px 20px; }}
            .report-cover h1 {{ font-size: 24px; }}
            .report-body {{ padding: 20px 12px 32px; }}
            .summary-metrics {{ grid-template-columns: 1fr 1fr; }}
            .step-body {{ padding-left: 0; }}
        }}
    </style>
</head>
<body>

<!-- 封面 -->
<div class="report-cover">
    <div class="cover-logo">BKMS</div>
    <div class="cover-badge">E2E TEST REPORT</div>
    <h1>E2E 测试报告<br><span>{test_cases}</span></h1>
    <div class="subtitle">
        测试时间: {test_time}<br>
        测试环境: {test_site}<br>
        认证方式: {auth_method}
    </div>
    <div class="cover-tags">
        <span class="cover-tag">空间: {space}</span>
        <span class="cover-tag">环境: {env}</span>
        <span class="cover-tag">报告目录: {report_dir}</span>
    </div>
</div>

<div class="report-body">

    <!-- 摘要指标 -->
    <div class="summary-metrics">
        <div class="metric-card">
            <div class="metric-value">{total_cases}</div>
            <div class="metric-label">测试用例</div>
        </div>
        <div class="metric-card">
            <div class="metric-value">{total_steps}</div>
            <div class="metric-label">总步骤</div>
        </div>
        <div class="metric-card metric-pass">
            <div class="metric-value">{total_passed}</div>
            <div class="metric-label">通过</div>
        </div>
        <div class="metric-card metric-fail">
            <div class="metric-value">{total_failed}</div>
            <div class="metric-label">失败</div>
        </div>
    </div>

    <!-- 结果摘要 -->
    <div class="card">
        <h2>结果摘要</h2>
        <div class="table-wrap">
            <table class="summary-table">
                <thead>
                    <tr><th>用例</th><th>总步骤</th><th>通过</th><th>失败</th><th>状态</th></tr>
                </thead>
                <tbody>
{summary_rows}
                </tbody>
            </table>
        </div>
    </div>

    <!-- 详细步骤 -->
{step_details}

    <!-- 问题汇总 -->
{issues_section}

    <!-- 建议 -->
    <div class="card">
        <h2>建议</h2>
        <p style="font-size:14px;color:var(--text-secondary);line-height:1.8;">{recommendations}</p>
    </div>

</div>

<div class="report-footer">
    BKMS E2E 测试报告 &mdash; {test_cases} &mdash; {test_time}
</div>

</body>
</html>"""


def render_report(data, screenshots_dir, output_path):
    """渲染 HTML 报告"""
    meta = data.get("meta", {})
    summary = data.get("summary", [])
    steps = data.get("steps", [])
    issues = data.get("issues", [])
    recommendations = data.get("recommendations", "")

    # 计算汇总指标
    total_cases = len(summary)
    total_steps = sum(s.get("total_steps", 0) for s in summary)
    total_passed = sum(s.get("passed", 0) for s in summary)
    total_failed = sum(s.get("failed", 0) for s in summary)

    html = HTML_TEMPLATE.format(
        font_face_css=build_font_face_css(),
        test_cases=esc(meta.get("test_cases", "-")),
        test_time=esc(meta.get("test_time", datetime.now().strftime("%Y-%m-%d %H:%M"))),
        test_site=esc(meta.get("test_site", "-")),
        auth_method=esc(meta.get("auth_method", "AccessToken (Bearer)")),
        space=esc(meta.get("space", "-")),
        env=esc(meta.get("env", "-")),
        report_dir=esc(meta.get("report_dir", "-")),
        total_cases=total_cases,
        total_steps=total_steps,
        total_passed=total_passed,
        total_failed=total_failed,
        summary_rows=build_summary_rows(summary),
        step_details=build_step_details(steps, screenshots_dir),
        issues_section=build_issues_section(issues),
        recommendations=esc(recommendations),
    )

    with open(output_path, "w", encoding="utf-8") as f:
        f.write(html)

    print(f"HTML 报告已生成: {output_path}")
    file_size = os.path.getsize(output_path)
    if file_size > 1024 * 1024:
        print(f"文件大小: {file_size / (1024 * 1024):.1f} MB")
    else:
        print(f"文件大小: {file_size / 1024:.1f} KB")


def main():
    parser = argparse.ArgumentParser(description="将 E2E 测试 JSON 结果渲染为 HTML 报告")
    parser.add_argument("--input", required=True, help="JSON 测试结果文件路径")
    parser.add_argument("--output", required=True, help="HTML 报告输出路径")
    parser.add_argument(
        "--screenshots-dir",
        default="",
        help="截图目录（默认与 JSON 文件同目录）",
    )

    args = parser.parse_args()

    if not os.path.exists(args.input):
        print(f"输入文件不存在: {args.input}", file=sys.stderr)
        sys.exit(1)

    try:
        with open(args.input, "r", encoding="utf-8") as f:
            data = json.load(f)
    except json.JSONDecodeError as e:
        print(f"JSON 解析错误: {e}", file=sys.stderr)
        sys.exit(1)

    screenshots_dir = args.screenshots_dir or os.path.dirname(os.path.abspath(args.input))

    # 确保输出目录存在
    output_dir = os.path.dirname(os.path.abspath(args.output))
    if output_dir:
        os.makedirs(output_dir, exist_ok=True)

    render_report(data, screenshots_dir, args.output)


if __name__ == "__main__":
    main()
