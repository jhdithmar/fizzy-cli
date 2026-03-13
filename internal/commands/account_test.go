package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestAccountShow(t *testing.T) {
	t.Run("shows account settings", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":                           "acc-1",
				"name":                         "37signals",
				"cards_count":                  float64(5),
				"auto_postpone_period_in_days": float64(30),
			},
		}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := accountShowCmd.RunE(accountShowCmd, []string{})
		assertExitCode(t, err, 0)

		if !result.Response.OK {
			t.Error("expected success response")
		}
		if len(mock.GetCalls) != 1 {
			t.Errorf("expected 1 Get call, got %d", len(mock.GetCalls))
		}
		if mock.GetCalls[0].Path != "/account/settings.json" {
			t.Errorf("expected path '/account/settings.json', got '%s'", mock.GetCalls[0].Path)
		}
	})

	t.Run("requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("", "account", "https://api.example.com")
		defer resetTest()

		err := accountShowCmd.RunE(accountShowCmd, []string{})
		assertExitCode(t, err, errors.ExitAuthFailure)
	})

	t.Run("requires account", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "", "https://api.example.com")
		defer resetTest()

		err := accountShowCmd.RunE(accountShowCmd, []string{})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestAccountEntropy(t *testing.T) {
	t.Run("updates account auto-postpone period", func(t *testing.T) {
		mock := NewMockClient()
		mock.PutResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":                           "acc-1",
				"name":                         "37signals",
				"auto_postpone_period_in_days": float64(30),
			},
		}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		accountEntropyAutoPostponePeriodInDays = 30
		err := accountEntropyCmd.RunE(accountEntropyCmd, []string{})
		accountEntropyAutoPostponePeriodInDays = 0

		assertExitCode(t, err, 0)

		if !result.Response.OK {
			t.Error("expected success response")
		}
		if len(mock.PutCalls) != 1 {
			t.Errorf("expected 1 Put call, got %d", len(mock.PutCalls))
		}
		if mock.PutCalls[0].Path != "/account/entropy.json" {
			t.Errorf("expected path '/account/entropy.json', got '%s'", mock.PutCalls[0].Path)
		}
		body, ok := mock.PutCalls[0].Body.(map[string]any)
		if !ok {
			t.Fatal("expected map body")
		}
		if body["auto_postpone_period_in_days"] != float64(30) {
			t.Errorf("expected auto_postpone_period_in_days 30, got %v", body["auto_postpone_period_in_days"])
		}
	})

	t.Run("requires auto_postpone_period_in_days flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		accountEntropyAutoPostponePeriodInDays = 0
		err := accountEntropyCmd.RunE(accountEntropyCmd, []string{})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("rejects invalid period", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		accountEntropyAutoPostponePeriodInDays = 45
		err := accountEntropyCmd.RunE(accountEntropyCmd, []string{})
		accountEntropyAutoPostponePeriodInDays = 0

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("handles API error", func(t *testing.T) {
		mock := NewMockClient()
		mock.PutError = errors.NewForbiddenError("Admin role required")

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		accountEntropyAutoPostponePeriodInDays = 30
		err := accountEntropyCmd.RunE(accountEntropyCmd, []string{})
		accountEntropyAutoPostponePeriodInDays = 0

		assertExitCode(t, err, errors.ExitForbidden)
	})
}
