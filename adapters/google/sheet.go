package googleSS

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/carlosdimatteo/fintrack-backend-go/types"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	credentialsFile = "credentials.json"
	scopes          = "https://www.googleapis.com/auth/spreadsheets"
	sheetName       = "2024 Fintrack"
)

func getSheetService() (*sheets.SpreadsheetsValuesService, error) {
	ctx := context.Background()
	data, err := os.ReadFile(credentialsFile)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	client := option.WithCredentialsJSON(data)
	sheetService, err := sheets.NewService(ctx, client)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	sheetValueService := sheets.NewSpreadsheetsValuesService(sheetService)
	return sheetValueService, err
}

func SubmitExpenseRow(expensedata types.Expense, config types.Config) (*sheets.SpreadsheetsValuesAppendCall, error) {
	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	sheetValueService, err := getSheetService()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	dataToWrite := sheets.ValueRange{
		Values: [][]interface{}{
			{expensedata.Date,
				expensedata.Category,
				expensedata.Expense,
				expensedata.Description,
				expensedata.Method,
				expensedata.OriginalAmount,
				expensedata.CategoryId,
				expensedata.AccountId,
				expensedata.AccountType},
		},
	}
	configString := fmt.Sprint(config.Sheet, config.A1Range)
	sheetAndRange := func() string {
		if configString != "" {
			return configString
		}
		return "2024 Fintrack!A:I"
	}()
	_, err = sheetValueService.Append(
		spreadsheetID,
		sheetAndRange,
		&dataToWrite).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		log.Fatalf("Unable to write data to sheet: %v", err)
		return nil, err
	}
	return nil, nil
}
func SubmitBudget(budgets []types.Budget, config types.Config) (*sheets.SpreadsheetsValuesAppendCall, error) {
	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	sheetValueService, err := getSheetService()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	dataToWrite := sheets.ValueRange{
		Values: [][]interface{}{},
	}
	for _, budget := range budgets {
		dataToWrite.Values = append(dataToWrite.Values, []interface{}{budget.Amount})
	}
	_, err = sheetValueService.Append(
		spreadsheetID,
		fmt.Sprint(config.Sheet, config.A1Range),
		&dataToWrite).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		log.Fatalf("Unable to write data to sheet: %v", err)
		return nil, err
	}
	return nil, nil
}
func SubmitInvestment(investment types.Investment, config types.Config) (*sheets.SpreadsheetsValuesAppendCall, error) {
	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	sheetValueService, err := getSheetService()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	dataToWrite := sheets.ValueRange{
		Values: [][]interface{}{
			{investment.Date,
				investment.AccountId,
				investment.AccountName,
				investment.Description,
				investment.Amount,
				investment.Type,
			},
		},
	}
	_, err = sheetValueService.Append(
		spreadsheetID,
		fmt.Sprint(config.Sheet, config.A1Range),
		&dataToWrite).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		log.Fatalf("Unable to write data to sheet: %v", err)
		return nil, err
	}
	return nil, nil
}

func SubmitIncome(income types.Income, config types.Config) (*sheets.SpreadsheetsValuesAppendCall, error) {
	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	sheetValueService, err := getSheetService()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	dataToWrite := sheets.ValueRange{
		Values: [][]interface{}{
			{income.Date,
				income.AccountId,
				income.AccountName,
				income.Description,
				income.Amount,
			},
		},
	}
	_, err = sheetValueService.Append(
		spreadsheetID,
		fmt.Sprint(config.Sheet, config.A1Range),
		&dataToWrite).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		log.Fatalf("Unable to write data to sheet: %v", err)
		return nil, err
	}
	return nil, nil
}

func SubmitDebt(debt types.Debt, config types.Config) (*sheets.SpreadsheetsValuesAppendCall, error) {
	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	sheetValueService, err := getSheetService()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	typeString := "Borrowed"
	if debt.Outbound {
		typeString = "Lent"
	}
	dataToWrite := sheets.ValueRange{
		Values: [][]interface{}{
			{debt.Date,
				debt.DebtorId,
				debt.DebtorName,
				debt.Description,
				debt.Amount,
				typeString,
				debt.Outbound,
			},
		},
	}
	_, err = sheetValueService.Append(
		spreadsheetID,
		fmt.Sprint(config.Sheet, config.A1Range),
		&dataToWrite).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		log.Fatalf("Unable to write data to sheet: %v", err)
		return nil, err
	}
	return nil, nil
}

func SubmitAccount(account types.Account, config types.Config) (*sheets.SpreadsheetsValuesAppendCall, error) {
	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	sheetValueService, err := getSheetService()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	dataToWrite := sheets.ValueRange{
		Values: [][]interface{}{
			{account.Id,
				account.Name,
				account.Type,
				account.Currency,
			},
		},
	}
	sheetRange := fmt.Sprint(config.Sheet, config.A1Range)
	fmt.Println(sheetRange)
	_, err = sheetValueService.Append(
		spreadsheetID,
		sheetRange,
		&dataToWrite).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		log.Fatalf("Unable to write data to sheet: %v", err)
		return nil, err
	}
	return nil, nil
}
func UpdateAccountBalances(accounts []types.Account, config types.Config) (*sheets.SpreadsheetsValuesUpdateCall, error) {
	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	sheetValueService, err := getSheetService()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	dataToWrite := sheets.ValueRange{
		Values: [][]interface{}{},
		Range:  fmt.Sprint(config.Sheet, config.A1Range),
	}
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].Id < accounts[j].Id
	})
	for _, account := range accounts {

		dataToWrite.Values = append(dataToWrite.Values, []interface{}{account.Balance})
	}
	batchUpdateValueReq := sheets.BatchUpdateValuesRequest{
		Data:             []*sheets.ValueRange{&dataToWrite},
		ValueInputOption: "USER_ENTERED",
	}
	_, err = sheetValueService.BatchUpdate(
		spreadsheetID,
		&batchUpdateValueReq).Do()
	if err != nil {
		log.Fatalf("Unable to write data to sheet: %v", err)
		return nil, err
	}
	return nil, nil

}
func SubmitInvestmentAccount(investmentAccount types.InvestmentAccount, config types.Config) (*sheets.SpreadsheetsValuesAppendCall, error) {
	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	sheetValueService, err := getSheetService()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	dataToWrite := sheets.ValueRange{
		Values: [][]interface{}{
			{investmentAccount.Id,
				investmentAccount.Name,
				investmentAccount.Type,
				investmentAccount.Currency,
			},
		},
	}
	_, err = sheetValueService.Append(
		spreadsheetID,
		fmt.Sprint(config.Sheet, config.A1Range),
		&dataToWrite).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		log.Fatalf("Unable to write data to sheet: %v", err)
		return nil, err
	}
	return nil, nil
}

func UpdateInvestmentAccountBalances(accounts []types.InvestmentAccount, config types.Config) (*sheets.SpreadsheetsValuesUpdateCall, error) {
	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	sheetValueService, err := getSheetService()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	dataToWrite := sheets.ValueRange{
		Values: [][]interface{}{},
		Range:  fmt.Sprint(config.Sheet, config.A1Range),
	}
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].Id < accounts[j].Id
	})
	for _, account := range accounts {

		dataToWrite.Values = append(dataToWrite.Values, []interface{}{account.Balance})
	}
	batchUpdateValueReq := sheets.BatchUpdateValuesRequest{
		Data:             []*sheets.ValueRange{&dataToWrite},
		ValueInputOption: "USER_ENTERED",
	}
	_, err = sheetValueService.BatchUpdate(
		spreadsheetID,
		&batchUpdateValueReq).Do()
	if err != nil {
		log.Fatalf("Unable to write data to sheet: %v", err)
		return nil, err
	}
	return nil, nil

}
func SubmitDebtor(debtor types.Debtor, config types.Config) (*sheets.SpreadsheetsValuesAppendCall, error) {
	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	sheetValueService, err := getSheetService()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	dataToWrite := sheets.ValueRange{
		Values: [][]interface{}{
			{debtor.Id,
				debtor.Name,
			},
		},
	}
	_, err = sheetValueService.Append(
		spreadsheetID,
		fmt.Sprint(config.Sheet, config.A1Range),
		&dataToWrite).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		log.Fatalf("Unable to write data to sheet: %v", err)
		return nil, err
	}
	return nil, nil
}

// UpdateSheetCell updates a single cell with a value (uses Update, not Append)
// sheetRange should be in format "SheetName!A1" (e.g., "2026!D3")
func UpdateSheetCell(sheetRange string, value interface{}) error {
	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	sheetValueService, err := getSheetService()
	if err != nil {
		return err
	}

	dataToWrite := sheets.ValueRange{
		Values: [][]interface{}{{value}},
	}

	_, err = sheetValueService.Update(
		spreadsheetID,
		sheetRange,
		&dataToWrite,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		log.Printf("Unable to update cell %s: %v", sheetRange, err)
		return err
	}

	log.Printf("Updated cell %s with value: %v", sheetRange, value)
	return nil
}

// CalculateMonthlyCellRange calculates the cell range for a given month
// baseRange is the config range (e.g., "!D3" for January)
// month is 1-12
// Returns full range like "2026!D5" for March (month=3)
func CalculateMonthlyCellRange(sheet string, baseRange string, month int) string {
	// baseRange format: "!D3" where 3 is January (month 1)
	// So row = baseRow + (month - 1)
	// For D3 as January: D3=Jan, D4=Feb, D5=Mar, etc.

	// Extract column letter and base row from baseRange
	// baseRange is like "!D3"
	if len(baseRange) < 3 {
		return sheet + baseRange
	}

	// Find where the number starts
	col := ""
	rowStr := ""
	for i, c := range baseRange[1:] { // skip the "!"
		if c >= '0' && c <= '9' {
			col = baseRange[1 : i+1]
			rowStr = baseRange[i+1:]
			break
		}
	}

	if col == "" || rowStr == "" {
		return sheet + baseRange
	}

	baseRow := 0
	fmt.Sscanf(rowStr, "%d", &baseRow)
	targetRow := baseRow + (month - 1)

	return fmt.Sprintf("%s!%s%d", sheet, col, targetRow)
}
