package googleSS

import (
	"context"
	"fmt"
	"log"
	"os"

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
				expensedata.OriginalAmount},
		},
	}
	configString := fmt.Sprint(config.Sheet, config.A1Range)
	sheetAndRange := func() string {
		if configString != "" {
			return configString
		}
		return "2024 Fintrack!A:F"
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
	dataToWrite := sheets.ValueRange{
		Values: [][]interface{}{
			{debt.Date,
				debt.DebtorId,
				debt.DebtorName,
				debt.Description,
				debt.Amount,
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
