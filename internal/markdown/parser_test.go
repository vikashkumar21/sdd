package markdown

import (
	"testing"
)

func TestCodeFences(t *testing.T) {
	tests := []struct {
		name    string
		doc     string
		wantErr bool
	}{
		{
			name:    "simple backtick fence closed",
			doc:     "```go\nfmt.Println(\"ok\")\n```",
			wantErr: false,
		},
		{
			name:    "simple tilde fence closed",
			doc:     "~~~python\nprint('ok')\n~~~",
			wantErr: false,
		},
		{
			name:    "unclosed backtick fence",
			doc:     "```go\nfmt.Println(\"unclosed\")\n",
			wantErr: true,
		},
		{
			name:    "unclosed tilde fence",
			doc:     "~~~python\nprint('unclosed')\n",
			wantErr: true,
		},
		{
			name:    "mismatched fence chars (opened with backtick, closed with tilde)",
			doc:     "```\ncontent\n~~~",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := CheckFences(tt.doc)
			if (len(findings) > 0) != tt.wantErr {
				t.Fatalf("CheckFences() findings = %v, wantErr = %v", findings, tt.wantErr)
			}
		})
	}
}

func TestNestedCodeFences(t *testing.T) {
	// 4 backticks enclosing 3 backticks (e.g. carousel snippets)
	nested := "````carousel\nHere is an inner block:\n```python\nprint('hello')\n```\n<!-- slide -->\n```go\nprintln(\"world\")\n```\n````\n"
	findings := CheckFences(nested)
	if len(findings) > 0 {
		t.Fatalf("expected 4-backtick fence enclosing 3-backtick fence to pass cleanly, got: %+v", findings)
	}

	// Unclosed outer 4-backtick fence
	unclosedOuter := "````carousel\n```python\nprint('hello')\n```\n"
	findingsUnclosed := CheckFences(unclosedOuter)
	if len(findingsUnclosed) == 0 {
		t.Fatal("expected unclosed 4-backtick fence to trigger an error, got 0 findings")
	}
	if findingsUnclosed[0].Line != 1 {
		t.Errorf("expected error on line 1, got line %d", findingsUnclosed[0].Line)
	}
}

func TestInlineCodePipes(t *testing.T) {
	doc := "|\tCommand\t|\tPurpose\t|\n|---|---|\n| `grep | wc -l` | Counts matching lines |\n| `cat file | sort -u` | Sorts and dedupes |\n"
	tables := ExtractTables(doc)
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	tb := tables[0]
	if tb.Header.Columns != 2 {
		t.Errorf("header columns = %d, want 2", tb.Header.Columns)
	}
	for i, r := range tb.Rows {
		if r.Columns != 2 {
			t.Errorf("row %d columns = %d, want 2", i, r.Columns)
		}
	}
}

func TestEscapedPipes(t *testing.T) {
	doc := "|\tOption\t|\tDescription\t|\n|---|---|\n| A \\| B | Both options A and B |\n| C | Option C |\n"
	tables := ExtractTables(doc)
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	tb := tables[0]
	if tb.Header.Columns != 2 {
		t.Errorf("header columns = %d, want 2", tb.Header.Columns)
	}
	for i, r := range tb.Rows {
		if r.Columns != 2 {
			t.Errorf("row %d columns = %d, want 2", i, r.Columns)
		}
	}
}

func TestTableInsideCodeFenceIgnored(t *testing.T) {
	doc := "Here is a table in documentation:\n\n| Name | Role |\n|---|---|\n| Alice | Dev |\n\nAnd here is a code fence with a fake table:\n```markdown\n| Not | A | Real | Table |\n```\n"
	tables := ExtractTables(doc)
	if len(tables) != 1 {
		t.Fatalf("expected only 1 table (ignoring table inside code block), got %d", len(tables))
	}
	if tables[0].Header.Columns != 2 {
		t.Errorf("expected 2 columns, got %d", tables[0].Header.Columns)
	}
}
