package sms

import (
	"testing"

	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	logger := zap.NewNop() // No-op logger for testing

	tests := []struct {
		name       string
		accountSID string
		authToken  string
		fromNumber string
		expected   bool // expected enabled status
	}{
		{
			name:       "valid credentials",
			accountSID: "AC1234567890123456789012345678901234",
			authToken:  "test-auth-token",
			fromNumber: "+15551234567",
			expected:   true,
		},
		{
			name:       "missing account SID",
			accountSID: "",
			authToken:  "test-auth-token",
			fromNumber: "+15551234567",
			expected:   false,
		},
		{
			name:       "missing auth token",
			accountSID: "AC1234567890123456789012345678901234",
			authToken:  "",
			fromNumber: "+15551234567",
			expected:   false,
		},
		{
			name:       "missing from number",
			accountSID: "AC1234567890123456789012345678901234",
			authToken:  "test-auth-token",
			fromNumber: "",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smsService := New(tt.accountSID, tt.authToken, tt.fromNumber, logger)

			if smsService == nil {
				t.Fatal("Expected non-nil SMS service")
			}

			if smsService.IsEnabled() != tt.expected {
				t.Errorf("Expected enabled status %v, got %v", tt.expected, smsService.IsEnabled())
			}
		})
	}
}

func TestFormatPhoneNumber(t *testing.T) {
	logger := zap.NewNop()
	smsService := New("AC1234567890123456789012345678901234", "test-key", "+15551234567", logger)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "kenyan number starting with 07",
			input:    "0712345678",
			expected: "+254712345678",
		},
		{
			name:     "kenyan number starting with 01",
			input:    "0112345678",
			expected: "+254112345678",
		},
		{
			name:     "already formatted international",
			input:    "+254712345678",
			expected: "+254712345678",
		},
		{
			name:     "9 digit number",
			input:    "712345678",
			expected: "+254712345678",
		},
		{
			name:     "number with spaces and dashes",
			input:    "071-234-5678",
			expected: "+254712345678",
		},
		{
			name:     "number with parentheses",
			input:    "(071) 234-5678",
			expected: "+254712345678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := smsService.formatPhoneNumber(tt.input)
			if result != tt.expected {
				t.Errorf("formatPhoneNumber(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncateMessage(t *testing.T) {
	logger := zap.NewNop()
	smsService := New("AC1234567890123456789012345678901234", "test-key", "+15551234567", logger)

	tests := []struct {
		name      string
		message   string
		maxLength int
		expected  string
	}{
		{
			name:      "short message",
			message:   "Hello",
			maxLength: 10,
			expected:  "Hello",
		},
		{
			name:      "exact length message",
			message:   "Hello World",
			maxLength: 11,
			expected:  "Hello World",
		},
		{
			name:      "long message",
			message:   "This is a very long message",
			maxLength: 10,
			expected:  "This is a ...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := smsService.truncateMessage(tt.message, tt.maxLength)
			if result != tt.expected {
				t.Errorf("truncateMessage(%q, %d) = %q, want %q", tt.message, tt.maxLength, result, tt.expected)
			}
		})
	}
}

func TestSendOrderConfirmation(t *testing.T) {
	logger := zap.NewNop()

	// Test with disabled service
	disabledService := New("", "", "", logger)
	err := disabledService.SendOrderConfirmation("+254712345678", 123, "1000.00")
	if err == nil {
		t.Error("Expected error for disabled service, got none")
	}

	// Note: We can't test the actual SMS sending without mocking the Twilio service
	// This would require more complex testing setup with interfaces and dependency injection
}

func TestSendSMSWithRealNumber(t *testing.T) {
	// Skip this test by default to avoid sending real SMS in CI
	if testing.Short() {
		t.Skip("Skipping SMS test with real number")
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	// Test with dummy Twilio credentials for testing
	accountSID := "AC1234567890123456789012345678901234"
	authToken := "test-auth-token-dummy-value"
	fromNumber := "+15551234567" // Dummy test number

	smsService := New(accountSID, authToken, fromNumber, logger)

	// Test phone number (your number)
	phoneNumber := "0723339727" // Will be formatted to +254723339727

	if !smsService.IsEnabled() {
		t.Skip("SMS service is disabled")
	}

	// Test sending order confirmation
	orderID := int32(12345)
	totalAmount := "2500.00"

	t.Logf("Sending SMS to %s", phoneNumber)
	t.Logf("Formatted number: %s", smsService.formatPhoneNumber(phoneNumber))

	err = smsService.SendOrderConfirmation(phoneNumber, orderID, totalAmount)
	if err != nil {
		t.Logf("SMS send result: %v", err)
		// Note: This might "fail" if you don't have a Twilio phone number yet
	} else {
		t.Logf("SMS sent successfully!")
	}

	t.Logf("Expected message: 'Hi! Your SavannaCart order #%d has been received and will be processed soon. Total: KES %s. Thank you for shopping with us!'", orderID, totalAmount)
	t.Logf("Check your phone to see if the message was received!")
	t.Logf("Also check Twilio Console: https://console.twilio.com/us1/develop/sms/logs")
}
