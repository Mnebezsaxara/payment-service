package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"

	"sportlife/types" // Import the shared types

	"github.com/gin-gonic/gin"
)

type TransactionController struct {
	db *sql.DB
}

func NewTransactionController(db *sql.DB) *TransactionController {
	return &TransactionController{db: db}
}

func (tc *TransactionController) getCartDetails(c *gin.Context, cart *types.Cart) error {
	cartID := c.Param("cart_id")
	// Query the database to get cart details
	row := tc.db.QueryRow("SELECT id, user_id, total FROM carts WHERE id = $1", cartID)
	err := row.Scan(&cart.ID, &cart.UserID, &cart.Total)
	if err != nil {
		return err
	}

	// Get cart items
	rows, err := tc.db.Query("SELECT id, name, price, quantity FROM cart_items WHERE cart_id = $1", cartID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item types.CartItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.Quantity); err != nil {
			return err
		}
		cart.Items = append(cart.Items, item)
	}
	return nil
}

func (tc *TransactionController) createTransaction(cart types.Cart) (int64, error) {
	var transactionID int64
	err := tc.db.QueryRow(
		"INSERT INTO transactions (cart_id, user_id, amount, status) VALUES ($1, $2, $3, $4) RETURNING id",
		cart.ID, cart.UserID, cart.Total, "PENDING_PAYMENT",
	).Scan(&transactionID)
	return transactionID, err
}

func (tc *TransactionController) updateTransactionStatus(transactionID int64, status string) error {
	_, err := tc.db.Exec(
		"UPDATE transactions SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		status, transactionID,
	)
	return err
}

func preparePaymentRequest(cart types.Cart, userID string) []byte {
	paymentReq := struct {
		CartItems []types.CartItem `json:"cartItems"`
		Customer  struct {
			ID    string  `json:"id"`
			Name  string  `json:"name"`
			Email string  `json:"email"`
		} `json:"customer"`
	}{
		CartItems: cart.Items,
	}
	
	// In a real application, you would fetch customer details from the database
	paymentReq.Customer.ID = userID
	
	reqBytes, _ := json.Marshal(paymentReq)
	return reqBytes
}

func (tc *TransactionController) ProcessTransaction(c *gin.Context) {
	var cart types.Cart
	if err := tc.getCartDetails(c, &cart); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create transaction record
	transactionID, err := tc.createTransaction(cart)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction"})
		return
	}

	// Prepare payment request for microservice
	paymentReq := preparePaymentRequest(cart, c.GetString("userID"))
	
	// Send to payment microservice
	resp, err := http.Post(
		"http://localhost:8081/process-payment",
		"application/json",
		bytes.NewBuffer(paymentReq),
	)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment service unavailable"})
		return
	}

	// Handle response
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	if result["status"] == "success" {
		tc.updateTransactionStatus(transactionID, "PAID")
		c.JSON(http.StatusOK, gin.H{"message": "Transaction completed successfully"})
	} else {
		tc.updateTransactionStatus(transactionID, "DECLINED")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment declined"})
	}
} 