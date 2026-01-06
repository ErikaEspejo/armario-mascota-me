package models

// Sale represents a sale in the database
type Sale struct {
	ID                int64  `json:"id"`
	ReservedOrderID   int64  `json:"reservedOrderId"`
	SoldAt            string `json:"soldAt"`
	CustomerName      string `json:"customerName,omitempty"`
	AmountPaid        int64  `json:"amountPaid"`
	PaymentMethod     string `json:"paymentMethod"`
	PaymentDestination string `json:"paymentDestination"`
	Status            string `json:"status"`
	Notes             string `json:"notes,omitempty"`
	CreatedAt         string `json:"createdAt"`
}

// SellRequest represents the request body for selling a reserved order
// Example: {"amountPaid": 100000, "paymentMethod": "transfer", "paymentDestination": "Nequi", "notes": "Pago completo"}
type SellRequest struct {
	AmountPaid         int64  `json:"amountPaid"`
	PaymentMethod      string `json:"paymentMethod"`
	PaymentDestination string `json:"paymentDestination"`
	Notes              string `json:"notes,omitempty"`
}

// SaleResponse represents the response for a sale
// Example response:
// {
//   "id": 10,
//   "reservedOrderId": 3,
//   "soldAt": "2026-01-04T10:30:00Z",
//   "customerName": "Juan Pérez",
//   "amountPaid": 100000,
//   "paymentMethod": "transfer",
//   "paymentDestination": "Nequi",
//   "status": "paid",
//   "notes": "Pago completo",
//   "createdAt": "2026-01-04T10:30:00Z"
// }
type SaleResponse struct {
	Sale
}

// SaleListItem represents a sale in a list response
type SaleListItem struct {
	ID                int64  `json:"id"`
	SoldAt            string `json:"soldAt"`
	ReservedOrderID   int64  `json:"reservedOrderId"`
	CustomerName      string `json:"customerName,omitempty"`
	AmountPaid        int64  `json:"amountPaid"`
	PaymentDestination string `json:"paymentDestination"`
	PaymentMethod     string `json:"paymentMethod"`
}

// SaleListResponse represents the response for listing sales
// Example response:
// {
//   "sales": [
//     {
//       "id": 10,
//       "soldAt": "2026-01-04T10:30:00Z",
//       "reservedOrderId": 3,
//       "customerName": "Juan Pérez",
//       "amountPaid": 100000,
//       "paymentDestination": "Nequi",
//       "paymentMethod": "transfer"
//     }
//   ]
// }
type SaleListResponse struct {
	Sales []SaleListItem `json:"sales"`
}

// SaleDetailResponse represents the response for a sale detail with order information
// Example response:
// {
//   "id": 10,
//   "reservedOrderId": 3,
//   "soldAt": "2026-01-04T10:30:00Z",
//   "customerName": "Juan Pérez",
//   "amountPaid": 100000,
//   "paymentMethod": "transfer",
//   "paymentDestination": "Nequi",
//   "status": "paid",
//   "notes": "Pago completo",
//   "createdAt": "2026-01-04T10:30:00Z",
//   "order": {
//     "id": 3,
//     "status": "completed",
//     ...
//   }
// }
type SaleDetailResponse struct {
	Sale
	Order *ReservedOrderResponse `json:"order"`
}


