package supabase

import (
	"context"
	"fmt"
	"log"
	"os"

	googleSS "github.com/carlosdimatteo/fintrack-backend-go/adapters/google"
	supa "github.com/nedpals/supabase-go"
)

func InsertIntoDatabase(rowToInsert googleSS.FintrackRow) ([]googleSS.FintrackRow, error) {
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
	var results []googleSS.FintrackRow
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
