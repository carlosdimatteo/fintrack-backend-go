package types

import "time"

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type Category struct {
	Id          int32     `json:"id,omitempty"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsEssential bool      `json:"is_essential"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

type Budget struct {
	Id         int32   `json:"id,omitempty"`
	CategoryId int32   `json:"category_id"`
	Amount     float64 `json:"amount"`
}

type Expense struct {
	Id             int32   `json:"id,omitempty"`
	Date           string  `json:"date"`
	Category       string  `json:"category"`
	CategoryId     int32   `json:"category_id"`
	Expense        float64 `json:"expense"`
	Description    string  `json:"description"`
	Method         string  `json:"method"`
	OriginalAmount float64 `json:"originalAmount"`
	AccountId      int32   `json:"account_id"`
	AccountType    string  `json:"account_type"`
}

type BudgetByCategory struct {
	Amount       float64 `json:"amount"`
	Spent        float64 `json:"spent"`
	CategoryName string  `json:"category_name"`
	CategoryId   int32   `json:"category_id"`
}

type DebtByDebtor struct {
	DebtorId            int32   `json:"debtor_id"`
	DebtorName          string  `json:"debtor_name"`
	NetAmount           float64 `json:"net_amount"`
	TotalAmountBorrowed float64 `json:"total_amount_borrowed"`
	TotalAmountLent     float64 `json:"total_amount_lent"`
	TransactionCount    int32   `json:"transaction_count"`
}

type Config struct {
	Sheet   string `json:"sheet"`
	A1Range string `json:"range"`
	Type    string `json:"type"`
}

type Debt struct {
	Id             int32     `json:"id,omitempty"`
	Description    string    `json:"description"`
	Amount         float64   `json:"amount"`
	DebtorId       int32     `json:"debtor_id"`
	DebtorName     string    `json:"debtor_name"`
	Date           string    `json:"date"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	OriginalAmount float64   `json:"original_amount"`
	Currency       string    `json:"currency"`
	Outbound       bool      `json:"outbound"`
}

type Investment struct {
	Id          int32   `json:"id,omitempty"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	AccountId   int32   `json:"account_id"`
	AccountName string  `json:"account_name"`
	Date        string  `json:"date"`
	Type        string  `json:"type"`
}

type Income struct {
	Id          int32     `json:"id,omitempty"`
	Date        string    `json:"date,omitempty"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	AccountId   int32     `json:"account_id"`
	AccountName string    `json:"account_name"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

type Account struct {
	Id          int32   `json:"id,omitempty"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Type        string  `json:"type"`
	Currency    string  `json:"currency"`
	Balance     float64 `json:"balance"`
}

type InvestmentAccount struct {
	Id          int32   `json:"id,omitempty"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Type        string  `json:"type"`
	Currency    string  `json:"currency"`
	Balance     float64 `json:"balance"`
	Capital     float64 `json:"capital,omitempty"`
}

type Debtor struct {
	Id          int32  `json:"id,omitempty"`
	Name        string `json:"name"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Description string `json:"description"`
}

type RealBalanceByAccounts struct {
	Accounts           []Account           `json:"accounts"`
	InvestmentAccounts []InvestmentAccount `json:"investment_accounts,omitempty"`
}

var ConfigType map[string]string

func init() {
	ConfigType = map[string]string{
		"expense":                        "expense",
		"budget":                         "budget",
		"investment":                     "investment",
		"category":                       "category",
		"accounting_accounts":            "accounting_accounts",
		"accounting_investment_accounts": "accounting_investment_accounts",
	}
}
