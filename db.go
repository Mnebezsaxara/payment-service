package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib" // Import the pgx driver
	_ "github.com/lib/pq"
)

var db *sql.DB

func initDB() {
	var err error
	connStr := "postgres://postgres:LUFFYtaroo111&&@localhost:5432/payment_service" // Update with your database name
	db, err = sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}

	// Test the connection
	if err = db.Ping(); err != nil {
		log.Fatalf("Unable to reach the database: %v", err)
	}

	fmt.Println("Database connection established")
}

func storePaymentRecord(payment SubscriptionPayment) error {
	query := `
        INSERT INTO payment_transactions 
        (transaction_id, customer_email, subscription_type, amount, payment_method, 
        card_last_four, payment_status, payment_time)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `
	
	// Get last 4 digits of card number
	cardLastFour := payment.CardNumber[len(payment.CardNumber)-4:]
	
	_, err := db.Exec(query,
		payment.TransactionID,
		payment.Customer.Email,
		payment.SubscriptionType,
		payment.Amount,
		payment.PaymentMethod,
		cardLastFour,
		payment.Status,
		payment.PaymentTime,
	)
	
	return err
}

func insertPaymentTransaction(transactionID, customerEmail, subscriptionType string, amount float64, paymentMethod, cardLastFour, paymentStatus string) error {
	query := `INSERT INTO payment_transactions (transaction_id, customer_email, subscription_type, amount, payment_method, card_last_four, payment_status, payment_time) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := db.Exec(query, transactionID, customerEmail, subscriptionType, amount, paymentMethod, cardLastFour, paymentStatus, time.Now())
	return err
}

func insertSubscriptionReceipt(transactionID, receiptPath, emailStatus string) error {
	query := `INSERT INTO subscription_receipts (transaction_id, receipt_path, email_status) 
			  VALUES ($1, $2, $3)`

	_, err := db.Exec(query, transactionID, receiptPath, emailStatus)
	return err
} 