# excelize-tables

[![Go Reference](https://pkg.go.dev/badge/github.com/go-mods/excelize-tables.svg)](https://pkg.go.dev/github.com/go-mods/excelize-tables)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/go-mods/excelize-tables/blob/master/LICENSE)

Table helpers for [excelize](https://github.com/qax-os/excelize) that are not
provided by the library itself, such as resizing an existing table.

The module depends on the official `github.com/xuri/excelize/v2` release: no
fork and no `replace` directive are required.

## Installation

```shell
go get github.com/go-mods/excelize-tables
```

The module requires Go 1.25 or later and excelize v2.11.0 or later.

## Usage

```go
import (
    "github.com/xuri/excelize/v2"

    tables "github.com/go-mods/excelize-tables"
)

f, err := excelize.OpenFile("articles.xlsx")
if err != nil {
    return err
}
defer f.Close()

// Rows were written below the table "Articles", which covered A1:D8.
if err := tables.Resize(f, "Articles", "A1:D20"); err != nil {
    return err
}
return f.Save()
```

`Resize` takes the name of the table, compared without case as in Excel, and
its new range. The range may use absolute references (`$A$1:$D$20`) and
includes the totals row when the table has one.

### What is kept

The table definition is rewritten in place rather than rebuilt, so everything
it holds is kept, including what excelize does not model:

- calculated column formulas, column formats and totals row functions;
- auto filter settings of the columns that stay in the table;
- the table style, extensions and namespace prefixes written by Excel or by
    other tools.

### What is updated in the worksheet

- **Added columns** take the text of their header cell as name, or `ColumnN`
    when that cell is empty or the name is already used; the name is written back
    to the header cell.
- **The totals row** moves to the last row of the new range and the row it
    leaves is cleared.
- **Added data rows** get the calculated column formulas in their empty cells,
    and the style of the last original data row in their cells without a style.
    Values written before the call are never overwritten.

### Errors

`Resize` returns an error, testable with `errors.Is`, when:

| Error | Cause |
|-------|-------|
| `ErrTableNotFound` | no table has this name |
| `ErrInvalidRange` | the range cannot be parsed |
| `ErrAnchorMoved` | the new range does not start at the top-left cell of the table: rows are added or removed at the bottom, columns at the right |
| `ErrNoDataRow` | no data row is left between the header row and the totals row |
| `ErrOverlap` | the new range overlaps another table of the worksheet |

## Limitations

- Data validations, conditional formats and comments are not extended to the
    new range.
- Formulas referring to a column removed from the table are not rewritten.
- When a table with a totals row shrinks, the row receiving the totals loses
    its data.
- Workbooks saved in the "Strict Open XML" format are not supported: excelize
    cannot save them back in a form that Excel opens.

## Development

The tasks are run with [Task](https://taskfile.dev):

```shell
task            # enables the pre-commit hook and lists the tasks
task check      # formats, lints and tests the code
task go:test    # runs the tests
```

## License

[MIT](LICENSE)
