package postgres

import "testing"

func int32p(v int32) *int32 { return &v }

func TestEvaluateBudgetYear(t *testing.T) {
	const viajes, actividades, bienestar, prestamos, comidaFuera = 3, 14, 4, 9, 13

	var d budgetYear
	for m := range d.budgets {
		d.budgets[m] = map[int32]budgetVersion{}
		d.spend[m] = map[int32]categorySpend{}
	}
	d.categories = []budgetCategory{
		{bienestar, "Bienestar"}, {comidaFuera, "Comida Fuera"}, {prestamos, "Prestamos"},
		{viajes, "Viajes"}, {actividades, "Viajes- Actividades"},
	}
	for m := 0; m < 12; m++ {
		d.budgets[m][viajes] = budgetVersion{Amount: 400}
		d.budgets[m][actividades] = budgetVersion{Amount: 0, RollupCategoryId: int32p(viajes)}
		d.budgets[m][prestamos] = budgetVersion{Amount: 0}
		// YTD-judged, raised mid-year: 100/month January-June, 200/month from July
		amount := 100.0
		if m >= 6 {
			amount = 200
		}
		d.budgets[m][bienestar] = budgetVersion{Amount: amount, EvaluateYTD: true}
		// comidaFuera has spend but no budget row
	}
	d.spend[0][viajes] = categorySpend{Spent: 300}
	d.spend[0][actividades] = categorySpend{Spent: 150}
	d.spend[0][prestamos] = categorySpend{Spent: 5000}
	d.spend[0][comidaFuera] = categorySpend{Spent: 80}
	d.spend[0][bienestar] = categorySpend{Spent: 250}
	d.spend[7][bienestar] = categorySpend{Spent: 300}

	out := evaluateBudgetYear(d)
	get := func(month int, id int32) (r struct {
		actual, ytdBudget, ytdActual, fullYear float64
		status, rollupName                     string
		hasBudget                              bool
	}) {
		for _, e := range out[month] {
			if e.CategoryId == id {
				r.actual, r.ytdBudget, r.ytdActual, r.fullYear = e.Actual, e.YTDBudget, e.YTDActual, e.FullYearBudget
				r.status, r.rollupName, r.hasBudget = e.Status, e.RollupCategoryName, e.HasBudget
				return
			}
		}
		t.Fatalf("category %d missing in month %d", id, month)
		return
	}

	if r := get(0, viajes); r.actual != 450 || r.status != "over" {
		t.Errorf("Viajes in Jan: actual %.2f status %s, want 450 (300 own + 150 rolled in) over", r.actual, r.status)
	}
	if r := get(0, actividades); r.actual != 150 || r.status != "neutral" || r.rollupName != "Viajes" {
		t.Errorf("Actividades in Jan: actual %.2f status %s rollup %q, want 150 neutral Viajes", r.actual, r.status, r.rollupName)
	}
	if r := get(0, prestamos); r.status != "neutral" {
		t.Errorf("Prestamos 5000 over a 0 budget: status %s, want neutral", r.status)
	}
	if r := get(0, comidaFuera); r.hasBudget || r.status != "neutral" {
		t.Errorf("Comida Fuera without a budget row: hasBudget %v status %s, want false neutral", r.hasBudget, r.status)
	}
	if r := get(0, bienestar); r.ytdBudget != 100 || r.ytdActual != 250 || r.status != "over" {
		t.Errorf("Bienestar in Jan: ytd %.2f/%.2f status %s, want 250/100 over", r.ytdActual, r.ytdBudget, r.status)
	}
	// Status is the month's (300 > 200), even though year to date it's under: 550 of 6*100 + 2*200
	if r := get(7, bienestar); r.ytdBudget != 1000 || r.ytdActual != 550 || r.fullYear != 1800 || r.status != "over" {
		t.Errorf("Bienestar in Aug: ytd %.2f/%.2f full year %.2f status %s, want 550/1000 1800 over",
			r.ytdActual, r.ytdBudget, r.fullYear, r.status)
	}
}

func TestBudgetStatus(t *testing.T) {
	cases := []struct {
		actual, budget float64
		want           string
	}{
		{79.99, 100, "ok"},
		{80, 100, "warning"},
		{100.004, 100, "warning"}, // float noise at the budget is not over
		{100.01, 100, "over"},
		{1, 0, "over"},
		{0, 0, "ok"},
	}
	for _, c := range cases {
		if got := budgetStatus(c.actual, c.budget); got != c.want {
			t.Errorf("budgetStatus(%v, %v) = %s, want %s", c.actual, c.budget, got, c.want)
		}
	}
}
