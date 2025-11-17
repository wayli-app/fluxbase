package auth

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTOTPSetupResponse_JSONFormat(t *testing.T) {
	// Create a response as the service would
	secret, qrCodeDataURI, otpauthURI, err := GenerateTOTPSecret("Fluxbase", "user@example.com")
	if err != nil {
		t.Fatalf("Failed to generate TOTP secret: %v", err)
	}

	response := &TOTPSetupResponse{
		ID:   "550e8400-e29b-41d4-a716-446655440000",
		Type: "totp",
	}
	response.TOTP.QRCode = qrCodeDataURI
	response.TOTP.Secret = secret
	response.TOTP.URI = otpauthURI

	// Marshal to JSON (as the API would)
	jsonData, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Unmarshal back to verify structure
	var unmarshaled TOTPSetupResponse
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify all fields
	if unmarshaled.ID != response.ID {
		t.Errorf("ID mismatch: expected %s, got %s", response.ID, unmarshaled.ID)
	}
	if unmarshaled.Type != "totp" {
		t.Errorf("Type should be 'totp', got %s", unmarshaled.Type)
	}
	if unmarshaled.TOTP.Secret == "" {
		t.Error("Secret should not be empty")
	}
	if !strings.HasPrefix(unmarshaled.TOTP.QRCode, "data:image/png;base64,") {
		t.Error("QR code should be a data URI")
	}
	if !strings.HasPrefix(unmarshaled.TOTP.URI, "otpauth://totp/") {
		t.Error("URI should be an otpauth URI")
	}

	// Verify JSON structure matches Supabase format
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &jsonMap); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// Check top-level fields
	if _, ok := jsonMap["id"]; !ok {
		t.Error("JSON should have 'id' field")
	}
	if _, ok := jsonMap["type"]; !ok {
		t.Error("JSON should have 'type' field")
	}
	if _, ok := jsonMap["totp"]; !ok {
		t.Error("JSON should have 'totp' field")
	}

	// Check nested TOTP fields
	totpMap, ok := jsonMap["totp"].(map[string]interface{})
	if !ok {
		t.Fatal("totp should be an object")
	}
	if _, ok := totpMap["qr_code"]; !ok {
		t.Error("totp should have 'qr_code' field")
	}
	if _, ok := totpMap["secret"]; !ok {
		t.Error("totp should have 'secret' field")
	}
	if _, ok := totpMap["uri"]; !ok {
		t.Error("totp should have 'uri' field")
	}

	t.Logf("✓ Response JSON format is correct")
	t.Logf("  JSON preview: %s", string(jsonData[:200])+"...")
}

func TestTOTPSetupResponse_SupabaseCompatibility(t *testing.T) {
	// This test verifies the response matches Supabase's mfa.enroll() response format
	secret, qrCodeDataURI, otpauthURI, _ := GenerateTOTPSecret("Fluxbase", "user@example.com")

	response := &TOTPSetupResponse{
		ID:   "test-factor-id",
		Type: "totp",
	}
	response.TOTP.QRCode = qrCodeDataURI
	response.TOTP.Secret = secret
	response.TOTP.URI = otpauthURI

	jsonData, _ := json.Marshal(response)

	// Expected Supabase format:
	// {
	//   "id": "uuid",
	//   "type": "totp",
	//   "totp": {
	//     "qr_code": "data:image/svg+xml;...",  // or PNG in our case
	//     "secret": "...",
	//     "uri": "otpauth://..."
	//   }
	// }

	var result map[string]interface{}
	json.Unmarshal(jsonData, &result)

	// Verify exact field names (Supabase uses snake_case in JSON)
	if result["type"] != "totp" {
		t.Error("Type field should be 'totp'")
	}

	totpObj := result["totp"].(map[string]interface{})

	// Verify qr_code field exists (not qrCode or qr-code)
	if _, ok := totpObj["qr_code"]; !ok {
		t.Error("TOTP object should have 'qr_code' field with underscore")
	}

	t.Logf("✓ Response is Supabase-compatible")
}
