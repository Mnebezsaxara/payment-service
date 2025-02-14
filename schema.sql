-- Add these tables to your existing schema
CREATE TABLE payment_transactions (
    id SERIAL PRIMARY KEY,
    transaction_id VARCHAR(50) NOT NULL,
    customer_email VARCHAR(255) NOT NULL,
    subscription_type VARCHAR(50) NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    payment_method VARCHAR(50) NOT NULL,
    card_last_four VARCHAR(4) NOT NULL,
    payment_status VARCHAR(20) NOT NULL,
    payment_time TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE subscription_receipts (
    id SERIAL PRIMARY KEY,
    transaction_id VARCHAR(50) NOT NULL,
    receipt_path VARCHAR(255) NOT NULL,
    email_status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
); 