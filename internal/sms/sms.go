package sms

import (
	"fmt"
	"strings"

	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
	"go.uber.org/zap"
)

// SMSService represents the SMS service configuration and client
type SMSService struct {
	client     *twilio.RestClient
	fromNumber string
	logger     *zap.Logger
	enabled    bool
}

// SMSResponse represents the response from sending an SMS
type SMSResponse struct {
	MessageSID string `json:"message_sid"`
	Status     string `json:"status"`
	From       string `json:"from"`
	To         string `json:"to"`
}

// New creates a new Twilio SMS service instance
func New(accountSID, authToken, fromNumber string, logger *zap.Logger) *SMSService {
	// Validate required parameters
	if accountSID == "" || authToken == "" {
		logger.Warn("SMS service disabled: missing Account SID or Auth Token")
		return &SMSService{
			enabled: false,
			logger:  logger,
		}
	}

	if fromNumber == "" {
		logger.Warn("SMS service disabled: missing From Number")
		return &SMSService{
			enabled: false,
			logger:  logger,
		}
	}

	// Create the Twilio client
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: authToken,
	})

	logger.Info("SMS service initialized with Twilio",
		zap.String("account_sid", accountSID),
		zap.String("from_number", fromNumber))

	return &SMSService{
		client:     client,
		fromNumber: fromNumber,
		logger:     logger,
		enabled:    true,
	}
}

// Send sends an SMS message to the specified phone number
func (s *SMSService) Send(phoneNumber, message string) (*SMSResponse, error) {
	if !s.enabled {
		s.logger.Warn("SMS service is disabled, skipping SMS send")
		return nil, fmt.Errorf("SMS service is disabled")
	}

	// Validate inputs
	if phoneNumber == "" {
		return nil, fmt.Errorf("phone number is required")
	}
	if message == "" {
		return nil, fmt.Errorf("message is required")
	}

	// Ensure phone number is in international format
	phoneNumber = s.formatPhoneNumber(phoneNumber)

	s.logger.Info("Sending SMS via Twilio",
		zap.String("phone_number", phoneNumber),
		zap.String("message_preview", s.truncateMessage(message, 50)))

	// Create SMS parameters
	params := &openapi.CreateMessageParams{}
	params.SetTo(phoneNumber)
	params.SetFrom(s.fromNumber)
	params.SetBody(message)
	// Send SMS using Twilio
	resp, err := s.client.Api.CreateMessage(params)
	if err != nil {
		// Handle specific Twilio errors gracefully
		errorMsg := err.Error()
		if strings.Contains(errorMsg, "21612") {
			s.logger.Warn("SMS sending restricted - likely trial account limitation",
				zap.String("phone_number", phoneNumber),
				zap.String("from_number", s.fromNumber),
				zap.String("twilio_error", "21612"),
				zap.String("solution", "Verify phone number in Twilio console or upgrade account"))
			return nil, fmt.Errorf("SMS sending restricted: Trial accounts can only send to verified numbers. Please verify %s in Twilio console", phoneNumber)
		} else if strings.Contains(errorMsg, "21608") {
			s.logger.Warn("SMS sending to unverified number",
				zap.String("phone_number", phoneNumber),
				zap.String("twilio_error", "21608"))
			return nil, fmt.Errorf("SMS sending restricted: Phone number %s must be verified in Twilio console for trial accounts", phoneNumber)
		}

		s.logger.Error("Failed to send SMS via Twilio",
			zap.String("phone_number", phoneNumber),
			zap.Error(err))
		return nil, fmt.Errorf("failed to send SMS: %w", err)
	}

	s.logger.Info("SMS sent successfully via Twilio",
		zap.String("phone_number", phoneNumber),
		zap.String("message_sid", *resp.Sid),
		zap.String("status", *resp.Status))

	// Convert Twilio response to our format
	smsResponse := &SMSResponse{
		MessageSID: *resp.Sid,
		Status:     *resp.Status,
		From:       *resp.From,
		To:         *resp.To,
	}

	return smsResponse, nil
}

// SendOrderConfirmation sends a simple order confirmation SMS
func (s *SMSService) SendOrderConfirmation(phoneNumber string, orderID int32, totalAmount string) error {
	message := fmt.Sprintf("Hi! Your SavannaCart order #%d has been received and will be processed soon. Total: KES %s. Thank you for shopping with us!", orderID, totalAmount)

	_, err := s.Send(phoneNumber, message)
	return err
}

// formatPhoneNumber ensures the phone number is in international format
func (s *SMSService) formatPhoneNumber(phoneNumber string) string {
	// Remove any whitespace and special characters (spaces, dashes, parentheses)
	cleaned := ""
	for _, char := range phoneNumber {
		if char >= '0' && char <= '9' || char == '+' {
			cleaned += string(char)
		}
	}
	phoneNumber = cleaned

	// If it starts with 0 and looks like a Kenyan number, convert to +254
	if len(phoneNumber) >= 10 && phoneNumber[0] == '0' {
		phoneNumber = "+254" + phoneNumber[1:]
	}

	// If it doesn't start with +, assume it's a Kenyan number and add +254
	if len(phoneNumber) > 0 && phoneNumber[0] != '+' {
		// If it's a 9-digit number, it's likely missing the country code
		if len(phoneNumber) == 9 {
			phoneNumber = "+254" + phoneNumber
		}
	}

	return phoneNumber
}

// truncateMessage truncates a message for logging purposes
func (s *SMSService) truncateMessage(message string, maxLength int) string {
	if len(message) <= maxLength {
		return message
	}
	return message[:maxLength] + "..."
}

// IsEnabled returns whether the SMS service is enabled
func (s *SMSService) IsEnabled() bool {
	return s.enabled
}
