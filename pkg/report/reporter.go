package report

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"api_auto_test/pkg/executor"
)

// Reporter 报告生成器
type Reporter struct {
	report *executor.TestReport
}

// NewReporter 创建报告生成器
func NewReporter(report *executor.TestReport) *Reporter {
	return &Reporter{
		report: report,
	}
}

// PrintConsole 打印控制台报告
func (r *Reporter) PrintConsole() {
	// 生成动态标题
	title := "API Test Report"
	if r.report.ConfigFileName != "" {
		title = r.report.ConfigFileName + " " + title
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("  %s\n", title)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("  Base URL:     %s\n", r.report.BaseURL)
	fmt.Printf("  Version:      %s\n", r.report.Version)
	fmt.Printf("  Start Time:   %s\n", r.report.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Duration:     %s\n", r.report.Duration)
	fmt.Printf("  Total Tests:  %d\n", r.report.TotalTests)
	fmt.Printf("  Passed:       %s%d%s\n", colorGreen, r.report.PassedTests, colorReset)
	fmt.Printf("  Failed:       %s%d%s\n", colorRed, r.report.FailedTests, colorReset)
	fmt.Printf("  Skipped:      %s%d%s\n", colorYellow, r.report.SkippedTests, colorReset)
	fmt.Printf("  Success Rate: %.2f%%\n", r.getSuccessRate())
	fmt.Println(strings.Repeat("=", 80))

	for i, result := range r.report.Results {
		r.printTestResult(i+1, result)
	}

	fmt.Println(strings.Repeat("=", 80))
	if r.report.PassedTests == r.report.TotalTests {
		fmt.Printf("%s  All tests passed! ✓%s\n", colorGreen, colorReset)
	} else {
		fmt.Printf("%s  Some tests failed! ✗%s\n", colorRed, colorReset)
	}
	fmt.Println(strings.Repeat("=", 80) + "\n")
}

// printTestResult 打印单个测试结果
func (r *Reporter) printTestResult(index int, result executor.TestResult) {
	status := colorGreen + "✓ PASS" + colorReset
	if result.Skipped {
		status = colorYellow + "⊘ SKIP" + colorReset
	} else if !result.Passed {
		status = colorRed + "✗ FAIL" + colorReset
	}

	fmt.Printf("\n%s [%d/%d] %s\n", status, index, r.report.TotalTests, result.Name)
	if result.Description != "" {
		fmt.Printf("    Description: %s\n", result.Description)
	}
	fmt.Printf("    Method:      %s %s\n", result.Request.Method, result.Request.Path)

	// 如果是跳过状态，显示跳过原因
	if result.Skipped {
		fmt.Printf("    %sReason: %s%s\n", colorYellow, result.SkipReason, colorReset)
	} else {
		fmt.Printf("    Status:      %d\n", result.StatusCode)
		fmt.Printf("    Duration:    %s\n", result.Duration)
		if result.RetryCount > 0 {
			fmt.Printf("    Retries:     %d\n", result.RetryCount)
		}

		if result.Error != nil {
			fmt.Printf("    %sError: %s%s\n", colorRed, result.Error.Error(), colorReset)
		}

		if result.Validation != nil && !result.Validation.Passed {
			fmt.Printf("    %sValidation Errors:%s\n", colorYellow, colorReset)
			for _, err := range result.Validation.Errors {
				fmt.Printf("      - %s: %s\n", err.Field, err.Message)
				if err.Expected != nil && err.Actual != nil {
					fmt.Printf("        Expected: %v\n", err.Expected)
					fmt.Printf("        Actual:   %v\n", err.Actual)
				}
			}
		}
	}
}

// SaveJSON 保存为JSON格式
func (r *Reporter) SaveJSON(filename string) error {
	data, err := json.MarshalIndent(r.report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

// SaveHTML 保存为HTML格式
func (r *Reporter) SaveHTML(filename string) error {
	html := r.generateHTML()
	if err := os.WriteFile(filename, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}
	return nil
}

// generateHTML 生成HTML报告
func (r *Reporter) generateHTML() string {
	var sb strings.Builder

	// 生成动态标题
	pageTitle := "API Test Report"
	if r.report.ConfigFileName != "" {
		pageTitle = r.report.ConfigFileName + " " + pageTitle
	}

	sb.WriteString(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + pageTitle + `</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: Arial, sans-serif; background: #f5f5f5; }
        html { scroll-behavior: smooth; }

        /* 布局容器 */
        .layout { display: flex; min-height: 100vh; }

        /* 左侧导航栏 */
        .sidebar {
            width: 300px;
            background: #2c3e50;
            color: white;
            position: fixed;
            height: 100vh;
            overflow-y: auto;
            left: 0;
            top: 0;
            box-shadow: 2px 0 5px rgba(0,0,0,0.1);
        }
        .sidebar-header {
            padding: 20px;
            background: #34495e;
            border-bottom: 2px solid #4CAF50;
        }
        .sidebar-header h2 {
            font-size: 18px;
            margin-bottom: 10px;
        }
        .sidebar-stats {
            font-size: 12px;
            color: #ecf0f1;
        }
        .nav-list {
            list-style: none;
            padding: 10px 0;
        }
        .nav-item {
            border-bottom: 1px solid #34495e;
        }
        .nav-link {
            display: flex;
            align-items: center;
            padding: 12px 20px;
            color: #ecf0f1;
            text-decoration: none;
            transition: background 0.2s;
            font-size: 13px;
        }
        .nav-link:hover {
            background: #34495e;
        }
        .nav-link.active {
            background: #34495e;
            border-left: 4px solid #4CAF50;
        }
        .nav-status {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            margin-right: 10px;
            flex-shrink: 0;
        }
        .nav-status.pass { background: #4CAF50; }
        .nav-status.fail { background: #f44336; }
        .nav-status.skip { background: #FF9800; }
        .nav-number {
            color: #95a5a6;
            margin-right: 8px;
            font-size: 11px;
            min-width: 25px;
        }
        .nav-text {
            flex: 1;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }

        /* 主内容区 */
        .main-content {
            margin-left: 300px;
            flex: 1;
            padding: 20px;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }

        h1 {
            color: #333;
            border-bottom: 3px solid #4CAF50;
            padding-bottom: 15px;
            margin-bottom: 25px;
        }
        h4 { color: #555; margin: 15px 0 8px 0; font-size: 14px; }

        /* 摘要信息 */
        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
            gap: 15px;
            margin: 25px 0;
        }
        .summary-item {
            background: #f9f9f9;
            padding: 15px;
            border-radius: 5px;
            border-left: 4px solid #4CAF50;
        }
        .summary-item h3 {
            margin: 0 0 10px 0;
            color: #666;
            font-size: 13px;
        }
        .summary-item .value {
            font-size: 22px;
            font-weight: bold;
            color: #333;
        }

        /* 测试结果 */
        .test-result {
            margin: 25px 0;
            padding: 20px;
            border-radius: 5px;
            border-left: 4px solid #4CAF50;
            background: #f9f9f9;
            scroll-margin-top: 20px;
        }
        .test-result.failed { border-left-color: #f44336; }
        .test-result.skipped { border-left-color: #FF9800; }
        .test-result h3 {
            margin: 0 0 10px 0;
            color: #333;
            font-size: 18px;
        }
        .test-result .status {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 3px;
            font-size: 12px;
            font-weight: bold;
            color: white;
        }
        .test-result .status.pass { background: #4CAF50; }
        .test-result .status.fail { background: #f44336; }
        .test-result .status.skip { background: #FF9800; }
        .test-details {
            margin: 15px 0;
            font-size: 14px;
            color: #666;
        }
        .test-details dt {
            font-weight: bold;
            margin-top: 8px;
        }
        .test-details dd {
            margin: 0 0 5px 20px;
        }
        .error {
            background: #fff3cd;
            padding: 12px;
            border-radius: 3px;
            margin: 10px 0;
            color: #856404;
            border: 1px solid #ffeeba;
        }
        .success-rate { font-size: 20px; font-weight: bold; }
        .success-rate.high { color: #4CAF50; }
        .success-rate.low { color: #f44336; }
        .code-block {
            background: #282c34;
            color: #abb2bf;
            padding: 12px;
            border-radius: 4px;
            overflow-x: auto;
            font-family: 'Courier New', monospace;
            font-size: 13px;
            line-height: 1.5;
            margin: 8px 0;
        }
        .section {
            background: white;
            padding: 10px;
            border-radius: 4px;
            margin: 10px 0;
            border: 1px solid #e0e0e0;
        }
        .toggle-btn {
            background: #2196F3;
            color: white;
            border: none;
            padding: 6px 14px;
            border-radius: 3px;
            cursor: pointer;
            font-size: 12px;
            margin-top: 5px;
        }
        .toggle-btn:hover { background: #1976D2; }
        .collapsible { display: none; }
        .collapsible.show { display: block; }

        /* 滚动条样式 */
        .sidebar::-webkit-scrollbar { width: 8px; }
        .sidebar::-webkit-scrollbar-track { background: #34495e; }
        .sidebar::-webkit-scrollbar-thumb {
            background: #4CAF50;
            border-radius: 4px;
        }
        .sidebar::-webkit-scrollbar-thumb:hover { background: #45a049; }
    </style>
    <script>
        function toggleSection(id) {
            var section = document.getElementById(id);
            if (section.classList.contains('show')) {
                section.classList.remove('show');
            } else {
                section.classList.add('show');
            }
        }

        // 高亮当前激活的导航项
        document.addEventListener('DOMContentLoaded', function() {
            const navLinks = document.querySelectorAll('.nav-link');
            const testResults = document.querySelectorAll('.test-result');

            // 点击导航项时高亮
            navLinks.forEach(link => {
                link.addEventListener('click', function() {
                    navLinks.forEach(l => l.classList.remove('active'));
                    this.classList.add('active');
                });
            });

            // 滚动时自动高亮对应的导航项
            window.addEventListener('scroll', function() {
                let current = '';
                testResults.forEach(result => {
                    const rect = result.getBoundingClientRect();
                    if (rect.top <= 100) {
                        current = result.id;
                    }
                });

                navLinks.forEach(link => {
                    link.classList.remove('active');
                    if (link.getAttribute('href') === '#' + current) {
                        link.classList.add('active');
                    }
                });
            });
        });
    </script>
</head>
<body>
    <div class="layout">
        <!-- 左侧导航栏 -->
        <nav class="sidebar">
            <div class="sidebar-header">
                <h2>🧪 ` + pageTitle + `</h2>
                <div class="sidebar-stats">
                    <div>✓ 通过: ` + fmt.Sprintf("%d", r.report.PassedTests) + `</div>
                    <div>✗ 失败: ` + fmt.Sprintf("%d", r.report.FailedTests) + `</div>
                    <div>⊘ 跳过: ` + fmt.Sprintf("%d", r.report.SkippedTests) + `</div>
                    <div>⏱ 耗时: ` + r.report.Duration.String() + `</div>
                </div>
            </div>
            <ul class="nav-list">`)

	// 生成导航列表
	for i, result := range r.report.Results {
		statusClass := "pass"
		if result.Skipped {
			statusClass = "skip"
		} else if !result.Passed {
			statusClass = "fail"
		}
		testID := fmt.Sprintf("test-%d", i)
		sb.WriteString(fmt.Sprintf(`
                <li class="nav-item">
                    <a href="#%s" class="nav-link">
                        <span class="nav-number">#%d</span>
                        <span class="nav-status %s"></span>
                        <span class="nav-text" title="%s">%s</span>
                    </a>
                </li>`,
			testID, i+1, statusClass, result.Name, result.Name))
	}

	sb.WriteString(`
            </ul>
        </nav>

        <!-- 主内容区 -->
        <div class="main-content">
            <div class="container">
                <h1>` + pageTitle + `</h1>
                <div class="summary">
                    <div class="summary-item">
                        <h3>Base URL</h3>
                        <div class="value" style="font-size: 15px;">` + r.report.BaseURL + `</div>
                    </div>
                    <div class="summary-item">
                        <h3>Version</h3>
                        <div class="value">` + r.report.Version + `</div>
                    </div>
                    <div class="summary-item">
                        <h3>Total Tests</h3>
                        <div class="value">` + fmt.Sprintf("%d", r.report.TotalTests) + `</div>
                    </div>
                    <div class="summary-item">
                        <h3>Passed</h3>
                        <div class="value" style="color: #4CAF50;">` + fmt.Sprintf("%d", r.report.PassedTests) + `</div>
                    </div>
                    <div class="summary-item">
                        <h3>Failed</h3>
                        <div class="value" style="color: #f44336;">` + fmt.Sprintf("%d", r.report.FailedTests) + `</div>
                    </div>
                    <div class="summary-item">
                        <h3>Skipped</h3>
                        <div class="value" style="color: #FF9800;">` + fmt.Sprintf("%d", r.report.SkippedTests) + `</div>
                    </div>
                    <div class="summary-item">
                        <h3>Success Rate</h3>
                        <div class="value success-rate ` + r.getSuccessRateClass() + `">` + fmt.Sprintf("%.1f%%", r.getSuccessRate()) + `</div>
                    </div>
                    <div class="summary-item">
                        <h3>Duration</h3>
                        <div class="value">` + r.report.Duration.String() + `</div>
                    </div>
                    <div class="summary-item">
                        <h3>Start Time</h3>
                        <div class="value" style="font-size: 13px;">` + r.report.StartTime.Format("2006-01-02 15:04:05") + `</div>
                    </div>
                </div>
                <h2 style="margin-top: 30px; color: #333;">测试结果详情</h2>`)

	for i, result := range r.report.Results {
		statusClass := "pass"
		statusText := "PASS"
		resultClass := ""
		if result.Skipped {
			statusClass = "skip"
			statusText = "SKIP"
			resultClass = "skipped"
		} else if !result.Passed {
			statusClass = "fail"
			statusText = "FAIL"
			resultClass = "failed"
		}

		testID := fmt.Sprintf("test-%d", i)
		sb.WriteString(fmt.Sprintf(`
        <div id="%s" class="test-result %s">
            <h3>[%d/%d] %s <span class="status %s">%s</span></h3>`,
			testID, resultClass, i+1, r.report.TotalTests, result.Name, statusClass, statusText))

		if result.Description != "" {
			sb.WriteString(fmt.Sprintf(`<p>%s</p>`, result.Description))
		}

		sb.WriteString(`<dl class="test-details">`)
		sb.WriteString(fmt.Sprintf(`<dt>Request:</dt><dd>%s %s</dd>`, result.Request.Method, result.Request.Path))

		// 如果是跳过状态，显示跳过原因
		if result.Skipped {
			sb.WriteString(fmt.Sprintf(`<dt>Skip Reason:</dt><dd style="color: #FF9800; font-weight: bold;">%s</dd>`, result.SkipReason))
		} else {
			sb.WriteString(fmt.Sprintf(`<dt>Status Code:</dt><dd>%d</dd>`, result.StatusCode))
			sb.WriteString(fmt.Sprintf(`<dt>Duration:</dt><dd>%s</dd>`, result.Duration))
			if result.RetryCount > 0 {
				sb.WriteString(fmt.Sprintf(`<dt>Retries:</dt><dd>%d</dd>`, result.RetryCount))
			}
		}
		sb.WriteString(`</dl>`)

		if result.Error != nil {
			sb.WriteString(fmt.Sprintf(`<div class="error">Error: %s</div>`, result.Error.Error()))
		}

		if result.Validation != nil && !result.Validation.Passed {
			sb.WriteString(`<div class="error"><strong>Validation Errors:</strong><ul>`)
			for _, err := range result.Validation.Errors {
				sb.WriteString(fmt.Sprintf(`<li>%s: %s</li>`, err.Field, err.Message))
			}
			sb.WriteString(`</ul></div>`)
		}

		// 只有在接口实际执行的情况下才显示请求和响应详情
		if !result.Skipped {
			// 添加请求详情
			sb.WriteString(fmt.Sprintf(`
            <div class="section">
                <h4>📤 Request Details</h4>
                <button class="toggle-btn" onclick="toggleSection('req-%d')">Show/Hide</button>
                <div id="req-%d" class="collapsible">`, i, i))

			// 请求Headers
			if len(result.Request.Headers) > 0 {
				sb.WriteString(`<h4>Headers:</h4><pre class="code-block">`)
				headersJSON, _ := json.MarshalIndent(result.Request.Headers, "", "  ")
				sb.WriteString(r.escapeHTML(string(headersJSON)))
				sb.WriteString(`</pre>`)
			}

			// 请求Body
			if result.Request.Body != nil {
				sb.WriteString(`<h4>Body:</h4><pre class="code-block">`)
				bodyJSON, _ := json.MarshalIndent(result.Request.Body, "", "  ")
				sb.WriteString(r.escapeHTML(string(bodyJSON)))
				sb.WriteString(`</pre>`)
			}

			// 请求Query参数
			if len(result.Request.Query) > 0 {
				sb.WriteString(`<h4>Query Parameters:</h4><pre class="code-block">`)
				queryJSON, _ := json.MarshalIndent(result.Request.Query, "", "  ")
				sb.WriteString(r.escapeHTML(string(queryJSON)))
				sb.WriteString(`</pre>`)
			}

			sb.WriteString(`</div></div>`)

			// 添加响应详情 - 默认展开
			if result.Response != nil {
				sb.WriteString(fmt.Sprintf(`
            <div class="section">
                <h4>📥 Response Details</h4>
                <button class="toggle-btn" onclick="toggleSection('resp-%d')">Show/Hide</button>
                <div id="resp-%d" class="collapsible show">`, i, i))

				// 响应Headers
				if len(result.Response.Headers) > 0 {
					sb.WriteString(`<h4>Headers:</h4><pre class="code-block">`)
					headerMap := make(map[string]string)
					for key, values := range result.Response.Headers {
						headerMap[key] = strings.Join(values, ", ")
					}
					headersJSON, _ := json.MarshalIndent(headerMap, "", "  ")
					sb.WriteString(r.escapeHTML(string(headersJSON)))
					sb.WriteString(`</pre>`)
				}

				// 响应Body - 默认展开
				sb.WriteString(`<h4>Body:</h4><pre class="code-block">`)
				if result.Response.BodyJSON != nil {
					// 如果是JSON，格式化输出
					bodyJSON, _ := json.MarshalIndent(result.Response.BodyJSON, "", "  ")
					sb.WriteString(r.escapeHTML(string(bodyJSON)))
				} else if len(result.Response.Body) > 0 {
					// 如果不是JSON，直接输出
					sb.WriteString(r.escapeHTML(string(result.Response.Body)))
				} else {
					sb.WriteString("(empty)")
				}
				sb.WriteString(`</pre>`)

				sb.WriteString(`</div></div>`)
			}
		}

		sb.WriteString(`</div>`)
	}

	sb.WriteString(`
            </div>
        </div>
    </div>
</body>
</html>`)

	return sb.String()
}

// escapeHTML 转义HTML特殊字符
func (r *Reporter) escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// getSuccessRate 计算成功率
func (r *Reporter) getSuccessRate() float64 {
	if r.report.TotalTests == 0 {
		return 0
	}
	return float64(r.report.PassedTests) / float64(r.report.TotalTests) * 100
}

// getSuccessRateClass 获取成功率CSS类
func (r *Reporter) getSuccessRateClass() string {
	rate := r.getSuccessRate()
	if rate >= 80 {
		return "high"
	}
	return "low"
}

// ANSI颜色代码
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)
