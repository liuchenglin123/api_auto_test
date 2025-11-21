package config_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"api_auto_test/pkg/config"
)

var _ = Describe("Loader", func() {
	var (
		loader     *config.Loader
		tmpDir     string
		configFile string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "config-test")
		Expect(err).NotTo(HaveOccurred())

		configFile = filepath.Join(tmpDir, "test-config.yaml")
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Describe("Load", func() {
		Context("当配置文件有效时", func() {
			BeforeEach(func() {
				configContent := `
base_url: https://api.example.com
version: v1
timeout: 30s
headers:
  Content-Type: application/json
apis:
  - name: test-api
    description: Test API
    request:
      method: GET
      path: /test
    response:
      status_code: 200
`
				err := os.WriteFile(configFile, []byte(configContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				loader = config.NewLoader(configFile)
			})

			It("应该成功加载配置", func() {
				cfg, err := loader.Load()
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.BaseURL).To(Equal("https://api.example.com"))
				Expect(cfg.Version).To(Equal("v1"))
				Expect(cfg.APIs).To(HaveLen(1))
				Expect(cfg.APIs[0].Name).To(Equal("test-api"))
			})
		})

		Context("当配置文件不存在时", func() {
			BeforeEach(func() {
				loader = config.NewLoader(filepath.Join(tmpDir, "non-existent.yaml"))
			})

			It("应该返回错误", func() {
				_, err := loader.Load()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("当配置文件格式无效时", func() {
			BeforeEach(func() {
				invalidContent := `
invalid: yaml: content:
  - broken
    - structure
`
				err := os.WriteFile(configFile, []byte(invalidContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				loader = config.NewLoader(configFile)
			})

			It("应该返回错误", func() {
				_, err := loader.Load()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("LoadWithVersion", func() {
		BeforeEach(func() {
			configContent := `
base_url: https://api.example.com
version: v1
apis:
  - name: api-v1
    version: v1
    request:
      method: GET
      path: /v1/test
    response:
      status_code: 200
  - name: api-v2
    version: v2
    request:
      method: GET
      path: /v2/test
    response:
      status_code: 200
  - name: api-all
    request:
      method: GET
      path: /test
    response:
      status_code: 200
`
			err := os.WriteFile(configFile, []byte(configContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			loader = config.NewLoader(configFile)
		})

		Context("当指定v1版本时", func() {
			It("应该只加载v1和通用API", func() {
				cfg, err := loader.LoadWithVersion("v1")
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.Version).To(Equal("v1"))
				Expect(cfg.APIs).To(HaveLen(2))

				names := []string{}
				for _, api := range cfg.APIs {
					names = append(names, api.Name)
				}
				Expect(names).To(ConsistOf("api-v1", "api-all"))
			})
		})

		Context("当指定v2版本时", func() {
			It("应该只加载v2和通用API", func() {
				cfg, err := loader.LoadWithVersion("v2")
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.Version).To(Equal("v2"))
				Expect(cfg.APIs).To(HaveLen(2))

				names := []string{}
				for _, api := range cfg.APIs {
					names = append(names, api.Name)
				}
				Expect(names).To(ConsistOf("api-v2", "api-all"))
			})
		})
	})

	Describe("MergeConfig", func() {
		var baseConfig *config.TestConfig

		BeforeEach(func() {
			baseConfig = &config.TestConfig{
				BaseURL: "https://api.example.com",
				Version: "v1",
				Certificate: config.CertConfig{
					CertFile: "old-cert.crt",
					KeyFile:  "old-key.key",
				},
			}
		})

		Context("当提供新的baseURL时", func() {
			It("应该覆盖原有值", func() {
				merged := config.MergeConfig(baseConfig, "https://new-api.com", "", "", "", "")
				Expect(merged.BaseURL).To(Equal("https://new-api.com"))
			})
		})

		Context("当提供新的版本时", func() {
			It("应该覆盖原有版本", func() {
				merged := config.MergeConfig(baseConfig, "", "", "", "", "v2")
				Expect(merged.Version).To(Equal("v2"))
			})
		})

		Context("当提供新的证书路径时", func() {
			It("应该覆盖原有证书配置", func() {
				merged := config.MergeConfig(baseConfig, "", "new-cert.crt", "new-key.key", "new-ca.crt", "")
				Expect(merged.Certificate.CertFile).To(Equal("new-cert.crt"))
				Expect(merged.Certificate.KeyFile).To(Equal("new-key.key"))
				Expect(merged.Certificate.CAFile).To(Equal("new-ca.crt"))
			})
		})

		Context("当参数为空时", func() {
			It("应该保持原有值不变", func() {
				merged := config.MergeConfig(baseConfig, "", "", "", "", "")
				Expect(merged.BaseURL).To(Equal("https://api.example.com"))
				Expect(merged.Version).To(Equal("v1"))
			})
		})
	})
})
