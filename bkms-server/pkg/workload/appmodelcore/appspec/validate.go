package appspec

import (
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	validateOnce sync.Once
	validate     *validator.Validate
)

func init() {
	validateOnce.Do(func() {
		validate = validator.New(validator.WithRequiredStructEnabled())
		// Register section-specific validators.
		for _, section := range registeredSections {
			section.registerValidation(validate)
		}
	})
}

// Validate validates an AppSpec and all configured sections.
func Validate(spec *AppSpec) error {
	if err := validate.Struct(spec); err != nil {
		return wrapValidationErr(err)
	}
	return nil
}
