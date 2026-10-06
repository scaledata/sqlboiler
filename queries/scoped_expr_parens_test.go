package queries_test

import (
	"testing"

	"github.com/volatiletech/sqlboiler/drivers"
	"github.com/volatiletech/sqlboiler/queries"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

// selectPrefix is what a select with no columns and one FROM table renders
// before its WHERE clause.
const selectPrefix = `SELECT * FROM "t"`

// TestScopedExprParens pins both halves of the ScopedExprParens contract: a
// query that does not apply the mod renders exactly as it did before the mod
// existed, and one that does keeps automatic parentheses on every condition
// outside the Expr group that suppressed them.
//
// The file is package queries_test because queries/qm imports queries, so an
// in-package test could not reach the mod constructors.
func TestScopedExprParens(t *testing.T) {
	t.Parallel()

	// An Expr group holding a bare OR. Every case carries it, so the
	// suppression that ScopedExprParens confines is always triggered.
	exprGroup := qm.Expr(qm.Where("a = 1"), qm.Or2(qm.Where("b = 2")))

	tests := []struct {
		name   string
		optIn  bool
		mods   []qm.QueryMod
		expect string
	}{
		{
			name:   "without the mod an Expr unbrackets the whole query",
			optIn:  false,
			mods:   []qm.QueryMod{exprGroup, qm.Where("c = 3 or d = 4")},
			expect: ` WHERE (a = 1 OR b = 2) AND c = 3 or d = 4`,
		},
		{
			name:   "with the mod conditions outside the group keep brackets",
			optIn:  true,
			mods:   []qm.QueryMod{exprGroup, qm.Where("c = 3 or d = 4")},
			expect: ` WHERE (a = 1 OR b = 2) AND (c = 3 or d = 4)`,
		},
		{
			name:  "nested groups gain brackets only at the top level",
			optIn: true,
			mods: []qm.QueryMod{
				qm.Expr(qm.Where("a = 1"), qm.Expr(qm.Where("b = 2"))),
				qm.Where("c = 3 or d = 4"),
			},
			expect: ` WHERE (a = 1 AND (b = 2)) AND (c = 3 or d = 4)`,
		},
		{
			name:   "an IN condition outside the group keeps brackets",
			optIn:  true,
			mods:   []qm.QueryMod{exprGroup, qm.WhereIn("c in ?", 1, 2)},
			expect: ` WHERE (a = 1 OR b = 2) AND ("c" IN ($1,$2))`,
		},
		{
			// The empty-IN path writes its own brackets and returns early,
			// so it must not pick up a second pair.
			name:   "an empty IN condition stays singly bracketed",
			optIn:  true,
			mods:   []qm.QueryMod{exprGroup, qm.WhereIn("c in ?")},
			expect: ` WHERE (a = 1 OR b = 2) AND (1=0)`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q := &queries.Query{}
			queries.SetDialect(q, &drivers.Dialect{LQ: '"', RQ: '"', UseIndexPlaceholders: true})
			queries.SetFrom(q, "t")
			qm.Apply(q, test.mods...)
			if test.optIn {
				qm.Apply(q, qm.ScopedExprParens())
			}

			result, _ := queries.BuildQuery(q)
			expect := selectPrefix + test.expect + ";"
			if result != expect {
				t.Errorf("Mismatch between expect and result:\n%s\n%s\n", expect, result)
			}
		})
	}
}
