package models

import "strconv"

// ValuationMeasures represents Yahoo Finance's key-statistics valuation table.
type ValuationMeasures struct {
	// Columns are the table value columns, such as "Current" and quarter-end dates.
	Columns []string `json:"columns"`

	// Rows contains one row per valuation measure.
	Rows []ValuationMeasureRow `json:"rows"`
}

// ValuationMeasureRow represents one row in the valuation measures table.
type ValuationMeasureRow struct {
	// Name is the valuation measure label, such as "Market Cap".
	Name string `json:"name"`

	// Values maps each column name to a string representation of the raw value.
	//
	// This is kept for compatibility with go-yfinance versions that exposed
	// valuation cells as strings from Yahoo's key-statistics HTML page.
	Values map[string]string `json:"values"`

	// RawValues maps each column name to Yahoo's raw numeric value.
	// Missing cells are stored as nil.
	RawValues map[string]*float64 `json:"rawValues,omitempty"`
}

// Empty reports whether the valuation table has no rows.
func (v *ValuationMeasures) Empty() bool {
	return v == nil || len(v.Rows) == 0
}

// Value returns a value by row name and column name.
func (v *ValuationMeasures) Value(rowName, columnName string) (string, bool) {
	if v == nil {
		return "", false
	}
	for _, row := range v.Rows {
		if row.Name == rowName {
			value, ok := row.Values[columnName]
			return value, ok
		}
	}
	return "", false
}

// FloatValue returns the raw numeric value by row name and column name.
// It returns false when the row/column is missing or the cell has no value.
func (v *ValuationMeasures) FloatValue(rowName, columnName string) (float64, bool) {
	if v == nil {
		return 0, false
	}
	for _, row := range v.Rows {
		if row.Name != rowName {
			continue
		}
		value, ok := row.RawValues[columnName]
		if !ok || value == nil {
			return 0, false
		}
		return *value, true
	}
	return 0, false
}

// SetValue stores a raw numeric valuation cell while maintaining the
// compatibility string map.
func (r *ValuationMeasureRow) SetValue(column string, value *float64) {
	if r.Values == nil {
		r.Values = make(map[string]string)
	}
	if r.RawValues == nil {
		r.RawValues = make(map[string]*float64)
	}
	r.RawValues[column] = value
	if value == nil {
		r.Values[column] = ""
		return
	}
	r.Values[column] = strconv.FormatFloat(*value, 'g', -1, 64)
}
