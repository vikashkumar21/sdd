package block

import "testing"

func TestLocateAndInner(t *testing.T) {
	doc := "# title\n\nhuman text\n\n" + Begin("1.0.0") + "\nbody line\n" + End + "\n\ntail\n"
	if !Has(doc) {
		t.Fatal("block not found")
	}
	inner, ok := Inner(doc)
	if !ok || inner != "body line" {
		t.Fatalf("inner = %q, ok=%v", inner, ok)
	}
	bs, _, _, be, _ := Locate(doc)
	if doc[:bs] != "# title\n\nhuman text\n\n" || doc[be:] != "\n\ntail\n" {
		t.Errorf("outside-the-block text not preserved: %q | %q", doc[:bs], doc[be:])
	}
}

func TestMalformedBlocksAreNotFound(t *testing.T) {
	for name, doc := range map[string]string{
		"no markers":       "just text",
		"begin only":       Begin("1") + "\nbody",
		"unterminated tag": "<!-- sdd:begin contract",
		"end only":         End,
	} {
		if Has(doc) {
			t.Errorf("%s: should not be treated as a block", name)
		}
	}
}
