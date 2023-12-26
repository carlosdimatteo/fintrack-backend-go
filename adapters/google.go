package googleSS

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	credentialsFile = "credentials.json"
	scopes          = "https://www.googleapis.com/auth/spreadsheets"
	spreadsheetID   = "1tufwF9c3WwhrZF7o79zL-lBZKs8ccrZ17lXR0cVWIio"
	sheetName       = "Fintrack"
)

type FintrackRow struct {
	Date           string
	Category       string
	Expense        string
	Description    string
	Method         string
	OriginalAmount string
}

func SubmitRow(expensedata FintrackRow) (*sheets.SpreadsheetsValuesAppendCall, error) {
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

	/** get metadata as a way to test if adapter works, erase later */
	// Get metadata about the spreadsheet
	spreadsheetService := sheets.NewSpreadsheetsService(sheetService)
	metaData, err := spreadsheetService.Get(spreadsheetID).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve spreadsheet metadata: %v", err)
		return nil, err
	}
	fmt.Printf("Spreadsheet title: %s\n", metaData.Properties.Title)

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
		"Fintrack!A:F",
		&dataToWrite).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		log.Fatalf("Unable to write data to sheet: %v", err)
		return nil, err
	}
	return nil, nil
}
