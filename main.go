package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []Item `json:"items"`
	Total        string `json:"total"`
}

type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

type PointsResponse struct {
	Points int `json:"points"`
}

type ReceiptResponse struct {
	ID string `json:"id"`
}

var (
	receipts = make(map[string]Receipt)
	points   = make(map[string]int)
	mutex    sync.Mutex
)

func processReceipt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var receipt Receipt
	if err := json.NewDecoder(r.Body).Decode(&receipt); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	id := uuid.New().String()
	mutex.Lock()
	receipts[id] = receipt
	points[id] = calculatePoints(receipt)
	mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ReceiptResponse{ID: id})
}

func getPoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/receipts/")
	id = strings.TrimSuffix(id, "/points")

	mutex.Lock()
	pts, exists := points[id]
	mutex.Unlock()

	if !exists {
		http.Error(w, "Receipt not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PointsResponse{Points: pts})
}

func calculatePoints(receipt Receipt) int {
	points := 0

	// 1 point for each alphanumeric character in retailer name
	alphanumeric := regexp.MustCompile(`[a-zA-Z0-9]`)
	points += len(alphanumeric.FindAllString(receipt.Retailer, -1))

	// 50 points if total is a round dollar amount
	if strings.HasSuffix(receipt.Total, ".00") {
		points += 50
	}

	// 25 points if total is a multiple of 0.25
	total, err := strconv.ParseFloat(receipt.Total, 64)
	if err == nil && math.Mod(total, 0.25) == 0 {
		points += 25
	}

	// 5 points for every two items
	points += (len(receipt.Items) / 2) * 5

	// Points for item descriptions that are multiples of 3
	for _, item := range receipt.Items {
		trimmedDesc := strings.TrimSpace(item.ShortDescription)
		if len(trimmedDesc)%3 == 0 {
			itemPrice, err := strconv.ParseFloat(item.Price, 64)
			if err == nil {
				points += int(math.Ceil(itemPrice * 0.2))
			}
		}
	}

	// 6 points if purchase day is odd
	purchaseDate, err := time.Parse("2006-01-02", receipt.PurchaseDate)
	if err == nil && purchaseDate.Day()%2 != 0 {
		points += 6
	}

	// 10 points if purchase time is between 2:00 PM and 4:00 PM
	purchaseTime, err := time.Parse("15:04", receipt.PurchaseTime)
	if err == nil && purchaseTime.Hour() == 14 {
		points += 10
	}

	return points
}

func main() {
	http.HandleFunc("/receipts/process", processReceipt)
	http.HandleFunc("/receipts/", getPoints)
	fmt.Println("Server is running on port 8080...")
	http.ListenAndServe(":8080", nil)
}
