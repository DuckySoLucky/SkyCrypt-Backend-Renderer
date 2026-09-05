package data

import (
	"encoding/json"
	"testing"
)

func TestStripJSONComments(t *testing.T) {
	input := `{
  "url": "https://example.com/a//b",
  "value": 1, // inline comment
  /* block comment */
  "text": "quoted \\\"// not a comment"
}
// trailing comment`

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(StripJSONComments(input)), &result); err != nil {
		t.Fatalf("comment-stripped JSON is invalid: %v", err)
	}
	if result["url"] != "https://example.com/a//b" {
		t.Fatalf("URL was changed: %v", result["url"])
	}
}