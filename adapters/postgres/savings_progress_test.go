package postgres

import (
	"math"
	"testing"

	"github.com/carlosdimatteo/fintrack-backend-go/types"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 0.005 }

func TestCalculateSavingsProgress(t *testing.T) {
	cases := []struct {
		name                       string
		goals                      types.YearlyGoals
		now                        types.NetWorthSnapshot
		wantReal, wantNoBroker     float64
		wantNoGains, wantRemaining float64
	}{
		{
			// 2025 actuals: crypto lost value over the year, broker gained.
			name: "crypto loss, broker gain",
			goals: types.YearlyGoals{SavingsGoal: 145000,
				BaselineFiatBalance: 7337, BaselineCryptoBalance: 15978, BaselineCryptoCapital: 14382,
				BaselineBrokerBalance: 326695.22, BaselineBrokerCapital: 270560},
			now: types.NetWorthSnapshot{TotalFiatBalance: 16634.19,
				CryptoBalance: 19000, CryptoCapital: 20829, BrokerBalance: 538485.9, BrokerCapital: 409060,
				TotalRealNetWorth: 574120.09},
			wantReal: 224109.87, wantNoBroker: 150819.19, wantNoGains: 150819.19, wantRemaining: -5819.19,
		},
		{
			name: "crypto gain, broker loss",
			goals: types.YearlyGoals{SavingsGoal: 2000,
				BaselineFiatBalance: 1000, BaselineCryptoBalance: 1000, BaselineCryptoCapital: 1000,
				BaselineBrokerBalance: 10000, BaselineBrokerCapital: 9000},
			now: types.NetWorthSnapshot{TotalFiatBalance: 2000,
				CryptoBalance: 1500, CryptoCapital: 1000, BrokerBalance: 9500, BrokerCapital: 10000,
				TotalRealNetWorth: 13000},
			wantReal: 1000, wantNoBroker: 1000, wantNoGains: 500, wantRemaining: 1500,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := CalculateSavingsProgress(c.goals, c.now)
			if !approx(p.SavedThisYear, c.wantReal) {
				t.Errorf("SavedThisYear = %.2f, want %.2f", p.SavedThisYear, c.wantReal)
			}
			if !approx(p.SavedWithoutBrokerWinnings, c.wantNoBroker) {
				t.Errorf("SavedWithoutBrokerWinnings = %.2f, want %.2f", p.SavedWithoutBrokerWinnings, c.wantNoBroker)
			}
			if !approx(p.SavedWithoutWinnings, c.wantNoGains) {
				t.Errorf("SavedWithoutWinnings = %.2f, want %.2f", p.SavedWithoutWinnings, c.wantNoGains)
			}
			if !approx(p.RemainingWithoutWinnings, c.wantRemaining) {
				t.Errorf("RemainingWithoutWinnings = %.2f, want %.2f", p.RemainingWithoutWinnings, c.wantRemaining)
			}
			if p.SavedWithoutWinnings > p.SavedWithoutBrokerWinnings || p.SavedWithoutBrokerWinnings > p.SavedThisYear {
				t.Errorf("want without-gains <= without-broker-gains <= real, got %.2f, %.2f, %.2f",
					p.SavedWithoutWinnings, p.SavedWithoutBrokerWinnings, p.SavedThisYear)
			}
		})
	}
}
