package validators

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

func Init() {
	Validate = validator.New()
	_ = Validate.RegisterValidation("slug", validateSlug)
	_ = Validate.RegisterValidation("phone", validatePhone)
}

// RegisterGin registers custom validators (phone, slug) on Gin's binding validator
// so that ShouldBindJSON uses them and returns proper validation errors instead of panics.
func RegisterGin() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("slug", validateSlug)
		_ = v.RegisterValidation("phone", validatePhone)
	}
}

func validateSlug(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	if len(s) < 1 || len(s) > 100 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-z0-9]+(?:-[a-z0-9]+)*$`, s)
	return matched
}

func validatePhone(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	if s == "" {
		return true
	}
	matched, _ := regexp.MatchString(`^\+?[0-9\s\-()]{10,20}$`, s)
	return matched
}

// Slugify creates a URL-safe slug from a string
func Slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
