package postgres

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	types "github.com/carlosdimatteo/fintrack-backend-go/types"
)

// ========== BUDGETS ==========

// prestamosCategoryId is the loans category: lending and repaying aren't consumption, so it is
// never judged against a budget.
const prestamosCategoryId int32 = 9

const (
	budgetOver    = "over"
	budgetWarning = "warning"
	budgetOK      = "ok"
	budgetNeutral = "neutral" // not evaluated: no budget, rolls into another category, or loans
)

type budgetVersion struct {
	Amount           float64
	EvaluateYTD      bool
	RollupCategoryId *int32
	Notes            *string
}

type categorySpend struct {
	Spent float64 // your share: expense minus what was lent from it
	Lent  float64
}

type budgetCategory struct {
	Id   int32
	Name string
}

// budgetYear holds one year of inputs, indexed by month (0 = January).
type budgetYear struct {
	categories []budgetCategory // ordered by name
	budgets    [12]map[int32]budgetVersion
	spend      [12]map[int32]categorySpend
	totals     [12]categorySpend // includes uncategorized expenses
	income     [12]float64       // excludes debt repayments
}

func forEachRow(ctx context.Context, pool *pgxpool.Pool, sql string, args []any, scans []any, fn func()) error {
	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	_, err = pgx.ForEachRow(rows, scans, func() error { fn(); return nil })
	return err
}

func loadBudgetYear(year int) (budgetYear, error) {
	pool, err := GetPool()
	if err != nil {
		return budgetYear{}, err
	}
	ctx := context.Background()

	var d budgetYear
	for m := range d.budgets {
		d.budgets[m] = map[int32]budgetVersion{}
		d.spend[m] = map[int32]categorySpend{}
	}

	var category budgetCategory
	err = forEachRow(ctx, pool, `SELECT id, name FROM categories ORDER BY name`, nil,
		[]any{&category.Id, &category.Name},
		func() { d.categories = append(d.categories, category) })
	if err != nil {
		return budgetYear{}, fmt.Errorf("error querying categories: %w", err)
	}

	var month int
	var categoryId int32
	var version budgetVersion
	err = forEachRow(ctx, pool,
		`SELECT EXTRACT(MONTH FROM m)::int, b.category_id, COALESCE(b.amount, 0), b.evaluate_ytd, b.rollup_category_id, b.notes
		 FROM generate_series(make_date($1, 1, 1), make_date($1, 12, 1), interval '1 month') AS m
		 CROSS JOIN LATERAL public.budget_for_month(m::date) b
		 WHERE b.category_id IS NOT NULL`,
		[]any{year},
		[]any{&month, &categoryId, &version.Amount, &version.EvaluateYTD, &version.RollupCategoryId, &version.Notes},
		func() {
			d.budgets[month-1][categoryId] = version
			// pgx may scan into existing pointees; fresh pointers keep stored rows independent
			version.RollupCategoryId, version.Notes = nil, nil
		})
	if err != nil {
		return budgetYear{}, fmt.Errorf("error querying monthly budgets: %w", err)
	}

	var spendCategory *int32
	var spend categorySpend
	err = forEachRow(ctx, pool,
		`WITH lent_by_expense AS (
			SELECT expense_id, SUM(amount) AS lent
			FROM debts
			WHERE outbound AND expense_id IS NOT NULL
			GROUP BY expense_id
		 )
		 SELECT EXTRACT(MONTH FROM e.created_at)::int, e.category_id,
			COALESCE(SUM(e.expense - COALESCE(l.lent, 0)), 0),
			COALESCE(SUM(COALESCE(l.lent, 0)), 0)
		 FROM expenses e
		 LEFT JOIN lent_by_expense l ON l.expense_id = e.id
		 WHERE EXTRACT(YEAR FROM e.created_at) = $1
		 GROUP BY 1, 2`,
		[]any{year},
		[]any{&month, &spendCategory, &spend.Spent, &spend.Lent},
		func() {
			d.totals[month-1].Spent += spend.Spent
			d.totals[month-1].Lent += spend.Lent
			if spendCategory != nil {
				d.spend[month-1][*spendCategory] = spend
			}
		})
	if err != nil {
		return budgetYear{}, fmt.Errorf("error querying monthly spending: %w", err)
	}

	// Repayments are excluded: lent money is already left out of spending, so counting it coming
	// back as income would double count it.
	var income float64
	err = forEachRow(ctx, pool,
		`SELECT EXTRACT(MONTH FROM i.created_at)::int, COALESCE(SUM(i.amount), 0)
		 FROM incomes i
		 WHERE EXTRACT(YEAR FROM i.created_at) = $1
		   AND NOT EXISTS (SELECT 1 FROM debts d WHERE d.income_id = i.id)
		 GROUP BY 1`,
		[]any{year},
		[]any{&month, &income},
		func() { d.income[month-1] = income })
	if err != nil {
		return budgetYear{}, fmt.Errorf("error querying monthly income: %w", err)
	}

	return d, nil
}

// evaluateBudgetYear computes budget vs actual for every category in every month. A category's
// actual includes the spend of categories rolled into it that month; rolled-up categories, loans
// and categories without a budget aren't evaluated. Status is always the month's; year-to-date
// figures are returned alongside for views that ask for them.
func evaluateBudgetYear(d budgetYear) [12][]types.BudgetHistoryCategory {
	names := make(map[int32]string, len(d.categories))
	for _, c := range d.categories {
		names[c.Id] = c.Name
	}
	fullYear := map[int32]float64{}
	for m := range d.budgets {
		for id, v := range d.budgets[m] {
			fullYear[id] += v.Amount
		}
	}

	var out [12][]types.BudgetHistoryCategory
	ytdBudget := map[int32]float64{}
	ytdActual := map[int32]float64{}
	for m := 0; m < 12; m++ {
		rolledIn := map[int32]float64{}
		for id, v := range d.budgets[m] {
			if v.RollupCategoryId != nil && *v.RollupCategoryId != id {
				rolledIn[*v.RollupCategoryId] += d.spend[m][id].Spent
			}
		}

		for _, c := range d.categories {
			v, hasBudget := d.budgets[m][c.Id]
			s := d.spend[m][c.Id]
			actual := s.Spent + rolledIn[c.Id]
			if hasBudget {
				ytdBudget[c.Id] += v.Amount
			}
			ytdActual[c.Id] += actual

			e := types.BudgetHistoryCategory{
				CategoryId: c.Id, CategoryName: c.Name, Budget: v.Amount, Spent: s.Spent, Lent: s.Lent,
				BudgetEvaluation: types.BudgetEvaluation{
					HasBudget: hasBudget, Actual: actual, EvaluateYTD: v.EvaluateYTD,
					RollupCategoryId: v.RollupCategoryId, Notes: v.Notes,
					YTDBudget: ytdBudget[c.Id], YTDActual: ytdActual[c.Id], FullYearBudget: fullYear[c.Id],
				},
			}
			if v.RollupCategoryId != nil {
				e.RollupCategoryName = names[*v.RollupCategoryId]
			}

			e.Difference = v.Amount - actual
			if !hasBudget || v.RollupCategoryId != nil || c.Id == prestamosCategoryId {
				e.Status = budgetNeutral
			} else {
				e.Status = budgetStatus(actual, v.Amount)
			}
			out[m] = append(out[m], e)
		}
	}
	return out
}

// budgetStatus compares in cents so float noise can't flip a category over its budget.
func budgetStatus(actual, budget float64) string {
	actualCents, budgetCents := math.Round(actual*100), math.Round(budget*100)
	switch {
	case actualCents > budgetCents:
		return budgetOver
	case budgetCents > 0 && actualCents >= 0.8*budgetCents:
		return budgetWarning
	default:
		return budgetOK
	}
}

// GetBudgets returns budget vs actual for the current month, for categories with a budget.
func GetBudgets() ([]types.BudgetByCategory, error) {
	now := time.Now()
	d, err := loadBudgetYear(now.Year())
	if err != nil {
		return nil, err
	}

	results := []types.BudgetByCategory{}
	for _, e := range evaluateBudgetYear(d)[now.Month()-1] {
		if !e.HasBudget {
			continue
		}
		results = append(results, types.BudgetByCategory{
			Amount: e.Budget, Spent: e.Spent, Lent: e.Lent,
			CategoryName: e.CategoryName, CategoryId: e.CategoryId,
			BudgetEvaluation: e.BudgetEvaluation,
		})
	}
	return results, nil
}

// GetBudgetHistory returns monthly budget breakdown for a year
func GetBudgetHistory(year int) (types.BudgetHistoryResponse, error) {
	d, err := loadBudgetYear(year)
	if err != nil {
		return types.BudgetHistoryResponse{}, err
	}
	evaluated := evaluateBudgetYear(d)

	response := types.BudgetHistoryResponse{Year: year, Months: make([]types.BudgetHistoryMonth, 0, 12)}
	yearly := types.BudgetHistoryYearlyTotals{Categories: []types.BudgetHistoryYearlyCategory{}}

	for m := 0; m < 12; m++ {
		month := types.BudgetHistoryMonth{
			Month:         m + 1,
			TotalExpenses: d.totals[m].Spent,
			TotalLent:     d.totals[m].Lent,
			TotalIncome:   d.income[m],
			Categories:    evaluated[m],
		}
		for i := range month.Categories {
			c := &month.Categories[i]
			if month.TotalExpenses > 0 {
				c.PctOfExpenses = (c.Spent / month.TotalExpenses) * 100
			}
			if month.TotalIncome > 0 {
				c.PctOfIncome = (c.Spent / month.TotalIncome) * 100
			}
			if c.HasBudget && c.RollupCategoryId == nil {
				month.TotalBudget += c.Budget
			}
		}
		yearly.TotalExpenses += month.TotalExpenses
		yearly.TotalIncome += month.TotalIncome
		yearly.TotalBudget += month.TotalBudget
		response.Months = append(response.Months, month)
	}

	yearly.Difference = yearly.TotalBudget - yearly.TotalExpenses
	if yearly.TotalIncome > 0 {
		yearly.SavingsRate = ((yearly.TotalIncome - yearly.TotalExpenses) / yearly.TotalIncome) * 100
	}

	for i, c := range d.categories {
		yc := types.BudgetHistoryYearlyCategory{
			CategoryId:   c.Id,
			CategoryName: c.Name,
			YearlyBudget: evaluated[11][i].FullYearBudget,
		}
		var lastBudgeted *types.BudgetHistoryCategory
		for m := 0; m < 12; m++ {
			e := &evaluated[m][i]
			yc.TotalSpent += e.Spent
			yc.TotalLent += e.Lent
			yc.TotalActual += e.Actual
			if e.HasBudget {
				lastBudgeted = e
			}
		}
		yc.Difference = yc.YearlyBudget - yc.TotalActual

		switch {
		case lastBudgeted == nil || c.Id == prestamosCategoryId:
			yc.Status = budgetNeutral
		case lastBudgeted.RollupCategoryId != nil:
			yc.Status = budgetNeutral
			yc.RollupCategoryName = lastBudgeted.RollupCategoryName
		default:
			yc.Status = budgetStatus(yc.TotalActual, yc.YearlyBudget)
		}

		if yearly.TotalExpenses > 0 {
			yc.PctOfExpenses = (yc.TotalSpent / yearly.TotalExpenses) * 100
		}
		if yearly.TotalIncome > 0 {
			yc.PctOfIncome = (yc.TotalSpent / yearly.TotalIncome) * 100
		}
		yearly.Categories = append(yearly.Categories, yc)
	}

	response.YearlyTotals = yearly
	return response, nil
}
