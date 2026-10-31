package tables_test

import (
	"fmt"
	"log"

	"github.com/xuri/excelize/v2"

	tables "github.com/go-mods/excelize-tables"
)

// This example appends rows below a table and resizes the table so that it
// covers them.
func ExampleResize() {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	const sheet = "Sheet1"
	rows := [][]any{
		{"Article", "Quantity", "UnitPrice"},
		{"Pencil", 10, 0.5},
		{"Notebook", 4, 2.25},
	}
	for i, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := f.SetSheetRow(sheet, cell, &row); err != nil {
			log.Fatal(err)
		}
	}
	if err := f.AddTable(sheet, &excelize.Table{Range: "A1:C3", Name: "Articles"}); err != nil {
		log.Fatal(err)
	}

	// Two more articles, then a new column next to the table.
	if err := f.SetSheetRow(sheet, "A4", &[]any{"Eraser", 6, 0.8}); err != nil {
		log.Fatal(err)
	}
	if err := f.SetSheetRow(sheet, "A5", &[]any{"Ruler", 2, 1.5}); err != nil {
		log.Fatal(err)
	}
	if err := f.SetCellStr(sheet, "D1", "Supplier"); err != nil {
		log.Fatal(err)
	}

	if err := tables.Resize(f, "Articles", "A1:D5"); err != nil {
		log.Fatal(err)
	}

	list, err := f.GetTables(sheet)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(list[0].Name, list[0].Range)
	// Output: Articles A1:D5
}
