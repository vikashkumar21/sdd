package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// prompter asks the human questions on a terminal. End of input answers every
// question with its default, except confirmations, which default to "no".
type prompter struct {
	r *bufio.Reader
	w io.Writer
}

func newPrompter(in io.Reader, out io.Writer) *prompter {
	return &prompter{r: bufio.NewReader(in), w: out}
}

func (p *prompter) ask(label, def string) string {
	if def != "" {
		fprintf(p.w, "%s [%s]: ", label, def)
	} else {
		fprintf(p.w, "%s: ", label)
	}
	line, err := p.r.ReadString('\n')
	line = strings.TrimSpace(strings.TrimRight(line, "\r\n"))
	if err != nil && line == "" {
		fprintln(p.w)
		return def
	}
	if line == "" {
		return def
	}
	return line
}

// askValid keeps asking (up to three times) until validate accepts the answer.
func (p *prompter) askValid(label, def string, validate func(string) error) (string, error) {
	var err error
	for i := 0; i < 3; i++ {
		v := p.ask(label, def)
		if err = validate(v); err == nil {
			return v, nil
		}
		fprintf(p.w, "  %v\n", err)
	}
	return "", fmt.Errorf("%s: no valid answer after 3 attempts: %w", label, err)
}

func (p *prompter) confirm(label string) bool {
	fprintf(p.w, "%s [y/N]: ", label)
	line, _ := p.r.ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	}
	return false
}

// askLines collects lines until a blank line or end of input.
func (p *prompter) askLines(label string) []string {
	fprintf(p.w, "%s (one per line, blank line to finish)\n", label)
	var out []string
	for {
		fprint(p.w, "  > ")
		line, err := p.r.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			if err != nil {
				fprintln(p.w)
			}
			return out
		}
		out = append(out, line)
	}
}
