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
)

/*
UI provides helpers for building Telegram-specific message layouts and keyboards.
It is stateless and depends on configuration passed per request.
*/
type UI struct{}

/*
NewUI creates a new UI helper instance.
*/
func NewUI() *UI {
	return &UI{}
}

/*
BuildDraftMessage creates the text and keyboard for a transaction draft.
*/
func (u *UI) BuildDraftMessage(tx domain.Transaction, mappingProvider ports.MappingProvider, settings domain.Settings, formatter ports.TransactionFormatter) (string, *telebot.ReplyMarkup) {
	selector := &telebot.ReplyMarkup{}

	btnConfirm := selector.Data("Confirm ✅", CallbackConfirm)
	btnEditTarget := selector.Data("Edit Target ✏️", CallbackEditAcc, "0")
	btnEditSource := selector.Data("Edit Source ✏️", CallbackEditAcc, "1")
	btnDiscard := selector.Data("Discard ❌", CallbackDiscard)

	rows := []telebot.Row{
		selector.Row(btnConfirm),
		selector.Row(btnEditTarget, btnEditSource),
		selector.Row(btnDiscard),
	}

	targetAccount := tx.Postings[0].Account
	msgSuffix := ""
	if strings.HasSuffix(targetAccount, ":Unknown") {
		msgSuffix = "\n\nUnknown account. Suggestions:"
		suggestions := mappingProvider.SearchAccounts(tx.Description, 5)
		for _, suggestion := range suggestions {
			btn := selector.Data(suggestion, CallbackSelectAcc, suggestion)
			rows = append(rows, selector.Row(btn))
		}
	}

	selector.Inline(rows...)

	formatted := formatter.FormatTransaction(tx, settings.LedgerAlignment)
	msg := fmt.Sprintf("Draft Transaction:\n<pre>%s</pre>%s", formatted, msgSuffix)

	return msg, selector
}

/*
BuildEditPrompt creates the text and keyboard for an account search prompt.
*/
func (u *UI) BuildEditPrompt(isSource bool, results []string) (string, *telebot.ReplyMarkup) {
	selector := &telebot.ReplyMarkup{}
	rows := make([]telebot.Row, 0)

	for _, result := range results {
		btn := selector.Data(result, CallbackSelectAcc, result)
		rows = append(rows, selector.Row(btn))
	}

	btnCancel := selector.Data("Cancel 🔙", CallbackCancelEdit)
	rows = append(rows, selector.Row(btnCancel))

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
	rows := make([]telebot.Row, 0)

	for _, result := range results {
		btn := selector.Data(result, CallbackSelectAcc, result)
		rows = append(rows, selector.Row(btn))
	}

	btnCreate := selector.Data("✨ Create New Account", CallbackCreateAcc)
	rows = append(rows, selector.Row(btnCreate))

	btnCancel := selector.Data("Cancel 🔙", CallbackCancelEdit)
	rows = append(rows, selector.Row(btnCancel))

	selector.Inline(rows...)

	return fmt.Sprintf("Search results for '%s':", query), selector
}

/*
BuildAccountParentSelector creates the keyboard for selecting the root account.
*/
func (u *UI) BuildAccountParentSelector(parents []string) (string, *telebot.ReplyMarkup) {
	selector := &telebot.ReplyMarkup{}
	rows := make([]telebot.Row, 0)

	for _, p := range parents {
		btn := selector.Data(p, CallbackSelectParent, p)
		rows = append(rows, selector.Row(btn))
	}

	btnCancel := selector.Data("Cancel 🔙", CallbackCancelEdit)
	rows = append(rows, selector.Row(btnCancel))

	selector.Inline(rows...)

	return "Select a top-level account:", selector
}

/*
BuildAccountChildPrompt creates the text for prompting a sub-account name.
*/
func (u *UI) BuildAccountChildPrompt(currentPath string) (string, *telebot.ReplyMarkup) {
	selector := &telebot.ReplyMarkup{}
	btnCancel := selector.Data("Cancel 🔙", CallbackCancelEdit)
	selector.Inline(selector.Row(btnCancel))

	return fmt.Sprintf("Current path: <code>%s</code>\n\nType the name of the sub-account (e.g., 'Transport'):", currentPath), selector
}

/*
BuildAccountReview creates the keyboard for finalizing or extending an account path.
*/
func (u *UI) BuildAccountReview(path string) (string, *telebot.ReplyMarkup) {
	selector := &telebot.ReplyMarkup{}

	btnDone := selector.Data("Done ✅", CallbackDoneAcc)
	btnAdd := selector.Data("Add Sub-account ➕", CallbackAddSubAcc)
	btnCancel := selector.Data("Cancel 🔙", CallbackCancelEdit)

	selector.Inline(
		selector.Row(btnDone),
		selector.Row(btnAdd),
		selector.Row(btnCancel),
	)

	return fmt.Sprintf("Account constructed: <code>%s</code>\n\nWhat would you like to do?", path), selector
}
