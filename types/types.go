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
	Id         int32   `json:"id"`
	CategoryId int32   `json:"category_id"`
	Amount     float64 `json:"amount"`
}

type Expense struct {
	Date           string  `json:"date"`
	Category       string  `json:"category"`
	CategoryId     int32   `json:"category_id"`
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

type Config struct {
	Sheet   string `json:"sheet"`
	A1Range string `json:"range"`
	Type    string `json:"type"`
}

var ConfigType map[string]string

func init() {
	ConfigType = map[string]string{
		"expense":    "expense",
		"budget":     "budget",
		"investment": "investment",
		"category":   "category",
		"accounting": "accounting",
	}
}
