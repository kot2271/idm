package validator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Validator struct {
	validate *validator.Validate
}
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (ve ValidationErrors) Error() string {
	var messages []string
	for _, err := range ve.Errors {
		messages = append(messages, err.Message)
	}
	return strings.Join(messages, "; ")
}

func New() *Validator {
	validate := validator.New()
	return &Validator{validate: validate}
}

func (v *Validator) Validate(request any) error {
	err := v.validate.Struct(request)
	if err != nil {
		var validateErrs validator.ValidationErrors
		if errors.As(err, &validateErrs) {
			return v.formatValidationErrors(validateErrs)
		}
		return err
	}
	return nil
}

func (v *Validator) formatValidationErrors(errs validator.ValidationErrors) ValidationErrors {
	var validationErrors []ValidationError

	for _, err := range errs {
		validationError := ValidationError{
			Field:   err.Field(),
			Tag:     err.Tag(),
			Value:   fmt.Sprintf("%v", err.Value()),
			Message: v.getErrorMessage(err),
		}
		validationErrors = append(validationErrors, validationError)
	}

	return ValidationErrors{Errors: validationErrors}
}

func (v *Validator) getErrorMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("Field '%s' required", err.Field())
	case "email":
		return fmt.Sprintf("Field '%s' must contain a valid email address", err.Field())
	case "min":
		return fmt.Sprintf("Field '%s' must contain at least %s characters", err.Field(), err.Param())
	case "max":
		return fmt.Sprintf("Field '%s' must contain a maximum of %s characters", err.Field(), err.Param())
	case "len":
		return fmt.Sprintf("Field '%s' must contain exactly %s characters", err.Field(), err.Param())
	case "numeric":
		return fmt.Sprintf("Field '%s' must contain only numbers", err.Field())
	case "alpha":
		return fmt.Sprintf("Field '%s' must contain only letters", err.Field())
	case "alphanum":
		return fmt.Sprintf("Field '%s' must contain only letters and numbers", err.Field())
	default:
		return fmt.Sprintf("Field '%s' contains an incorrect value", err.Field())
	}
}
