package googleSS

import (
	"context"
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

func SubmitExpenseRow(expensedata types.Expense) (*sheets.SpreadsheetsValuesAppendCall, error) {
	ctx := context.Background()
	spreadsheetID := os.Getenv("SPREADSHEET_ID")
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
	/**

		 [
	            new Date().toDateString(),
	            category,
	            expense,
	            description,
	            method,
	            originalAmount,
	          ],
	*/
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
	_, err = sheetValueService.Append(
		spreadsheetID,
		"2024 Fintrack!A:F",
		&dataToWrite).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		log.Fatalf("Unable to write data to sheet: %v", err)
		return nil, err
	}
	return nil, nil
}
