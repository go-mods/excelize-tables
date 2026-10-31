package tables

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// tablesDir is the folder of the table parts inside the package.
const tablesDir = "xl/tables/"

// span is a byte range [start, end) of the table part.
type span struct {
	start, end int
}

// tableColumn is a <tableColumn> element of the table part.
type tableColumn struct {
	id      int
	name    string
	formula string // text of <calculatedColumnFormula>, if any
	element span
}

// filterColumn is a <filterColumn> element of the table auto filter.
type filterColumn struct {
	colID   int
	element span
}

// tablePart is a parsed table part. The parsing records where each value lives
// in the original bytes, so that the part can be rewritten by replacing those
// bytes only: everything else, including elements and attributes this package
// does not know about, is kept exactly as it was written.
type tablePart struct {
	path       string
	data       []byte
	prefix     string // namespace prefix of the root element, often empty
	headerRows int
	totalsRows int

	ref           span  // value of table@ref
	filterRef     *span // value of autoFilter@ref, when there is an auto filter
	filterColumns []filterColumn
	columnsCount  span // value of tableColumns@count
	columnsEnd    int  // offset of the </tableColumns> end tag
	columns       []tableColumn
}

// findPart returns the table part whose name matches, ignoring case as Excel
// does. The parts are read from the package of the file.
func findPart(f *excelize.File, name string) (*tablePart, error) {
	var (
		found    *tablePart
		parseErr error
	)
	f.Pkg.Range(func(key, value any) bool {
		path, ok := key.(string)
		if !ok || !strings.HasPrefix(path, tablesDir) || !strings.HasSuffix(path, ".xml") {
			return true
		}
		data, ok := value.([]byte)
		if !ok || !strings.EqualFold(rootName(data), name) {
			return true
		}
		found, parseErr = parsePart(path, data)
		return false
	})
	if parseErr != nil {
		return nil, parseErr
	}
	if found == nil {
		return nil, fmt.Errorf("%w: %q", ErrTableNotFound, name)
	}
	return found, nil
}

// rootName returns the name attribute of the root element of a table part.
func rootName(data []byte) string {
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.RawToken()
		if err != nil {
			return ""
		}
		if start, ok := tok.(xml.StartElement); ok {
			return attrValue(start, "name")
		}
	}
}

// parsePart parses a table part. Only the elements of the namespace of the
// root element are considered: an extension such as <x14:table> has the same
// local name as the root element but must be left alone.
func parsePart(path string, data []byte) (*tablePart, error) {
	p := &tablePart{path: path, data: data, headerRows: 1, columnsEnd: -1}
	dec := xml.NewDecoder(bytes.NewReader(data))
	var (
		stack   []string // local names of the open elements of the root namespace, "" otherwise
		opened  []int    // start offsets of the open elements
		column  *tableColumn
		filter  *filterColumn
		formula strings.Builder
		rooted  bool
	)
	for {
		before := int(dec.InputOffset())
		tok, err := dec.RawToken()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrMalformedTable, path, err)
		}
		after := int(dec.InputOffset())
		switch t := tok.(type) {
		case xml.StartElement:
			if !rooted {
				p.prefix, rooted = t.Name.Space, true
			}
			local := ""
			if t.Name.Space == p.prefix {
				local = t.Name.Local
			}
			parent := ""
			if len(stack) > 0 {
				parent = stack[len(stack)-1]
			}
			stack = append(stack, local)
			opened = append(opened, before)
			if err := p.start(t, local, parent, len(stack), before, after, &column, &filter); err != nil {
				return nil, err
			}
			if local == "calculatedColumnFormula" {
				formula.Reset()
			}
		case xml.CharData:
			if len(stack) > 0 && stack[len(stack)-1] == "calculatedColumnFormula" {
				formula.Write(t)
			}
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, fmt.Errorf("%w: %s: unbalanced elements", ErrMalformedTable, path)
			}
			local := stack[len(stack)-1]
			begin := opened[len(opened)-1]
			stack, opened = stack[:len(stack)-1], opened[:len(opened)-1]
			switch {
			case local == "calculatedColumnFormula" && column != nil:
				column.formula = formula.String()
			case local == "tableColumn" && column != nil:
				column.element = span{begin, after}
				p.columns = append(p.columns, *column)
				column = nil
			case local == "filterColumn" && filter != nil:
				filter.element = span{begin, after}
				p.filterColumns = append(p.filterColumns, *filter)
				filter = nil
			case local == "tableColumns":
				p.columnsEnd = before
			}
		}
	}
	if p.ref == (span{}) || p.columnsEnd < 0 || p.columnsCount == (span{}) {
		return nil, fmt.Errorf("%w: %s: missing ref or tableColumns", ErrMalformedTable, path)
	}
	return p, nil
}

// start records what Resize needs from a start tag found at depth (1 for the
// root element) between the offsets before and after.
func (p *tablePart) start(t xml.StartElement, local, parent string, depth, before, after int,
	column **tableColumn, filter **filterColumn) error {
	var err error
	switch {
	case depth == 1 && local == "table":
		if p.ref, err = p.attrSpan(before, after, "ref"); err != nil {
			return err
		}
		if v := attrValue(t, "headerRowCount"); v != "" {
			p.headerRows, err = strconv.Atoi(v)
		}
		if v := attrValue(t, "totalsRowCount"); v != "" && err == nil {
			p.totalsRows, err = strconv.Atoi(v)
		}
	case depth == 2 && local == "autoFilter":
		var s span
		if s, err = p.attrSpan(before, after, "ref"); err == nil {
			p.filterRef = &s
		}
	case depth == 2 && local == "tableColumns":
		p.columnsCount, err = p.attrSpan(before, after, "count")
	case depth == 3 && parent == "autoFilter" && local == "filterColumn":
		var id int
		id, err = strconv.Atoi(attrValue(t, "colId"))
		*filter = &filterColumn{colID: id}
	case depth == 3 && parent == "tableColumns" && local == "tableColumn":
		var id int
		id, err = strconv.Atoi(attrValue(t, "id"))
		*column = &tableColumn{id: id, name: attrValue(t, "name")}
	}
	if err != nil {
		return fmt.Errorf("%w: %s: <%s>: %w", ErrMalformedTable, p.path, local, err)
	}
	return nil
}

// attrSpan returns the span of the value of an unprefixed attribute of the
// start tag located between the offsets start and end.
func (p *tablePart) attrSpan(start, end int, name string) (span, error) {
	re := regexp.MustCompile(`\s` + regexp.QuoteMeta(name) + `\s*=\s*(?:"([^"]*)"|'([^']*)')`)
	loc := re.FindSubmatchIndex(p.data[start:end])
	if loc == nil {
		return span{}, fmt.Errorf("attribute %q not found", name)
	}
	if loc[2] >= 0 {
		return span{start + loc[2], start + loc[3]}, nil
	}
	return span{start + loc[4], start + loc[5]}, nil
}

// attrValue returns the value of an unprefixed attribute.
func attrValue(t xml.StartElement, name string) string {
	for _, a := range t.Attr {
		if a.Name.Space == "" && a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

// maxColumnID returns the highest column id of the table.
func (p *tablePart) maxColumnID() int {
	highest := 0
	for _, c := range p.columns {
		highest = max(highest, c.id)
	}
	return highest
}

// edit replaces the bytes of a span; an empty span inserts at its offset.
type edit struct {
	at   span
	with []byte
}

// rewrite returns the table part for the new range: ref and filter ref are
// updated, the columns beyond newCount are removed with their filters, and the
// added columns are appended.
func (p *tablePart) rewrite(to area, added []tableColumn) []byte {
	newCount := to.columns()
	edits := []edit{
		{p.ref, []byte(to.String())},
		{p.columnsCount, []byte(strconv.Itoa(newCount))},
	}
	if p.filterRef != nil {
		filtered := to
		filtered.bottom -= p.totalsRows
		edits = append(edits, edit{*p.filterRef, []byte(filtered.String())})
	}
	for _, fc := range p.filterColumns {
		if fc.colID >= newCount {
			edits = append(edits, edit{fc.element, nil})
		}
	}
	for i := newCount; i < len(p.columns); i++ {
		edits = append(edits, edit{p.columns[i].element, nil})
	}
	if len(added) > 0 {
		var b bytes.Buffer
		for _, c := range added {
			b.WriteString(`<` + p.qualified("tableColumn") + ` id="` + strconv.Itoa(c.id) + `" name="`)
			_ = xml.EscapeText(&b, []byte(c.name))
			b.WriteString(`"/>`)
		}
		edits = append(edits, edit{span{p.columnsEnd, p.columnsEnd}, b.Bytes()})
	}
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].at.start < edits[j].at.start })

	var out bytes.Buffer
	out.Grow(len(p.data) + 64*len(added))
	pos := 0
	for _, e := range edits {
		out.Write(p.data[pos:e.at.start])
		out.Write(e.with)
		pos = e.at.end
	}
	out.Write(p.data[pos:])
	return out.Bytes()
}

// qualified returns the name of an element in the namespace of the root.
func (p *tablePart) qualified(local string) string {
	if p.prefix == "" {
		return local
	}
	return p.prefix + ":" + local
}
