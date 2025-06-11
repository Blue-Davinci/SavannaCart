package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	err := godotenv.Load("cmd/api/.env")
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	accountSID := os.Getenv("SAVANNACART_SMS_ACCOUNT_SID")
	fromNumber := os.Getenv("SAVANNACART_SMS_FROM_NUMBER")
	fmt.Println("🚨 TWILIO TRIAL ACCOUNT SMS LIMITATIONS 🚨")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Your Twilio Account: %s\n", accountSID)
	fmt.Printf("Your Twilio Number: %s\n", fromNumber)
	fmt.Println()

	fmt.Println("❌ PROBLEM:")
	fmt.Println("   • Your Twilio account is in TRIAL mode")
	fmt.Println("   • Trial accounts can only send SMS to VERIFIED phone numbers")
	fmt.Println("   • You tried to send to: +254723339727 (unverified Kenyan number)")
	fmt.Println("   • Your number is US-based: +12253141384")
	fmt.Println()

	fmt.Println("✅ SOLUTIONS:")
	fmt.Println("   1. VERIFY PHONE NUMBERS (FREE):")
	fmt.Println("      • Go to: https://console.twilio.com/us1/develop/phone-numbers/manage/verified")
	fmt.Println("      • Click 'Verify a new number'")
	fmt.Println("      • Enter: +254723339727")
	fmt.Println("      • Follow SMS verification process")
	fmt.Println()
	fmt.Println("   2. UPGRADE ACCOUNT (PAID):")
	fmt.Println("      • Add credit to your Twilio account")
	fmt.Println("      • This removes trial restrictions")
	fmt.Println("      • You can send to any valid phone number")
	fmt.Println()
	fmt.Println("   3. TEST WITH YOUR OWN NUMBER:")
	fmt.Println("      • Use your own verified phone number for testing")
	fmt.Println("      • Update user phone number in database")
	fmt.Println()

	fmt.Println("🔧 WHAT WE'VE FIXED:")
	fmt.Println("   • SMS service now handles trial limitations gracefully")
	fmt.Println("   • Better error messages (INFO instead of ERROR logs)")
	fmt.Println("   • Orders still complete successfully even if SMS fails")
	fmt.Println()

	fmt.Println("📱 TO TEST SMS IMMEDIATELY:")
	fmt.Println("   1. Verify +254723339727 in Twilio console")
	fmt.Println("   2. Create a new order with that phone number")
	fmt.Println("   3. SMS will be sent successfully!")
}
