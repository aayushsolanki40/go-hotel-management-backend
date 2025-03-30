// backend/internal/models/hotel.go
package models

import "time"

type Hotel struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Bed struct {
	ID        int       `json:"id"`
	HotelID   int       `json:"hotel_id"`
	BedNumber string    `json:"bed_number"`
	Position  string    `json:"position"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Customer struct {
	ID           int       `json:"id"`
	FullName     string    `json:"full_name"`
	MobileNumber string    `json:"mobile_number"`
	CheckIn      time.Time `json:"check_in"`
	CheckOut     time.Time `json:"check_out"`
	AmountPaid   float64   `json:"amount_paid"`
	PaymentMode  string    `json:"payment_mode"`
	BedID        *int      `json:"bed_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateCustomerRequest struct {
	FullName     string    `json:"full_name" validate:"required"`
	MobileNumber string    `json:"mobile_number" validate:"required"`
	CheckIn      time.Time `json:"check_in" validate:"required"`
	CheckOut     time.Time `json:"check_out" validate:"required"`
	AmountPaid   float64   `json:"amount_paid" validate:"required"`
	PaymentMode  string    `json:"payment_mode" validate:"required"`
	BedID        int       `json:"bed_id" validate:"required"`
}

type CheckoutRequest struct {
	CustomerID int `json:"customer_id" validate:"required"`
}