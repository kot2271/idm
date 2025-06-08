package validator

import (
	"errors"

	"github.com/go-playground/validator/v10"
)

type Validator struct {
	validate *validator.Validate
}

func New() *Validator {
	validate := validator.New()
	return &Validator{validate: validate}
}

func (v Validator) Validate(request any) (err error) {
	err = v.validate.Struct(request)
	if err != nil {
		var validateErrs validator.ValidationErrors
		if errors.As(err, &validateErrs) {
			return validateErrs
		}
	}
	return err
}
