package plugins

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testHTML = `
<html>
<head>
<title>FakePage</title>
</head>
<body>
<script src="https://example.com/example-framework.js"
        integrity="sha384-oqVuAfXRKap7fdgcCY5uykM6+R9GqQ8K/uxy9rx7HNQlGYl1kPzQho1wx4JwY8wC"
        crossorigin="anonymous"></script>
</body>
</html>
`

func TestSRI(t *testing.T) {

	sri := NewSRIPlugin()

	t.Run("Parses and locates SRI tags", func(t *testing.T) {
		replaced := sri.HandleResponseBody([]byte(testHTML))
		assert.False(t, strings.Contains(string(replaced), "integrity"))
	})

}
