package golinters

import (
	"context"

	"github.com/golangci/golangci-lint/pkg/lint/linter"
	"github.com/golangci/golangci-lint/pkg/result"
	govetAPI "github.com/golangci/govet"
)

type Govet struct{}

func (Govet) Name() string {
	return "govet"
}

func (Govet) Desc() string {
	return "Vet examines Go source code and reports suspicious constructs, such as Printf calls whose arguments do not align with the format string"
}

func (g Govet) Run(ctx context.Context, lintCtx *linter.Context) ([]result.Issue, error) {
	// TODO: check .S asm files: govet can do it if pass dirs
	var govetIssues []govetAPI.Issue
	for _, pkg := range lintCtx.PkgProgram.Packages() {
		issues, err := govetAPI.Run(pkg.Files(lintCtx.Cfg.Run.AnalyzeTests), lintCtx.Settings().Govet.CheckShadowing)
		if err != nil {
			return nil, err
		}
		govetIssues = append(govetIssues, issues...)
	}
	if len(govetIssues) == 0 {
		return nil, nil
	}

	res := make([]result.Issue, 0, len(govetIssues))
	for _, i := range govetIssues {
		res = append(res, result.Issue{
			Pos:        i.Pos,
			Text:       i.Message,
			FromLinter: g.Name(),
		})
	}
	return res, nil
}
