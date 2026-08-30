package seed

import (
	"testing"

	"github.com/arias9306/schema-api/db"
	"github.com/arias9306/schema-api/schema"
)

func BenchmarkSeed(b *testing.B) {
	sch := &schema.Schema{Tables: []schema.Table{
		{Name: "users", Columns: []schema.Column{
			{Name: "name", Type: "string", Required: true},
			{Name: "email", Type: "string", Unique: true},
		}},
		{Name: "posts", Columns: []schema.Column{
			{Name: "title", Type: "string", Required: true},
			{Name: "user_id", Type: "int", ForeignKey: &schema.ForeignKey{Table: "users"}},
		}},
	}}

	b.ResetTimer()
	for b.Loop() {
		database, err := db.InitDB(":memory:")
		if err != nil {
			b.Fatal(err)
		}
		for _, table := range sch.Tables {
			if err := db.CreateTable(database, table); err != nil {
				b.Fatal(err)
			}
		}

		const rowsPerTable = 1000
		if err := Seed(database, sch, rowsPerTable); err != nil {
			b.Fatal(err)
		}
		database.Close()
	}
}
