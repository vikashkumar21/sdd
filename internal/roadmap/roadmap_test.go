package roadmap

import "testing"

const sample = `| Milestone | Tasks |
|---|---|
| M0 | T-000 |

| ID | Title | Size | Status |
|---|---|---|---|
| T-000 | Scaffolding | S | Done (` + "`COMMITTED T-000`" + `) |
| T-000.1 | Amendments | S | **Awaiting review** |
| T-001 | Skeleton | S | Blocked (D-030: handle needed) |
| T-002 | Spike | L | Not started |
`

func TestParse(t *testing.T) {
	tasks := Parse(sample)
	if len(tasks) != 4 {
		t.Fatalf("got %d tasks: %+v", len(tasks), tasks)
	}
	want := []struct{ id, title, kind string }{
		{"T-000", "Scaffolding", "done"}, {"T-000.1", "Amendments", "awaiting review"},
		{"T-001", "Skeleton", "blocked"}, {"T-002", "Spike", "not started"},
	}
	for i, w := range want {
		if tasks[i].ID != w.id || tasks[i].Title != w.title || Kind(tasks[i].Status) != w.kind {
			t.Errorf("task %d = %+v, want %+v", i, tasks[i], w)
		}
	}
	if tasks[0].Line != 7 {
		t.Errorf("line = %d, want 7", tasks[0].Line)
	}
}

func TestValidStatus(t *testing.T) {
	for _, ok := range []string{"Not started", "**Awaiting review**", "Done (x)", "blocked (y)", "Specifying", "Implementing", "Awaiting plan approval"} {
		if !ValidStatus(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "Almost done", "WIP"} {
		if ValidStatus(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func TestSetStatus(t *testing.T) {
	got, ok := SetStatus(sample, "T-002", "Not started", "Specifying")
	if !ok {
		t.Fatal("expected a change")
	}
	if Kind(Parse(got)[3].Status) != "specifying" {
		t.Errorf("status not updated: %+v", Parse(got)[3])
	}
	if _, ok := SetStatus(sample, "T-001", "Not started", "Specifying"); ok {
		t.Error("must not change a Blocked task")
	}
	if _, ok := SetStatus(sample, "T-099", "Not started", "Specifying"); ok {
		t.Error("must not change an unknown task")
	}
	if got2, _ := SetStatus(sample, "T-002", "Not started", "Specifying"); got2 == sample {
		t.Error("text should differ after a successful change")
	}
}
