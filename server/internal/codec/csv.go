package codec

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
)

const (
	maxCSVRows      = 1_000_000
	maxCSVCellBytes = 1 << 20 // 1 MiB per cell
)

// ValidateCSV checks that raw parses as CSV within row/cell bounds. Ragged rows
// (varying field counts) are allowed; the bounds guard against pathological
// in-memory expansion.
func ValidateCSV(raw []byte) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		return &ParseError{Code: "E_EMPTY", Message: "empty CSV document"}
	}
	r := csv.NewReader(bytes.NewReader(raw))
	r.FieldsPerRecord = -1 // allow ragged rows
	r.ReuseRecord = true

	rows := 0
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return &ParseError{Code: "E_BAD_CSV", Message: err.Error()}
		}
		if rows++; rows > maxCSVRows {
			return &ParseError{Code: "E_TOO_MANY", Message: "too many CSV rows"}
		}
		for _, cell := range rec {
			if len(cell) > maxCSVCellBytes {
				return &ParseError{Code: "E_CELL_TOO_LARGE", Message: "CSV cell exceeds size limit"}
			}
		}
	}
	if rows == 0 {
		return &ParseError{Code: "E_EMPTY", Message: "CSV has no rows"}
	}
	return nil
}
