package store

import "testing"

func TestExecuteStatement_DDLThenSelect(t *testing.T) {
	s := newTestStore(t)

	if r, err := s.ExecuteStatement("CREATE TABLE orders (id INTEGER, amount DOUBLE)"); err != nil || r.State != "SUCCEEDED" {
		t.Fatalf("CREATE TABLE: state=%v err=%v msg=%v", r, err, r.ErrorMessage)
	}
	if r, err := s.ExecuteStatement("INSERT INTO orders VALUES (1, 9.5), (2, 3.0)"); err != nil || r.State != "SUCCEEDED" {
		t.Fatalf("INSERT: state=%v err=%v", r, err)
	}

	r, err := s.ExecuteStatement("SELECT id, amount FROM orders ORDER BY id")
	if err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	if r.State != "SUCCEEDED" {
		t.Fatalf("SELECT state = %s, err = %s", r.State, r.ErrorMessage)
	}
	if len(r.Columns) != 2 || r.Columns[0].Name != "id" {
		t.Fatalf("columns = %+v", r.Columns)
	}
	if len(r.Rows) != 2 {
		t.Fatalf("rows = %+v, want 2", r.Rows)
	}
}

func TestExecuteStatement_InvalidSQLFails(t *testing.T) {
	s := newTestStore(t)

	r, err := s.ExecuteStatement("SELEKT nonsense")
	if err != nil {
		t.Fatalf("ExecuteStatement returned a transport error: %v", err)
	}
	if r.State != "FAILED" || r.ErrorMessage == "" {
		t.Fatalf("expected a FAILED result with an error message, got %+v", r)
	}
}

func TestGetStatement_RoundTrip(t *testing.T) {
	s := newTestStore(t)

	r, err := s.ExecuteStatement("SELECT 1 AS one")
	if err != nil {
		t.Fatal(err)
	}

	got, ok := s.GetStatement(r.StatementID)
	if !ok {
		t.Fatal("GetStatement did not find the statement just executed")
	}
	if got.State != "SUCCEEDED" || len(got.Rows) != 1 {
		t.Fatalf("got = %+v", got)
	}

	if _, ok := s.GetStatement("does-not-exist"); ok {
		t.Error("GetStatement found a statement id that was never created")
	}
}
