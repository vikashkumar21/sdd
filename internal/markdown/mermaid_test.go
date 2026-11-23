package markdown_test

import (
	"testing"

	"github.com/vikashkumar21/sdd/internal/markdown"
)

func TestCheckMermaid_ValidDiagrams(t *testing.T) {
	tests := []struct {
		name string
		md   string
	}{
		{
			name: "valid flowchart with various node shapes",
			md: "```mermaid\n" +
				"flowchart TD\n" +
				"    A[Standard Box] --> B(Rounded Box)\n" +
				"    B --> C([Stadium Node])\n" +
				"    C --> D[[Subroutine]]\n" +
				"    D --> E[(Database Cylinder)]\n" +
				"    E --> F((Circle Node))\n" +
				"    F --> G{\"Decision?\"}\n" +
				"    G -->|Yes| H[Done]\n" +
				"```",
		},
		{
			name: "cylinder shape with quotes and brackets inside",
			md: "```mermaid\n" +
				"flowchart TD\n" +
				"    DB[(\"User Store [PostgreSQL]\")] --> C[\"Label with (parens) and [brackets]\"]\n" +
				"```",
		},
		{
			name: "comments with unmatched delimiters",
			md: "```mermaid\n" +
				"flowchart LR\n" +
				"    %% This comment has unclosed ( paren and [ bracket\n" +
				"    A --> B %% another comment with )\n" +
				"```",
		},
		{
			name: "sequence diagram with messages containing colons and parens",
			md: "```mermaid\n" +
				"sequenceDiagram\n" +
				"    autonumber\n" +
				"    actor User\n" +
				"    participant System\n" +
				"    User->>System: 1) Initiate login (with params)\n" +
				"    Note over User,System: Step 2) Validate session\n" +
				"    System-->>User: 3) 200 OK (JWT token)\n" +
				"```",
		},
		{
			name: "er diagram with crow foot relationships and entity blocks",
			md: "```mermaid\n" +
				"erDiagram\n" +
				"    CUSTOMER ||--o{ ORDER : places\n" +
				"    ORDER ||--|{ ORDER_ITEM : contains\n" +
				"    CUSTOMER {\n" +
				"        string id PK\n" +
				"        string name\n" +
				"    }\n" +
				"```",
		},
		{
			name: "er diagram with odd pipe cardinality and entity attributes",
			md: "```mermaid\n" +
				"erDiagram\n" +
				"    ORDER ||--|{ ORDER_ITEM : contains\n" +
				"    CUSTOMER {\n" +
				"        string id PK\n" +
				"        string name \"[Required] Full Name\"\n" +
				"    }\n" +
				"```",
		},
		{
			name: "state diagram with composite states",
			md: "```mermaid\n" +
				"stateDiagram-v2\n" +
				"    [*] --> Idle\n" +
				"    state Processing {\n" +
				"        [*] --> Validating\n" +
				"        Validating --> Ready\n" +
				"    }\n" +
				"    Processing --> Completed\n" +
				"```",
		},
		{
			name: "class diagram",
			md: "```mermaid\n" +
				"classDiagram\n" +
				"    class Account {\n" +
				"        +String id\n" +
				"        +deposit(amount)\n" +
				"    }\n" +
				"```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := markdown.CheckMermaid(tt.md)
			if len(findings) > 0 {
				t.Fatalf("expected 0 findings for valid mermaid, got %d: %+v", len(findings), findings)
			}
		})
	}
}

func TestCheckMermaid_InvalidDiagrams(t *testing.T) {
	tests := []struct {
		name        string
		md          string
		expectedMsg string
	}{
		{
			name: "empty mermaid block",
			md: "```mermaid\n" +
				"\n" +
				"```",
			expectedMsg: "empty Mermaid code block",
		},
		{
			name: "unrecognized diagram declaration",
			md: "```mermaid\n" +
				"unknownTopology TD\n" +
				"    A --> B\n" +
				"```",
			expectedMsg: "unrecognized or missing Mermaid diagram type",
		},
		{
			name: "unclosed bracket",
			md: "```mermaid\n" +
				"flowchart TD\n" +
				"    A[Unclosed Box --> B\n" +
				"```",
			expectedMsg: "unclosed delimiter '['",
		},
		{
			name: "unexpected closing bracket",
			md: "```mermaid\n" +
				"flowchart TD\n" +
				"    A --> B]\n" +
				"```",
			expectedMsg: "unexpected closing bracket ']'",
		},
		{
			name: "unclosed parenthesis",
			md: "```mermaid\n" +
				"flowchart TD\n" +
				"    A(Unclosed Round\n" +
				"```",
			expectedMsg: "unclosed delimiter '('",
		},
		{
			name: "unexpected closing brace",
			md: "```mermaid\n" +
				"classDiagram\n" +
				"    class Foo\n" +
				"    }\n" +
				"```",
			expectedMsg: "unexpected closing brace '}'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := markdown.CheckMermaid(tt.md)
			if len(findings) == 0 {
				t.Fatalf("expected findings for invalid mermaid %s, got 0", tt.name)
			}
			found := false
			for _, f := range findings {
				if f.Rule == "mermaid-syntax" && (tt.expectedMsg == "" || contains(f.Message, tt.expectedMsg)) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected finding matching %q, got: %+v", tt.expectedMsg, findings)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr))))
}
