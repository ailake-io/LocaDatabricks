package store

import "testing"

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestClusterLifecycle(t *testing.T) {
	s := newTestStore(t)

	c := s.CreateCluster("dev")
	if c.State != "RUNNING" {
		t.Errorf("new cluster state = %q, want RUNNING", c.State)
	}

	got, ok := s.GetCluster(c.ClusterID)
	if !ok || got.ClusterID != c.ClusterID {
		t.Fatalf("GetCluster(%q) = %v, %v", c.ClusterID, got, ok)
	}

	if !s.DeleteCluster(c.ClusterID) {
		t.Fatal("DeleteCluster returned false for an existing cluster")
	}
	got, _ = s.GetCluster(c.ClusterID)
	if got.State != "TERMINATED" {
		t.Errorf("state after delete = %q, want TERMINATED", got.State)
	}

	if s.DeleteCluster("does-not-exist") {
		t.Error("DeleteCluster returned true for a nonexistent cluster")
	}
}

func TestJobRunLifecycle(t *testing.T) {
	s := newTestStore(t)

	job := s.CreateJob("etl", "/workspace/etl.py")
	got, ok := s.GetJob(job.JobID)
	if !ok || got.ScriptPath != "/workspace/etl.py" {
		t.Fatalf("GetJob(%d) = %v, %v", job.JobID, got, ok)
	}

	run := s.NewRun()
	if run.LifeCycleState != "RUNNING" {
		t.Errorf("new run state = %q, want RUNNING", run.LifeCycleState)
	}

	s.SetRunResult(run.RunID, "TERMINATED", "SUCCESS")
	updated, ok := s.GetRun(run.RunID)
	if !ok || updated.LifeCycleState != "TERMINATED" || updated.ResultState != "SUCCESS" {
		t.Fatalf("GetRun(%d) after SetRunResult = %+v, %v", run.RunID, updated, ok)
	}

	if _, ok := s.GetRun(99999); ok {
		t.Error("GetRun found a run that was never created")
	}
}

func TestUnityCatalogRegistry(t *testing.T) {
	s := newTestStore(t)

	if err := s.CreateCatalog("main", "test catalog"); err != nil {
		t.Fatalf("CreateCatalog: %v", err)
	}
	if err := s.CreateSchema("main", "default", ""); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	if err := s.CreateTable("main", "default", "orders", "DELTA", "/tmp/orders"); err != nil {
		t.Fatalf("CreateTable: %v", err)
	}

	tables, err := s.ListTables("main", "default")
	if err != nil {
		t.Fatalf("ListTables: %v", err)
	}
	if len(tables) != 1 || tables[0].Name != "orders" {
		t.Fatalf("ListTables = %+v, want one table named orders", tables)
	}

	// Duplicate catalog name should fail (primary key).
	if err := s.CreateCatalog("main", "again"); err == nil {
		t.Error("expected CreateCatalog to reject a duplicate name")
	}
}
