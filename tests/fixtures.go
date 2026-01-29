package tests

// ========== TEST FIXTURE CONSTANTS ==========
// Single source of truth for all test data

// Test Account IDs (use high IDs to avoid conflicts with production data)
const (
	TestAccountBankID    int32 = 100
	TestAccountSavingsID int32 = 101
	TestAccountCOPID     int32 = 102
)

// Test Investment Account IDs
const (
	TestInvAccountCryptoID int32 = 100
	TestInvAccountBrokerID int32 = 101
)

// Test Category IDs
const (
	TestCategoryFoodID      int32 = 100
	TestCategoryTransportID int32 = 101
	TestCategoryUtilitiesID int32 = 102
)

// Test Debtor IDs
const (
	TestDebtorJohnID int32 = 100
	TestDebtorJaneID int32 = 101
)

// ========== TEST FIXTURE DATA ==========

// TestAccount represents a test fiat account
type TestAccountData struct {
	ID              int32
	Name            string
	Description     string
	Type            string
	Currency        string
	Balance         float64
	StartingBalance float64
	StartingDate    string
}

// TestInvestmentAccount represents a test investment account
type TestInvestmentAccountData struct {
	ID              int32
	Name            string
	Description     string
	Type            string
	Currency        string
	Balance         float64
	Capital         float64
	StartingCapital float64
	StartingDate    string
}

// TestCategory represents a test category
type TestCategoryData struct {
	ID          int32
	Name        string
	Description string
	IsEssential bool
}

// TestDebtor represents a test debtor
type TestDebtorData struct {
	ID        int32
	Name      string
	FirstName string
	LastName  string
}

// ========== FIXTURE INSTANCES ==========

var TestAccounts = []TestAccountData{
	{
		ID:              TestAccountBankID,
		Name:            "TestBank",
		Description:     "Test checking account",
		Type:            "Fiat",
		Currency:        "USD",
		Balance:         1000,
		StartingBalance: 1000,
		StartingDate:    "2026-01-01",
	},
	{
		ID:              TestAccountSavingsID,
		Name:            "TestSavings",
		Description:     "Test savings account",
		Type:            "Fiat",
		Currency:        "USD",
		Balance:         500,
		StartingBalance: 500,
		StartingDate:    "2026-01-01",
	},
	{
		ID:              TestAccountCOPID,
		Name:            "TestCOP",
		Description:     "Test COP account",
		Type:            "Fiat",
		Currency:        "COP",
		Balance:         2000000,
		StartingBalance: 2000000,
		StartingDate:    "2026-01-01",
	},
}

var TestInvestmentAccounts = []TestInvestmentAccountData{
	{
		ID:              TestInvAccountCryptoID,
		Name:            "TestCrypto",
		Description:     "Test crypto account",
		Type:            "Crypto",
		Currency:        "USD",
		Balance:         1200, // Real balance from "reconciliation"
		Capital:         800,
		StartingCapital: 800,
		StartingDate:    "2026-01-01",
	},
	{
		ID:              TestInvAccountBrokerID,
		Name:            "TestBroker",
		Description:     "Test broker account",
		Type:            "Broker",
		Currency:        "USD",
		Balance:         5500, // Real balance from "reconciliation"
		Capital:         4000,
		StartingCapital: 4000,
		StartingDate:    "2026-01-01",
	},
}

var TestCategories = []TestCategoryData{
	{ID: TestCategoryFoodID, Name: "TestFood", Description: "Test food category", IsEssential: true},
	{ID: TestCategoryTransportID, Name: "TestTransport", Description: "Test transport category", IsEssential: false},
	{ID: TestCategoryUtilitiesID, Name: "TestUtilities", Description: "Test utilities category", IsEssential: true},
}

var TestDebtors = []TestDebtorData{
	{ID: TestDebtorJohnID, Name: "TestJohn", FirstName: "John", LastName: "Doe"},
	{ID: TestDebtorJaneID, Name: "TestJane", FirstName: "Jane", LastName: "Smith"},
}

// ========== HELPER FUNCTIONS ==========

// GetTestAccount returns the test account data by ID
func GetTestAccount(id int32) *TestAccountData {
	for _, acc := range TestAccounts {
		if acc.ID == id {
			return &acc
		}
	}
	return nil
}

// GetTestInvestmentAccount returns the test investment account data by ID
func GetTestInvestmentAccount(id int32) *TestInvestmentAccountData {
	for _, acc := range TestInvestmentAccounts {
		if acc.ID == id {
			return &acc
		}
	}
	return nil
}

// GetTestCategory returns the test category data by ID
func GetTestCategory(id int32) *TestCategoryData {
	for _, cat := range TestCategories {
		if cat.ID == id {
			return &cat
		}
	}
	return nil
}

// GetTestDebtor returns the test debtor data by ID
func GetTestDebtor(id int32) *TestDebtorData {
	for _, d := range TestDebtors {
		if d.ID == id {
			return &d
		}
	}
	return nil
}
