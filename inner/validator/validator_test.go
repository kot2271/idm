package validator

import (
	"idm/inner/employee"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockValidator struct {
	mock.Mock
}

func (m *MockValidator) Validate(request any) error {
	args := m.Called(request)
	return args.Error(0)
}

func TestCreateRequest_Validation(t *testing.T) {
	validator_ := validator.New()

	t.Run("Valid request - all fields correct", func(t *testing.T) {
		req := employee.CreateRequest{
			Name:       "John Doe",
			Email:      "john.doe@example.com",
			Position:   "Software Engineer",
			Department: "IT",
			RoleId:     1,
		}

		err := validator_.Struct(req)
		assert.NoError(t, err)
	})

	t.Run("Invalid Name - empty", func(t *testing.T) {
		req := employee.CreateRequest{
			Name:       "",
			Email:      "john.doe@example.com",
			Position:   "Software Engineer",
			Department: "IT",
			RoleId:     1,
		}

		err := validator_.Struct(req)
		require.Error(t, err)

		validationErrors := err.(validator.ValidationErrors)
		assert.Len(t, validationErrors, 1)
		assert.Equal(t, "Name", validationErrors[0].Field())
		assert.Equal(t, "required", validationErrors[0].Tag())
	})

	t.Run("Invalid Name - too short (less than 2 characters)", func(t *testing.T) {
		req := employee.CreateRequest{
			Name:       "J",
			Email:      "john.doe@example.com",
			Position:   "Software Engineer",
			Department: "IT",
			RoleId:     1,
		}

		err := validator_.Struct(req)
		require.Error(t, err)

		validationErrors := err.(validator.ValidationErrors)
		assert.Len(t, validationErrors, 1)
		assert.Equal(t, "Name", validationErrors[0].Field())
		assert.Equal(t, "min", validationErrors[0].Tag())
	})

	t.Run("Invalid Name - too long (more than 155 characters)", func(t *testing.T) {
		longName := "John Doe with a very very very very very very very very very very very very very very very very very very very very very very very very very long name that exceeds the limit"
		req := employee.CreateRequest{
			Name:       longName,
			Email:      "john.doe@example.com",
			Position:   "Software Engineer",
			Department: "IT",
			RoleId:     1,
		}

		err := validator_.Struct(req)
		require.Error(t, err)

		validationErrors := err.(validator.ValidationErrors)
		assert.Len(t, validationErrors, 1)
		assert.Equal(t, "Name", validationErrors[0].Field())
		assert.Equal(t, "max", validationErrors[0].Tag())
	})

	t.Run("Invalid Email - empty", func(t *testing.T) {
		req := employee.CreateRequest{
			Name:       "John Doe",
			Email:      "",
			Position:   "Software Engineer",
			Department: "IT",
			RoleId:     1,
		}

		err := validator_.Struct(req)
		require.Error(t, err)

		validationErrors := err.(validator.ValidationErrors)
		assert.Len(t, validationErrors, 1)
		assert.Equal(t, "Email", validationErrors[0].Field())
		assert.Equal(t, "required", validationErrors[0].Tag())
	})

	t.Run("Invalid Email - incorrect format", func(t *testing.T) {
		req := employee.CreateRequest{
			Name:       "John Doe",
			Email:      "invalid-email",
			Position:   "Software Engineer",
			Department: "IT",
			RoleId:     1,
		}

		err := validator_.Struct(req)
		require.Error(t, err)

		validationErrors := err.(validator.ValidationErrors)
		assert.Len(t, validationErrors, 1)
		assert.Equal(t, "Email", validationErrors[0].Field())
		assert.Equal(t, "email", validationErrors[0].Tag())
	})

	t.Run("Invalid Email - missing @ symbol", func(t *testing.T) {
		req := employee.CreateRequest{
			Name:       "John Doe",
			Email:      "john.doeexample.com",
			Position:   "Software Engineer",
			Department: "IT",
			RoleId:     1,
		}

		err := validator_.Struct(req)
		require.Error(t, err)

		validationErrors := err.(validator.ValidationErrors)
		assert.Len(t, validationErrors, 1)
		assert.Equal(t, "Email", validationErrors[0].Field())
		assert.Equal(t, "email", validationErrors[0].Tag())
	})

	t.Run("Invalid Position - empty", func(t *testing.T) {
		req := employee.CreateRequest{
			Name:       "John Doe",
			Email:      "john.doe@example.com",
			Position:   "",
			Department: "IT",
			RoleId:     1,
		}

		err := validator_.Struct(req)
		require.Error(t, err)

		validationErrors := err.(validator.ValidationErrors)
		assert.Len(t, validationErrors, 1)
		assert.Equal(t, "Position", validationErrors[0].Field())
		assert.Equal(t, "required", validationErrors[0].Tag())
	})

	t.Run("Invalid Department - empty", func(t *testing.T) {
		req := employee.CreateRequest{
			Name:       "John Doe",
			Email:      "john.doe@example.com",
			Position:   "Software Engineer",
			Department: "",
			RoleId:     1,
		}

		err := validator_.Struct(req)
		require.Error(t, err)

		validationErrors := err.(validator.ValidationErrors)
		assert.Len(t, validationErrors, 1)
		assert.Equal(t, "Department", validationErrors[0].Field())
		assert.Equal(t, "required", validationErrors[0].Tag())
	})

	t.Run("Invalid RoleId - zero value", func(t *testing.T) {
		req := employee.CreateRequest{
			Name:       "John Doe",
			Email:      "john.doe@example.com",
			Position:   "Software Engineer",
			Department: "IT",
			RoleId:     0,
		}

		err := validator_.Struct(req)
		require.Error(t, err)

		validationErrors := err.(validator.ValidationErrors)
		assert.Len(t, validationErrors, 1)
		assert.Equal(t, "RoleId", validationErrors[0].Field())
		assert.Equal(t, "required", validationErrors[0].Tag())
	})

	t.Run("Multiple validation errors", func(t *testing.T) {
		req := employee.CreateRequest{
			Name:       "",
			Email:      "invalid-email",
			Position:   "",
			Department: "",
			RoleId:     0,
		}

		err := validator_.Struct(req)
		require.Error(t, err)

		validationErrors := err.(validator.ValidationErrors)
		assert.Len(t, validationErrors, 5)

		// Проверяем, что все поля имеют ошибки валидации
		fieldErrors := make(map[string]string)
		for _, validationError := range validationErrors {
			fieldErrors[validationError.Field()] = validationError.Tag()
		}

		assert.Equal(t, "required", fieldErrors["Name"])
		assert.Equal(t, "email", fieldErrors["Email"])
		assert.Equal(t, "required", fieldErrors["Position"])
		assert.Equal(t, "required", fieldErrors["Department"])
		assert.Equal(t, "required", fieldErrors["RoleId"])
	})
}

func TestCreateRequest_WithCustomValidator(t *testing.T) {
	customValidator := New()

	t.Run("Valid request with custom validator", func(t *testing.T) {
		req := employee.CreateRequest{
			Name:       "John Doe",
			Email:      "john.doe@example.com",
			Position:   "Software Engineer",
			Department: "IT",
			RoleId:     1,
		}

		err := customValidator.Validate(req)
		assert.NoError(t, err)
	})

	t.Run("Invalid request with custom validator", func(t *testing.T) {
		req := employee.CreateRequest{
			Name:       "",
			Email:      "invalid-email",
			Position:   "",
			Department: "",
			RoleId:     0,
		}

		err := customValidator.Validate(req)
		require.Error(t, err)

		validationErrors, ok := err.(validator.ValidationErrors)
		require.True(t, ok)
		assert.Len(t, validationErrors, 5)
	})
}

// Тест с использованием мока валидатора
func TestCreateRequest_WithMockValidator(t *testing.T) {
	t.Run("Mock validator returns no error", func(t *testing.T) {
		mockValidator := &MockValidator{}
		req := employee.CreateRequest{
			Name:       "John Doe",
			Email:      "john.doe@example.com",
			Position:   "Software Engineer",
			Department: "IT",
			RoleId:     1,
		}

		mockValidator.On("Validate", req).Return(nil)

		err := mockValidator.Validate(req)
		assert.NoError(t, err)
		mockValidator.AssertExpectations(t)
	})

	t.Run("Mock validator returns validation error", func(t *testing.T) {
		mockValidator := &MockValidator{}
		req := employee.CreateRequest{
			Name:       "",
			Email:      "john.doe@example.com",
			Position:   "Software Engineer",
			Department: "IT",
			RoleId:     1,
		}

		expectedError := validator.ValidationErrors{}
		mockValidator.On("Validate", req).Return(expectedError)

		err := mockValidator.Validate(req)
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
		mockValidator.AssertExpectations(t)
	})
}

// Benchmark тесты для проверки производительности валидации
func BenchmarkCreateRequest_Validation(b *testing.B) {
	validator := validator.New()
	req := employee.CreateRequest{
		Name:       "John Doe",
		Email:      "john.doe@example.com",
		Position:   "Software Engineer",
		Department: "IT",
		RoleId:     1,
	}

	for b.Loop() {
		_ = validator.Struct(req)
	}
}
