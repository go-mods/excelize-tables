package tables

import (
	"strings"

	"github.com/xuri/excelize/v2"
)

// cellContent is what is moved when a totals row cell changes row.
type cellContent struct {
	style   int
	formula string
	kind    excelize.CellType
	raw     string
}

func readCell(f *excelize.File, sheet, cell string) (cellContent, error) {
	var (
		c   cellContent
		err error
	)
	if c.style, err = f.GetCellStyle(sheet, cell); err != nil {
		return c, err
	}
	if c.formula, err = f.GetCellFormula(sheet, cell); err != nil {
		return c, err
	}
	if c.kind, err = f.GetCellType(sheet, cell); err != nil {
		return c, err
	}
	c.raw, err = f.GetCellValue(sheet, cell, excelize.Options{RawCellValue: true})
	return c, err
}

func writeCell(f *excelize.File, sheet, cell string, c cellContent) error {
	if err := f.SetCellStyle(sheet, cell, cell, c.style); err != nil {
		return err
	}
	if c.formula != "" {
		return f.SetCellFormula(sheet, cell, c.formula)
	}
	switch {
	case c.raw == "":
		return nil
	case c.kind == excelize.CellTypeSharedString || c.kind == excelize.CellTypeInlineString:
		return f.SetCellStr(sheet, cell, c.raw)
	case c.kind == excelize.CellTypeBool:
		return f.SetCellBool(sheet, cell, c.raw == "1" || strings.EqualFold(c.raw, "true"))
	default:
		return f.SetCellDefault(sheet, cell, c.raw)
	}
}

// clearCell removes the value and the formula of a cell and keeps its style.
func clearCell(f *excelize.File, sheet, cell string) error {
	if err := f.SetCellFormula(sheet, cell, ""); err != nil {
		return err
	}
	return f.SetCellValue(sheet, cell, nil)
}

// isEmpty reports whether a cell has neither a value nor a formula.
func isEmpty(f *excelize.File, sheet, cell string) (bool, error) {
	formula, err := f.GetCellFormula(sheet, cell)
	if err != nil || formula != "" {
		return false, err
	}
	raw, err := f.GetCellValue(sheet, cell, excelize.Options{RawCellValue: true})
	return raw == "", err
}

// moveTotalsRow moves the totals row cells of the given columns from row
// `from` to row `to`. The cells left behind are cleared, and so are the cells
// receiving the totals first: a value left in a totals row cell that has no
// totals function makes Excel reject the workbook.
func moveTotalsRow(f *excelize.File, sheet string, left, right, from, to int) error {
	contents := make([]cellContent, 0, right-left+1)
	for col := left; col <= right; col++ {
		c, err := readCell(f, sheet, cellName(col, from))
		if err != nil {
			return err
		}
		contents = append(contents, c)
	}
	for col := left; col <= right; col++ {
		for _, row := range []int{from, to} {
			if err := clearCell(f, sheet, cellName(col, row)); err != nil {
				return err
			}
		}
	}
	for i, c := range contents {
		if err := writeCell(f, sheet, cellName(left+i, to), c); err != nil {
			return err
		}
	}
	return nil
}

// fillDataRows prepares the rows added to the data area, from firstRow to
// lastRow, for the columns kept from the original table: an empty cell gets
// the calculated column formula, and a cell without a style, or emptied by
// the move of the totals row, gets the style of the last original data row.
func fillDataRows(f *excelize.File, sheet string, left int, columns []tableColumn,
	templateRow, firstRow, lastRow, clearedRow int) error {
	for i, column := range columns {
		col := left + i
		style, err := f.GetCellStyle(sheet, cellName(col, templateRow))
		if err != nil {
			return err
		}
		for row := firstRow; row <= lastRow; row++ {
			cell := cellName(col, row)
			current, err := f.GetCellStyle(sheet, cell)
			if err != nil {
				return err
			}
			if current == 0 || row == clearedRow {
				if err := f.SetCellStyle(sheet, cell, cell, style); err != nil {
					return err
				}
			}
			if column.formula == "" {
				continue
			}
			empty, err := isEmpty(f, sheet, cell)
			if err != nil {
				return err
			}
			if empty {
				if err := f.SetCellFormula(sheet, cell, column.formula); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
