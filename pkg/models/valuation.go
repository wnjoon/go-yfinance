package models

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

	// Values maps each column name to the displayed Yahoo Finance value.
	Values map[string]string `json:"values"`
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
