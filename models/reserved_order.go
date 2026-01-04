package models

// ReservedOrder represents a reserved order in the database
type ReservedOrder struct {
	ID           int64  `json:"id"`
	Status       string `json:"status"` // reserved, completed, canceled
	AssignedTo   string `json:"assignedTo"`
	OrderType    string `json:"orderType"`
	CustomerName string `json:"customerName,omitempty"`
	CustomerPhone string `json:"customerPhone,omitempty"`
	Notes        string `json:"notes,omitempty"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// ReservedOrderLine represents a line item in a reserved order
type ReservedOrderLine struct {
	ID             int64  `json:"id"`
	ReservedOrderID int64  `json:"reservedOrderId"`
	ItemID         int64  `json:"itemId"`
	Qty            int    `json:"qty"`
	UnitPrice      int64  `json:"unitPrice"`
	CreatedAt      string `json:"createdAt"`
	// Item details (populated when joining with items table)
	ItemSKU   string `json:"itemSku,omitempty"`
	ItemSize  string `json:"itemSize,omitempty"`
	ItemPrice int64  `json:"itemPrice,omitempty"`
}

// CreateReservedOrderRequest represents the request body for creating a reserved order
// Example: {"assignedTo": "Erika", "orderType": "detal", "customerName": "Juan Pérez", "customerPhone": "+1234567890", "notes": "Cliente VIP"}
// orderType values: "detal" (retail) or "mayorista" (wholesale) - case-insensitive, will be normalized to lowercase
type CreateReservedOrderRequest struct {
	AssignedTo    string `json:"assignedTo"`
	OrderType     string `json:"orderType"` // "detal" or "mayorista" (case-insensitive)
	CustomerName  string `json:"customerName,omitempty"`
	CustomerPhone string `json:"customerPhone,omitempty"`
	Notes         string `json:"notes,omitempty"`
}

// AddItemToOrderRequest represents the request body for adding an item to a reserved order
// Example: {"itemId": 123, "qty": 2}
type AddItemToOrderRequest struct {
	ItemID int64 `json:"itemId"`
	Qty    int   `json:"qty"`
}

// ReservedOrderResponse represents the response for a single reserved order with its lines
// Example response:
// {
//   "id": 1,
//   "status": "reserved",
//   "assignedTo": "Erika",
//   "orderType": "detal",
//   "customerName": "Juan Pérez",
//   "customerPhone": "+1234567890",
//   "notes": "Cliente VIP",
//   "createdAt": "2024-01-15T10:30:00Z",
//   "updatedAt": "2024-01-15T10:30:00Z",
//   "lines": [
//     {
//       "id": 1,
//       "itemId": 123,
//       "itemSku": "MN_ABC123",
//       "itemSize": "MN",
//       "qty": 2,
//       "unitPrice": 50000,
//       "createdAt": "2024-01-15T10:30:00Z"
//     }
//   ],
//   "total": 100000
// }
type ReservedOrderResponse struct {
	ReservedOrder
	Lines []ReservedOrderLine `json:"lines"`
	Total int64               `json:"total"` // Sum of qty * unit_price for all lines
}

// ReservedOrderListItem represents a reserved order in a list response
type ReservedOrderListItem struct {
	ReservedOrder
	LineCount int   `json:"lineCount"` // Number of line items
	Total     int64 `json:"total"`     // Sum of qty * unit_price for all lines
}

// ReservedOrderListResponse represents the response for listing reserved orders
// Example response:
// {
//   "orders": [
//     {
//       "id": 1,
//       "status": "reserved",
//       "assignedTo": "Erika",
//       "orderType": "detal",
//       "customerName": "Juan Pérez",
//       "createdAt": "2024-01-15T10:30:00Z",
//       "updatedAt": "2024-01-15T10:30:00Z",
//       "lineCount": 2,
//       "total": 100000
//     }
//   ]
// }
type ReservedOrderListResponse struct {
	Orders []ReservedOrderListItem `json:"orders"`
}

// ItemFullInfo represents complete item information with design asset details
type ItemFullInfo struct {
	ID            int64  `json:"id"`
	SKU           string `json:"sku"`
	Size          string `json:"size"`
	Price         int64  `json:"price"`
	StockTotal    int    `json:"stockTotal"`
	StockReserved int    `json:"stockReserved"`
	DesignAssetID int    `json:"designAssetId"`
	// Design asset information
	Description    string `json:"description"`
	ColorPrimary   string `json:"colorPrimary"`
	ColorSecondary string `json:"colorSecondary"`
	HoodieType     string `json:"hoodieType"`
	ImageType      string `json:"imageType"`
	DecoID         string `json:"decoId"`
	DecoBase       string `json:"decoBase"`
	// Image endpoints
	ImageUrlThumb  string `json:"imageUrlThumb"`
	ImageUrlMedium string `json:"imageUrlMedium"`
}

// ReservedOrderLineWithItem represents a line item with complete item and design asset information
type ReservedOrderLineWithItem struct {
	ID             int64        `json:"id"`
	ReservedOrderID int64       `json:"reservedOrderId"`
	ItemID         int64        `json:"itemId"`
	Qty            int          `json:"qty"`
	UnitPrice      int64        `json:"unitPrice"`
	CreatedAt      string       `json:"createdAt"`
	Item           ItemFullInfo `json:"item"`
}

// ReservedOrderWithFullItems represents a reserved order with complete item information
type ReservedOrderWithFullItems struct {
	ReservedOrder
	Lines []ReservedOrderLineWithItem `json:"lines"`
	Total int64                        `json:"total"` // Sum of qty * unit_price for all lines
}

// SeparatedCartsResponse represents the response for separated carts endpoint
// Example response:
// {
//   "carts": [
//     {
//       "id": 1,
//       "status": "reserved",
//       "assignedTo": "Erika",
//       "orderType": "detal",
//       "customerName": "Juan Pérez",
//       "customerPhone": "+1234567890",
//       "notes": "Cliente VIP",
//       "createdAt": "2024-01-15T10:30:00Z",
//       "updatedAt": "2024-01-15T10:30:00Z",
//       "lines": [
//         {
//           "id": 1,
//           "reservedOrderId": 1,
//           "itemId": 123,
//           "qty": 2,
//           "unitPrice": 50000,
//           "createdAt": "2024-01-15T10:30:00Z",
//           "item": {
//             "id": 123,
//             "sku": "MN_ABC123",
//             "size": "MN",
//             "price": 50000,
//             "stockTotal": 10,
//             "stockReserved": 2,
//             "designAssetId": 45,
//             "description": "Hoodie con diseño especial",
//             "colorPrimary": "BL",
//             "colorSecondary": "NG",
//             "hoodieType": "HO",
//             "imageType": "FR",
//             "decoId": "123",
//             "decoBase": "BA",
//             "imageUrlThumb": "/admin/design-assets/pending/45/image?size=thumb",
//             "imageUrlMedium": "/admin/design-assets/pending/45/image?size=medium"
//           }
//         }
//       ],
//       "total": 100000
//     }
//   ]
// }
type SeparatedCartsResponse struct {
	Carts []ReservedOrderWithFullItems `json:"carts"`
}

