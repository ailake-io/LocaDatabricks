package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"regexp"
	"strings"
)

// ColumnSchema describes one column of a statement's result set.
type ColumnSchema struct {
	Name     string `json:"name"`
	TypeText string `json:"type_text"`
}

// StatementResult mirrors the shape of the Databricks SQL Statement
// Execution API (/api/2.0/sql/statements), executed synchronously here
// instead of asynchronously against a real warehouse.
type StatementResult struct {
	StatementID  string
	State        string // "SUCCEEDED" or "FAILED"
	ErrorMessage string
	Columns      []ColumnSchema
	Rows         [][]any
}

// returnsRows guesses whether a statement produces a result set (SELECT-like)
// versus a DDL/DML statement that must go through Exec instead of Query.
var returnsRowsRe = regexp.MustCompile(`(?i)^\s*(SELECT|WITH|SHOW|DESCRIBE|EXPLAIN|PRAGMA|SUMMARIZE)\b`)

// ExecuteStatement runs sqlText against the embedded DuckDB warehouse and
// caches the result under a generated statement ID for later retrieval via
// GetStatement (matching the real API's submit-then-poll shape, collapsed
// here into one synchronous call since there's no real cluster to wait on).
func (s *Store) ExecuteStatement(sqlText string) (*StatementResult, error) {
	id, err := randomID()
	if err != nil {
		return nil, err
	}

	result := &StatementResult{StatementID: id}

	if returnsRowsRe.MatchString(sqlText) {
		rows, err := s.warehouseDB.Query(sqlText)
		if err != nil {
			result.State = "FAILED"
			result.ErrorMessage = err.Error()
		} else {
			defer rows.Close()
			if err := scanRows(rows, result); err != nil {
				result.State = "FAILED"
				result.ErrorMessage = err.Error()
			} else {
				result.State = "SUCCEEDED"
			}
		}
	} else {
		if _, err := s.warehouseDB.Exec(sqlText); err != nil {
			result.State = "FAILED"
			result.ErrorMessage = err.Error()
		} else {
			result.State = "SUCCEEDED"
		}
	}

	s.mu.Lock()
	s.statements[id] = result
	s.mu.Unlock()

	return result, nil
}

func (s *Store) GetStatement(id string) (*StatementResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.statements[id]
	return r, ok
}

func scanRows(rows *sql.Rows, result *StatementResult) error {
	types, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	result.Columns = make([]ColumnSchema, len(types))
	for i, t := range types {
		result.Columns[i] = ColumnSchema{Name: t.Name(), TypeText: strings.ToUpper(t.DatabaseTypeName())}
	}

	for rows.Next() {
		vals := make([]any, len(types))
		ptrs := make([]any, len(types))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		result.Rows = append(result.Rows, vals)
	}
	return rows.Err()
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
