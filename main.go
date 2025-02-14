package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v4/stdlib" // Import the pgx driver
	"github.com/jung-kurt/gofpdf"
	"github.com/rs/cors"
	"gopkg.in/gomail.v2"
)

var db *sql.DB

// Initialize the database connection
func initDB() {
	var err error
	connStr := "postgres://postgres:LUFFYtaroo111&&&@localhost:5432/payment_service" // Update with your database name
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

// Insert payment transaction into the database
func insertPaymentTransaction(transactionID, customerEmail, subscriptionType string, amount float64, paymentMethod, cardLastFour, paymentStatus string) error {
	query := `INSERT INTO payment_transactions (transaction_id, customer_email, subscription_type, amount, payment_method, card_last_four, payment_status, payment_time) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := db.Exec(query, transactionID, customerEmail, subscriptionType, amount, paymentMethod, cardLastFour, paymentStatus, time.Now())
	return err
}

// Insert subscription receipt into the database
func insertSubscriptionReceipt(transactionID, receiptPath, emailStatus string) error {
	query := `INSERT INTO subscription_receipts (transaction_id, receipt_path, email_status) 
			  VALUES ($1, $2, $3)`

	_, err := db.Exec(query, transactionID, receiptPath, emailStatus)
	return err
}

type InitPaymentRequest struct {
	SubscriptionType string  `json:"subscriptionType"`
	BasePrice       float64 `json:"basePrice"`
}

type InitPaymentResponse struct {
	Success      bool   `json:"success"`
	TransactionId string `json:"transactionId"`
	Message      string `json:"message,omitempty"`
}

type PaymentData struct {
	Email       string  `json:"email"`
	Name        string  `json:"name"`
	Phone       string  `json:"phone"`
	CardNumber  string  `json:"cardNumber"`
	Amount      float64 `json:"amount"`
}

type SubscriptionPayment struct {
	TransactionID string    `json:"transactionId"`
	Customer     PaymentData `json:"customer"`
	Status       string    `json:"status"`
	PaymentTime  time.Time `json:"paymentTime"`
}

type Config struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

var config Config

func init() {
	// Load configuration
	file, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatal("Error reading config file:", err)
	}

	if err := json.Unmarshal(file, &config); err != nil {
		log.Fatal("Error parsing config file:", err)
	}

	// Create font directory if it doesn't exist
	if err := os.MkdirAll("font", 0755); err != nil {
		log.Fatal("Error creating font directory:", err)
	}

	// Check if DejaVu fonts exist
	fontPath := "font/DejaVuSans.ttf"
	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		// Download DejaVu Sans font if it doesn't exist
		log.Println("Please download DejaVuSans.ttf and DejaVuSans-Bold.ttf and place them in the font directory")
		log.Println("You can download them from: https://dejavu-fonts.github.io/")
		log.Fatal("Required fonts are missing")
	}
}

func main() {
	initDB() // Initialize the database connection

	r := mux.NewRouter()

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:5500", "http://127.0.0.1:5500"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})

	r.HandleFunc("/init-payment", handleInitPayment).Methods("POST", "OPTIONS")
	r.HandleFunc("/payment", servePaymentPage).Methods("GET")
	r.HandleFunc("/process-payment", handleProcessPayment).Methods("POST")

	handler := c.Handler(r)

	fmt.Println("Payment service starting on :8081")
	log.Fatal(http.ListenAndServe(":8081", handler))
}

func generateReceipt(data PaymentData) (string, error) {
	if err := os.MkdirAll("receipts", 0755); err != nil {
		return "", err
	}

	// Create new PDF with Unicode support
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Set UTF-8 encoding and font
	pdf.AddUTF8Font("DejaVu", "", "font/DejaVuSans.ttf")
	pdf.AddUTF8Font("DejaVu", "B", "font/DejaVuSans-Bold.ttf")
	
	// Title
	pdf.SetFont("DejaVu", "B", 16)
	pdf.Cell(190, 10, "Квитанция об оплате")
	pdf.Ln(15)
	
	// Content
	pdf.SetFont("DejaVu", "", 12)
	
	// Function to add a row with label and value
	addRow := func(label, value string) {
		pdf.CellFormat(50, 8, label, "", 0, "L", false, 0, "")
		pdf.CellFormat(140, 8, value, "", 0, "L", false, 0, "")
		pdf.Ln(10)
	}
	
	// Add content rows
	currentTime := time.Now().Format("02.01.2006 15:04:05")
	addRow("Дата:", currentTime)
	addRow("ФИО:", data.Name)
	addRow("Email:", data.Email)
	addRow("Телефон:", data.Phone)
	addRow("Сумма:", fmt.Sprintf("%.2f тенге", data.Amount))
	
	// Add some space before footer
	pdf.Ln(10)
	
	// Footer
	pdf.SetFont("DejaVu", "", 10)
	pdf.Cell(190, 8, "Спасибо за оплату!")
	pdf.Ln(8)
	pdf.Cell(190, 8, "С уважением, SportLife")
	
	filename := fmt.Sprintf("receipts/receipt_%s_%s.pdf", 
		data.Email, 
		time.Now().Format("20060102150405"))
	
	err := pdf.OutputFileAndClose(filename)
	return filename, err
}

func sendEmail(to, receiptPath string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", config.Email)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Квитанция об оплате - SportLife")
	m.SetBody("text/html", `
		<h2>Спасибо за оплату!</h2>
		<p>Ваша квитанция во вложении.</p>
		<br>
		<p>С уважением,<br>SportLife</p>
	`)
	m.Attach(receiptPath)

	// Use Mail.ru SMTP settings
	d := gomail.NewDialer("smtp.mail.ru", 587, config.Email, config.Password)

	if err := d.DialAndSend(m); err != nil {
		log.Printf("Error sending email: %v", err)
		return err
	}

	log.Printf("Email sent successfully to %s with receipt %s", to, receiptPath)
	return nil
}

func handleInitPayment(w http.ResponseWriter, r *http.Request) {
	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Simulate processing delay
	time.Sleep(5 * time.Second)

	// Handle preflight request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Parse request body
	var req InitPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request format",
		})
		return
	}

	// Generate transaction ID
	transactionId := fmt.Sprintf("TRX-%d", time.Now().Unix())

	// Send response
	json.NewEncoder(w).Encode(InitPaymentResponse{
		Success:      true,
		TransactionId: transactionId,
	})
}

func servePaymentPage(w http.ResponseWriter, r *http.Request) {
	transactionId := r.URL.Query().Get("transactionId")
	if transactionId == "" {
		http.Error(w, "Transaction ID is required", http.StatusBadRequest)
		return
	}

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Оплата | SportLife</title>
	<style>
		body { 
			font-family: Arial, sans-serif; 
			margin: 0; 
			padding: 20px;
			background-color: #f8f9fa;
		}
		.payment-form { 
			max-width: 500px; 
			margin: 20px auto;
			background: white;
			padding: 30px;
			border-radius: 10px;
			box-shadow: 0 2px 10px rgba(0,0,0,0.1);
		}
		.form-group { 
			margin-bottom: 20px; 
		}
		label { 
			display: block; 
			margin-bottom: 8px;
			color: #333;
			font-weight: 500;
		}
		input, select { 
			width: 100%; 
			padding: 10px;
			border: 1px solid #ddd;
			border-radius: 4px;
			font-size: 14px;
			box-sizing: border-box;
		}
		button { 
			background: #47a447; 
			color: white; 
			padding: 12px 24px; 
			border: none; 
			border-radius: 4px;
			cursor: pointer;
			width: 100%;
			font-size: 16px;
			font-weight: 500;
		}
		button:hover {
			background: #3d8b3d;
		}
		.loading-overlay {
			position: fixed;
			top: 0;
			left: 0;
			width: 100%;
			height: 100%;
			background: rgba(255, 255, 255, 0.9);
			display: none;
			justify-content: center;
			align-items: center;
			flex-direction: column;
			z-index: 1000;
		}
		.spinner {
			width: 50px;
			height: 50px;
			border: 5px solid #f3f3f3;
			border-top: 5px solid #47a447;
			border-radius: 50%;
			animation: spin 1s linear infinite;
			margin-bottom: 20px;
		}
		@keyframes spin {
			0% { transform: rotate(0deg); }
			100% { transform: rotate(360deg); }
		}
		.loading-text {
			font-size: 18px;
			color: #333;
			margin-top: 15px;
		}
		.status {
			margin-top: 20px;
			padding: 15px;
			border-radius: 4px;
			display: none;
		}
	</style>
</head>
<body>
	<div class="payment-form">
		<h2>Оформление платежа</h2>
		<form id="paymentForm">
			<input type="hidden" id="transactionId" value="` + transactionId + `">
			<div class="form-group">
				<label>Email:</label>
				<input type="email" id="email" required placeholder="example@mail.com">
			</div>
			<div class="form-group">
				<label>ФИО:</label>
				<input type="text" id="name" required placeholder="Иванов Иван Иванович">
			</div>
			<div class="form-group">
				<label>Номер телефона:</label>
				<input type="tel" 
					   id="phone" 
					   required 
					   placeholder="+7XXXXXXXXXX"
					   maxlength="12">
			</div>
			<div class="form-group">
				<label>Номер карты:</label>
				<input type="text" id="cardNumber" required pattern="[0-9]{16}" placeholder="XXXX XXXX XXXX XXXX">
			</div>
			<div class="form-group">
				<label>Способ оплаты:</label>
				<select id="paymentMethod" required>
					<option value="">Выберите способ оплаты</option>
					<option value="card">Банковская карта</option>
					<option value="googlepay">Google Pay</option>
					<option value="applepay">Apple Pay</option>
				</select>
			</div>
			<button type="submit">Оплатить</button>
		</form>
		<div id="status" class="status"></div>
	</div>

	<div id="loadingOverlay" class="loading-overlay">
		<div class="spinner"></div>
		<div class="loading-text">Обработка платежа...</div>
	</div>

	<script>
		document.getElementById('paymentForm').addEventListener('submit', async (e) => {
			e.preventDefault();
			
			const loadingOverlay = document.getElementById('loadingOverlay');
			const status = document.getElementById('status');
			
			// Get form data
			const formData = {
				email: document.getElementById('email').value,
				name: document.getElementById('name').value,
				phone: document.getElementById('phone').value,
				cardNumber: document.getElementById('cardNumber').value,
				amount: 25000 // You might want to pass this from the subscription selection
			};
			
			loadingOverlay.style.display = 'flex';
			
			try {
				// Send payment data to server
				const response = await fetch('/process-payment', {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json'
					},
					body: JSON.stringify(formData)
				});

				const result = await response.json();

				setTimeout(() => {
					loadingOverlay.style.display = 'none';
					status.style.display = 'block';
					
					if (result.success) {
						status.style.background = '#d4edda';
						status.style.color = '#155724';
						status.innerHTML = '<h3 style="margin: 0 0 10px 0">Платёж успешно обработан!</h3><p style="margin: 0">Чек был отправлен на указанный email.</p>';
					} else {
						status.style.background = '#f8d7da';
						status.style.color = '#721c24';
						status.innerHTML = '<h3 style="margin: 0 0 10px 0">Ошибка при обработке платежа</h3><p style="margin: 0">' + (result.message || 'Пожалуйста, попробуйте позже.') + '</p>';
					}
				}, 5000);
			} catch (error) {
				console.error('Error:', error);
				loadingOverlay.style.display = 'none';
				status.style.display = 'block';
				status.style.background = '#f8d7da';
				status.style.color = '#721c24';
				status.innerHTML = '<h3 style="margin: 0 0 10px 0">Ошибка при обработке платежа</h3><p style="margin: 0">Пожалуйста, попробуйте позже.</p>';
			}
		});

		// Format card number input
		document.getElementById('cardNumber').addEventListener('input', function(e) {
			let value = e.target.value.replace(/\\D/g, '');
			if (value.length > 16) value = value.slice(0, 16);
			e.target.value = value;
		});

		// Improved phone number input handling
		const phoneInput = document.getElementById('phone');
		
		// Set initial +7 prefix
		if (!phoneInput.value) {
			phoneInput.value = '+7';
		}
		
		phoneInput.addEventListener('input', function(e) {
			let value = e.target.value;
			
			// Ensure starts with +7
			if (!value.startsWith('+7')) {
				value = '+7';
			}
			
			// Remove any non-digits after +7
			value = '+7' + value.substring(2).replace(/[^\d]/g, '');
			
			// Limit to +7 plus 10 digits
			if (value.length > 12) {
				value = value.slice(0, 12);
			}
			
			e.target.value = value;
		});

		// Prevent deletion of +7 prefix
		phoneInput.addEventListener('keydown', function(e) {
			if (e.target.selectionStart <= 2 && e.key === 'Backspace') {
				e.preventDefault();
			}
		});

		// Add form validation
		document.getElementById('paymentForm').addEventListener('submit', function(e) {
			const phoneValue = phoneInput.value;
			const phoneRegex = /^\+7\d{10}$/;
			
			if (!phoneRegex.test(phoneValue)) {
				phoneInput.setCustomValidity('Введите номер в формате +7XXXXXXXXXX');
			} else {
				phoneInput.setCustomValidity('');
			}
		});
	</script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func handleProcessPayment(w http.ResponseWriter, r *http.Request) {
	var data PaymentData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request format",
		})
		return
	}

	// Generate receipt
	receiptPath, err := generateReceipt(data)
	if err != nil {
		log.Printf("Error generating receipt: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Error generating receipt",
		})
		return
	}

	// Send email with receipt
	if err := sendEmail(data.Email, receiptPath); err != nil {
		log.Printf("Error sending email: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Error sending email",
		})
		return
	}

	// Insert payment transaction into the database
	err = insertPaymentTransaction("TRX-"+data.Email, data.Email, "Your Subscription Type", data.Amount, "Credit Card", data.CardNumber[len(data.CardNumber)-4:], "Success")
	if err != nil {
		log.Printf("Error inserting payment transaction: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Error saving payment transaction",
		})
		return
	}

	// Insert receipt into the database
	err = insertSubscriptionReceipt("TRX-"+data.Email, receiptPath, "Sent")
	if err != nil {
		log.Printf("Error inserting subscription receipt: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Error saving subscription receipt",
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Payment processed successfully",
	})
}