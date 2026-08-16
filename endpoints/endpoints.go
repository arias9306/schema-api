// Package endpoints collects and prints the HTTP endpoints exposed by a schema.
package endpoints

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/arias9306/schema-api/schema"
)

type Info struct {
	Method string
	Path   string
	Source string
	Status string
}

var crudRoutes = []struct {
	method string
	suffix string
	status int
}{
	{"GET", "", 200},
	{"GET", "/{id}", 200},
	{"POST", "", 201},
	{"PUT", "/{id}", 200},
	{"DELETE", "/{id}", 204},
}

func Collect(s *schema.Schema) []Info {
	var routes []Info

	for _, table := range s.Tables {
		for _, route := range crudRoutes {
			path := "/" + table.Name + route.suffix
			routes = append(routes, Info{
				Method: route.method,
				Path:   path,
				Source: "crud",
				Status: strconv.Itoa(route.status),
			})
		}
	}

	for _, endpoint := range s.Endpoints {
		status := endpoint.Status
		if status == 0 {
			status = 200
		}
		routes = append(routes, Info{
			Method: endpoint.Method,
			Path:   endpoint.Path,
			Source: "mock",
			Status: strconv.Itoa(status),
		})
	}

	return routes
}

func PrintTable(routes []Info) {
	fmt.Print(formatTable(routes))
}

func formatTable(routes []Info) string {
	if len(routes) == 0 {
		return ""
	}

	var b strings.Builder

	header := []string{"METHOD", "PATH", "SOURCE", "STATUS"}
	rows := make([][]string, 0, len(routes))
	for _, route := range routes {
		rows = append(rows, []string{route.Method, route.Path, route.Source, route.Status})
	}

	widths := make([]int, len(header))
	for i, h := range header {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	b.WriteString("\nEndpoints\n\n")
	printRow(&b, header, widths)
	for i, w := range widths {
		if i > 0 {
			b.WriteString("  ")
		}
		for j := 0; j < w; j++ {
			b.WriteString("-")
		}
	}
	b.WriteString("\n")
	for _, row := range rows {
		printRow(&b, row, widths)
	}

	return b.String()
}

func printRow(b *strings.Builder, cells []string, widths []int) {
	for i, cell := range cells {
		if i > 0 {
			b.WriteString("  ")
		}
		fmt.Fprintf(b, "%-*s", widths[i], cell)
	}
	b.WriteString("\n")
}
