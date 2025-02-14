package main

import (
	"fmt"
	"log"

	"github.com/jung-kurt/gofpdf"
)

func generateReceipt(payment SubscriptionPayment) error {
	// Create PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	
	// Add company logo and header
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(190, 10, "SportLife Gym")
	
	// Add receipt details
	pdf.SetFont("Arial", "", 12)
	pdf.Ln(10)
	pdf.Cell(190, 10, fmt.Sprintf("Transaction ID: %s", payment.TransactionID))
	pdf.Ln(10)
	pdf.Cell(190, 10, fmt.Sprintf("Date: %s", payment.PaymentTime.Format("2006-01-02 15:04:05")))
	pdf.Ln(10)
	pdf.Cell(190, 10, fmt.Sprintf("Customer: %s", payment.Customer.Name))
	pdf.Ln(10)
	pdf.Cell(190, 10, fmt.Sprintf("Subscription: %s", payment.SubscriptionType))
	pdf.Ln(10)
	pdf.Cell(190, 10, fmt.Sprintf("Amount: %.2f KZT", payment.Amount))
	
	// Save PDF
	filename := fmt.Sprintf("receipts/receipt_%s.pdf", payment.TransactionID)
	if err := pdf.OutputFileAndClose(filename); err != nil {
		return err
	}
	
	// Send email with receipt
	return sendEmail(payment.Customer.Email, filename, payment)
}

func sendReceiptEmail(email, receiptPath string) error {
	// Implement email sending logic here
	// For now, just log it
	log.Printf("Would send receipt to %s: %s", email, receiptPath)
	return nil
} 