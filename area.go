package tables

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// area is a rectangular cell range, in 1-based column and row numbers.
type area struct {
	left, top, right, bottom int
}

// parseArea parses a range reference such as "A1:D8". Absolute references
// ("$A$1:$D$8") are accepted and the corners are put back in order, so
// "D8:A1" gives the same area as "A1:D8".
func parseArea(ref string) (area, error) {
	first, last, ok := strings.Cut(strings.ReplaceAll(ref, "$", ""), ":")
	if !ok {
		return area{}, fmt.Errorf("%w: %q", ErrInvalidRange, ref)
	}
	x1, y1, err := excelize.CellNameToCoordinates(first)
	if err != nil {
		return area{}, fmt.Errorf("%w: %q: %w", ErrInvalidRange, ref, err)
	}
	x2, y2, err := excelize.CellNameToCoordinates(last)
	if err != nil {
		return area{}, fmt.Errorf("%w: %q: %w", ErrInvalidRange, ref, err)
	}
	return area{
		left:   min(x1, x2),
		top:    min(y1, y2),
		right:  max(x1, x2),
		bottom: max(y1, y2),
	}, nil
}

// String returns the relative reference of the area, as Excel writes it.
func (a area) String() string {
	return cellName(a.left, a.top) + ":" + cellName(a.right, a.bottom)
}

func (a area) columns() int { return a.right - a.left + 1 }

func (a area) rows() int { return a.bottom - a.top + 1 }

func (a area) overlaps(b area) bool {
	return a.left <= b.right && b.left <= a.right && a.top <= b.bottom && b.top <= a.bottom
}

// cellName returns the name of a cell whose coordinates are known to be valid.
func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}
