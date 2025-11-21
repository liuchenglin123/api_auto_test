package validator_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"api_auto_test/pkg/client"
	"api_auto_test/pkg/config"
	"api_auto_test/pkg/validator"
	"net/http"
)

var _ = Describe("Validator", func() {
	var (
		v    *validator.Validator
		resp *client.Response
	)

	Describe("验证状态码", func() {
		BeforeEach(func() {
			resp = &client.Response{
				StatusCode: 200,
				Headers:    http.Header{},
				Body:       []byte(`{"success":true}`),
				BodyJSON:   map[string]interface{}{"success": true},
			}
		})

		Context("当状态码匹配时", func() {
			It("应该验证通过", func() {
				expectation := config.ResponseExpectation{
					StatusCode: 200,
				}
				v = validator.NewValidator(expectation)
				result := v.Validate(resp)

				Expect(result.Passed).To(BeTrue())
				Expect(result.Errors).To(BeEmpty())
			})
		})

		Context("当状态码不匹配时", func() {
			It("应该验证失败", func() {
				expectation := config.ResponseExpectation{
					StatusCode: 404,
				}
				v = validator.NewValidator(expectation)
				result := v.Validate(resp)

				Expect(result.Passed).To(BeFalse())
				Expect(result.Errors).To(HaveLen(1))
				Expect(result.Errors[0].Field).To(Equal("StatusCode"))
			})
		})
	})

	Describe("验证响应头", func() {
		BeforeEach(func() {
			headers := http.Header{}
			headers.Set("Content-Type", "application/json")
			resp = &client.Response{
				StatusCode: 200,
				Headers:    headers,
				Body:       []byte(`{}`),
				BodyJSON:   map[string]interface{}{},
			}
		})

		Context("当响应头匹配时", func() {
			It("应该验证通过", func() {
				expectation := config.ResponseExpectation{
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				}
				v = validator.NewValidator(expectation)
				result := v.Validate(resp)

				Expect(result.Passed).To(BeTrue())
			})
		})

		Context("当响应头不匹配时", func() {
			It("应该验证失败", func() {
				expectation := config.ResponseExpectation{
					Headers: map[string]string{
						"Content-Type": "text/html",
					},
				}
				v = validator.NewValidator(expectation)
				result := v.Validate(resp)

				Expect(result.Passed).To(BeFalse())
				Expect(result.Errors).To(HaveLen(1))
			})
		})
	})

	Describe("验证Body字段", func() {
		BeforeEach(func() {
			resp = &client.Response{
				StatusCode: 200,
				Headers:    http.Header{},
				Body:       []byte(`{"success":true,"data":{"id":"123","name":"test"}}`),
				BodyJSON: map[string]interface{}{
					"success": true,
					"data": map[string]interface{}{
						"id":   "123",
						"name": "test",
					},
				},
			}
		})

		Context("当字段值匹配时", func() {
			It("应该验证通过", func() {
				expectation := config.ResponseExpectation{
					Body: map[string]interface{}{
						"success": true,
					},
				}
				v = validator.NewValidator(expectation)
				result := v.Validate(resp)

				Expect(result.Passed).To(BeTrue())
			})
		})

		Context("当嵌套字段值匹配时", func() {
			It("应该验证通过", func() {
				expectation := config.ResponseExpectation{
					Body: map[string]interface{}{
						"data.id": "123",
					},
				}
				v = validator.NewValidator(expectation)
				result := v.Validate(resp)

				Expect(result.Passed).To(BeTrue())
			})
		})

		Context("当字段值不匹配时", func() {
			It("应该验证失败", func() {
				expectation := config.ResponseExpectation{
					Body: map[string]interface{}{
						"success": false,
					},
				}
				v = validator.NewValidator(expectation)
				result := v.Validate(resp)

				Expect(result.Passed).To(BeFalse())
				Expect(result.Errors).To(HaveLen(1))
			})
		})
	})

	Describe("验证Body包含内容", func() {
		BeforeEach(func() {
			resp = &client.Response{
				StatusCode: 200,
				Headers:    http.Header{},
				Body:       []byte(`{"message":"User created successfully"}`),
				BodyJSON:   map[string]interface{}{"message": "User created successfully"},
			}
		})

		Context("当Body包含指定内容时", func() {
			It("应该验证通过", func() {
				expectation := config.ResponseExpectation{
					BodyContains: []string{"User created", "successfully"},
				}
				v = validator.NewValidator(expectation)
				result := v.Validate(resp)

				Expect(result.Passed).To(BeTrue())
			})
		})

		Context("当Body不包含指定内容时", func() {
			It("应该验证失败", func() {
				expectation := config.ResponseExpectation{
					BodyContains: []string{"error", "failed"},
				}
				v = validator.NewValidator(expectation)
				result := v.Validate(resp)

				Expect(result.Passed).To(BeFalse())
				Expect(result.Errors).To(HaveLen(2))
			})
		})
	})

	Describe("自定义验证器", func() {
		BeforeEach(func() {
			resp = &client.Response{
				StatusCode: 200,
				Headers:    http.Header{},
				Body:       []byte(`{"email":"test@example.com","count":10}`),
				BodyJSON: map[string]interface{}{
					"email": "test@example.com",
					"count": float64(10),
				},
			}
		})

		Context("equals验证器", func() {
			It("应该正确验证相等", func() {
				expectation := config.ResponseExpectation{
					Validators: []config.Validator{
						{
							Type:  "equals",
							Field: "count",
							Value: float64(10),
						},
					},
				}
				v = validator.NewValidator(expectation)
				result := v.Validate(resp)

				Expect(result.Passed).To(BeTrue())
			})
		})

		Context("regex验证器", func() {
			It("应该正确验证正则表达式", func() {
				expectation := config.ResponseExpectation{
					Validators: []config.Validator{
						{
							Type:  "regex",
							Field: "email",
							Value: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
						},
					},
				}
				v = validator.NewValidator(expectation)
				result := v.Validate(resp)

				Expect(result.Passed).To(BeTrue())
			})
		})

		Context("contains验证器", func() {
			It("应该正确验证包含关系", func() {
				expectation := config.ResponseExpectation{
					Validators: []config.Validator{
						{
							Type:  "contains",
							Field: "email",
							Value: "example.com",
						},
					},
				}
				v = validator.NewValidator(expectation)
				result := v.Validate(resp)

				Expect(result.Passed).To(BeTrue())
			})
		})

		Context("not_empty验证器", func() {
			It("应该正确验证非空", func() {
				expectation := config.ResponseExpectation{
					Validators: []config.Validator{
						{
							Type:  "not_empty",
							Field: "email",
						},
					},
				}
				v = validator.NewValidator(expectation)
				result := v.Validate(resp)

				Expect(result.Passed).To(BeTrue())
			})
		})
	})
})
