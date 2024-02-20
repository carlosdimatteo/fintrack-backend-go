package supabase

import (
	"context"
	"fmt"
	"log"
	"os"

	types "github.com/carlosdimatteo/fintrack-backend-go/types"
	supa "github.com/nedpals/supabase-go"
)

func InsertIntoDatabase(rowToInsert types.Expense) ([]types.Expense, error) {
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
		log.Fatal(err)
		return nil, err
	}
	var results []types.Expense
	fmt.Println("before insert")
	supabase.DB.AddHeader("Authorization", "Bearer "+user.AccessToken)
	header := supabase.DB.Headers().Get("Authorization")
	fmt.Println(header)
	err = supabase.DB.From("expenses").Insert(rowToInsert).Execute(&results)
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
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
		log.Fatal(err)
		return nil, err
	}
	var results []types.Category
	supabase.DB.AddHeader("Authorization", "Bearer "+user.AccessToken)
	header := supabase.DB.Headers().Get("Authorization")
	fmt.Println(header)
	err = supabase.DB.From("categories").Select().Execute(&results)
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
		return nil, err
	}

	fmt.Println(results)
	return results, nil
}
func GetBudgets() ([]types.BudgetByCategory, error) {
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
		log.Fatal(err)
		return nil, err
	}
	var results []types.BudgetByCategory
	supabase.DB.AddHeader("Authorization", "Bearer "+user.AccessToken)
	header := supabase.DB.Headers().Get("Authorization")
	fmt.Println(header)
	err = supabase.DB.From("budget_by_category_current_month").Select().Execute(&results)
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
		return nil, err
	}

	fmt.Println(results)
	return results, nil
}
