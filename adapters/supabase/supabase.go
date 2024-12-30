package supabase

import (
	"context"
	"fmt"
	"os"

	types "github.com/carlosdimatteo/fintrack-backend-go/types"
	supa "github.com/nedpals/supabase-go"
)

func getSupabaseClient() (*supa.Client, error) {
	supabaseUrl := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_KEY")
	supabase := supa.CreateClient(supabaseUrl, supabaseKey)
	ctx := context.Background()
	user, err := supabase.Auth.SignIn(ctx, supa.UserCredentials{
		Email:    os.Getenv("SUPABASE_USER"),
		Password: os.Getenv("SUPABASE_USER_PWD"),
	})

	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	supabase.DB.AddHeader("Authorization", "Bearer "+user.AccessToken)
	supabase.DB.Headers().Get("Authorization")

	return supabase, nil
}

func GetCategories() ([]types.Category, error) {
	supabaseUrl := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_KEY")
	supabase := supa.CreateClient(supabaseUrl, supabaseKey)
	ctx := context.Background()
	user, err := supabase.Auth.SignIn(ctx, supa.UserCredentials{
		Email:    os.Getenv("SUPABASE_USER"),
		Password: os.Getenv("SUPABASE_USER_PWD"),
	})
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}
	fmt.Println("supabase user", user.User.Email)
	var results []types.Category
	supabase.DB.AddHeader("Authorization", "Bearer "+user.AccessToken)
	err = supabase.DB.From("categories").Select().Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	fmt.Println(results)
	return results, nil
}

func GetConfig() ([]types.Config, error) {
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}
	var results []types.Config
	err = supabase.DB.From("config").Select().Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	fmt.Println(results)
	return results, nil

}

func GetConfigByType(configType string) (types.Config, error) {

	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return types.Config{}, err
	}
	var results types.Config
	err = supabase.DB.From("config").Select().Single().Eq("type", configType).Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return types.Config{}, err
	}

	fmt.Println(results)
	return results, nil
}
func InsertConfigIntoDatabase(toInsert []types.Config) ([]types.Config, error) {
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}
	var results []types.Config
	query := supabase.DB.From("config").Upsert(toInsert)
	err = query.Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	fmt.Println("results", results) // Inserted rows
	return results, nil
}
func GetBudgets() ([]types.BudgetByCategory, error) {
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}
	results := []types.BudgetByCategory{}
	err = supabase.DB.From("budget_by_category_current_month").Select().Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	fmt.Println(results)
	return results, nil
}
func InsertBudgetsIntoDatabase(toInsert []types.Budget) ([]types.Budget, error) {
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}
	var results []types.Budget

	query := supabase.DB.From("budgets").Upsert(toInsert)

	err = query.Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	fmt.Println("results", results) // Inserted rows
	return results, nil
}

func InsertExpenseIntoDatabase(rowToInsert types.Expense) ([]types.Expense, error) {
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}
	var results []types.Expense
	err = supabase.DB.From("expenses").Insert(rowToInsert).Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	fmt.Println(results) // Inserted rows
	return results, nil
}

func GetExpenses(limit int, offset int) ([]types.Expense, int, error) {
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, 0, err
	}
	results := []types.Expense{}

	// Get total count first
	var count int
	err = supabase.DB.From("expenses").Select("id", "count").Single().Execute(&count)
	if err != nil {
		fmt.Println(err)
		return nil, 0, err
	}

	err = supabase.DB.From("expenses").Select().LimitWithOffset(limit, offset).Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, 0, err
	}

	fmt.Println(results)
	return results, count, nil
}

func InsertDebtIntoDatabase(rowToinsert types.Debt) ([]types.Debt, error) {
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}
	var results []types.Debt
	err = supabase.DB.From("debts").Insert(rowToinsert).Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	fmt.Println(results) // Inserted rows
	return results, nil
}

func InsertIncomeIntoDatabase(rowToinsert types.Income) ([]types.Income, error) {
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}
	var results []types.Income
	err = supabase.DB.From("incomes").Insert(rowToinsert).Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	fmt.Println(results) // Inserted rows
	return results, nil
}

func GetIncomes(limit int, offset int) ([]types.Income, int, error) {
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, 0, err
	}
	results := []types.Income{}

	// Get total count first
	var count int
	err = supabase.DB.From("incomes").Select("id", "count").Single().Execute(&count)
	if err != nil {
		fmt.Println(err)
		return nil, 0, err
	}

	err = supabase.DB.From("incomes").Select().LimitWithOffset(limit, offset).Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, 0, err
	}

	fmt.Println(results)
	return results, count, nil
}

func InsertInvestmentIntoDatabase(rowToinsert types.Investment) ([]types.Investment, error) {
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}
	var results []types.Investment
	err = supabase.DB.From("investments").Insert(rowToinsert).Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	fmt.Println(results) // Inserted rows
	return results, nil
}

func GetAccounts() ([]types.Account, error) {
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}
	results := []types.Account{}
	err = supabase.DB.From("accounts").Select().Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	fmt.Println(results)
	return results, nil
}
func InsertAccountIntoDatabase(rowToinsert types.Account) (types.Account, error) {
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return types.Account{}, err
	}
	var results []types.Account
	err = supabase.DB.From("accounts").Insert(rowToinsert).Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return types.Account{}, err
	}
	fmt.Println("results: ", results) // Inserted rows
	return results[0], nil
}

func UpdateAccountBalances(toUpdate []types.Account) ([]types.Account, error) {
	var results []types.Account
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	for _, account := range toUpdate {
		var accountResult []types.Account
		stringId := fmt.Sprint(account.Id)

		query := supabase.DB.From("accounts").Update(account).Eq("id", stringId)
		err = query.Execute(&accountResult)
		if err != nil {
			fmt.Println(err)
			// log.Fatal(err)
			return nil, err
		}
		if len(accountResult) > 0 {
			results = append(results, accountResult[0])
		}
	}
	fmt.Println("results", results) // Inserted rows
	return results, nil

}

func GetInvestmentAccounts() ([]types.InvestmentAccount, error) {
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}
	results := []types.InvestmentAccount{}
	err = supabase.DB.From("investment_accounts").Select().Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	fmt.Println(results)
	return results, nil
}

func InsertInvestmentAccountIntoDatabase(rowToinsert types.InvestmentAccount) ([]types.InvestmentAccount, error) {
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}
	var results []types.InvestmentAccount
	err = supabase.DB.From("investment_accounts").Insert(rowToinsert).Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	fmt.Println(results) // Inserted rows
	return results, nil
}

func UpdateInvestmentAccountBalances(toUpdate []types.InvestmentAccount) ([]types.InvestmentAccount, error) {

	var results []types.InvestmentAccount
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	for _, account := range toUpdate {
		var accountResult []types.InvestmentAccount

		query := supabase.DB.From("investment_accounts").Update(account).Eq("id", fmt.Sprint(account.Id))
		err = query.Execute(&accountResult)
		if err != nil {
			fmt.Println(err)
			// log.Fatal(err)
			return nil, err
		}
		if len(accountResult) > 0 {

			results = append(results, accountResult[0])
		}
	}

	fmt.Println("results", results) // Inserted rows
	return results, nil

}

func GetDebtors() ([]types.Debtor, error) {
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}
	results := []types.Debtor{}
	err = supabase.DB.From("debtors").Select().Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	fmt.Println(results)
	return results, nil
}

func GetDebtorsWithDebts() ([]types.DebtByDebtor, error) {
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}
	results := []types.DebtByDebtor{}
	err = supabase.DB.From("debt_by_debtor").Select().Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	fmt.Println(results)
	return results, nil
}

func InsertDebtorIntoDatabase(rowToinsert types.Debtor) ([]types.Debtor, error) {
	supabase, err := getSupabaseClient()
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}
	var results []types.Debtor
	err = supabase.DB.From("debtors").Insert(rowToinsert).Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	fmt.Println(results) // Inserted rows
	return results, nil
}
