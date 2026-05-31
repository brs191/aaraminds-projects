package teams

import "testing"

func TestStubModeNotDelivered(t *testing.T) {
	p := NewPoster("")
	delivered, err := p.Post(Message{Title: "t", Markdown: "m"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if delivered {
		t.Fatal("expected delivered=false in stub mode (no webhook)")
	}
}
