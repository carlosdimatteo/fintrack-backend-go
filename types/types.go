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
	DebtorId         int32   `json:"debtor_id"`
	DebtorName       string  `json:"debtor_name"`
	TotalLent        float64 `json:"total_lent"`     // Phase 1B - renamed
	TotalReceived    float64 `json:"total_received"` // Phase 1B - renamed
	NetOwed          float64 `json:"net_owed"`       // Phase 1B - renamed (positive = they owe you)
	TransactionCount int32   `json:"transaction_count"`
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
	// Phase 1B additions
	AccountId *int32 `json:"account_id,omitempty"` // Which account was affected
	ExpenseId *int32 `json:"expense_id,omitempty"` // Link to expense that caused this debt
	IncomeId  *int32 `json:"income_id,omitempty"`  // Link to income (for repayments)
}

type Investment struct {
	Id              int32   `json:"id,omitempty"`
	Description     string  `json:"description"`
	Amount          float64 `json:"amount"`
	AccountId       int32   `json:"account_id"`   // Investment account ID
	AccountName     string  `json:"account_name"` // Investment account name
	Date            string  `json:"date"`
	Type            string  `json:"type"`                        // "deposit" or "withdrawal"
	SourceAccountId *int32  `json:"source_account_id,omitempty"` // Fiat account funds come from/go to
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
	Id              int32     `json:"id,omitempty"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Type            string    `json:"type"`
	Currency        string    `json:"currency"`
	Balance         float64   `json:"balance"`
	StartingBalance float64   `json:"starting_balance,omitempty"` // Phase 1 addition
	StartingDate    time.Time `json:"starting_date,omitempty"`    // Phase 1 addition
}

type InvestmentAccount struct {
	Id              int32     `json:"id,omitempty"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Type            string    `json:"type"`
	Currency        string    `json:"currency"`
	Balance         float64   `json:"balance"`                    // Real balance from reconciliation
	Capital         float64   `json:"capital,omitempty"`          // Running total capital
	StartingCapital float64   `json:"starting_capital,omitempty"` // Phase 1 - fixed starting point
	StartingDate    time.Time `json:"starting_date,omitempty"`    // Phase 1 - when tracking started
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

// MonthlyIncomeSummary represents aggregated income for a month (from view)
type MonthlyIncomeSummary struct {
	Year        int     `json:"year"`
	Month       int     `json:"month"`
	TotalIncome float64 `json:"total_income"`
}

// Transfer represents a fiat-to-fiat money transfer (Phase 1B)
type Transfer struct {
	Id                int32     `json:"id,omitempty"`
	CreatedAt         time.Time `json:"created_at,omitempty"`
	Date              string    `json:"date"`
	Description       string    `json:"description"`
	SourceAccountId   int32     `json:"source_account_id"`
	SourceAccountName string    `json:"source_account_name,omitempty"`
	SourceAmount      float64   `json:"source_amount"`
	DestAccountId     int32     `json:"dest_account_id"`
	DestAccountName   string    `json:"dest_account_name,omitempty"`
	DestAmount        float64   `json:"dest_amount"`
	ExchangeRate      float64   `json:"exchange_rate,omitempty"` // dest_amount / source_amount
}

// AccountExpectedBalance from the view (Phase 1B)
type AccountExpectedBalance struct {
	Id                         int32     `json:"id"`
	Name                       string    `json:"name"`
	Currency                   string    `json:"currency"`
	StartingBalance            float64   `json:"starting_balance"`
	StartingDate               time.Time `json:"starting_date"`
	TotalIncome                float64   `json:"total_income"`
	TotalExpenses              float64   `json:"total_expenses"`
	TotalInvestmentDeposits    float64   `json:"total_investment_deposits"`
	TotalInvestmentWithdrawals float64   `json:"total_investment_withdrawals"`
	TotalTransfersOut          float64   `json:"total_transfers_out"`
	TotalTransfersIn           float64   `json:"total_transfers_in"`
	ExpectedBalance            float64   `json:"expected_balance"`
	RealBalance                float64   `json:"real_balance"`
	Discrepancy                float64   `json:"discrepancy"`
}

// InvestmentAccountSummary from the view (Phase 1)
type InvestmentAccountSummary struct {
	Id              int32   `json:"id"`
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	Currency        string  `json:"currency"`
	RealBalance     float64 `json:"real_balance"`
	TotalCapital    float64 `json:"total_capital"`
	StartingCapital float64 `json:"starting_capital"`
	PnL             float64 `json:"pnl"`
	PnLPercent      float64 `json:"pnl_percent"`
}

// InvestmentAccountExpectedCapital shows the breakdown of capital calculation
type InvestmentAccountExpectedCapital struct {
	Id              int32   `json:"id"`
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	Currency        string  `json:"currency"`
	StartingCapital float64 `json:"starting_capital"`
	TotalDeposits   float64 `json:"total_deposits"`
	TotalWithdrawals float64 `json:"total_withdrawals"`
	TotalExpenses   float64 `json:"total_expenses"`
	ExpectedCapital float64 `json:"expected_capital"`
	RealBalance     float64 `json:"real_balance"`
	Discrepancy     float64 `json:"discrepancy"` // real_balance - expected_capital = PnL
}

// YearlyGoals represents annual financial goals (Phase 4)
type YearlyGoals struct {
	Id              int32     `json:"id,omitempty"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
	Year            int       `json:"year"`
	SavingsGoal     float64   `json:"savings_goal"`
	InvestmentGoal  float64   `json:"investment_goal"`
	IdealInvestment float64   `json:"ideal_investment"`
}

// NetWorthSnapshot represents a monthly snapshot of net worth (Phase 4)
type NetWorthSnapshot struct {
	Id                     int32     `json:"id,omitempty"`
	CreatedAt              time.Time `json:"created_at,omitempty"`
	Date                   time.Time `json:"date"`
	Year                   int       `json:"year"`
	Month                  int       `json:"month"`
	// Real balances (from accounting/reconciliation)
	TotalFiatBalance       float64 `json:"total_fiat_balance"`
	CryptoBalance          float64 `json:"crypto_balance"`
	CryptoCapital          float64 `json:"crypto_capital"`
	BrokerBalance          float64 `json:"broker_balance"`
	BrokerCapital          float64 `json:"broker_capital"`
	TotalInvestmentBalance float64 `json:"total_investment_balance"`
	TotalInvestmentCapital float64 `json:"total_investment_capital"`
	TotalRealNetWorth      float64 `json:"total_real_net_worth"`
	TotalPnL               float64 `json:"total_pnl"`
	// Expected balances (from transactions)
	ExpectedFiatBalance float64 `json:"expected_fiat_balance"`
	ExpectedNetWorth    float64 `json:"expected_net_worth"`
	// Discrepancy
	FiatDiscrepancy     float64 `json:"fiat_discrepancy"`
	TotalDiscrepancy    float64 `json:"total_discrepancy"`
	// Percentages
	FiatPercent   float64 `json:"fiat_percent"`
	CryptoPercent float64 `json:"crypto_percent"`
	BrokerPercent float64 `json:"broker_percent"`
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
