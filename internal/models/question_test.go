package models

import "testing"

func TestIsValidQuestionStatus(t *testing.T) {
	for _, s := range []QuestionStatus{QuestionStatusOpen, QuestionStatusAnswered, QuestionStatusClosed} {
		if !IsValidQuestionStatus(s) {
			t.Fatalf("expected valid status %q", s)
		}
	}
	if IsValidQuestionStatus(QuestionStatus("hidden")) {
		t.Fatal("expected invalid status to fail")
	}
}
