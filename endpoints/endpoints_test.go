package endpoints

import (
	"testing"

	"github.com/arias9306/schema-api/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollect(t *testing.T) {
	sch := &schema.Schema{
		Tables: []schema.Table{
			{Name: "users"},
			{Name: "posts"},
		},
		Endpoints: []schema.Endpoint{
			{Method: "GET", Path: "/health", Status: 200, Response: map[string]any{}},
			{Method: "POST", Path: "/echo", Status: 201, Response: map[string]any{}},
			{Method: "PATCH", Path: "/no-status", Response: "x"},
		},
	}

	routes := Collect(sch)
	require.Len(t, routes, 2*5+3)

	assert.Equal(t, Info{Method: "GET", Path: "/users", Source: "crud", Status: "200"}, routes[0])
	assert.Equal(t, Info{Method: "GET", Path: "/users/{id}", Source: "crud", Status: "200"}, routes[1])
	assert.Equal(t, Info{Method: "POST", Path: "/users", Source: "crud", Status: "201"}, routes[2])
	assert.Equal(t, Info{Method: "PUT", Path: "/users/{id}", Source: "crud", Status: "200"}, routes[3])
	assert.Equal(t, Info{Method: "DELETE", Path: "/users/{id}", Source: "crud", Status: "204"}, routes[4])
	assert.Equal(t, Info{Method: "GET", Path: "/posts", Source: "crud", Status: "200"}, routes[5])

	assert.Equal(t, Info{Method: "GET", Path: "/health", Source: "mock", Status: "200"}, routes[10])
	assert.Equal(t, Info{Method: "POST", Path: "/echo", Source: "mock", Status: "201"}, routes[11])
	assert.Equal(t, Info{Method: "PATCH", Path: "/no-status", Source: "mock", Status: "200"}, routes[12])
}

func TestCollectEmptySchema(t *testing.T) {
	routes := Collect(&schema.Schema{})
	assert.Empty(t, routes)
}

func TestCollectOnlyEndpoints(t *testing.T) {
	sch := &schema.Schema{
		Endpoints: []schema.Endpoint{
			{Method: "DELETE", Path: "/items/{id}", Status: 404, Response: nil},
		},
	}

	routes := Collect(sch)
	require.Len(t, routes, 1)
	assert.Equal(t, Info{Method: "DELETE", Path: "/items/{id}", Source: "mock", Status: "404"}, routes[0])
}

func TestFormatTable(t *testing.T) {
	routes := []Info{
		{Method: "GET", Path: "/users", Source: "crud", Status: "200"},
		{Method: "DELETE", Path: "/users/{id}", Source: "crud", Status: "204"},
	}

	expected := "\nEndpoints\n\n" +
		"METHOD  PATH         SOURCE  STATUS\n" +
		"------  -----------  ------  ------\n" +
		"GET     /users       crud    200   \n" +
		"DELETE  /users/{id}  crud    204   \n"

	assert.Equal(t, expected, formatTable(routes))
}

func TestFormatTableEmpty(t *testing.T) {
	assert.Empty(t, formatTable(nil))
}
