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

## Development

The tasks are run with [Task](https://taskfile.dev):

```shell
task            # enables the pre-commit hook and lists the tasks
task check      # formats, lints and tests the code
task go:test    # runs the tests
```

## License

[MIT](LICENSE)
