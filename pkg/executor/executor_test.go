package executor

import (
	"api_auto_test/pkg/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Type Conversion", func() {
	var executor *Executor

	BeforeEach(func() {
		executor = &Executor{}
	})

	Describe("convertToSchemaType", func() {
		Context("converting to int", func() {
			It("should keep int as is", func() {
				result := executor.convertToSchemaType(123, "int")
				Expect(result).To(Equal(123))
			})

			It("should convert int64 to int", func() {
				result := executor.convertToSchemaType(int64(456), "int")
				Expect(result).To(Equal(456))
			})

			It("should convert float64 to int", func() {
				result := executor.convertToSchemaType(float64(789), "int")
				Expect(result).To(Equal(789))
			})

			It("should convert string to int", func() {
				result := executor.convertToSchemaType("123", "int")
				Expect(result).To(Equal(123))
			})

			It("should return original value if string cannot be converted", func() {
				result := executor.convertToSchemaType("abc", "int")
				Expect(result).To(Equal("abc"))
			})
		})

		Context("converting to float64", func() {
			It("should keep float64 as is", func() {
				result := executor.convertToSchemaType(19.99, "float64")
				Expect(result).To(Equal(19.99))
			})

			It("should convert int to float64", func() {
				result := executor.convertToSchemaType(123, "float64")
				Expect(result).To(Equal(float64(123)))
			})

			It("should convert int64 to float64", func() {
				result := executor.convertToSchemaType(int64(456), "float64")
				Expect(result).To(Equal(float64(456)))
			})

			It("should convert string to float64", func() {
				result := executor.convertToSchemaType("19.99", "float64")
				Expect(result).To(Equal(19.99))
			})
		})

		Context("converting to string", func() {
			It("should convert int to string", func() {
				result := executor.convertToSchemaType(123, "string")
				Expect(result).To(Equal("123"))
			})

			It("should convert float to string", func() {
				result := executor.convertToSchemaType(19.99, "string")
				Expect(result).To(Equal("19.99"))
			})

			It("should convert bool to string", func() {
				result := executor.convertToSchemaType(true, "string")
				Expect(result).To(Equal("true"))
			})
		})

		Context("converting to bool", func() {
			It("should keep bool as is", func() {
				result := executor.convertToSchemaType(true, "bool")
				Expect(result).To(Equal(true))
			})

			It("should convert string to bool", func() {
				result := executor.convertToSchemaType("true", "bool")
				Expect(result).To(Equal(true))
			})

			It("should convert string 'false' to bool", func() {
				result := executor.convertToSchemaType("false", "bool")
				Expect(result).To(Equal(false))
			})
		})

		Context("with nil value", func() {
			It("should return nil", func() {
				result := executor.convertToSchemaType(nil, "int")
				Expect(result).To(BeNil())
			})
		})

		Context("with array and object types", func() {
			It("should keep array as is", func() {
				arr := []int{1, 2, 3}
				result := executor.convertToSchemaType(arr, "array")
				Expect(result).To(Equal(arr))
			})

			It("should keep object as is", func() {
				obj := map[string]interface{}{"key": "value"}
				result := executor.convertToSchemaType(obj, "object")
				Expect(result).To(Equal(obj))
			})
		})
	})

	Describe("setNestedValue", func() {
		Context("with top-level field", func() {
			It("should convert top-level field type", func() {
				data := map[string]interface{}{
					"id": "123",
				}
				executor.setNestedValue(data, "id", "int")
				Expect(data["id"]).To(Equal(123))
			})
		})

		Context("with nested field", func() {
			It("should convert nested field type", func() {
				data := map[string]interface{}{
					"user": map[string]interface{}{
						"id": float64(456),
					},
				}
				executor.setNestedValue(data, "user.id", "int")
				nested := data["user"].(map[string]interface{})
				Expect(nested["id"]).To(Equal(456))
			})

			It("should convert deeply nested field type", func() {
				data := map[string]interface{}{
					"data": map[string]interface{}{
						"user": map[string]interface{}{
							"age": "25",
						},
					},
				}
				executor.setNestedValue(data, "data.user.age", "int")
				nested1 := data["data"].(map[string]interface{})
				nested2 := nested1["user"].(map[string]interface{})
				Expect(nested2["age"]).To(Equal(25))
			})
		})

		Context("with missing field", func() {
			It("should not panic if field does not exist", func() {
				data := map[string]interface{}{
					"name": "test",
				}
				Expect(func() {
					executor.setNestedValue(data, "id", "int")
				}).NotTo(Panic())
			})

			It("should not panic if nested path is broken", func() {
				data := map[string]interface{}{
					"user": "not a map",
				}
				Expect(func() {
					executor.setNestedValue(data, "user.id", "int")
				}).NotTo(Panic())
			})
		})
	})

	Describe("convertBodyToSchemaTypes", func() {
		Context("with simple fields", func() {
			It("should convert multiple fields", func() {
				body := map[string]interface{}{
					"id":     "123",
					"name":   "test",
					"price":  "19.99",
					"active": "true",
				}
				schema := map[string]string{
					"id":     "int",
					"name":   "string",
					"price":  "float64",
					"active": "bool",
				}

				result := executor.convertBodyToSchemaTypes(body, schema)
				resultMap := result.(map[string]interface{})

				Expect(resultMap["id"]).To(Equal(123))
				Expect(resultMap["name"]).To(Equal("test"))
				Expect(resultMap["price"]).To(Equal(19.99))
				Expect(resultMap["active"]).To(Equal(true))
			})
		})

		Context("with nested fields", func() {
			It("should convert nested fields", func() {
				body := map[string]interface{}{
					"id":   "123",
					"name": "测试部门",
					"extend": map[string]interface{}{
						"source": "iam",
						"count":  float64(10),
					},
				}
				schema := map[string]string{
					"id":            "int",
					"name":          "string",
					"extend":        "object",
					"extend.source": "string",
					"extend.count":  "int",
				}

				result := executor.convertBodyToSchemaTypes(body, schema)
				resultMap := result.(map[string]interface{})

				Expect(resultMap["id"]).To(Equal(123))
				Expect(resultMap["name"]).To(Equal("测试部门"))

				extend := resultMap["extend"].(map[string]interface{})
				Expect(extend["source"]).To(Equal("iam"))
				Expect(extend["count"]).To(Equal(10))
			})
		})

		Context("with non-map body", func() {
			It("should return original body", func() {
				body := "not a map"
				schema := map[string]string{"id": "int"}

				result := executor.convertBodyToSchemaTypes(body, schema)
				Expect(result).To(Equal(body))
			})
		})

		Context("with empty schema", func() {
			It("should return original body", func() {
				body := map[string]interface{}{
					"id": "123",
				}
				schema := map[string]string{}

				result := executor.convertBodyToSchemaTypes(body, schema)
				resultMap := result.(map[string]interface{})
				Expect(resultMap["id"]).To(Equal("123"))
			})
		})
	})

	Describe("Type conversion in variable replacement scenario", func() {
		Context("simulating dependency result", func() {
			It("should convert float64 id from dependency to int", func() {
				// 模拟依赖接口返回的 JSON 数据（id 是 float64）
				body := map[string]interface{}{
					"id":        float64(123), // JSON 解析后的数字类型
					"name":      "更新后的部门名称",
					"parent_id": 1,
				}

				schema := map[string]string{
					"id":        "int", // 期望是 int
					"name":      "string",
					"parent_id": "int",
				}

				result := executor.convertBodyToSchemaTypes(body, schema)
				resultMap := result.(map[string]interface{})

				// 验证 float64 已经被转换为 int
				Expect(resultMap["id"]).To(Equal(123))
				_, isInt := resultMap["id"].(int)
				Expect(isInt).To(BeTrue())
			})

			It("should handle string id from variable substitution", func() {
				// 模拟变量替换后的情况（有时候可能是字符串）
				body := map[string]interface{}{
					"id":        "456",
					"name":      "更新后的名称",
					"parent_id": 1,
				}

				schema := map[string]string{
					"id":        "int",
					"name":      "string",
					"parent_id": "int",
				}

				result := executor.convertBodyToSchemaTypes(body, schema)
				resultMap := result.(map[string]interface{})

				// 验证字符串已经被转换为 int
				Expect(resultMap["id"]).To(Equal(456))
				_, isInt := resultMap["id"].(int)
				Expect(isInt).To(BeTrue())
			})
		})
	})

	Describe("extractFieldValue with array access", func() {
		Context("accessing array elements", func() {
			It("should extract value from simple array by index", func() {
				body := map[string]interface{}{
					"data": []interface{}{
						map[string]interface{}{"id": 1, "name": "Item1"},
						map[string]interface{}{"id": 2, "name": "Item2"},
						map[string]interface{}{"id": 3, "name": "Item3"},
					},
				}

				// 访问 data[0].id
				result := executor.extractFieldValue(body, "data[0].id")
				Expect(result).To(Equal(1))
			})

			It("should extract value from nested array by index", func() {
				body := map[string]interface{}{
					"data": map[string]interface{}{
						"result": []interface{}{
							map[string]interface{}{"id": 10, "name": "Dept1"},
							map[string]interface{}{"id": 20, "name": "Dept2"},
						},
					},
				}

				// 访问 data.result[0].id
				result := executor.extractFieldValue(body, "data.result[0].id")
				Expect(result).To(Equal(10))
			})

			It("should extract value from multi-level array", func() {
				body := map[string]interface{}{
					"data": []interface{}{
						map[string]interface{}{
							"id": 1,
							"children": []interface{}{
								map[string]interface{}{"id": 101, "name": "Child1"},
								map[string]interface{}{"id": 102, "name": "Child2"},
							},
						},
					},
				}

				// 访问 data[0].children[1].name
				result := executor.extractFieldValue(body, "data[0].children[1].name")
				Expect(result).To(Equal("Child2"))
			})

			It("should return nil for out of bounds index", func() {
				body := map[string]interface{}{
					"data": []interface{}{
						map[string]interface{}{"id": 1},
					},
				}

				// 访问不存在的索引
				result := executor.extractFieldValue(body, "data[5].id")
				Expect(result).To(BeNil())
			})

			It("should extract entire array element", func() {
				body := map[string]interface{}{
					"data": []interface{}{
						map[string]interface{}{"id": 1, "name": "Item1"},
						map[string]interface{}{"id": 2, "name": "Item2"},
					},
				}

				// 访问 data[0]（整个对象）
				result := executor.extractFieldValue(body, "data[0]")
				resultMap := result.(map[string]interface{})
				Expect(resultMap["id"]).To(Equal(1))
				Expect(resultMap["name"]).To(Equal("Item1"))
			})
		})
	})

	Describe("parseFieldPath", func() {
		Context("parsing field paths with array indices", func() {
			It("should parse simple array access", func() {
				parts := executor.parseFieldPath("data[0]")
				Expect(parts).To(HaveLen(1))
				Expect(parts[0].name).To(Equal("data"))
				Expect(parts[0].isArray).To(BeTrue())
				Expect(parts[0].index).To(Equal(0))
			})

			It("should parse nested array access", func() {
				parts := executor.parseFieldPath("data.result[0].id")
				Expect(parts).To(HaveLen(3))

				Expect(parts[0].name).To(Equal("data"))
				Expect(parts[0].isArray).To(BeFalse())

				Expect(parts[1].name).To(Equal("result"))
				Expect(parts[1].isArray).To(BeTrue())
				Expect(parts[1].index).To(Equal(0))

				Expect(parts[2].name).To(Equal("id"))
				Expect(parts[2].isArray).To(BeFalse())
			})

			It("should parse multi-level array access", func() {
				parts := executor.parseFieldPath("data[0].children[1].name")
				Expect(parts).To(HaveLen(3))

				Expect(parts[0].name).To(Equal("data"))
				Expect(parts[0].isArray).To(BeTrue())
				Expect(parts[0].index).To(Equal(0))

				Expect(parts[1].name).To(Equal("children"))
				Expect(parts[1].isArray).To(BeTrue())
				Expect(parts[1].index).To(Equal(1))

				Expect(parts[2].name).To(Equal("name"))
				Expect(parts[2].isArray).To(BeFalse())
			})

			It("should parse simple field path without array", func() {
				parts := executor.parseFieldPath("data.user.id")
				Expect(parts).To(HaveLen(3))
				Expect(parts[0].name).To(Equal("data"))
				Expect(parts[0].isArray).To(BeFalse())
				Expect(parts[1].name).To(Equal("user"))
				Expect(parts[1].isArray).To(BeFalse())
				Expect(parts[2].name).To(Equal("id"))
				Expect(parts[2].isArray).To(BeFalse())
			})
		})
	})
})

var _ = Describe("Dependency Tracking", func() {
	var executor *Executor

	BeforeEach(func() {
		executor = &Executor{
			results: make(map[string]*TestResult),
		}
	})

	Describe("findRootCause", func() {
		Context("with direct failure", func() {
			It("should return the failed test itself", func() {
				// 设置一个失败的测试结果
				executor.storeResult(&TestResult{
					Name:    "test1",
					Passed:  false,
					Skipped: false,
				})

				rootCause := executor.findRootCause("test1")
				Expect(rootCause).To(Equal("test1"))
			})
		})

		Context("with dependency chain", func() {
			BeforeEach(func() {
				// 模拟配置
				executor.config = &config.TestConfig{
					APIs: []config.APITest{
						{Name: "test1", DependsOn: ""},
						{Name: "test2", DependsOn: "test1"},
						{Name: "test3", DependsOn: "test2"},
					},
				}
			})

			It("should find root cause in dependency chain", func() {
				// test1 失败
				executor.storeResult(&TestResult{
					Name:    "test1",
					Passed:  false,
					Skipped: false,
				})

				// test2 因为 test1 失败而被跳过
				executor.storeResult(&TestResult{
					Name:       "test2",
					Passed:     false,
					Skipped:    true,
					SkipReason: "依赖接口 'test1' 执行失败",
				})

				// test3 因为 test2 被跳过而被跳过
				executor.storeResult(&TestResult{
					Name:       "test3",
					Passed:     false,
					Skipped:    true,
					SkipReason: "依赖接口 'test2' 被跳过",
				})

				// 从 test3 查找根本原因，应该找到 test1
				rootCause := executor.findRootCause("test3")
				Expect(rootCause).To(Equal("test1"))

				// 从 test2 查找根本原因，应该找到 test1
				rootCause = executor.findRootCause("test2")
				Expect(rootCause).To(Equal("test1"))
			})
		})

		Context("with missing result", func() {
			It("should return the test name itself", func() {
				rootCause := executor.findRootCause("non-existent")
				Expect(rootCause).To(Equal("non-existent"))
			})
		})

		Context("with circular dependency", func() {
			BeforeEach(func() {
				// 模拟循环依赖配置
				executor.config = &config.TestConfig{
					APIs: []config.APITest{
						{Name: "test1", DependsOn: "test2"},
						{Name: "test2", DependsOn: "test1"},
					},
				}
			})

			It("should handle circular dependency gracefully", func() {
				executor.storeResult(&TestResult{
					Name:    "test1",
					Passed:  false,
					Skipped: true,
				})
				executor.storeResult(&TestResult{
					Name:    "test2",
					Passed:  false,
					Skipped: true,
				})

				rootCause := executor.findRootCause("test1")
				// 应该返回检测到循环依赖的节点
				Expect(rootCause).To(Or(Equal("test1"), Equal("test2")))
			})
		})
	})

	Describe("findDependsOn", func() {
		BeforeEach(func() {
			executor.config = &config.TestConfig{
				APIs: []config.APITest{
					{Name: "test1", DependsOn: ""},
					{Name: "test2", DependsOn: "test1"},
					{Name: "test3", DependsOn: "test2"},
				},
			}
		})

		It("should find dependency for existing test", func() {
			dep := executor.findDependsOn("test2")
			Expect(dep).To(Equal("test1"))

			dep = executor.findDependsOn("test3")
			Expect(dep).To(Equal("test2"))
		})

		It("should return empty string for test with no dependency", func() {
			dep := executor.findDependsOn("test1")
			Expect(dep).To(Equal(""))
		})

		It("should return empty string for non-existent test", func() {
			dep := executor.findDependsOn("non-existent")
			Expect(dep).To(Equal(""))
		})
	})

	Describe("getDependencyFailureReason", func() {
		It("should return '被跳过' for skipped result", func() {
			result := &TestResult{
				Passed:  false,
				Skipped: true,
			}
			reason := executor.getDependencyFailureReason(result)
			Expect(reason).To(Equal("被跳过"))
		})

		It("should return '执行失败' for failed result", func() {
			result := &TestResult{
				Passed:  false,
				Skipped: false,
			}
			reason := executor.getDependencyFailureReason(result)
			Expect(reason).To(Equal("执行失败"))
		})
	})
})
