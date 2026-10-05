package policy

import (
	"context"
	"fmt"
	"scim-ai-gateway/pkg/scim"

	// ⚠️ Use the modern v1 AST package path to prevent deprecation alerts
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
)

type Evaluator struct {
	preparedQuery rego.PreparedEvalQuery
}

// EvalInput defines the exact JSON payload shape expected by your Rego policy
type EvalInput struct {
	Agent               *scim.Agent `json:"agent"`
	Owner               *scim.User  `json:"owner"`
	Action              string      `json:"action"`
	TargetRequiredGroup string      `json:"target_required_group"`
}

func NewEvaluator(regoFilePath string) (*Evaluator, error) {
	ctx := context.Background()
	query, err := rego.New(
		rego.Query("data.scim.authz.allow"),
		rego.Load([]string{regoFilePath}, nil),
	).PrepareForEval(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare rego query: %w", err)
	}

	return &Evaluator{preparedQuery: query}, nil
}

// EvaluateParsed evaluates rules natively via a pre-compiled AST input tree
func (e *Evaluator) EvaluateParsed(ctx context.Context, input ast.Value) (bool, error) {
	results, err := e.preparedQuery.Eval(ctx, rego.EvalParsedInput(input))
	if err != nil {
		return false, err
	}

	if len(results) > 0 && len(results[0].Expressions) > 0 {
		if allowed, ok := results[0].Expressions[0].Value.(bool); ok {
			return allowed, nil
		}
	}
	return false, nil
}
