package client

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Body Schema Validation", func() {
	Describe("validateBodySchema", func() {
		Context("with valid types", func() {
			It("should pass for string type", func() {
				body := map[string]interface{}{
					"name": "test",
				}
				schema := map[string]string{
					"name": "string",
				}
				err := validateBodySchema(body, schema)
				Expect(err).To(BeNil())
			})

			It("should pass for int type (as float64)", func() {
				body := map[string]interface{}{
					"id": float64(123), // JSON 解析后数字是 float64
				}
				schema := map[string]string{
					"id": "int",
				}
				err := validateBodySchema(body, schema)
				Expect(err).To(BeNil())
			})

			It("should pass for int type (actual int)", func() {
				body := map[string]interface{}{
					"id": 123,
				}
				schema := map[string]string{
					"id": "int",
				}
				err := validateBodySchema(body, schema)
				Expect(err).To(BeNil())
			})

			It("should pass for bool type", func() {
				body := map[string]interface{}{
					"enabled": true,
				}
				schema := map[string]string{
					"enabled": "bool",
				}
				err := validateBodySchema(body, schema)
				Expect(err).To(BeNil())
			})

			It("should pass for float type", func() {
				body := map[string]interface{}{
					"price": 19.99,
				}
				schema := map[string]string{
					"price": "float",
				}
				err := validateBodySchema(body, schema)
				Expect(err).To(BeNil())
			})

			It("should pass for array type", func() {
				body := map[string]interface{}{
					"ids": []int{1, 2, 3},
				}
				schema := map[string]string{
					"ids": "array",
				}
				err := validateBodySchema(body, schema)
				Expect(err).To(BeNil())
			})

			It("should pass for object type", func() {
				body := map[string]interface{}{
					"config": map[string]interface{}{
						"key": "value",
					},
				}
				schema := map[string]string{
					"config": "object",
				}
				err := validateBodySchema(body, schema)
				Expect(err).To(BeNil())
			})

			It("should pass for nested fields", func() {
				body := map[string]interface{}{
					"extend": map[string]interface{}{
						"source": "iam",
						"data":   map[string]interface{}{},
					},
				}
				schema := map[string]string{
					"extend":        "object",
					"extend.source": "string",
					"extend.data":   "object",
				}
				err := validateBodySchema(body, schema)
				Expect(err).To(BeNil())
			})

			It("should pass for multiple fields", func() {
				body := map[string]interface{}{
					"name":      "test",
					"id":        float64(123),
					"enabled":   true,
					"tags":      []string{"a", "b"},
					"parent_id": float64(1),
				}
				schema := map[string]string{
					"name":      "string",
					"id":        "int",
					"enabled":   "bool",
					"tags":      "array",
					"parent_id": "int",
				}
				err := validateBodySchema(body, schema)
				Expect(err).To(BeNil())
			})
		})

		Context("with invalid types", func() {
			It("should fail when string expected but got int", func() {
				body := map[string]interface{}{
					"name": 123,
				}
				schema := map[string]string{
					"name": "string",
				}
				err := validateBodySchema(body, schema)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("expected 'string'"))
			})

			It("should fail when int expected but got string", func() {
				body := map[string]interface{}{
					"id": "abc",
				}
				schema := map[string]string{
					"id": "int",
				}
				err := validateBodySchema(body, schema)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("expected 'int'"))
			})

			It("should fail when int expected but got float", func() {
				body := map[string]interface{}{
					"id": float64(12.5), // 非整数的浮点数
				}
				schema := map[string]string{
					"id": "int",
				}
				err := validateBodySchema(body, schema)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("expected 'int'"))
			})

			It("should fail when bool expected but got string", func() {
				body := map[string]interface{}{
					"enabled": "true",
				}
				schema := map[string]string{
					"enabled": "bool",
				}
				err := validateBodySchema(body, schema)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("expected 'bool'"))
			})

			It("should fail when array expected but got object", func() {
				body := map[string]interface{}{
					"data": map[string]interface{}{},
				}
				schema := map[string]string{
					"data": "array",
				}
				err := validateBodySchema(body, schema)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("expected 'array/slice'"))
			})

			It("should fail when object expected but got array", func() {
				body := map[string]interface{}{
					"data": []int{1, 2, 3},
				}
				schema := map[string]string{
					"data": "object",
				}
				err := validateBodySchema(body, schema)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("expected 'object/map'"))
			})
		})

		Context("with missing fields", func() {
			It("should fail when field not found", func() {
				body := map[string]interface{}{
					"name": "test",
				}
				schema := map[string]string{
					"id": "int",
				}
				err := validateBodySchema(body, schema)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})

			It("should fail when nested field not found", func() {
				body := map[string]interface{}{
					"extend": map[string]interface{}{},
				}
				schema := map[string]string{
					"extend.source": "string",
				}
				err := validateBodySchema(body, schema)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})

		Context("with nil values", func() {
			It("should fail when value is nil", func() {
				body := map[string]interface{}{
					"name": nil,
				}
				schema := map[string]string{
					"name": "string",
				}
				err := validateBodySchema(body, schema)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("is nil"))
			})
		})

		Context("with unsupported types", func() {
			It("should fail for unsupported type", func() {
				body := map[string]interface{}{
					"name": "test",
				}
				schema := map[string]string{
					"name": "unknown_type",
				}
				err := validateBodySchema(body, schema)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("unsupported type"))
			})
		})
	})

	Describe("getNestedValue", func() {
		It("should get top-level value", func() {
			data := map[string]interface{}{
				"name": "test",
			}
			value, exists := getNestedValue(data, "name")
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal("test"))
		})

		It("should get nested value", func() {
			data := map[string]interface{}{
				"user": map[string]interface{}{
					"name": "test",
				},
			}
			value, exists := getNestedValue(data, "user.name")
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal("test"))
		})

		It("should get deeply nested value", func() {
			data := map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": "deep",
					},
				},
			}
			value, exists := getNestedValue(data, "a.b.c")
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal("deep"))
		})

		It("should return false for missing field", func() {
			data := map[string]interface{}{}
			_, exists := getNestedValue(data, "missing")
			Expect(exists).To(BeFalse())
		})
	})

	Describe("getValueType", func() {
		It("should return 'int' for integers", func() {
			Expect(getValueType(123)).To(Equal("int"))
			Expect(getValueType(int64(123))).To(Equal("int"))
		})

		It("should return 'float64' for floats", func() {
			Expect(getValueType(1.23)).To(Equal("float64"))
		})

		It("should return 'string' for strings", func() {
			Expect(getValueType("test")).To(Equal("string"))
		})

		It("should return 'bool' for booleans", func() {
			Expect(getValueType(true)).To(Equal("bool"))
		})

		It("should return 'slice' for arrays", func() {
			Expect(getValueType([]int{1, 2, 3})).To(Equal("slice"))
		})

		It("should return 'map' for maps", func() {
			Expect(getValueType(map[string]interface{}{})).To(Equal("map"))
		})

		It("should return 'nil' for nil", func() {
			Expect(getValueType(nil)).To(Equal("nil"))
		})
	})
})
