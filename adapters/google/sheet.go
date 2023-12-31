package googleSS

import (
	"context"
	"log"
	"os"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	credentialsFile = "credentials.json"
	scopes          = "https://www.googleapis.com/auth/spreadsheets"
	sheetName       = "2024 Fintrack"
)

type FintrackRow struct {
	Date           string  `json:"date"`
	Category       string  `json:"category"`
	Expense        float64 `json:"expense"`
	Description    string  `json:"description"`
	Method         string  `json:"method"`
	OriginalAmount float64 `json:"originalAmount"`
}

func SubmitRow(expensedata FintrackRow) (*sheets.SpreadsheetsValuesAppendCall, error) {
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

	/** get metadata as a way to test if adapter works, erase later */
	// Get metadata about the spreadsheet
	// spreadsheetService := sheets.NewSpreadsheetsService(sheetService)
	// metaData, err := spreadsheetService.Get(spreadsheetID).Do()
	// if err != nil {
	// 	log.Fatalf("Unable to retrieve spreadsheet metadata: %v", err)
	// 	return nil, err
	// }
	// fmt.Printf("Spreadsheet title: %s\n", metaData.Properties.Title)

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
