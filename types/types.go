package types

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type Category struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsEssential bool   `json:"is_essential"`
}

type Budget struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Spent    float64 `json:"spent"`
}

type Expense struct {
	Date           string  `json:"date"`
	Category       string  `json:"category"`
	Expense        float64 `json:"expense"`
	Description    string  `json:"description"`
	Method         string  `json:"method"`
	OriginalAmount float64 `json:"originalAmount"`
}

type BudgetByCategory struct {
	Amount       float64 `json:"amount"`
	Spent        float64 `json:"spent"`
	CategoryName string  `json:"category_name"`
	CategoryId   int32   `json:"category_id"`
}
