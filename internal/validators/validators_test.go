package validators

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlugify(t *testing.T) {
	assert.Equal(t, "brake-pads", Slugify("Brake Pads"))
	assert.Equal(t, "oil-filter", Slugify("Oil Filter!"))
	assert.Equal(t, "abc-123", Slugify("ABC 123"))
}
