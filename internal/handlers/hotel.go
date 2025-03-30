// backend/internal/handlers/hotel.go
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yourusername/hotel-bed-management/backend/internal/db"
	"github.com/yourusername/hotel-bed-management/backend/internal/models"
)

type HotelHandler struct {
	db *db.Database
}

func NewHotelHandler(db *db.Database) *HotelHandler {
	return &HotelHandler{db: db}
}

func (h *HotelHandler) GetHotels(c *gin.Context) {
	rows, err := h.db.Pool.Query(c, "SELECT id, name, description, created_at FROM hotels")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch hotels"})
		return
	}
	defer rows.Close()

	var hotels []models.Hotel
	for rows.Next() {
		var hotel models.Hotel
		if err := rows.Scan(&hotel.ID, &hotel.Name, &hotel.Description, &hotel.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan hotel"})
			return
		}
		hotels = append(hotels, hotel)
	}

	c.JSON(http.StatusOK, hotels)
}

func (h *HotelHandler) GetBeds(c *gin.Context) {
	hotelID := c.Param("hotelId")

	rows, err := h.db.Pool.Query(c, `
		SELECT b.id, b.hotel_id, b.bed_number, b.position, b.status, b.created_at,
		       c.id, c.full_name, c.mobile_number, c.check_in, c.check_out, 
		       c.amount_paid, c.payment_mode
		FROM beds b
		LEFT JOIN customers c ON b.id = c.bed_id AND c.check_out IS NULL
		WHERE b.hotel_id = $1
		ORDER BY b.position, b.bed_number
	`, hotelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch beds"})
		return
	}
	defer rows.Close()

	type BedWithCustomer struct {
		models.Bed
		Customer *models.Customer `json:"customer,omitempty"`
	}

	var beds []BedWithCustomer
	for rows.Next() {
		var bed BedWithCustomer
		var customerID *int
		var fullName, mobileNumber, paymentMode *string
		var checkIn, checkOut *time.Time
		var amountPaid *float64

		err := rows.Scan(
			&bed.ID, &bed.HotelID, &bed.BedNumber, &bed.Position, &bed.Status, &bed.CreatedAt,
			&customerID, &fullName, &mobileNumber, &checkIn, &checkOut, &amountPaid, &paymentMode,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan bed"})
			return
		}

		if customerID != nil {
			// Initialize with zero values first
			customer := &models.Customer{
				BedID: &bed.ID,
			}

			// Safely assign values with nil checks
			if customerID != nil {
				customer.ID = *customerID
			}
			if fullName != nil {
				customer.FullName = *fullName
			}
			if mobileNumber != nil {
				customer.MobileNumber = *mobileNumber
			}
			if checkIn != nil {
				customer.CheckIn = *checkIn
			}
			if checkOut != nil {
				customer.CheckOut = *checkOut
			} else {
				// Set to zero time if nil
				customer.CheckOut = time.Time{}
			}
			if amountPaid != nil {
				customer.AmountPaid = *amountPaid
			}
			if paymentMode != nil {
				customer.PaymentMode = *paymentMode
			}

			bed.Customer = customer
		}

		beds = append(beds, bed)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing beds"})
		return
	}

	c.JSON(http.StatusOK, beds)
}

func (h *HotelHandler) CreateCustomer(c *gin.Context) {
	var req models.CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	tx, err := h.db.Pool.Begin(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(c)

	// Check if bed is available
	var bedStatus string
	err = tx.QueryRow(c, "SELECT status FROM beds WHERE id = $1 FOR UPDATE", req.BedID).Scan(&bedStatus)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bed not found"})
		return
	}

	if bedStatus != "available" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bed is not available"})
		return
	}

	// Create customer
	var customerID int
	err = tx.QueryRow(c, `
		INSERT INTO customers (
			full_name, mobile_number, check_in, check_out, 
			amount_paid, payment_mode, bed_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, req.FullName, req.MobileNumber, req.CheckIn, req.CheckOut, req.AmountPaid, req.PaymentMode, req.BedID,
	).Scan(&customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create customer"})
		return
	}

	// Update bed status
	_, err = tx.Exec(c, "UPDATE beds SET status = 'occupied' WHERE id = $1", req.BedID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bed status"})
		return
	}

	if err := tx.Commit(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": customerID})
}

func (h *HotelHandler) CheckoutCustomer(c *gin.Context) {
	var req models.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	tx, err := h.db.Pool.Begin(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(c)

	// Get customer details including bed_id
	var bedID int
	err = tx.QueryRow(c, "SELECT bed_id FROM customers WHERE id = $1 AND check_out IS NULL FOR UPDATE", req.CustomerID).Scan(&bedID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Customer not found or already checked out"})
		return
	}

	// Update customer checkout time
	_, err = tx.Exec(c, "UPDATE customers SET check_out = NOW() WHERE id = $1", req.CustomerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update customer checkout"})
		return
	}

	// Update bed status
	_, err = tx.Exec(c, "UPDATE beds SET status = 'available' WHERE id = $1", bedID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bed status"})
		return
	}

	if err := tx.Commit(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Customer checked out successfully"})
}
