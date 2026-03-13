package commands

import (
	"fmt"

	"github.com/basecamp/fizzy-sdk/go/pkg/generated"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage account settings",
	Long:  "Commands for managing account settings.",
}

var accountShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show account settings",
	Long:  "Shows the current account settings including auto-postpone period.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		data, _, err := getSDK().Account().GetSettings(cmd.Context())
		if err != nil {
			return convertSDKError(err)
		}

		items := normalizeAny(data)

		summary := "Account"
		if account, ok := items.(map[string]any); ok {
			if name, ok := account["name"].(string); ok && name != "" {
				summary = fmt.Sprintf("Account: %s", name)
			}
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("boards", "fizzy board list", "List boards"),
			breadcrumb("entropy", "fizzy account entropy --auto_postpone_period_in_days N", "Update auto-postpone period"),
		}

		printDetail(items, summary, breadcrumbs)
		return nil
	},
}

// Account entropy flags
var accountEntropyAutoPostponePeriodInDays int

var accountEntropyCmd = &cobra.Command{
	Use:   "entropy",
	Short: "Update account auto-postpone period",
	Long:  "Updates the account-level default auto-postpone period. Requires admin role.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if accountEntropyAutoPostponePeriodInDays == 0 {
			return newRequiredFlagError("auto_postpone_period_in_days")
		}
		if err := validateAutoPostponePeriodInDays(accountEntropyAutoPostponePeriodInDays); err != nil {
			return err
		}

		req := &generated.UpdateAccountEntropyRequest{
			AutoPostponePeriodInDays: int32(accountEntropyAutoPostponePeriodInDays),
		}

		data, _, err := getSDK().Account().UpdateEntropy(cmd.Context(), req)
		if err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", "fizzy account show", "View account settings"),
			breadcrumb("boards", "fizzy board list", "List boards"),
		}

		printMutation(normalizeAny(data), "", breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(accountCmd)

	// Show
	accountCmd.AddCommand(accountShowCmd)

	// Entropy
	accountEntropyCmd.Flags().IntVar(&accountEntropyAutoPostponePeriodInDays, "auto_postpone_period_in_days", 0, "Auto postpone period in days ("+validAutoPostponePeriodsHelp+")")
	accountCmd.AddCommand(accountEntropyCmd)
}
