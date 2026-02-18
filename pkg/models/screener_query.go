package models

import (
	"fmt"
	"strings"
)

// ScreenerQueryBuilder is the interface for both EquityQuery and FundQuery.
// It provides methods for serializing queries and identifying the quote type.
type ScreenerQueryBuilder interface {
	// ToDict serializes the query to a map suitable for JSON encoding.
	// IS-IN operators are expanded to OR-of-EQ queries.
	ToDict() map[string]any

	// QuoteType returns "EQUITY" for EquityQuery or "MUTUALFUND" for FundQuery.
	QuoteType() string
}

// queryBase holds the common query state and validation logic.
type queryBase struct {
	operator string
	operands []any
}

// EquityQuery represents a validated equity screener query.
type EquityQuery struct {
	queryBase
}

// FundQuery represents a validated fund screener query.
type FundQuery struct {
	queryBase
}

// NewEquityQuery creates a new validated equity screener query.
// Operator is case-insensitive and will be uppercased.
// Valid operators: "EQ", "GT", "LT", "GTE", "LTE", "BTWN", "AND", "OR", "IS-IN"
//
// Example:
//
//	q, err := models.NewEquityQuery("and", []any{
//	    mustEquityQuery("eq", []any{"region", "us"}),
//	    mustEquityQuery("gt", []any{"intradayprice", 10}),
//	})
func NewEquityQuery(operator string, operands []any) (*EquityQuery, error) {
	op := strings.ToUpper(operator)
	q := &EquityQuery{queryBase{operator: op, operands: operands}}
	if err := q.validate(); err != nil {
		return nil, err
	}
	return q, nil
}

// NewFundQuery creates a new validated fund screener query.
// Same operator rules as NewEquityQuery.
func NewFundQuery(operator string, operands []any) (*FundQuery, error) {
	op := strings.ToUpper(operator)
	q := &FundQuery{queryBase{operator: op, operands: operands}}
	if err := q.validate(); err != nil {
		return nil, err
	}
	return q, nil
}

// QuoteType returns "EQUITY".
func (q *EquityQuery) QuoteType() string { return "EQUITY" }

// QuoteType returns "MUTUALFUND".
func (q *FundQuery) QuoteType() string { return "MUTUALFUND" }

// validFields returns the flattened set of valid field names for equity screener.
func (q *EquityQuery) validFields() map[string]bool {
	return allEquityValidFields()
}

// validFields returns the flattened set of valid field names for fund screener.
func (q *FundQuery) validFields() map[string]bool {
	return allFundValidFields()
}

// validValues returns the constrained field->values map for equity screener.
// Fields: "exchange" -> all equity exchange codes, "region" -> all regions,
// "sector" -> all sectors, "industry" -> all industries, "peer_group" -> all peer groups.
func (q *EquityQuery) validValues() map[string]map[string]bool {
	return map[string]map[string]bool{
		"exchange":   allExchangeCodes(EquityScreenerExchangeMap),
		"region":     regionCodes(EquityScreenerExchangeMap),
		"sector":     sliceToSet(EquityScreenerSectors),
		"industry":   allIndustries(),
		"peer_group": sliceToSet(EquityScreenerPeerGroups),
	}
}

// validValues returns the constrained field->values map for fund screener.
func (q *FundQuery) validValues() map[string]map[string]bool {
	return map[string]map[string]bool{
		"exchange": allExchangeCodes(FundScreenerExchangeMap),
	}
}

// sliceToSet converts a string slice to a bool map.
func sliceToSet(s []string) map[string]bool {
	m := make(map[string]bool, len(s))
	for _, v := range s {
		m[v] = true
	}
	return m
}

// validate performs all validation checks on the query.
func (q *EquityQuery) validate() error {
	return validateQuery(q.operator, q.operands, q.validFields(), q.validValues(), "EquityQuery")
}

func (q *FundQuery) validate() error {
	return validateQuery(q.operator, q.operands, q.validFields(), q.validValues(), "FundQuery")
}

// validateQuery is the shared validation logic matching Python's QueryBase.__init__.
func validateQuery(operator string, operands []any, validFields map[string]bool, validValues map[string]map[string]bool, typeName string) error {
	if len(operands) == 0 {
		return fmt.Errorf("invalid field for EquityQuery")
		// NOTE: Python has a bug - it says "EquityQuery" even for FundQuery. We replicate it.
	}

	switch operator {
	case "IS-IN":
		return validateISIN(operands, validFields, validValues, typeName)
	case "OR", "AND":
		return validateORAND(operands)
	case "EQ":
		return validateEQ(operands, validFields, validValues, typeName)
	case "BTWN":
		return validateBTWN(operands, validFields, typeName)
	case "GT", "LT", "GTE", "LTE":
		return validateGTLT(operands, validFields, typeName)
	default:
		return fmt.Errorf("invalid Operator Value")
	}
}

func validateORAND(operands []any) error {
	if len(operands) <= 1 {
		return fmt.Errorf("operand must be length longer than 1")
	}
	for _, op := range operands {
		if _, ok := op.(ScreenerQueryBuilder); !ok {
			return fmt.Errorf("operand must be type ScreenerQueryBuilder for OR/AND")
		}
	}
	return nil
}

func validateEQ(operands []any, validFields map[string]bool, validValues map[string]map[string]bool, typeName string) error {
	if len(operands) != 2 {
		return fmt.Errorf("operand must be length 2 for EQ")
	}
	fieldName, ok := operands[0].(string)
	if !ok {
		return fmt.Errorf("first operand must be a string field name")
	}
	if !validFields[fieldName] {
		return fmt.Errorf("invalid field for %s %q", typeName, fieldName)
	}
	// Check value constraint if field has one
	if vv, exists := validValues[fieldName]; exists {
		valStr := fmt.Sprintf("%v", operands[1])
		if !vv[valStr] {
			return fmt.Errorf("invalid EQ value %q", operands[1])
		}
	}
	return nil
}

func validateBTWN(operands []any, validFields map[string]bool, typeName string) error {
	if len(operands) != 3 {
		return fmt.Errorf("operand must be length 3 for BTWN")
	}
	fieldName, ok := operands[0].(string)
	if !ok {
		return fmt.Errorf("first operand must be a string field name")
	}
	if !validFields[fieldName] {
		return fmt.Errorf("invalid field for %s", typeName)
	}
	if !isNumeric(operands[1]) {
		return fmt.Errorf("invalid comparison type for BTWN")
	}
	if !isNumeric(operands[2]) {
		return fmt.Errorf("invalid comparison type for BTWN")
	}
	return nil
}

func validateGTLT(operands []any, validFields map[string]bool, typeName string) error {
	if len(operands) != 2 {
		return fmt.Errorf("operand must be length 2 for GT/LT")
	}
	fieldName, ok := operands[0].(string)
	if !ok {
		return fmt.Errorf("first operand must be a string field name")
	}
	if !validFields[fieldName] {
		return fmt.Errorf("invalid field for %s %q", typeName, fieldName)
	}
	if !isNumeric(operands[1]) {
		return fmt.Errorf("invalid comparison type for GT/LT")
	}
	return nil
}

func validateISIN(operands []any, validFields map[string]bool, validValues map[string]map[string]bool, typeName string) error {
	if len(operands) < 2 {
		return fmt.Errorf("operand must be length 2+ for IS-IN")
	}
	fieldName, ok := operands[0].(string)
	if !ok {
		return fmt.Errorf("first operand must be a string field name")
	}
	if !validFields[fieldName] {
		return fmt.Errorf("invalid field for %s %q", typeName, fieldName)
	}
	if vv, exists := validValues[fieldName]; exists {
		for i := 1; i < len(operands); i++ {
			valStr := fmt.Sprintf("%v", operands[i])
			if !vv[valStr] {
				return fmt.Errorf("invalid EQ value %q", operands[i])
			}
		}
	}
	return nil
}

// isNumeric checks if a value is a numeric type (int, int64, float64, etc.).
func isNumeric(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	}
	return false
}

// ToDict serializes the EquityQuery to a map for JSON encoding.
// IS-IN is expanded to OR of EQ queries.
func (q *EquityQuery) ToDict() map[string]any {
	return toDict(q.operator, q.operands, func(op string, operands []any) ScreenerQueryBuilder {
		eq := &EquityQuery{queryBase{operator: op, operands: operands}}
		return eq
	})
}

// ToDict serializes the FundQuery to a map for JSON encoding.
func (q *FundQuery) ToDict() map[string]any {
	return toDict(q.operator, q.operands, func(op string, operands []any) ScreenerQueryBuilder {
		fq := &FundQuery{queryBase{operator: op, operands: operands}}
		return fq
	})
}

// toDict is the shared serialization logic.
func toDict(operator string, operands []any, makeChild func(string, []any) ScreenerQueryBuilder) map[string]any {
	op := operator
	ops := operands

	if operator == "IS-IN" {
		// Expand to OR of EQ queries
		op = "OR"
		expanded := make([]any, 0, len(operands)-1)
		fieldName := operands[0]
		for i := 1; i < len(operands); i++ {
			child := makeChild("EQ", []any{fieldName, operands[i]})
			expanded = append(expanded, child)
		}
		ops = expanded
	}

	result := map[string]any{
		"operator": op,
	}

	serializedOps := make([]any, len(ops))
	for i, o := range ops {
		if qb, ok := o.(ScreenerQueryBuilder); ok {
			serializedOps[i] = qb.ToDict()
		} else {
			serializedOps[i] = o
		}
	}
	result["operands"] = serializedOps

	return result
}

// String returns a human-readable representation of the query.
func (q *EquityQuery) String() string {
	return queryString("EquityQuery", q.operator, q.operands, 0)
}

// String returns a human-readable representation of the query.
func (q *FundQuery) String() string {
	return queryString("FundQuery", q.operator, q.operands, 0)
}

func queryString(typeName, operator string, operands []any, indent int) string {
	prefix := strings.Repeat("  ", indent)
	hasNested := false
	for _, op := range operands {
		if _, ok := op.(ScreenerQueryBuilder); ok {
			hasNested = true
			break
		}
	}

	if hasNested {
		parts := make([]string, len(operands))
		for i, op := range operands {
			if eq, ok := op.(*EquityQuery); ok {
				parts[i] = queryString("EquityQuery", eq.operator, eq.operands, indent+1)
			} else if fq, ok := op.(*FundQuery); ok {
				parts[i] = queryString("FundQuery", fq.operator, fq.operands, indent+1)
			} else {
				parts[i] = fmt.Sprintf("%s  %v", prefix, op)
			}
		}
		return fmt.Sprintf("%s%s(%s,\n%s)", prefix, typeName, operator, strings.Join(parts, ",\n"))
	}
	return fmt.Sprintf("%s%s(%s, %v)", prefix, typeName, operator, operands)
}
