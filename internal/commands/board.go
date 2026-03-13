package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/basecamp/fizzy-sdk/go/pkg/generated"
	"github.com/spf13/cobra"
)

var validAutoPostponePeriods = []int{3, 7, 11, 30, 90, 365}

var validAutoPostponePeriodsHelp = func() string {
	parts := make([]string, len(validAutoPostponePeriods))
	for i, v := range validAutoPostponePeriods {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ", ")
}()

func validateAutoPostponePeriodInDays(days int) error {
	for _, v := range validAutoPostponePeriods {
		if days == v {
			return nil
		}
	}
	return errors.NewInvalidArgsError(fmt.Sprintf("--auto_postpone_period_in_days must be one of: %s (got %d)", validAutoPostponePeriodsHelp, days))
}

var boardCmd = &cobra.Command{
	Use:   "board",
	Short: "Manage boards",
	Long:  "Commands for managing Fizzy boards.",
}

// Board list flags
var boardListPage int
var boardListAll bool

var boardListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all boards",
	Long:  "Lists all boards you have access to.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}
		if err := checkLimitAll(boardListAll); err != nil {
			return err
		}

		ac := getSDK()
		var items any
		var linkNext string

		path := "/boards.json"
		if boardListPage > 0 {
			path += "?page=" + strconv.Itoa(boardListPage)
		}

		if boardListAll {
			pages, err := ac.GetAll(cmd.Context(), path)
			if err != nil {
				return convertSDKError(err)
			}
			items = jsonAnySlice(pages)
		} else {
			data, resp, err := ac.Boards().List(cmd.Context(), path)
			if err != nil {
				return convertSDKError(err)
			}
			items = normalizeAny(data)
			linkNext = parseSDKLinkNext(resp)
		}

		// Build summary
		count := dataCount(items)
		summary := fmt.Sprintf("%d boards", count)
		if boardListAll {
			summary += " (all)"
		} else if boardListPage > 0 {
			summary += fmt.Sprintf(" (page %d)", boardListPage)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", "fizzy board show <id>", "View board details"),
			breadcrumb("cards", "fizzy card list --board <id>", "List cards on board"),
			breadcrumb("columns", "fizzy column list --board <id>", "List board columns"),
		}

		hasNext := linkNext != ""
		if hasNext {
			nextPage := boardListPage + 1
			if boardListPage == 0 {
				nextPage = 2
			}
			breadcrumbs = append(breadcrumbs, breadcrumb("next", fmt.Sprintf("fizzy board list --page %d", nextPage), "Next page"))
		}

		printListPaginated(items, boardColumns, hasNext, linkNext, boardListAll, summary, breadcrumbs)
		return nil
	},
}

var boardShowCmd = &cobra.Command{
	Use:   "show BOARD_ID",
	Short: "Show a board",
	Long:  "Shows details of a specific board.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		boardID := args[0]

		resp, err := getSDK().Get(cmd.Context(), "/boards/"+boardID+".json")
		if err != nil {
			return convertSDKError(err)
		}

		items := normalizeAny(resp.Data)

		summary := "Board"
		if board, ok := items.(map[string]any); ok {
			if name, ok := board["name"].(string); ok && name != "" {
				summary = fmt.Sprintf("Board: %s", name)
			}
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("cards", fmt.Sprintf("fizzy card list --board %s", boardID), "List cards"),
			breadcrumb("columns", fmt.Sprintf("fizzy column list --board %s", boardID), "List columns"),
			breadcrumb("create-card", fmt.Sprintf("fizzy card create --board %s --title \"title\"", boardID), "Create card"),
		}
		if board, ok := items.(map[string]any); ok {
			if publicURL, ok := board["public_url"].(string); ok && publicURL != "" {
				breadcrumbs = append(breadcrumbs, breadcrumb("unpublish", fmt.Sprintf("fizzy board unpublish %s", boardID), "Disable public board link"))
			} else {
				breadcrumbs = append(breadcrumbs, breadcrumb("publish", fmt.Sprintf("fizzy board publish %s", boardID), "Create public board link"))
			}
		}

		printDetail(items, summary, breadcrumbs)
		return nil
	},
}

// Board create flags
var boardCreateName string
var boardCreateAllAccess string
var boardCreateAutoPostponePeriodInDays int

var boardCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a board",
	Long:  "Creates a new board.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if boardCreateName == "" {
			return newRequiredFlagError("name")
		}

		req := &generated.CreateBoardRequest{
			Name: boardCreateName,
		}
		if boardCreateAllAccess != "" {
			req.AllAccess = boardCreateAllAccess == "true"
		}
		if boardCreateAutoPostponePeriodInDays != 0 {
			if err := validateAutoPostponePeriodInDays(boardCreateAutoPostponePeriodInDays); err != nil {
				return err
			}
			req.AutoPostponePeriodInDays = int32(boardCreateAutoPostponePeriodInDays)
		}

		ac := getSDK()
		data, resp, err := ac.Boards().Create(cmd.Context(), req)
		if err != nil {
			return convertSDKError(err)
		}

		items := normalizeAny(data)
		boardID := ""
		if board, ok := items.(map[string]any); ok {
			if id, ok := board["id"]; ok {
				boardID = fmt.Sprintf("%v", id)
			}
		}

		var breadcrumbs []Breadcrumb
		if boardID != "" {
			breadcrumbs = []Breadcrumb{
				breadcrumb("show", fmt.Sprintf("fizzy board show %s", boardID), "View board details"),
				breadcrumb("cards", fmt.Sprintf("fizzy card list --board %s", boardID), "List cards"),
				breadcrumb("columns", fmt.Sprintf("fizzy column list --board %s", boardID), "List columns"),
			}
		}

		if location := resp.Headers.Get("Location"); location != "" {
			printMutationWithLocation(items, location, breadcrumbs)
		} else {
			printMutation(items, "", breadcrumbs)
		}
		return nil
	},
}

// Board update flags
var boardUpdateName string
var boardUpdateAllAccess string
var boardUpdateAutoPostponePeriodInDays int

var boardUpdateCmd = &cobra.Command{
	Use:   "update BOARD_ID",
	Short: "Update a board",
	Long:  "Updates an existing board.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		boardID := args[0]

		if boardUpdateAutoPostponePeriodInDays != 0 {
			if err := validateAutoPostponePeriodInDays(boardUpdateAutoPostponePeriodInDays); err != nil {
				return err
			}
		}

		// When --all_access false is set, we must send `"all_access": false`
		// explicitly. The SDK's UpdateBoardRequest uses `omitempty` on the
		// AllAccess bool, which silently drops false values. Use raw Patch
		// when all_access is being set to false.
		ac := getSDK()
		var data any
		if boardUpdateAllAccess == "false" {
			body := map[string]any{"all_access": false}
			if boardUpdateName != "" {
				body["name"] = boardUpdateName
			}
			if boardUpdateAutoPostponePeriodInDays != 0 {
				body["auto_postpone_period_in_days"] = boardUpdateAutoPostponePeriodInDays
			}
			resp, patchErr := ac.Patch(cmd.Context(), "/boards/"+boardID+".json", body)
			if patchErr != nil {
				return convertSDKError(patchErr)
			}
			data = resp.Data
		} else {
			req := &generated.UpdateBoardRequest{}
			if boardUpdateName != "" {
				req.Name = boardUpdateName
			}
			if boardUpdateAllAccess == "true" {
				req.AllAccess = true
			}
			if boardUpdateAutoPostponePeriodInDays != 0 {
				req.AutoPostponePeriodInDays = int32(boardUpdateAutoPostponePeriodInDays)
			}
			var updateErr error
			data, _, updateErr = ac.Boards().Update(cmd.Context(), boardID, req)
			if updateErr != nil {
				return convertSDKError(updateErr)
			}
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy board show %s", boardID), "View board"),
			breadcrumb("cards", fmt.Sprintf("fizzy card list --board %s", boardID), "List cards"),
		}

		printMutation(normalizeAny(data), "", breadcrumbs)
		return nil
	},
}

var boardDeleteCmd = &cobra.Command{
	Use:   "delete BOARD_ID",
	Short: "Delete a board",
	Long:  "Deletes a board.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		_, err := getSDK().Boards().Delete(cmd.Context(), args[0])
		if err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("boards", "fizzy board list", "List boards"),
			breadcrumb("create", "fizzy board create --name \"name\"", "Create new board"),
		}

		printMutation(map[string]any{
			"deleted": true,
		}, "", breadcrumbs)
		return nil
	},
}

var boardPublishCmd = &cobra.Command{
	Use:   "publish BOARD_ID",
	Short: "Publish a board",
	Long:  "Publishes a board and returns its public share URL.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		boardID := args[0]

		client := getClient()
		resp, err := client.Post("/boards/"+boardID+"/publication.json", nil)
		if err != nil {
			return err
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy board show %s", boardID), "View board"),
			breadcrumb("cards", fmt.Sprintf("fizzy card list --board %s", boardID), "List cards"),
			breadcrumb("unpublish", fmt.Sprintf("fizzy board unpublish %s", boardID), "Disable public board link"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]any{"published": true}
		}

		printMutation(data, "", breadcrumbs)
		return nil
	},
}

var boardUnpublishCmd = &cobra.Command{
	Use:   "unpublish BOARD_ID",
	Short: "Unpublish a board",
	Long:  "Removes a board's public share URL.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		boardID := args[0]

		client := getClient()
		_, err := client.Delete("/boards/" + boardID + "/publication.json")
		if err != nil {
			return err
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy board show %s", boardID), "View board"),
			breadcrumb("cards", fmt.Sprintf("fizzy card list --board %s", boardID), "List cards"),
			breadcrumb("publish", fmt.Sprintf("fizzy board publish %s", boardID), "Create public board link"),
		}

		printMutation(map[string]any{
			"unpublished": true,
		}, "", breadcrumbs)
		return nil
	},
}

// Board entropy flags
var boardEntropyAutoPostponePeriodInDays int

var boardEntropyCmd = &cobra.Command{
	Use:   "entropy BOARD_ID",
	Short: "Update board auto-postpone period",
	Long:  "Updates the auto-postpone period for a specific board. Requires board admin permission.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if boardEntropyAutoPostponePeriodInDays == 0 {
			return newRequiredFlagError("auto_postpone_period_in_days")
		}
		if err := validateAutoPostponePeriodInDays(boardEntropyAutoPostponePeriodInDays); err != nil {
			return err
		}

		boardID := args[0]

		req := &generated.UpdateBoardEntropyRequest{
			AutoPostponePeriodInDays: int32(boardEntropyAutoPostponePeriodInDays),
		}

		data, _, err := getSDK().Boards().UpdateEntropy(cmd.Context(), boardID, req)
		if err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy board show %s", boardID), "View board"),
			breadcrumb("cards", fmt.Sprintf("fizzy card list --board %s", boardID), "List cards"),
		}

		printMutation(normalizeAny(data), "", breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(boardCmd)

	// List
	boardListCmd.Flags().IntVar(&boardListPage, "page", 0, "Page number")
	boardListCmd.Flags().BoolVar(&boardListAll, "all", false, "Fetch all pages")
	boardCmd.AddCommand(boardListCmd)

	// Show
	boardCmd.AddCommand(boardShowCmd)

	// Create
	boardCreateCmd.Flags().StringVar(&boardCreateName, "name", "", "Board name (required)")
	boardCreateCmd.Flags().StringVar(&boardCreateAllAccess, "all_access", "", "Allow all team members access (true/false)")
	boardCreateCmd.Flags().IntVar(&boardCreateAutoPostponePeriodInDays, "auto_postpone_period_in_days", 0, "Auto postpone period in days ("+validAutoPostponePeriodsHelp+")")
	boardCmd.AddCommand(boardCreateCmd)

	// Update
	boardUpdateCmd.Flags().StringVar(&boardUpdateName, "name", "", "Board name")
	boardUpdateCmd.Flags().StringVar(&boardUpdateAllAccess, "all_access", "", "Allow all team members access (true/false)")
	boardUpdateCmd.Flags().IntVar(&boardUpdateAutoPostponePeriodInDays, "auto_postpone_period_in_days", 0, "Auto postpone period in days ("+validAutoPostponePeriodsHelp+")")
	boardCmd.AddCommand(boardUpdateCmd)

	// Delete
	boardCmd.AddCommand(boardDeleteCmd)

	// Publication
	boardCmd.AddCommand(boardPublishCmd)
	boardCmd.AddCommand(boardUnpublishCmd)

	// Entropy
	boardEntropyCmd.Flags().IntVar(&boardEntropyAutoPostponePeriodInDays, "auto_postpone_period_in_days", 0, "Auto postpone period in days ("+validAutoPostponePeriodsHelp+")")
	boardCmd.AddCommand(boardEntropyCmd)
}
