package validation

import (
	"log"
	"regexp"

	"github.com/go-playground/validator/v10"
)

var (
	validate               = validator.New()
	customRegexValidations = map[string]string{
		"customcoordinates": "^\\s*[+-]?\\d*\\.?\\d{1,8}\\s*,\\s*[+-]?\\d*\\.?\\d{1,8}\\s*$",
		"customdatetime":    "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}[+-]\\d{2}:\\d{2}$",
	}
)

// RegisterCustomValidations registers custom validation functions for the validator
func RegisterCustomValidations(validate *validator.Validate) {
	for tag, regex := range customRegexValidations {
		err := validate.RegisterValidation(tag, func(fl validator.FieldLevel) bool {
			matched, err := regexp.MatchString(regex, fl.Field().String())

			if err != nil {
				log.Printf("error matching regex for tag '%s': %v\n", tag, err)
				return false
			}

			return matched
		})

		if err != nil {
			log.Fatalf("failed to register validation for tag '%s': %v\n", tag, err)
		}
	}
}
