package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"api_auto_test/pkg/config"
	"api_auto_test/pkg/executor"
	"api_auto_test/pkg/report"
)

var (
	configFile   = flag.String("config", "testdata/api_tests.yaml", "配置文件路径")
	baseURL      = flag.String("url", "", "基础URL（覆盖配置文件）")
	version      = flag.String("version", "", "API版本（覆盖配置文件）")
	certFile     = flag.String("cert", "", "客户端证书文件路径")
	keyFile      = flag.String("key", "", "客户端密钥文件路径")
	caFile       = flag.String("ca", "", "CA证书文件路径")
	outputFormat = flag.String("format", "console", "输出格式: console, json, html")
	outputFile   = flag.String("output", "", "输出文件路径（用于json和html格式）")
	concurrent   = flag.Bool("concurrent", false, "是否并发执行测试")
	maxWorkers   = flag.Int("workers", 5, "并发执行时的最大工作线程数")
	testName     = flag.String("test", "", "只运行指定名称的测试")
	listTests    = flag.Bool("list", false, "列出所有测试名称")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// 加载配置
	loader := config.NewLoader(*configFile)
	cfg, err := loader.LoadWithVersion(*version)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 合并命令行参数
	cfg = config.MergeConfig(cfg, *baseURL, *certFile, *keyFile, *caFile, *version)

	// 创建执行器
	exec, err := executor.NewExecutor(cfg)
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// 列出所有测试
	if *listTests {
		fmt.Println("Available tests:")
		for i, name := range exec.GetTestNames() {
			fmt.Printf("  %d. %s\n", i+1, name)
		}
		return nil
	}

	// 执行单个测试
	if *testName != "" {
		result, err := exec.ExecuteByName(*testName)
		if err != nil {
			return fmt.Errorf("failed to execute test: %w", err)
		}

		// 创建单测试报告
		testReport := &executor.TestReport{
			TotalTests:     1,
			Results:        []executor.TestResult{*result},
			StartTime:      result.ExecutedAt,
			EndTime:        result.ExecutedAt.Add(result.Duration),
			Duration:       result.Duration,
			Version:        cfg.Version,
			BaseURL:        cfg.BaseURL,
			ConfigFileName: getConfigFileName(*configFile),
		}
		if result.Passed {
			testReport.PassedTests = 1
		} else {
			testReport.FailedTests = 1
		}

		return generateReport(testReport)
	}

	// 执行所有测试
	var testReport *executor.TestReport
	if *concurrent {
		fmt.Printf("Running %d tests concurrently (max workers: %d)...\n", len(cfg.APIs), *maxWorkers)
		testReport = exec.ExecuteConcurrent(*maxWorkers)
	} else {
		fmt.Printf("Running %d tests sequentially...\n", len(cfg.APIs))
		testReport = exec.Execute()
	}

	// 设置配置文件名称
	testReport.ConfigFileName = getConfigFileName(*configFile)

	// 生成报告
	if err := generateReport(testReport); err != nil {
		return err
	}

	// 如果有失败的测试，返回错误退出码
	if testReport.FailedTests > 0 {
		os.Exit(1)
	}

	return nil
}

func generateReport(testReport *executor.TestReport) error {
	reporter := report.NewReporter(testReport)

	switch *outputFormat {
	case "console":
		reporter.PrintConsole()
	case "json":
		filename := *outputFile
		if filename == "" {
			filename = "test-report.json"
		}
		if err := reporter.SaveJSON(filename); err != nil {
			return fmt.Errorf("failed to save JSON report: %w", err)
		}
		fmt.Printf("JSON report saved to: %s\n", filename)
	case "html":
		filename := *outputFile
		if filename == "" {
			filename = "test-report.html"
		}
		if err := reporter.SaveHTML(filename); err != nil {
			return fmt.Errorf("failed to save HTML report: %w", err)
		}
		fmt.Printf("HTML report saved to: %s\n", filename)
	default:
		return fmt.Errorf("unknown output format: %s", *outputFormat)
	}

	return nil
}

// getConfigFileName 从配置文件路径中提取文件名（不含扩展名）
func getConfigFileName(configPath string) string {
	// 获取文件名（不含路径）
	fileName := filepath.Base(configPath)
	// 去掉扩展名
	ext := filepath.Ext(fileName)
	if ext != "" {
		fileName = fileName[:len(fileName)-len(ext)]
	}
	return fileName
}
