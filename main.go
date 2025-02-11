package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	// Import the shared types

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
)

type PaymentRequest struct {
	CartItems []CartItem `json:"cartItems"`
	Customer  Customer   `json:"customer"`
	Payment   Payment    `json:"payment"`
	BookingID string    `json:"bookingId"`
}

type Payment struct {
	Amount           float64 `json:"amount"`
	Currency        string  `json:"currency"`
	IsStudentDiscount bool    `json:"isStudentDiscount"`
}

type CartItem struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

type Customer struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func main() {
	// Ensure receipts directory exists
	os.MkdirAll("receipts", 0755)
	
	r := gin.Default()
	
	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})
	
	r.POST("/process-payment", handlePayment)
	r.Run(":8081")
}

func handlePayment(c *gin.Context) {
	// Check authentication
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Authentication required",
		})
		return
	}

	// Verify token
	userID, err := verifyToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Invalid token",
		})
		return
	}

	var req PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": fmt.Sprintf("Invalid request format: %v", err),
		})
		return
	}

	// Verify that the customer ID matches the authenticated user
	if req.Customer.ID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	// Validate payment details
	if err := validatePayment(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	// Process payment
	if success := processPayment(req); success {
		// Send confirmation email
		go sendConfirmationEmail(req)

		c.JSON(http.StatusOK, gin.H{
			"status":   "success",
			"message":  "Payment processed successfully",
			"bookingId": req.BookingID,
			"amount":   req.Payment.Amount,
			"currency": req.Payment.Currency,
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Payment processing failed",
		})
	}
}

func validatePayment(req PaymentRequest) error {
	if req.Payment.Amount <= 0 {
		return fmt.Errorf("invalid payment amount")
	}
	if req.Payment.Currency != "KZT" {
		return fmt.Errorf("invalid currency")
	}
	if len(req.CartItems) == 0 {
		return fmt.Errorf("cart is empty")
	}
	return nil
}

func processPayment(req PaymentRequest) bool {
	// Here you would integrate with your actual payment processing system
	// For now, we'll simulate success
	log.Printf("Processing payment: Amount=%v, Currency=%s, BookingID=%s",
		req.Payment.Amount, req.Payment.Currency, req.BookingID)
	return true
}

func generateAndSendReceipt(req PaymentRequest) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	
	// Add company header
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "SportLife")
	
	// Add transaction details
	pdf.SetFont("Arial", "", 12)
	pdf.Ln(10)
	pdf.Cell(40, 10, "Transaction Date: "+time.Now().Format("2006-01-02 15:04:05"))
	
	// Add items
	pdf.Ln(10)
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(80, 10, "Item")
	pdf.Cell(30, 10, "Quantity")
	pdf.Cell(30, 10, "Price")
	pdf.Cell(30, 10, "Total")
	
	var total float64
	for _, item := range req.CartItems {
		pdf.Ln(8)
		pdf.SetFont("Arial", "", 12)
		pdf.Cell(80, 10, item.Name)
		pdf.Cell(30, 10, fmt.Sprintf("%d", item.Quantity))
		pdf.Cell(30, 10, fmt.Sprintf("%.2f", item.Price))
		itemTotal := float64(item.Quantity) * item.Price
		pdf.Cell(30, 10, fmt.Sprintf("%.2f", itemTotal))
		total += itemTotal
	}

	// Add total
	pdf.Ln(10)
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(140, 10, "Total:")
	pdf.Cell(30, 10, fmt.Sprintf("%.2f", total))
	
	// Save PDF
	filename := fmt.Sprintf("receipts/receipt_%s.pdf", time.Now().Format("20060102150405"))
	if err := pdf.OutputFileAndClose(filename); err != nil {
		return fmt.Errorf("failed to generate PDF: %v", err)
	}
	
	// Use your existing email sending function
	return sendEmailWithReceipt(req.Customer.Email, filename)
}

func sendEmailWithReceipt(to, filename string) error {
	// Use your existing email sending function
	// You can import and use the sendEmailWithAttachment function from your main server
	// For now, return nil to mock successful email sending
	return nil
}

func sendConfirmationEmail(req PaymentRequest) {
	// Email template
	emailBody := fmt.Sprintf(`
		Здравствуйте, %s!

		Спасибо за покупку абонемента в SportLife Gym.

		Детали вашего заказа:
		- План: %s
		- Сумма: %.2f %s
		- Номер заказа: %s

		С уважением,
		Команда SportLife Gym
	`, req.Customer.Name, req.CartItems[0].Name, req.Payment.Amount, req.Payment.Currency, req.BookingID)

	// Send email using your email service
	// This is a placeholder - implement with your actual email service
	sendEmail(req.Customer.Email, "Подтверждение покупки абонемента", emailBody)
}

func verifyToken(token string) (string, error) {
	// Remove "Bearer " prefix
	tokenString := strings.TrimPrefix(token, "Bearer ")
	
	// For now, just check if the token is not empty
	if tokenString == "" {
		return "", fmt.Errorf("empty token")
	}

	// TODO: Implement proper JWT verification
	// This is a simplified version - you should implement actual JWT verification
	if strings.HasPrefix(tokenString, "eyJ") {
		// Simulate extracting user ID from token
		return "user_id", nil
	}

	return "", fmt.Errorf("invalid token format")
}

func sendEmail(to, subject, body string) error {
	// Implement your email sending logic here
	// You can use packages like gomail or AWS SES
	return nil
} 