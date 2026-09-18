package extractor

import "testing"

func TestParseCandidateOutputAcceptsFencedJSON(t *testing.T) {
	raw := "```json\n{\"candidates\":[{\"scope\":\"project\",\"kind\":\"project_fact\",\"memory\":\"ContextGateway prepares turns.\",\"confidence\":0.9}]}\n```"
	got, err := ParseCandidateOutput(raw)
	if err != nil {
		t.Fatalf("ParseCandidateOutput() = %v", err)
	}
	if len(got) != 1 || got[0].Memory != "ContextGateway prepares turns." {
		t.Fatalf("candidates = %+v", got)
	}
}

func TestParseCandidateOutputRejectsInvalidCandidate(t *testing.T) {
	raw := `{"candidates":[{"scope":"project","kind":"project_fact","memory":"","confidence":0.9}]}`
	if _, err := ParseCandidateOutput(raw); err == nil {
		t.Fatal("invalid candidate should fail")
	}
}
