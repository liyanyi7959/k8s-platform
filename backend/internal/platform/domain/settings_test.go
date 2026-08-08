package domain

import "testing"

func TestSettingsDefaults(t *testing.T) {
	settings := Defaults()
	if settings.SiteName == "" || settings.SessionTimeout != 3600 {
		t.Fatalf("unexpected defaults: %+v", settings)
	}
	if err := settings.Validate(); err != nil {
		t.Fatalf("defaults should validate: %v", err)
	}
}

func TestSettingsValidateRejectsInvalidValues(t *testing.T) {
	settings := Defaults()
	settings.SessionTimeout = 60
	if err := settings.Validate(); err == nil {
		t.Fatal("expected validation error for session timeout")
	}
	settings = Defaults()
	settings.SiteName = "   "
	if err := settings.Validate(); err == nil {
		t.Fatal("expected validation error for empty site name")
	}
}

func TestSettingsValuesRoundTrip(t *testing.T) {
	settings := Defaults()
	settings.SiteName = "运维平台"
	settings.SessionTimeout = 7200
	settings.MaxLoginAttempts = 6
	settings.PasswordExpirationDays = 120
	settings.EnableAuditLog = false
	settings.EnableTwoFactorAuth = true

	roundTrip := FromValues(settings.Values())
	if roundTrip.SiteName != settings.SiteName ||
		roundTrip.SessionTimeout != settings.SessionTimeout ||
		roundTrip.MaxLoginAttempts != settings.MaxLoginAttempts ||
		roundTrip.PasswordExpirationDays != settings.PasswordExpirationDays ||
		roundTrip.EnableAuditLog != settings.EnableAuditLog ||
		roundTrip.EnableTwoFactorAuth != settings.EnableTwoFactorAuth {
		t.Fatalf("round trip mismatch: %+v vs %+v", roundTrip, settings)
	}
}

func TestFromValuesFallsBackToDefaults(t *testing.T) {
	settings := FromValues(map[string]string{"site_name": "demo"})
	if settings.SessionTimeout != 3600 || settings.MaxLoginAttempts != 5 {
		t.Fatalf("unexpected fallback: %+v", settings)
	}
}
