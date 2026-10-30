-- Budget counts only your share of each expense. The part lent to others (outbound debts
-- created from the expense) is reported in a new `lent` column instead of `spent`.
-- Apply BEFORE deploying the backend that reads `lent`.

CREATE OR REPLACE VIEW public.budget_by_category_current_month AS
WITH lent_by_expense AS (
    SELECT expense_id, SUM(amount) AS lent
    FROM public.debts
    WHERE outbound AND expense_id IS NOT NULL
    GROUP BY expense_id
),
monthlyexpenses AS (
    SELECT e.category_id, e.expense, COALESCE(l.lent, 0) AS lent
    FROM public.expenses e
    LEFT JOIN lent_by_expense l ON l.expense_id = e.id
    WHERE e.created_at >= date_trunc('MONTH', CURRENT_DATE::timestamp with time zone)
      AND e.created_at < date_trunc('MONTH', CURRENT_DATE::timestamp with time zone) + '1 mon'::interval
)
SELECT b.amount,
    c.name AS category_name,
    b.category_id,
    COALESCE(SUM(e.expense - e.lent), 0::double precision) AS spent,
    COALESCE(SUM(e.lent), 0::double precision) AS lent
FROM public.budgets b
JOIN public.categories c ON b.category_id = c.id
LEFT JOIN monthlyexpenses e ON b.category_id = e.category_id
GROUP BY b.amount, c.name, b.category_id
ORDER BY b.category_id;
