package tables

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Resize changes the range of the table with the given name to rangeRef, such
// as "A1:F20". The name is compared without case, as in Excel, and rangeRef
// may use absolute references.
//
// The top-left cell of the table cannot move: rows are added or removed at the
// bottom, columns at the right. When the table has a totals row, rangeRef
// includes it. The table must keep at least one data row and must not overlap
// another table of the worksheet.
//
// The table part is rewritten in place, so everything it holds is kept:
// calculated column formulas, column formats, totals row functions, filters
// and extensions. The worksheet is updated the way Excel does it:
//
//   - a column added to the table takes the text of its header cell as name,
//     or "ColumnN" when that cell is empty or the name is already used, and
//     the name is written back to the header cell;
//   - the totals row moves to the last row of the new range and the row it
//     leaves is cleared; when the table shrinks, the row receiving the totals
//     loses its data;
//   - in the data rows added to the table, empty cells of calculated columns
//     get the column formula and cells without a style get the style of the
//     last original data row.
//
// Data validations, conditional formats and comments are not extended to the
// new range, and formulas referring to a removed column are not rewritten.
func Resize(f *excelize.File, name, rangeRef string) error {
	if f == nil {
		return ErrNilFile
	}
	to, err := parseArea(rangeRef)
	if err != nil {
		return err
	}
	sheet, from, others, err := locate(f, name)
	if err != nil {
		return err
	}
	part, err := findPart(f, name)
	if err != nil {
		return err
	}
	if err := check(part, name, from, to, others); err != nil {
		return err
	}

	kept := part.columns[:min(len(part.columns), to.columns())]
	added, err := addedColumns(f, sheet, part, kept, to)
	if err != nil {
		return err
	}
	if part.headerRows > 0 {
		for i, c := range added {
			if err := f.SetCellStr(sheet, cellName(from.right+1+i, to.top), c.name); err != nil {
				return err
			}
		}
	}
	if err := updateRows(f, sheet, part, kept, from, to); err != nil {
		return err
	}
	f.Pkg.Store(part.path, part.rewrite(to, added))
	return nil
}

// locate returns the worksheet and the range of the table, and the ranges of
// the other tables of that worksheet.
func locate(f *excelize.File, name string) (sheet string, current area, others []area, err error) {
	if name == "" {
		return "", area{}, nil, fmt.Errorf("%w: empty name", ErrTableNotFound)
	}
	for _, s := range f.GetSheetList() {
		tables, err := f.GetTables(s)
		if err != nil {
			return "", area{}, nil, err
		}
		found := false
		var ranges []area
		for _, t := range tables {
			a, err := parseArea(t.Range)
			if err != nil {
				return "", area{}, nil, fmt.Errorf("%w: table %q", err, t.Name)
			}
			if strings.EqualFold(t.Name, name) {
				current, found = a, true
				continue
			}
			ranges = append(ranges, a)
		}
		if found {
			return s, current, ranges, nil
		}
	}
	return "", area{}, nil, fmt.Errorf("%w: %q", ErrTableNotFound, name)
}

// check validates the new range against the table and its worksheet.
func check(part *tablePart, name string, from, to area, others []area) error {
	if len(part.columns) != from.columns() {
		return fmt.Errorf("%w: %s: %d columns for range %s", ErrMalformedTable, part.path, len(part.columns), from)
	}
	if to.left != from.left || to.top != from.top {
		return fmt.Errorf("%w: table %q starts at %s, range %s", ErrAnchorMoved, name, cellName(from.left, from.top), to)
	}
	if to.rows()-part.headerRows-part.totalsRows < 1 {
		return fmt.Errorf("%w: table %q, range %s", ErrNoDataRow, name, to)
	}
	for _, o := range others {
		if to.overlaps(o) {
			return fmt.Errorf("%w: table %q, range %s overlaps %s", ErrOverlap, name, to, o)
		}
	}
	return nil
}

// addedColumns names the columns added by the new range. A column takes the
// text of its header cell when there is one and the name is not used yet,
// otherwise "ColumnN" where N is its position, increased until it is unique.
// Names are compared without case, as in Excel.
func addedColumns(f *excelize.File, sheet string, part *tablePart, kept []tableColumn, to area) ([]tableColumn, error) {
	used := make(map[string]bool, to.columns())
	for _, c := range kept {
		used[strings.ToLower(c.name)] = true
	}
	var added []tableColumn
	nextID := part.maxColumnID() + 1
	for position := len(kept) + 1; position <= to.columns(); position++ {
		name := ""
		if part.headerRows > 0 {
			text, err := f.GetCellValue(sheet, cellName(to.left+position-1, to.top))
			if err != nil {
				return nil, err
			}
			name = strings.TrimSpace(text)
		}
		for n := position; name == "" || used[strings.ToLower(name)]; n++ {
			name = "Column" + strconv.Itoa(n)
		}
		used[strings.ToLower(name)] = true
		added = append(added, tableColumn{id: nextID, name: name})
		nextID++
	}
	return added, nil
}

// updateRows moves the totals row and prepares the data rows added to the
// table.
func updateRows(f *excelize.File, sheet string, part *tablePart, kept []tableColumn, from, to area) error {
	oldLastData := from.bottom - part.totalsRows
	newLastData := to.bottom - part.totalsRows
	cleared := 0
	if part.totalsRows > 0 && to.bottom != from.bottom {
		if err := moveTotalsRow(f, sheet, from.left, from.left+len(kept)-1, from.bottom, to.bottom); err != nil {
			return err
		}
		cleared = from.bottom
	}
	if newLastData > oldLastData {
		return fillDataRows(f, sheet, from.left, kept, oldLastData, oldLastData+1, newLastData, cleared)
	}
	return nil
}
