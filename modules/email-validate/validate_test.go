package emailvalidate

import (
	"testing"
	"time"
)

// TestValidate_RealEmails runs the real AfterShip verifier (no mocks) against a
// clearly valid, well-known address and a clearly malformed one, asserting on
// the actual returned Result. The valid case performs a live DNS/MX lookup, so
// this test requires outbound network.
func TestValidate_RealEmails(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network-dependent test in short mode")
	}
	val := NewValidator(15 * time.Second)

	t.Run("valid well-known address", func(t *testing.T) {
		res, audit := val.Validate("support@github.com")

		if res.Status != "ok" {
			t.Fatalf("status = %q, want %q (error: %q)", res.Status, "ok", res.Error)
		}
		if !res.SyntaxValid {
			t.Errorf("syntax_valid = false, want true for support@github.com")
		}
		if !res.HasMXRecords {
			t.Errorf("has_mx_records = false, want true for github.com")
		}
		if !res.IsRoleAccount {
			t.Errorf("is_role_account = false, want true for support@ (a role address)")
		}
		if res.Error != "" {
			t.Errorf("error = %q, want empty", res.Error)
		}
		assertAudit(t, audit, "support@github.com", "ok")
	})

	t.Run("clearly malformed address", func(t *testing.T) {
		res, audit := val.Validate("not-an-email!!")

		// The check itself runs successfully; the finding is that syntax is bad.
		if res.Status != "ok" {
			t.Fatalf("status = %q, want %q", res.Status, "ok")
		}
		if res.SyntaxValid {
			t.Errorf("syntax_valid = true, want false for %q", "not-an-email!!")
		}
		if res.HasMXRecords {
			t.Errorf("has_mx_records = true, want false for a malformed address")
		}
		assertAudit(t, audit, "not-an-email!!", "ok")
	})
}

// TestValidate_MissingEmail confirms the graceful-degradation contract: an empty
// email must not error out — it yields status "unknown" with an error note so
// the pipeline keeps running.
func TestValidate_MissingEmail(t *testing.T) {
	val := NewValidator(0) // exercises the DefaultTimeout fallback

	res, audit := val.Validate("   ")
	if res.Status != "unknown" {
		t.Errorf("status = %q, want %q", res.Status, "unknown")
	}
	if res.Deliverable != "unknown" {
		t.Errorf("deliverable = %q, want %q", res.Deliverable, "unknown")
	}
	if res.Error == "" {
		t.Errorf("error is empty, want a note explaining the missing email")
	}
	if res.SourceTool != SourceTool {
		t.Errorf("source_tool = %q, want %q", res.SourceTool, SourceTool)
	}
	assertAudit(t, audit, "", "unknown")
}

func assertAudit(t *testing.T, a AuditRecord, wantEmail, wantStatus string) {
	t.Helper()
	if a.Tool != SourceTool {
		t.Errorf("audit.tool = %q, want %q", a.Tool, SourceTool)
	}
	if a.Email != wantEmail {
		t.Errorf("audit.email = %q, want %q", a.Email, wantEmail)
	}
	if a.Status != wantStatus {
		t.Errorf("audit.status = %q, want %q", a.Status, wantStatus)
	}
	if a.LegalBasis != LegalBasis {
		t.Errorf("audit.legal_basis = %q, want %q", a.LegalBasis, LegalBasis)
	}
	if _, err := time.Parse(time.RFC3339, a.CheckedAt); err != nil {
		t.Errorf("audit.checked_at = %q, not RFC3339: %v", a.CheckedAt, err)
	}
}
