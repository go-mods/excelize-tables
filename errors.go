package tables

import "errors"

// Errors returned by Resize. They are wrapped with context and can be tested
// with errors.Is.
var (
	// ErrNilFile is returned when the given *excelize.File is nil.
	ErrNilFile = errors.New("tables: nil file")
	// ErrTableNotFound is returned when no table has the given name.
	ErrTableNotFound = errors.New("tables: table not found")
	// ErrInvalidRange is returned when the range reference cannot be parsed.
	ErrInvalidRange = errors.New("tables: invalid range")
	// ErrAnchorMoved is returned when the new range does not start at the
	// top-left cell of the table.
	ErrAnchorMoved = errors.New("tables: the top-left cell of a table cannot move")
	// ErrNoDataRow is returned when the new range leaves no data row between
	// the header row and the totals row.
	ErrNoDataRow = errors.New("tables: a table needs at least one data row")
	// ErrOverlap is returned when the new range overlaps another table of the
	// worksheet.
	ErrOverlap = errors.New("tables: range overlaps another table")
	// ErrMalformedTable is returned when the table part cannot be understood.
	ErrMalformedTable = errors.New("tables: malformed table part")
)
