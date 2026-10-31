// Package tables provides table operations for workbooks handled by
// github.com/xuri/excelize/v2 that excelize does not offer itself.
//
// Resize changes the range of an existing table the way Excel does: the table
// part is rewritten in place, so everything it holds is kept (calculated column
// formulas, column formats, totals row functions, filters, extensions), and the
// header and totals row cells are kept consistent with the new range.
//
// Workbooks saved in the "Strict Open XML" format are not supported: excelize
// itself cannot save them back in a form that Excel opens.
package tables
