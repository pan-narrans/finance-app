package telegram

import (
	"fmt"
	"strings"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
	"gopkg.in/telebot.v3"
)

// Callback constants to remove magic strings from handlers.
const (
	CallbackConfirm      = "confirm"
	CallbackDiscard      = "discard"
	CallbackEditAcc      = "edit_acc"
	CallbackSelectAcc    = "select_acc"
	CallbackCancelEdit   = "cancel_edit"
	CallbackCreateAcc    = "create_acc"
	CallbackAddSubAcc    = "add_sub_acc"
	CallbackDoneAcc      = "done_acc"
	CallbackSelectParent = "sel_parent"
	CallbackCancelImport = "cancel_import"
	CallbackAcceptAll    = "accept_all"
)

// UI message constants.
const (
	MsgSessionExpired         = "Session expired."
	MsgTransactionSaved       = "Transaction saved! ✅"
	MsgTransactionDiscarded   = "Transaction discarded. ❌"
	MsgAccountUpdated         = "Account updated."
	MsgAccountCreatedSelected = "Account created and selected."
	MsgImportCancelled        = "Remaining transactions cancelled. 🛑"
	MsgUseButtons             = "Please use the buttons provided to continue or click Cancel."
	MsgSelectTopLevel         = "Select a top-level account:"
	MsgWelcome                = "Welcome to Finance App Bot! Send me an amount and description (e.g., '12.50 dinner') or upload a bank file."
	MsgPromptTransaction      = "Please send transaction details in format:\n<code>[source] &lt;amount&gt; &lt;description/target&gt;</code>\n\nExample: <code>Cash 10 steam</code>"

	LabelConfirm    = "Confirm ✅"
	LabelDiscard    = "Discard ❌"
	LabelEditTarget = "Edit Target ✏️"
	LabelEditSource = "Edit Source ✏️"
	LabelAcceptAll  = "Accept All Remaining ⏩"
	LabelCancelRem  = "Cancel Remaining 🛑"
	LabelDone       = "Done ✅"
	LabelAddSub     = "Add Sub-account ➕"
	LabelCancel     = "Cancel 🔙"
	LabelCreateAcc  = "✨ Create New Account"
)

/*
UI provides helpers for building Telegram-specific message layouts and keyboards.
It is stateless and depends on configuration passed per request.
*/
type UI struct {
	webAppBaseURL string
}

/*
NewUI creates a new UI helper instance.
*/
func NewUI(webAppBaseURL string) *UI {
	return &UI{
		webAppBaseURL: webAppBaseURL,
	}
}

/*
BuildDraftMessage creates the text and keyboard for a transaction draft.
*/
func (u *UI) BuildDraftMessage(tx domain.Transaction, mappingProvider ports.MappingProvider, settings domain.Settings, formatter ports.TransactionFormatter, isPrivate bool) (string, *telebot.ReplyMarkup) {
	selector := &telebot.ReplyMarkup{}

	var editRow telebot.Row
	if isPrivate {
		// WebApp buttons only work in Private Chats
		editRow = selector.Row(
			selector.WebApp(LabelEditSource, &telebot.WebApp{URL: fmt.Sprintf("%s?type=source", u.webAppBaseURL)}),
			selector.WebApp(LabelEditTarget, &telebot.WebApp{URL: fmt.Sprintf("%s?type=target", u.webAppBaseURL)}),
		)
	} else {
		// Fallback to standard buttons in Groups
		editRow = selector.Row(
			selector.Data(LabelEditSource, CallbackEditAcc, "1"),
			selector.Data(LabelEditTarget, CallbackEditAcc, "0"),
		)
	}

	rows := []telebot.Row{
		makeRow(selector, LabelConfirm, CallbackConfirm),
		editRow,
		makeRow(selector, LabelDiscard, CallbackDiscard),
	}

	targetAccount := tx.Postings[0].Account
	msgSuffix := ""
	if strings.HasSuffix(targetAccount, ":Unknown") {
		msgSuffix = "\n\nUnknown account. Suggestions:"
		suggestions := mappingProvider.SearchAccounts(tx.Description, 5)
		rows = append(rows, mapToRows(selector, suggestions, CallbackSelectAcc)...)
	}

	selector.Inline(rows...)

	formatted := formatter.FormatTransaction(tx, settings.LedgerAlignment)
	msg := fmt.Sprintf("Draft Transaction:\n<pre>%s</pre>%s", formatted, msgSuffix)

	return msg, selector
}

/*
BuildImportReviewMessage is a specialized version of BuildDraftMessage for the import review flow.
It includes "Accept All" and "Cancel Import" options.
*/
func (u *UI) BuildImportReviewMessage(tx domain.Transaction, pendingCount int, mappingProvider ports.MappingProvider, settings domain.Settings, formatter ports.TransactionFormatter, isPrivate bool) (string, *telebot.ReplyMarkup) {
	selector := &telebot.ReplyMarkup{}

	var editRow telebot.Row
	if isPrivate {
		editRow = selector.Row(
			selector.WebApp(LabelEditSource, &telebot.WebApp{URL: fmt.Sprintf("%s?type=source", u.webAppBaseURL)}),
			selector.WebApp(LabelEditTarget, &telebot.WebApp{URL: fmt.Sprintf("%s?type=target", u.webAppBaseURL)}),
		)
	} else {
		editRow = selector.Row(
			selector.Data(LabelEditSource, CallbackEditAcc, "1"),
			selector.Data(LabelEditTarget, CallbackEditAcc, "0"),
		)
	}

	rows := []telebot.Row{
		makeRow(selector, LabelConfirm, CallbackConfirm),
		editRow,
		makeRow(selector, LabelDiscard, CallbackDiscard),
		selector.Row(
			selector.Data(LabelAcceptAll, CallbackAcceptAll),
			selector.Data(LabelCancelRem, CallbackCancelImport),
		),
	}

	targetAccount := tx.Postings[0].Account
	msgSuffix := ""
	if strings.HasSuffix(targetAccount, ":Unknown") {
		msgSuffix = "\n\nUnknown account. Suggestions:"
		suggestions := mappingProvider.SearchAccounts(tx.Description, 5)
		rows = append(rows, mapToRows(selector, suggestions, CallbackSelectAcc)...)
	}

	selector.Inline(rows...)

	formatted := formatter.FormatTransaction(tx, settings.LedgerAlignment)
	msg := fmt.Sprintf("Reviewing Import (%d left):\n<pre>%s</pre>%s", pendingCount+1, formatted, msgSuffix)

	return msg, selector
}

/*
BuildEditPrompt creates the text and keyboard for an account search prompt.
*/
func (u *UI) BuildEditPrompt(isSource bool, results []string) (string, *telebot.ReplyMarkup) {
	selector := &telebot.ReplyMarkup{}
	rows := mapToRows(selector, results, CallbackSelectAcc)
	rows = append(rows, searchFooter(selector)...)

	selector.Inline(rows...)

	accountType := "target"
	if isSource {
		accountType = "source"
	}

	if len(results) > 0 {
		return fmt.Sprintf("Suggestions for <b>%s</b> (or type to search):", accountType), selector
	}

	return fmt.Sprintf("Send text to search for <b>%s</b> account.", accountType), selector
}

/*
BuildSearchResults creates the text and keyboard for account search results.
*/
func (u *UI) BuildSearchResults(query string, results []string) (string, *telebot.ReplyMarkup) {
	selector := &telebot.ReplyMarkup{}
	rows := mapToRows(selector, results, CallbackSelectAcc)
	rows = append(rows, searchFooter(selector)...)

	selector.Inline(rows...)

	return fmt.Sprintf("Search results for '%s':", query), selector
}

/*
BuildAccountParentSelector creates the keyboard for selecting the root account.
*/
func (u *UI) BuildAccountParentSelector(parents []string) (string, *telebot.ReplyMarkup) {
	selector := &telebot.ReplyMarkup{}
	rows := mapToRows(selector, parents, CallbackSelectParent)
	rows = append(rows, makeRow(selector, LabelCancel, CallbackCancelEdit))

	selector.Inline(rows...)

	return MsgSelectTopLevel, selector
}

/*
BuildAccountChildPrompt creates the text for prompting a sub-account name.
*/
func (u *UI) BuildAccountChildPrompt(currentPath string) (string, *telebot.ReplyMarkup) {
	selector := &telebot.ReplyMarkup{}
	selector.Inline(makeRow(selector, LabelCancel, CallbackCancelEdit))

	return fmt.Sprintf("Current path: <code>%s</code>\n\nType the name of the sub-account (e.g., 'Transport'):", currentPath), selector
}

/*
BuildAccountReview creates the keyboard for finalizing or extending an account path.
*/
func (u *UI) BuildAccountReview(path string) (string, *telebot.ReplyMarkup) {
	selector := &telebot.ReplyMarkup{}

	selector.Inline(
		makeRow(selector, LabelDone, CallbackDoneAcc),
		makeRow(selector, LabelAddSub, CallbackAddSubAcc),
		makeRow(selector, LabelCancel, CallbackCancelEdit),
	)

	return fmt.Sprintf("Account constructed: <code>%s</code>\n\nWhat would you like to do?", path), selector
}

// Helpers

/*
makeRow creates a single-button row in the provided markup.
*/
func makeRow(m *telebot.ReplyMarkup, text, unique string, data ...string) telebot.Row {
	return m.Row(m.Data(text, unique, data...))
}

/*
mapToRows converts a slice of strings into a slice of single-button rows.
Each button uses the item string as both its label and its callback data.
*/
func mapToRows(m *telebot.ReplyMarkup, items []string, unique string) []telebot.Row {
	rows := make([]telebot.Row, 0, len(items))
	for _, item := range items {
		rows = append(rows, makeRow(m, item, unique, item))
	}
	return rows
}

/*
searchFooter returns the standard action rows for search-related keyboards,
containing "Create New Account" and "Cancel" buttons.
*/
func searchFooter(m *telebot.ReplyMarkup) []telebot.Row {
	return []telebot.Row{
		makeRow(m, LabelCreateAcc, CallbackCreateAcc),
		makeRow(m, LabelCancel, CallbackCancelEdit),
	}
}
