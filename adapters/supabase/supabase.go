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
	fmt.Println("supabase user", user.User.Email)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	supabase.DB.AddHeader("Authorization", "Bearer "+user.AccessToken)
	header := supabase.DB.Headers().Get("Authorization")
	fmt.Println(header)

	return supabase, nil
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

func GetCategories() ([]types.Category, error) {
	supabaseUrl := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_KEY")
	supabase := supa.CreateClient(supabaseUrl, supabaseKey)
	ctx := context.Background()
	user, err := supabase.Auth.SignIn(ctx, supa.UserCredentials{
		Email:    os.Getenv("SUPABASE_USER"),
		Password: os.Getenv("SUPABASE_USER_PWD"),
	})
	fmt.Println("supabase user", user.User.Email)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}
	var results []types.Category
	supabase.DB.AddHeader("Authorization", "Bearer "+user.AccessToken)
	header := supabase.DB.Headers().Get("Authorization")
	fmt.Println(header)
	err = supabase.DB.From("categories").Select().Execute(&results)
	if err != nil {
		fmt.Println(err)
		// log.Fatal(err)
		return nil, err
	}

	fmt.Println(results)
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
