package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	zm "github.com/miilg/zenmoney-go"
)

// cmdTx dispatches `zm tx <subcommand>`; when no recognized subcommand is
// given (or args[0] starts with a dash), falls through to the read-only
// list view in cmdTransactions for backward compatibility.
func cmdTx(args []string) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "add":
			cmdTxAdd(args[1:])
			return
		case "delete", "del", "rm":
			cmdTxDelete(args[1:])
			return
		}
	}
	cmdTransactions(args)
}

// ── tx add ────────────────────────────────────────────────────────────

func cmdTxAdd(args []string) {
	fs := flag.NewFlagSet("tx add", flag.ExitOnError)
	amount := fs.String("amount", "", "amount (positive, in account currency)")
	account := fs.String("account", "", "account title or ID (substring, case-insensitive)")
	tag := fs.String("tag", "", "category title or ID (substring, optional)")
	merchant := fs.String("merchant", "", "merchant title or ID (substring, optional)")
	payee := fs.String("payee", "", "free-form payee text (optional)")
	comment := fs.String("comment", "", "comment text (optional)")
	date := fs.String("date", "", "date YYYY-MM-DD (default: today)")
	income := fs.Bool("income", false, "treat amount as income (default: outcome/expense)")
	yes := fs.Bool("yes", false, "skip confirmation prompt")
	fs.Parse(args)

	if *amount == "" || *account == "" {
		fatal("tx add requires --amount and --account")
	}
	amt, err := strconv.ParseFloat(strings.Replace(*amount, ",", ".", 1), 64)
	if err != nil || amt <= 0 {
		fatal("invalid amount %q (must be positive number)", *amount)
	}

	s := mustStore()
	state := mustLoadSync(s)
	c := mustClient(s)

	acc := resolveAccount(state.Account, *account)
	if acc.Instrument == nil {
		fatal("account %q has no instrument (currency) set", acc.Title)
	}

	// Non-nil empty slice; the server's diff endpoint distinguishes
	// `"tag": []` (no categories) from a missing key (rejected as validation
	// error) and we always send an explicit value.
	tagIDs := []string{}
	if *tag != "" {
		tagIDs = []string{resolveTag(state.Tag, *tag, *income).ID}
	}
	var merchantID *string
	if *merchant != "" {
		m := resolveMerchant(state.Merchant, *merchant)
		merchantID = &m.ID
	}
	var payeePtr *string
	if *payee != "" {
		payeePtr = payee
	}
	var commentPtr *string
	if *comment != "" {
		commentPtr = comment
	}

	txDate := *date
	if txDate == "" {
		txDate = time.Now().Format("2006-01-02")
	} else if _, err := time.Parse("2006-01-02", txDate); err != nil {
		fatal("invalid date %q (expected YYYY-MM-DD)", txDate)
	}

	now := time.Now().Unix()
	tx := zm.Transaction{
		ID:                zm.NewUUID(),
		Changed:           now,
		Created:           now,
		User:              acc.User,
		IncomeInstrument:  *acc.Instrument,
		IncomeAccount:     acc.ID,
		OutcomeInstrument: *acc.Instrument,
		OutcomeAccount:    acc.ID,
		Tag:               tagIDs,
		Merchant:          merchantID,
		Payee:             payeePtr,
		Comment:           commentPtr,
		Date:              txDate,
	}
	if *income {
		tx.Income = amt
	} else {
		tx.Outcome = amt
	}

	cur := instrumentSymbol(state.Instrument, *acc.Instrument)
	sign := "-"
	if *income {
		sign = "+"
	}
	fmt.Printf("Going to create:\n  %s%s %s  on %s  [%s]\n",
		sign, formatMoney(amt), cur, txDate, acc.Title)
	if len(tagIDs) > 0 {
		fmt.Printf("  category: %s\n", tagTitle(state.Tag, tagIDs[0]))
	}
	if merchantID != nil {
		fmt.Printf("  merchant: %s\n", merchantTitle(state.Merchant, *merchantID))
	}
	if payeePtr != nil {
		fmt.Printf("  payee:    %s\n", *payeePtr)
	}
	if commentPtr != nil {
		fmt.Printf("  comment:  %s\n", *commentPtr)
	}
	if !*yes && !confirm("Proceed?") {
		fmt.Println("aborted.")
		return
	}

	fmt.Print("Pushing...")
	resp, err := c.Push(state.ServerTimestamp, zm.WithTransactions([]zm.Transaction{tx}))
	if err != nil {
		fatal("\npush failed: %v", err)
	}
	state.MergeSync(resp)
	if err := s.SaveSync(state); err != nil {
		fatal("\nsave data: %v", err)
	}
	fmt.Printf(" done\n")
	fmt.Printf("Created transaction %s\n", tx.ID)
}

// ── tx delete ─────────────────────────────────────────────────────────

func cmdTxDelete(args []string) {
	fs := flag.NewFlagSet("tx delete", flag.ExitOnError)
	yes := fs.Bool("yes", false, "skip confirmation prompt")
	fs.Parse(args)
	rest := fs.Args()
	if len(rest) != 1 {
		fatal("usage: zm tx delete [--yes] <transaction-id-or-prefix>")
	}
	idArg := rest[0]

	s := mustStore()
	state := mustLoadSync(s)
	c := mustClient(s)

	target := resolveTransaction(state.Transaction, idArg)

	accs := makeAccountMap(state.Account)
	instr := makeInstrumentMap(state.Instrument)
	tagsMap := makeTagMap(state.Tag)
	amtStr, cur := formatTransaction(target, accs, instr)
	fmt.Printf("Going to delete:\n  %s  %s %s  [%s]  %s  %s\n",
		target.Date, amtStr, cur,
		accountTitle(accs, target.IncomeAccount, target.OutcomeAccount),
		tagTitles(tagsMap, target.Tag),
		deref(target.Comment))
	fmt.Printf("  id: %s\n", target.ID)

	if !*yes && !confirm("Proceed?") {
		fmt.Println("aborted.")
		return
	}

	target.Deleted = true
	target.Changed = time.Now().Unix()

	fmt.Print("Pushing...")
	resp, err := c.Push(state.ServerTimestamp, zm.WithTransactions([]zm.Transaction{target}))
	if err != nil {
		fatal("\npush failed: %v", err)
	}
	state.MergeSync(resp)
	if err := s.SaveSync(state); err != nil {
		fatal("\nsave data: %v", err)
	}
	fmt.Printf(" done\n")
	fmt.Printf("Deleted transaction %s\n", target.ID)
}

// ── resolvers ─────────────────────────────────────────────────────────

func resolveAccount(accounts []zm.Account, q string) zm.Account {
	var hits []zm.Account
	for _, a := range accounts {
		if a.ID == q {
			return a
		}
		if a.Archive {
			continue
		}
		if containsLower(a.Title, q) {
			hits = append(hits, a)
		}
	}
	switch len(hits) {
	case 0:
		fatal("no active account matches %q. Run: zm accounts", q)
	case 1:
		return hits[0]
	}
	for _, a := range hits {
		if strings.EqualFold(a.Title, q) {
			return a
		}
	}
	fatal("multiple accounts match %q (%d):\n%s", q, len(hits), formatAccountList(hits))
	return zm.Account{}
}

// resolveTag picks a category by substring. When isIncome is set, prefers
// categories with ShowIncome; otherwise prefers ShowOutcome. Falls back to
// any direction match if the preferred list is empty.
func resolveTag(tags []zm.Tag, q string, isIncome bool) zm.Tag {
	var hits []zm.Tag
	for _, t := range tags {
		if t.ID == q {
			return t
		}
		if containsLower(t.Title, q) {
			hits = append(hits, t)
		}
	}
	if len(hits) == 0 {
		fatal("no category matches %q. Run: zm cats", q)
	}
	// Narrow by direction
	var directional []zm.Tag
	for _, t := range hits {
		if isIncome && t.ShowIncome {
			directional = append(directional, t)
		} else if !isIncome && t.ShowOutcome {
			directional = append(directional, t)
		}
	}
	if len(directional) > 0 {
		hits = directional
	}
	if len(hits) == 1 {
		return hits[0]
	}
	for _, t := range hits {
		if strings.EqualFold(t.Title, q) {
			return t
		}
	}
	fatal("multiple categories match %q (%d):\n%s", q, len(hits), formatTagList(hits))
	return zm.Tag{}
}

func resolveMerchant(merchants []zm.Merchant, q string) zm.Merchant {
	var hits []zm.Merchant
	for _, m := range merchants {
		if m.ID == q {
			return m
		}
		if containsLower(m.Title, q) {
			hits = append(hits, m)
		}
	}
	switch len(hits) {
	case 0:
		fatal("no merchant matches %q (use --payee for free text instead)", q)
	case 1:
		return hits[0]
	}
	for _, m := range hits {
		if strings.EqualFold(m.Title, q) {
			return m
		}
	}
	fatal("multiple merchants match %q (%d):\n%s", q, len(hits), formatMerchantList(hits))
	return zm.Merchant{}
}

func resolveTransaction(txs []zm.Transaction, q string) zm.Transaction {
	var hits []zm.Transaction
	for _, t := range txs {
		if t.Deleted {
			continue
		}
		if t.ID == q {
			return t
		}
		if strings.HasPrefix(t.ID, q) {
			hits = append(hits, t)
		}
	}
	switch len(hits) {
	case 0:
		fatal("no active transaction with id or prefix %q", q)
	case 1:
		return hits[0]
	}
	fatal("ambiguous id prefix %q matches %d transactions — use a longer prefix", q, len(hits))
	return zm.Transaction{}
}

// ── small helpers ─────────────────────────────────────────────────────

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

func tagTitle(tags []zm.Tag, id string) string {
	for _, t := range tags {
		if t.ID == id {
			return t.Title
		}
	}
	return id
}

func merchantTitle(merchants []zm.Merchant, id string) string {
	for _, m := range merchants {
		if m.ID == id {
			return m.Title
		}
	}
	return id
}

func instrumentSymbol(instruments []zm.Instrument, id int) string {
	for _, i := range instruments {
		if i.ID == id {
			return i.Symbol
		}
	}
	return ""
}

func formatAccountList(accounts []zm.Account) string {
	var b strings.Builder
	for _, a := range accounts {
		fmt.Fprintf(&b, "  %s  (%s)\n", a.Title, a.ID)
	}
	return b.String()
}

func formatTagList(tags []zm.Tag) string {
	var b strings.Builder
	for _, t := range tags {
		fmt.Fprintf(&b, "  %s  (%s)\n", t.Title, t.ID)
	}
	return b.String()
}

func formatMerchantList(merchants []zm.Merchant) string {
	var b strings.Builder
	for _, m := range merchants {
		fmt.Fprintf(&b, "  %s  (%s)\n", m.Title, m.ID)
	}
	return b.String()
}
