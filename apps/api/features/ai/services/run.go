package ai_services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pengi-med-saas/core/ai"

	"gorm.io/gorm"
)

// openFunction prepares one charged use of fn for the tenant: resolves the
// provider (backoffice settings, else env), dials it and checks the monthly
// budget. ErrNotConfigured when there is no usable key, ErrQuotaExceeded when
// the month's budget is used up; the model is not called in either case.
func openFunction(dial Dialer, db *gorm.DB, tenantID uint, fn string, now time.Time) (Provider, Completer, error) {
	provider, client, err := dialFunction(dial, db, fn)
	if err != nil {
		return provider, nil, err
	}
	budget, err := MonthlyBudget(db, tenantID, now)
	if err != nil {
		return provider, nil, err
	}
	if budget.Reached() {
		return provider, nil, ErrQuotaExceeded
	}
	return provider, client, nil
}

// dialFunction resolves and dials fn's provider without looking at the
// quota (a use already counted, or one that is never counted).
func dialFunction(dial Dialer, db *gorm.DB, fn string) (Provider, Completer, error) {
	provider, err := ResolveProvider(db, fn)
	if err != nil {
		return provider, nil, fmt.Errorf("%w: %v", ErrNotConfigured, err)
	}
	if !provider.Configured() {
		return provider, nil, ErrNotConfigured
	}
	client := dial(provider)
	if client == nil || !client.Configured() {
		return provider, nil, ErrNotConfigured
	}
	return provider, client, nil
}

// callAndLog sends req with the provider's model and reasoning effort and,
// once the provider answered (and billed the tokens), logs the use: it counts
// against the quota even if the answer turns out unusable. A failed call is
// not logged nor counted.
func callAndLog(ctx context.Context, db *gorm.DB, client Completer, provider Provider, req ai.ChatRequest, entry UsageEntry, now time.Time) (*ai.ChatResponse, error) {
	res, err := call(ctx, client, provider, req)
	if err != nil {
		return nil, err
	}
	entry.Model, entry.Host = provider.Model, provider.Host()
	entry.PromptTokens, entry.CompletionTokens = res.Usage.PromptTokens, res.Usage.CompletionTokens
	if err := LogUsage(db, entry, now); err != nil {
		return nil, err
	}
	return res, nil
}

// call sends req with the provider's model and reasoning effort, nothing
// logged.
func call(ctx context.Context, client Completer, provider Provider, req ai.ChatRequest) (*ai.ChatResponse, error) {
	req.Model = provider.Model
	req.ReasoningEffort = provider.ReasoningEffort
	res, err := client.Chat(ctx, req)
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) {
			return nil, ErrNotConfigured
		}
		return nil, fmt.Errorf("%w: %v", ErrProvider, err)
	}
	return res, nil
}
