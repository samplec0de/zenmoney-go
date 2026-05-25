package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	zm "github.com/miilg/zenmoney-go"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "auth":
		cmdAuth(os.Args[2:])
	case "sync":
		cmdSync(os.Args[2:])
	case "accounts":
		cmdAccounts(os.Args[2:])
	case "transactions", "tx":
		cmdTx(os.Args[2:])
	case "categories", "cats":
		cmdCategories(os.Args[2:])
	case "summary":
		cmdSummary(os.Args[2:])
	case "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`zm — ZenMoney CLI

Usage: zm <command> [flags]

Commands:
  auth           Save access token
  sync           Sync data from ZenMoney
  accounts       List accounts with balances
  transactions   List transactions (alias: tx)
  tx add         Create a new transaction
  tx delete      Soft-delete a transaction (aliases: tx del, tx rm)
  categories     List categories (alias: cats)
  summary        Monthly income/expense summary
  help           Show this help`)
}

func fatal(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+msg+"\n", args...)
	os.Exit(1)
}

func mustStore() *zm.Store {
	s, err := zm.NewStore()
	if err != nil {
		fatal("init store: %v", err)
	}
	return s
}

func mustClient(s *zm.Store) *zm.Client {
	tok, err := s.LoadToken()
	if err != nil {
		fatal("not authenticated — run: zm auth --token <TOKEN>")
	}
	return zm.NewClient(tok.AccessToken)
}

func mustLoadSync(s *zm.Store) *zm.SyncState {
	state, err := s.LoadSync()
	if err != nil {
		fatal("no data — run: zm sync")
	}
	return state
}

// ── auth ──────────────────────────────────────────────────────────────

func cmdAuth(args []string) {
	fs := flag.NewFlagSet("auth", flag.ExitOnError)
	token := fs.String("token", "", "access token (get one at https://zerro.app/token)")
	fs.Parse(args)

	if *token == "" {
		fmt.Println("Get your token at https://zerro.app/token then run:")
		fmt.Println("  zm auth --token <TOKEN>")
		os.Exit(1)
	}

	s := mustStore()
	if err := s.SaveToken(&zm.Token{AccessToken: *token}); err != nil {
		fatal("save token: %v", err)
	}
	fmt.Println("Token saved. Run: zm sync")
}

// ── sync ──────────────────────────────────────────────────────────────

func cmdSync(args []string) {
	fs := flag.NewFlagSet("sync", flag.ExitOnError)
	full := fs.Bool("full", false, "force full re-sync")
	fs.Parse(args)

	s := mustStore()
	c := mustClient(s)

	var serverTS int64
	var state *zm.SyncState

	if !*full {
		existing, err := s.LoadSync()
		if err == nil {
			serverTS = existing.ServerTimestamp
			state = existing
		}
	}

	fmt.Print("Syncing...")
	diff, err := c.Diff(&zm.DiffRequest{ServerTimestamp: serverTS})
	if err != nil {
		fatal("\nsync failed: %v", err)
	}

	if state == nil || *full {
		state = &zm.SyncState{
			ServerTimestamp: diff.ServerTimestamp,
			Instrument:     diff.Instrument,
			Company:        diff.Company,
			User:           diff.User,
			Account:        diff.Account,
			Tag:            diff.Tag,
			Merchant:       diff.Merchant,
			Reminder:       diff.Reminder,
			ReminderMarker: diff.ReminderMarker,
			Transaction:    diff.Transaction,
			Budget:         diff.Budget,
		}
	} else {
		state.MergeSync(diff)
	}

	if err := s.SaveSync(state); err != nil {
		fatal("\nsave data: %v", err)
	}

	fmt.Printf(" done\n")
	fmt.Printf("  %d accounts, %d transactions, %d categories\n",
		len(state.Account), len(state.Transaction), len(state.Tag))
}

// ── accounts ──────────────────────────────────────────────────────────

func cmdAccounts(args []string) {
	fs := flag.NewFlagSet("accounts", flag.ExitOnError)
	all := fs.Bool("all", false, "include archived accounts")
	fs.Parse(args)

	s := mustStore()
	state := mustLoadSync(s)

	instruments := makeInstrumentMap(state.Instrument)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "TITLE\tTYPE\tBALANCE\tCURRENCY\n")

	for _, a := range state.Account {
		if a.Archive && !*all {
			continue
		}
		bal := ""
		if a.Balance != nil {
			bal = formatMoney(*a.Balance)
		}
		cur := ""
		if a.Instrument != nil {
			if inst, ok := instruments[*a.Instrument]; ok {
				cur = inst.ShortTitle
			}
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.Title, a.Type, bal, cur)
	}
	w.Flush()
}

// ── transactions ──────────────────────────────────────────────────────

func cmdTransactions(args []string) {
	fs := flag.NewFlagSet("transactions", flag.ExitOnError)
	from := fs.String("from", "", "start date (YYYY-MM-DD)")
	to := fs.String("to", "", "end date (YYYY-MM-DD)")
	account := fs.String("account", "", "filter by account title (substring)")
	tag := fs.String("tag", "", "filter by category title (substring)")
	limit := fs.Int("limit", 30, "max rows to show")
	showID := fs.Bool("id", false, "include the transaction id as the first column (for `zm tx delete`)")
	fs.Parse(args)

	s := mustStore()
	state := mustLoadSync(s)

	tags := makeTagMap(state.Tag)
	accounts := makeAccountMap(state.Account)
	instruments := makeInstrumentMap(state.Instrument)

	// Filter
	var txs []zm.Transaction
	for _, t := range state.Transaction {
		if t.Deleted {
			continue
		}
		if *from != "" && t.Date < *from {
			continue
		}
		if *to != "" && t.Date > *to {
			continue
		}
		if *account != "" {
			accTitle := accountTitle(accounts, t.IncomeAccount, t.OutcomeAccount)
			if !containsLower(accTitle, *account) {
				continue
			}
		}
		if *tag != "" {
			tagTitle := tagTitles(tags, t.Tag)
			if !containsLower(tagTitle, *tag) {
				continue
			}
		}
		txs = append(txs, t)
	}

	// Sort by date desc
	sort.Slice(txs, func(i, j int) bool {
		if txs[i].Date == txs[j].Date {
			return txs[i].Changed > txs[j].Changed
		}
		return txs[i].Date > txs[j].Date
	})

	if *limit > 0 && len(txs) > *limit {
		txs = txs[:*limit]
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if *showID {
		fmt.Fprintf(w, "ID\tDATE\tAMOUNT\tACCOUNT\tCATEGORY\tPAYEE\tCOMMENT\n")
	} else {
		fmt.Fprintf(w, "DATE\tAMOUNT\tACCOUNT\tCATEGORY\tPAYEE\tCOMMENT\n")
	}

	for _, t := range txs {
		amount, cur := formatTransaction(t, accounts, instruments)
		acc := accountTitle(accounts, t.IncomeAccount, t.OutcomeAccount)
		cat := tagTitles(tags, t.Tag)
		payee := deref(t.Payee)
		comment := deref(t.Comment)
		if len(comment) > 30 {
			comment = comment[:27] + "..."
		}
		if *showID {
			fmt.Fprintf(w, "%s\t%s\t%s %s\t%s\t%s\t%s\t%s\n", t.ID, t.Date, amount, cur, acc, cat, payee, comment)
		} else {
			fmt.Fprintf(w, "%s\t%s %s\t%s\t%s\t%s\t%s\n", t.Date, amount, cur, acc, cat, payee, comment)
		}
	}
	w.Flush()
	fmt.Printf("\nShowing %d of %d transactions\n", len(txs), countActive(state.Transaction))
}

// ── categories ────────────────────────────────────────────────────────

func cmdCategories(args []string) {
	flag.NewFlagSet("categories", flag.ExitOnError).Parse(args)

	s := mustStore()
	state := mustLoadSync(s)

	// Group by parent
	roots := make([]zm.Tag, 0)
	children := make(map[string][]zm.Tag)
	for _, t := range state.Tag {
		if t.Parent == nil || *t.Parent == "" {
			roots = append(roots, t)
		} else {
			children[*t.Parent] = append(children[*t.Parent], t)
		}
	}

	sort.Slice(roots, func(i, j int) bool { return roots[i].Title < roots[j].Title })

	for _, r := range roots {
		dir := dirLabel(r.ShowIncome, r.ShowOutcome)
		fmt.Printf("%s %s\n", r.Title, dir)
		kids := children[r.ID]
		sort.Slice(kids, func(i, j int) bool { return kids[i].Title < kids[j].Title })
		for _, c := range kids {
			cdir := dirLabel(c.ShowIncome, c.ShowOutcome)
			fmt.Printf("  └ %s %s\n", c.Title, cdir)
		}
	}
}

// ── summary ───────────────────────────────────────────────────────────

func cmdSummary(args []string) {
	fs := flag.NewFlagSet("summary", flag.ExitOnError)
	months := fs.Int("months", 3, "number of months to show")
	fs.Parse(args)

	s := mustStore()
	state := mustLoadSync(s)
	instruments := makeInstrumentMap(state.Instrument)
	accounts := makeAccountMap(state.Account)

	// Find base currency from first user
	baseCur := "RUB"
	if len(state.User) > 0 {
		if inst, ok := instruments[state.User[0].Currency]; ok {
			baseCur = inst.ShortTitle
		}
	}

	type monthStats struct {
		income  float64
		expense float64
	}
	byMonth := make(map[string]*monthStats)

	cutoff := time.Now().AddDate(0, -*months, 0).Format("2006-01-02")

	for _, t := range state.Transaction {
		if t.Deleted || t.Date < cutoff {
			continue
		}
		month := t.Date[:7] // YYYY-MM
		ms, ok := byMonth[month]
		if !ok {
			ms = &monthStats{}
			byMonth[month] = ms
		}

		isTransfer := t.IncomeAccount != t.OutcomeAccount && t.Income > 0 && t.Outcome > 0
		if isTransfer {
			// skip transfers between own accounts
			_, incOwn := accounts[t.IncomeAccount]
			_, outOwn := accounts[t.OutcomeAccount]
			if incOwn && outOwn {
				continue
			}
		}

		if t.Income > 0 && t.Outcome == 0 {
			ms.income += t.Income
		}
		if t.Outcome > 0 && t.Income == 0 {
			ms.expense += t.Outcome
		}
	}

	// Sort months
	monthKeys := make([]string, 0, len(byMonth))
	for k := range byMonth {
		monthKeys = append(monthKeys, k)
	}
	sort.Strings(monthKeys)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "MONTH\tINCOME\tEXPENSE\tNET\n")
	for _, m := range monthKeys {
		ms := byMonth[m]
		net := ms.income - ms.expense
		sign := ""
		if net >= 0 {
			sign = "+"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s%s\t%s\n", m,
			formatMoney(ms.income), formatMoney(ms.expense),
			sign, formatMoney(net), baseCur)
	}
	w.Flush()

	// Net worth
	fmt.Println()
	var totalNetWorth float64
	for _, a := range state.Account {
		if a.Archive || !a.InBalance || a.Balance == nil {
			continue
		}
		bal := *a.Balance
		// Convert to base currency via instrument rate
		if a.Instrument != nil {
			if inst, ok := instruments[*a.Instrument]; ok {
				if baseCur != inst.ShortTitle && inst.Rate > 0 {
					// inst.Rate is rate to RUB; if base is RUB this works directly
					bal *= inst.Rate
				}
			}
		}
		totalNetWorth += bal
	}
	fmt.Printf("Net worth: %s %s\n", formatMoney(totalNetWorth), baseCur)
}

// ── helpers ───────────────────────────────────────────────────────────

func makeInstrumentMap(instruments []zm.Instrument) map[int]zm.Instrument {
	m := make(map[int]zm.Instrument, len(instruments))
	for _, i := range instruments {
		m[i.ID] = i
	}
	return m
}

func makeTagMap(tags []zm.Tag) map[string]zm.Tag {
	m := make(map[string]zm.Tag, len(tags))
	for _, t := range tags {
		m[t.ID] = t
	}
	return m
}

func makeAccountMap(accounts []zm.Account) map[string]zm.Account {
	m := make(map[string]zm.Account, len(accounts))
	for _, a := range accounts {
		m[a.ID] = a
	}
	return m
}

func accountTitle(accounts map[string]zm.Account, incomeAcc, outcomeAcc string) string {
	if outcomeAcc != "" {
		if a, ok := accounts[outcomeAcc]; ok {
			return a.Title
		}
	}
	if incomeAcc != "" {
		if a, ok := accounts[incomeAcc]; ok {
			return a.Title
		}
	}
	return ""
}

func tagTitles(tags map[string]zm.Tag, ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if t, ok := tags[id]; ok {
			names = append(names, t.Title)
		}
	}
	return strings.Join(names, ", ")
}

func formatTransaction(t zm.Transaction, accounts map[string]zm.Account, instruments map[int]zm.Instrument) (string, string) {
	cur := ""
	if t.Outcome > 0 {
		if a, ok := accounts[t.OutcomeAccount]; ok && a.Instrument != nil {
			if inst, ok := instruments[*a.Instrument]; ok {
				cur = inst.Symbol
			}
		}
		return "-" + formatMoney(t.Outcome), cur
	}
	if a, ok := accounts[t.IncomeAccount]; ok && a.Instrument != nil {
		if inst, ok := instruments[*a.Instrument]; ok {
			cur = inst.Symbol
		}
	}
	return "+" + formatMoney(t.Income), cur
}

func formatMoney(v float64) string {
	neg := v < 0
	v = math.Abs(v)
	whole := int64(v)
	frac := int64(math.Round((v - float64(whole)) * 100))

	// Add thousands separator
	s := fmt.Sprintf("%d", whole)
	if len(s) > 3 {
		var parts []string
		for len(s) > 3 {
			parts = append([]string{s[len(s)-3:]}, parts...)
			s = s[:len(s)-3]
		}
		parts = append([]string{s}, parts...)
		s = strings.Join(parts, " ")
	}

	result := s
	if frac > 0 {
		result = fmt.Sprintf("%s.%02d", s, frac)
	}
	if neg {
		result = "-" + result
	}
	return result
}

func containsLower(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func countActive(txs []zm.Transaction) int {
	n := 0
	for _, t := range txs {
		if !t.Deleted {
			n++
		}
	}
	return n
}

func dirLabel(income, outcome bool) string {
	switch {
	case income && outcome:
		return "(in/out)"
	case income:
		return "(in)"
	case outcome:
		return "(out)"
	default:
		return ""
	}
}
