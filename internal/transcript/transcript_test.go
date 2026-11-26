package transcript

import (
	"fmt"
	"strings"
	"testing"
)

func TestSummarize(t *testing.T) {
	jsonl := strings.Join([]string{
		`{"step_index":0,"source":"USER_EXPLICIT","type":"USER_INPUT","content":"hi"}`,
		`{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":100,"cache_read_tokens":500,"output_tokens":20}`,
		`{"step_index":2,"source":"SYSTEM","type":"CHECKPOINT","input_tokens":9999,"cache_read_tokens":9999,"output_tokens":9999}`,
		`{"step_index":3,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":200,"cache_read_tokens":1000,"output_tokens":40}`,
		`{"step_index":4,"source":"MODEL","type":"PLANNER_RESPONSE"}`,
		`{"step_index":5,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":null,"cache_read_tokens":null,"output_tokens":null}`,
		`{"step_index":6,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":300,"cache_read_tokens":1500,"output_tokens":60}`,
	}, "\n") + "\n"

	cases := []struct {
		name         string
		input        string
		opts         Options
		wantIn       int64
		wantCache    int64
		wantOut      int64
		wantSteps    int
		wantSkipped  int
		wantLastSeen int
	}{
		{
			name:         "open ended from step 1",
			input:        jsonl,
			opts:         Options{FromStep: 1, ToStep: NoLimit},
			wantIn:       600,
			wantCache:    3000,
			wantOut:      120,
			wantSteps:    3,
			wantSkipped:  2,
			wantLastSeen: 6,
		},
		{
			name:         "inclusive window 1..3",
			input:        jsonl,
			opts:         Options{FromStep: 1, ToStep: 3},
			wantIn:       300,
			wantCache:    1500,
			wantOut:      60,
			wantSteps:    2,
			wantSkipped:  0,
			wantLastSeen: 6,
		},
		{
			name:         "past end returns zero steps",
			input:        jsonl,
			opts:         Options{FromStep: 7, ToStep: NoLimit},
			wantIn:       0,
			wantCache:    0,
			wantOut:      0,
			wantSteps:    0,
			wantSkipped:  0,
			wantLastSeen: 6,
		},
		{
			name:         "empty input",
			input:        "",
			opts:         Options{FromStep: 0, ToStep: NoLimit},
			wantIn:       0,
			wantCache:    0,
			wantOut:      0,
			wantSteps:    0,
			wantSkipped:  0,
			wantLastSeen: -1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Summarize(strings.NewReader(tc.input), tc.opts)
			if err != nil {
				t.Fatalf("Summarize: %v", err)
			}
			if got.Input != tc.wantIn || got.CacheRead != tc.wantCache || got.Output != tc.wantOut ||
				got.Steps != tc.wantSteps || got.SkippedNoUsage != tc.wantSkipped || got.LastStepSeen != tc.wantLastSeen {
				t.Errorf("got %+v, want in=%d cache=%d out=%d steps=%d skip=%d last=%d",
					got, tc.wantIn, tc.wantCache, tc.wantOut, tc.wantSteps, tc.wantSkipped, tc.wantLastSeen)
			}
		})
	}
}

func TestSummarizeErrors(t *testing.T) {
	cases := []struct {
		name string
		data string
		opts Options
	}{
		{
			name: "malformed json",
			data: `{"step_index":0,"source":"MODEL"` + "\n",
			opts: Options{FromStep: 0, ToStep: NoLimit},
		},
		{
			name: "missing step_index",
			data: `{"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":1,"cache_read_tokens":1,"output_tokens":1}` + "\n",
			opts: Options{FromStep: 0, ToStep: NoLimit},
		},
		{
			name: "non-integer step_index",
			data: `{"step_index":1.5,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":1,"cache_read_tokens":1,"output_tokens":1}` + "\n",
			opts: Options{FromStep: 0, ToStep: NoLimit},
		},
		{
			name: "negative step_index",
			data: `{"step_index":-1,"source":"USER_EXPLICIT","type":"USER_INPUT"}` + "\n",
			opts: Options{FromStep: 0, ToStep: NoLimit},
		},
		{
			name: "negative token out of window",
			data: `{"step_index":0,"source":"SYSTEM","type":"CHECKPOINT","input_tokens":-5,"cache_read_tokens":0,"output_tokens":0}` + "\n",
			opts: Options{FromStep: 10, ToStep: NoLimit},
		},
		{
			name: "non-integer token",
			data: `{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":1.5,"cache_read_tokens":0,"output_tokens":0}` + "\n",
			opts: Options{FromStep: 0, ToStep: NoLimit},
		},
		{
			name: "partial usage",
			data: `{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":10}` + "\n",
			opts: Options{FromStep: 0, ToStep: NoLimit},
		},
		{
			name: "int64 overflow",
			data: `{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":9223372036854775807,"cache_read_tokens":0,"output_tokens":0}` + "\n" +
				`{"step_index":2,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":1,"cache_read_tokens":0,"output_tokens":0}` + "\n",
			opts: Options{FromStep: 0, ToStep: NoLimit},
		},
		{
			name: "line exceeds MaxLineBytes",
			data: `{"step_index":0,"content":"` + strings.Repeat("a", 200) + `"}` + "\n",
			opts: Options{FromStep: 0, ToStep: NoLimit, MaxLineBytes: 64},
		},
		{
			name: "input exceeds MaxBytes",
			data: `{"step_index":0,"source":"USER_EXPLICIT","type":"USER_INPUT"}` + "\n" +
				`{"step_index":1,"source":"USER_EXPLICIT","type":"USER_INPUT"}` + "\n",
			opts: Options{FromStep: 0, ToStep: NoLimit, MaxBytes: 70},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Summarize(strings.NewReader(tc.data), tc.opts)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
			if got != (Summary{}) {
				t.Errorf("expected zero Summary on error, got %+v", got)
			}
			if !strings.Contains(err.Error(), "line ") {
				t.Errorf("expected line number in error, got %v", err)
			}
		})
	}

	// Invalid Options / nil reader
	if _, err := Summarize(nil, Options{FromStep: 0, ToStep: NoLimit}); err == nil {
		t.Errorf("expected error for nil reader")
	}
	for _, badOpts := range []Options{
		{FromStep: -1, ToStep: NoLimit},
		{FromStep: 0, ToStep: -2},
		{FromStep: 5, ToStep: 3},
	} {
		if _, err := Summarize(strings.NewReader(""), badOpts); err == nil {
			t.Errorf("expected error for invalid options %+v", badOpts)
		}
	}
}

func TestNoContentLeak(t *testing.T) {
	sentinel := "SECRET_PROMPT_SENTINEL_94827"
	// Valid record followed by malformed JSON containing the sentinel
	data := fmt.Sprintf(`{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":1,"cache_read_tokens":2,"output_tokens":3,"content":%q}`+"\n"+
		`{"step_index":2,"content":%q,bad`, sentinel, sentinel)
	_, err := Summarize(strings.NewReader(data), Options{FromStep: 0, ToStep: NoLimit})
	if err == nil {
		t.Fatalf("expected error on malformed line")
	}
	if strings.Contains(err.Error(), sentinel) {
		t.Errorf("error leaked transcript content: %v", err)
	}
}

func BenchmarkSummarize(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&sb, `{"step_index":%d,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":100,"cache_read_tokens":5000,"output_tokens":50,"content":"synthetic payload"}`+"\n", i)
	}
	payload := sb.String()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Summarize(strings.NewReader(payload), Options{FromStep: 0, ToStep: NoLimit}); err != nil {
			b.Fatal(err)
		}
	}
}
