package main

import (
	"fmt"
	"net/smtp"
	"os"
)

type EmailConfig struct {
    Host     string
    Port     int
    Username string
    Password string
    From     string
}

var emailConfig = EmailConfig{
    Host:     "smtp.gmail.com",
    Port:     587,
    Username: os.Getenv("EMAIL_USERNAME"),
    Password: os.Getenv("EMAIL_PASSWORD"),
    From:     "your-email@gmail.com",
}

func sendEmail(to, receiptPath string, payment SubscriptionPayment) error {
    subject := "Your SportLife Gym Subscription Receipt"
    body := fmt.Sprintf(`
        Dear %s,

        Thank you for your subscription to SportLife Gym!

        Transaction Details:
        - Transaction ID: %s
        - Subscription Type: %s
        - Amount: %.2f KZT
        - Date: %s

        Your payment has been processed successfully.
        Please find your receipt attached to this email.

        If you have any questions, please don't hesitate to contact us.

        Best regards,
        SportLife Gym Team
    `, payment.Customer.Name, payment.TransactionID, payment.SubscriptionType, 
       payment.Amount, payment.PaymentTime.Format("2006-01-02 15:04:05"))

    // Create email headers
    headers := make(map[string]string)
    headers["From"] = emailConfig.From
    headers["To"] = to
    headers["Subject"] = subject
    headers["MIME-Version"] = "1.0"
    headers["Content-Type"] = "text/plain; charset=UTF-8"

    // Compose message
    message := ""
    for key, value := range headers {
        message += fmt.Sprintf("%s: %s\r\n", key, value)
    }
    message += "\r\n" + body

    // Connect to SMTP server
    auth := smtp.PlainAuth("", emailConfig.Username, emailConfig.Password, emailConfig.Host)
    addr := fmt.Sprintf("%s:%d", emailConfig.Host, emailConfig.Port)

    return smtp.SendMail(addr, auth, emailConfig.From, []string{to}, []byte(message))
} 