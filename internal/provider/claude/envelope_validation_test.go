package claude

import "testing"

func TestValidateSuccess(t *testing.T) {
	for _, raw := range []string{
		"null", "{}", "[]", "not json",
		`{"type":"result","subtype":"success","is_error":false}`,
		`{"type":"result","subtype":"success","is_error":false,"result":null}`,
		`{"type":"result","subtype":"success","result":""}`,
		`{"type":"result","subtype":"error_max_turns","is_error":false,"result":""}`,
		`{"type":"result","subtype":"success","is_error":true,"result":""}`,
		`{"type":"message","subtype":"success","is_error":false,"result":""}`,
	} {
		if err := validateSuccess([]byte(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	for _, raw := range []string{probeEnvelope, `{"type":"result","subtype":"success","is_error":false,"result":""}`} {
		if err := validateSuccess([]byte(raw)); err != nil {
			t.Errorf("rejected valid result: %v", err)
		}
	}
}
