package tables

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

const (
	sheet            = "Articles"
	table            = "Articles"
	part1            = "xl/tables/table1.xml"
	totalPriceFormat = "Articles[[#This Row],[Quantity]]*Articles[[#This Row],[UnitPrice]]"
)

// open opens a workbook of testdata. The table "Articles" of these workbooks
// covers A1:D8: a header row, six data rows and a totals row; TotalPrice is a
// calculated column and UnitPrice and TotalPrice have a number format.
func open(t *testing.T, name string) *excelize.File {
	t.Helper()
	f, err := excelize.OpenFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func tableXML(t *testing.T, f *excelize.File) string {
	t.Helper()
	v, ok := f.Pkg.Load(part1)
	require.True(t, ok)
	return string(v.([]byte))
}

// reopen saves the workbook and opens it again.
func reopen(t *testing.T, f *excelize.File) *excelize.File {
	t.Helper()
	path := filepath.Join(t.TempDir(), "resized.xlsx")
	require.NoError(t, f.SaveAs(path))
	g, err := excelize.OpenFile(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = g.Close() })
	return g
}

func tableRange(t *testing.T, f *excelize.File, sheet string) string {
	t.Helper()
	tables, err := f.GetTables(sheet)
	require.NoError(t, err)
	require.Len(t, tables, 1)
	return tables[0].Range
}

func value(t *testing.T, f *excelize.File, cell string) string {
	t.Helper()
	v, err := f.GetCellValue(sheet, cell)
	require.NoError(t, err)
	return v
}

func formula(t *testing.T, f *excelize.File, cell string) string {
	t.Helper()
	v, err := f.GetCellFormula(sheet, cell)
	require.NoError(t, err)
	return v
}

func style(t *testing.T, f *excelize.File, cell string) int {
	t.Helper()
	v, err := f.GetCellStyle(sheet, cell)
	require.NoError(t, err)
	return v
}

func TestResizeAddRowsMovesTotalsRow(t *testing.T) {
	f := open(t, "articles.xlsx")
	before := tableXML(t, f)

	require.NoError(t, Resize(f, table, "A1:D11"))

	x := tableXML(t, f)
	assert.Contains(t, x, `ref="A1:D11"`)
	assert.Contains(t, x, `<autoFilter ref="A1:D10"`)
	assert.Contains(t, x, `<calculatedColumnFormula>`+totalPriceFormat+`</calculatedColumnFormula>`)
	assert.Contains(t, x, `totalsRowDxfId="0"`)
	assert.Equal(t, len(before), len(x)-len("11")-len("10")+len("8")+len("7"), "only the two ranges change")

	assert.Equal(t, "Total", value(t, f, "A11"))
	assert.Equal(t, "SUBTOTAL(109,Articles[Quantity])", formula(t, f, "B11"))
	assert.Equal(t, "SUBTOTAL(109,Articles[TotalPrice])", formula(t, f, "D11"))
	for _, cell := range []string{"A8", "B8", "C8"} {
		assert.Empty(t, value(t, f, cell), cell)
		assert.Empty(t, formula(t, f, cell), cell)
	}
	for _, cell := range []string{"D8", "D9", "D10"} {
		assert.Equal(t, totalPriceFormat, formula(t, f, cell), cell)
	}
	assert.Equal(t, style(t, f, "C7"), style(t, f, "C9"), "format of the last data row")

	assert.Equal(t, "A1:D11", tableRange(t, reopen(t, f), sheet))
}

func TestResizeMovesTotalsRowValues(t *testing.T) {
	f := open(t, "articles.xlsx")
	require.NoError(t, f.SetCellFormula(sheet, "B8", ""))
	require.NoError(t, f.SetCellValue(sheet, "B8", 1296))
	require.NoError(t, f.SetCellBool(sheet, "C8", true))

	require.NoError(t, Resize(f, table, "A1:D9"))

	assert.Equal(t, "1296", value(t, f, "B9"))
	kind, err := f.GetCellType(sheet, "C9")
	require.NoError(t, err)
	assert.Equal(t, excelize.CellTypeBool, kind)
	assert.Equal(t, "TRUE", value(t, f, "C9"))
	assert.Empty(t, value(t, f, "B8"))
}

func TestResizeRemoveRowsMovesTotalsRow(t *testing.T) {
	f := open(t, "articles.xlsx")

	require.NoError(t, Resize(f, table, "A1:D5"))

	x := tableXML(t, f)
	assert.Contains(t, x, `ref="A1:D5"`)
	assert.Contains(t, x, `<autoFilter ref="A1:D4"`)
	assert.Equal(t, "Total", value(t, f, "A5"))
	assert.Equal(t, "SUBTOTAL(109,Articles[Quantity])", formula(t, f, "B5"))
	assert.Empty(t, value(t, f, "C5"), "a totals cell without function is emptied")
	assert.Equal(t, "Cable tie 200 mm", value(t, f, "A6"), "rows below the table are kept")
	assert.Equal(t, "Wall plug 8 mm", value(t, f, "A7"))
	assert.Empty(t, value(t, f, "A8"))
	assert.Empty(t, formula(t, f, "B8"))
}

func TestResizeWithoutTotalsRow(t *testing.T) {
	f := excelize.NewFile()
	t.Cleanup(func() { _ = f.Close() })
	rows := [][]any{{"Article", "Quantity"}, {"Hex bolt M8", 120}, {"Lock nut M8", 120}}
	for i, r := range rows {
		require.NoError(t, f.SetSheetRow("Sheet1", cellName(1, i+1), &r))
	}
	require.NoError(t, f.AddTable("Sheet1", &excelize.Table{Range: "A1:B3", Name: "Stock"}))

	require.NoError(t, Resize(f, "Stock", "A1:C6"))

	x := tableXML(t, f)
	assert.Contains(t, x, `ref="A1:C6"`)
	assert.Contains(t, x, `<autoFilter ref="A1:C6"`)
	assert.Contains(t, x, `<tableColumn id="3" name="Column3"/>`)
	got, err := f.GetCellValue("Sheet1", "C1")
	require.NoError(t, err)
	assert.Equal(t, "Column3", got)
	assert.Equal(t, "A1:C6", tableRange(t, reopen(t, f), "Sheet1"))
}

func TestResizeWithHiddenHeaderRow(t *testing.T) {
	f := excelize.NewFile()
	t.Cleanup(func() { _ = f.Close() })
	hidden := false
	require.NoError(t, f.SetSheetRow("Sheet1", "A1", &[]any{"Hex bolt M8", 120}))
	require.NoError(t, f.AddTable("Sheet1", &excelize.Table{Range: "A1:B2", Name: "Stock", ShowHeaderRow: &hidden}))
	require.Contains(t, tableXML(t, f), `headerRowCount="0"`)
	// excelize moves the range of a table without header row one row down
	require.Equal(t, "A2:B2", tableRange(t, f, "Sheet1"))

	require.NoError(t, Resize(f, "Stock", "A2:C4"))

	x := tableXML(t, f)
	assert.Contains(t, x, `ref="A2:C4"`)
	assert.Contains(t, x, `name="Column3"`)
	for _, cell := range []string{"C1", "C2"} {
		got, err := f.GetCellValue("Sheet1", cell)
		require.NoError(t, err)
		assert.Empty(t, got, "no header cell to write")
	}
}

func TestResizeAddColumnNames(t *testing.T) {
	tests := []struct {
		name   string
		header any
		want   string
	}{
		{"empty header", nil, "Column5"},
		{"header text", "  Discount ", "Discount"},
		{"name already used", "ARTICLE", "Column5"},
		{"numeric header", 2024, "2024"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := open(t, "articles.xlsx")
			if tt.header != nil {
				require.NoError(t, f.SetCellValue(sheet, "E1", tt.header))
			}

			require.NoError(t, Resize(f, table, "A1:E8"))

			x := tableXML(t, f)
			assert.Contains(t, x, `<tableColumns count="5">`)
			assert.Contains(t, x, `<tableColumn id="5" name="`+tt.want+`"/></tableColumns>`)
			assert.Contains(t, x, `<autoFilter ref="A1:E7"`)
			assert.Equal(t, tt.want, value(t, f, "E1"))
			kind, err := f.GetCellType(sheet, "E1")
			require.NoError(t, err)
			assert.Equal(t, excelize.CellTypeSharedString, kind, "a header cell holds text")
		})
	}
}

func TestResizeAddColumnNameIsEscaped(t *testing.T) {
	f := open(t, "articles.xlsx")
	require.NoError(t, f.SetCellValue(sheet, "E1", `Price "net" & <VAT>`))

	require.NoError(t, Resize(f, table, "A1:E8"))

	assert.Contains(t, tableXML(t, f), `name="Price &#34;net&#34; &amp; &lt;VAT&gt;"`)
	require.NoError(t, Resize(f, table, "A1:E9"), "the rewritten part can be parsed again")
}

func TestResizeRemoveColumns(t *testing.T) {
	f := open(t, "articles.xlsx")

	require.NoError(t, Resize(f, table, "A1:C8"))

	x := tableXML(t, f)
	assert.Contains(t, x, `<tableColumns count="3">`)
	assert.NotContains(t, x, "TotalPrice")
	assert.NotContains(t, x, "calculatedColumnFormula")
	assert.Contains(t, x, `<autoFilter ref="A1:C7"`)
	assert.Equal(t, "A1:C8", tableRange(t, reopen(t, f), sheet))
}

func TestResizeRemoveColumnsDropsTheirFilters(t *testing.T) {
	tests := []struct {
		rangeRef   string
		keepFilter bool
	}{
		{"A1:B8", true},
		{"A1:A8", false},
	}
	for _, tt := range tests {
		t.Run(tt.rangeRef, func(t *testing.T) {
			f := open(t, "articles-filtered.xlsx")
			require.Contains(t, tableXML(t, f), `<filterColumn colId="1">`)

			require.NoError(t, Resize(f, table, tt.rangeRef))

			x := tableXML(t, f)
			assert.Equal(t, tt.keepFilter, strings.Contains(x, `<filterColumn colId="1">`))
			assert.Equal(t, tt.keepFilter, strings.Contains(x, `<customFilter operator="greaterThan" val="100"/>`))
		})
	}
}

func TestResizeKeepsExtensions(t *testing.T) {
	f := open(t, "articles-extlst.xlsx")

	require.NoError(t, Resize(f, table, "A1:E11"))

	x := tableXML(t, f)
	assert.Contains(t, x, `<x14:table altText="List of articles" altTextSummary="Quantities, unit prices and totals"/>`)
	assert.Contains(t, x, `mc:Ignorable="xr xr3"`)
	assert.Contains(t, x, `xr3:uid=`)
	assert.Equal(t, "A1:E11", tableRange(t, reopen(t, f), sheet))
}

func TestResizeKeepsNamespacePrefix(t *testing.T) {
	f := open(t, "articles-prefixed.xlsx")

	require.NoError(t, Resize(f, table, "A1:E8"))

	x := tableXML(t, f)
	assert.Contains(t, x, `<x:table xmlns:x="http://schemas.openxmlformats.org/spreadsheetml/2006/main"`)
	assert.Contains(t, x, `<x:tableColumn id="5" name="Column5"/></x:tableColumns>`)
	assert.Contains(t, x, `<x:autoFilter ref="A1:E7"`)
	assert.Equal(t, "A1:E8", tableRange(t, reopen(t, f), sheet))
}

func TestResizeAcceptsAnyCaseAndRangeForm(t *testing.T) {
	f := open(t, "articles.xlsx")

	require.NoError(t, Resize(f, "ARTICLES", "$D$11:$A$1"))

	assert.Contains(t, tableXML(t, f), `ref="A1:D11"`)
}

func TestResizeTwiceIsStable(t *testing.T) {
	f := open(t, "articles.xlsx")
	before := tableXML(t, f)

	require.NoError(t, Resize(f, table, "A1:D11"))
	require.NoError(t, Resize(f, table, "A1:D8"))

	assert.Equal(t, before, tableXML(t, f))
	assert.Equal(t, "Total", value(t, f, "A8"))
	assert.Equal(t, "SUBTOTAL(109,Articles[TotalPrice])", formula(t, f, "D8"))
	assert.Empty(t, value(t, f, "A11"))
}

func TestResizeErrors(t *testing.T) {
	tests := []struct {
		name     string
		table    string
		rangeRef string
		prepare  func(t *testing.T, f *excelize.File)
		want     error
	}{
		{name: "empty name", table: "", rangeRef: "A1:D9", want: ErrTableNotFound},
		{name: "unknown table", table: "Orders", rangeRef: "A1:D9", want: ErrTableNotFound},
		{name: "single cell", table: table, rangeRef: "A1", want: ErrInvalidRange},
		{name: "invalid cell", table: table, rangeRef: "A1:D0", want: ErrInvalidRange},
		{name: "column moved", table: table, rangeRef: "B1:D9", want: ErrAnchorMoved},
		{name: "row moved", table: table, rangeRef: "A2:D9", want: ErrAnchorMoved},
		{name: "no data row", table: table, rangeRef: "A1:D2", want: ErrNoDataRow},
		{
			name: "overlap", table: table, rangeRef: "A1:G8", want: ErrOverlap,
			prepare: func(t *testing.T, f *excelize.File) {
				require.NoError(t, f.SetSheetRow(sheet, "F1", &[]any{"Code", "Label"}))
				require.NoError(t, f.AddTable(sheet, &excelize.Table{Range: "F1:G3", Name: "Codes"}))
			},
		},
		{
			name: "malformed part", table: table, rangeRef: "A1:D9", want: ErrMalformedTable,
			prepare: func(t *testing.T, f *excelize.File) {
				f.Pkg.Store(part1, []byte(`<table xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" `+
					`id="1" name="Articles" displayName="Articles" ref="A1:D8"/>`))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := open(t, "articles.xlsx")
			if tt.prepare != nil {
				tt.prepare(t, f)
			}
			before := tableXML(t, f)

			err := Resize(f, tt.table, tt.rangeRef)

			require.ErrorIs(t, err, tt.want)
			assert.Equal(t, before, tableXML(t, f), "the table is left unchanged")
		})
	}
}

func TestResizeNilFile(t *testing.T) {
	assert.ErrorIs(t, Resize(nil, table, "A1:D9"), ErrNilFile)
}
