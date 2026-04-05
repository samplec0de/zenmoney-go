package zenmoney

// Instrument represents a currency with exchange rate information.
// System entity — read-only.
type Instrument struct {
	ID         int     `json:"id"`
	Changed    int64   `json:"changed"`
	Title      string  `json:"title"`
	ShortTitle string  `json:"shortTitle"`
	Symbol     string  `json:"symbol"`
	Rate       float64 `json:"rate"`
}

// Company represents a financial institution (bank, payment org).
// System entity — read-only.
type Company struct {
	ID          int     `json:"id"`
	Changed     int64   `json:"changed"`
	Title       string  `json:"title"`
	FullTitle   *string `json:"fullTitle"`
	WWW         string  `json:"www"`
	Country     *int    `json:"country"`
	CountryCode *string `json:"countryCode,omitempty"`
	Deleted     bool    `json:"deleted,omitempty"`
}

// User represents an account owner. Supports family account hierarchies.
// System entity — read-only.
type User struct {
	ID          int     `json:"id"`
	Changed     int64   `json:"changed"`
	Login       *string `json:"login"`
	Currency    int     `json:"currency"`    // -> Instrument.ID
	Parent      *int    `json:"parent"`      // -> User.ID, nil for admin
	CountryCode *string `json:"countryCode,omitempty"`
	Email       *string `json:"email,omitempty"`
}

// AccountType enumerates the possible account types.
type AccountType string

const (
	AccountCash     AccountType = "cash"
	AccountCCard    AccountType = "ccard"
	AccountChecking AccountType = "checking"
	AccountLoan     AccountType = "loan"
	AccountDeposit  AccountType = "deposit"
	AccountEMoney   AccountType = "emoney"
	AccountDebt     AccountType = "debt"
)

// IntervalType enumerates scheduling intervals.
type IntervalType string

const (
	IntervalDay   IntervalType = "day"
	IntervalWeek  IntervalType = "week"
	IntervalMonth IntervalType = "month"
	IntervalYear  IntervalType = "year"
)

// Account represents a user financial account.
type Account struct {
	ID         string  `json:"id"`
	Changed    int64   `json:"changed"`
	User       int     `json:"user"`
	Role       *int    `json:"role,omitempty"`
	Instrument *int    `json:"instrument,omitempty"`
	Company    *int    `json:"company,omitempty"`

	Type   AccountType `json:"type"`
	Title  string      `json:"title"`
	SyncID []string    `json:"syncID,omitempty"`

	Balance      *float64 `json:"balance,omitempty"`
	StartBalance *float64 `json:"startBalance,omitempty"`
	CreditLimit  *float64 `json:"creditLimit,omitempty"`

	InBalance        bool  `json:"inBalance"`
	Savings          *bool `json:"savings,omitempty"`
	EnableCorrection bool  `json:"enableCorrection"`
	EnableSMS        bool  `json:"enableSMS"`
	Archive          bool  `json:"archive"`

	// Loan/Deposit fields
	Capitalization        *bool         `json:"capitalization,omitempty"`
	Percent               *float64      `json:"percent,omitempty"`
	StartDate             *string       `json:"startDate,omitempty"`
	EndDateOffset         *int          `json:"endDateOffset,omitempty"`
	EndDateOffsetInterval *IntervalType `json:"endDateOffsetInterval,omitempty"`
	PayoffStep            *int          `json:"payoffStep,omitempty"`
	PayoffInterval        *IntervalType `json:"payoffInterval,omitempty"`
}

// Tag represents a transaction category. Supports one level of nesting.
type Tag struct {
	ID      string `json:"id"`
	Changed int64  `json:"changed"`
	User    int    `json:"user"`

	Title   string  `json:"title"`
	Parent  *string `json:"parent,omitempty"` // -> Tag.ID
	Icon    *string `json:"icon,omitempty"`
	Picture *string `json:"picture,omitempty"`
	Color   *int    `json:"color,omitempty"` // ARGB: (a<<24) + (r<<16) + (g<<8) + b

	ShowIncome    bool  `json:"showIncome"`
	ShowOutcome   bool  `json:"showOutcome"`
	BudgetIncome  bool  `json:"budgetIncome"`
	BudgetOutcome bool  `json:"budgetOutcome"`
	Required      *bool `json:"required,omitempty"`
}

// Merchant represents a payee/payer entity.
type Merchant struct {
	ID      string `json:"id"`
	Changed int64  `json:"changed"`
	User    int    `json:"user"`
	Title   string `json:"title"`
}

// Reminder represents a recurring transaction template.
type Reminder struct {
	ID      string `json:"id"`
	Changed int64  `json:"changed"`
	User    int    `json:"user"`

	IncomeInstrument  int     `json:"incomeInstrument"`
	IncomeAccount     string  `json:"incomeAccount"`
	Income            float64 `json:"income"`
	OutcomeInstrument int     `json:"outcomeInstrument"`
	OutcomeAccount    string  `json:"outcomeAccount"`
	Outcome           float64 `json:"outcome"`

	Tag      []string `json:"tag,omitempty"`
	Merchant *string  `json:"merchant,omitempty"`
	Payee    *string  `json:"payee,omitempty"`
	Comment  *string  `json:"comment,omitempty"`

	Interval  *IntervalType `json:"interval,omitempty"` // nil = non-recurring
	Step      *int          `json:"step,omitempty"`
	Points    []int         `json:"points,omitempty"`
	StartDate string        `json:"startDate"`
	EndDate   *string       `json:"endDate,omitempty"`
	Notify    bool          `json:"notify"`
}

// ReminderMarkerState enumerates the possible states of a reminder marker.
type ReminderMarkerState string

const (
	MarkerPlanned   ReminderMarkerState = "planned"
	MarkerProcessed ReminderMarkerState = "processed"
	MarkerDeleted   ReminderMarkerState = "deleted"
)

// ReminderMarker represents an individual instance of a planned transaction.
type ReminderMarker struct {
	ID      string `json:"id"`
	Changed int64  `json:"changed"`
	User    int    `json:"user"`

	IncomeInstrument  int     `json:"incomeInstrument"`
	IncomeAccount     string  `json:"incomeAccount"`
	Income            float64 `json:"income"`
	OutcomeInstrument int     `json:"outcomeInstrument"`
	OutcomeAccount    string  `json:"outcomeAccount"`
	Outcome           float64 `json:"outcome"`

	Tag      []string `json:"tag,omitempty"`
	Merchant *string  `json:"merchant,omitempty"`
	Payee    *string  `json:"payee,omitempty"`
	Comment  *string  `json:"comment,omitempty"`

	Date     string              `json:"date"`
	Reminder string              `json:"reminder"` // -> Reminder.ID
	State    ReminderMarkerState `json:"state"`
	Notify   bool                `json:"notify"`
}

// Transaction represents a monetary operation.
type Transaction struct {
	ID      string `json:"id"`
	Changed int64  `json:"changed"`
	Created int64  `json:"created"`
	User    int    `json:"user"`
	Deleted bool   `json:"deleted"`
	Hold    *bool  `json:"hold,omitempty"`

	IncomeInstrument  int     `json:"incomeInstrument"`
	IncomeAccount     string  `json:"incomeAccount"`
	Income            float64 `json:"income"`
	OutcomeInstrument int     `json:"outcomeInstrument"`
	OutcomeAccount    string  `json:"outcomeAccount"`
	Outcome           float64 `json:"outcome"`

	Tag            []string `json:"tag,omitempty"`
	Merchant       *string  `json:"merchant,omitempty"`
	Payee          *string  `json:"payee,omitempty"`
	OriginalPayee  *string  `json:"originalPayee,omitempty"`
	Comment        *string  `json:"comment,omitempty"`
	Date           string   `json:"date"` // yyyy-MM-dd
	MCC            *int     `json:"mcc,omitempty"`
	ReminderMarker *string  `json:"reminderMarker,omitempty"`

	// Original operation currency (when different from account currency)
	OpIncome           *float64 `json:"opIncome,omitempty"`
	OpIncomeInstrument *int     `json:"opIncomeInstrument,omitempty"`
	OpOutcome          *float64 `json:"opOutcome,omitempty"`
	OpOutcomeInstrument *int    `json:"opOutcomeInstrument,omitempty"`

	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

// Budget represents a monthly income/expense budget for a category.
type Budget struct {
	Changed int64 `json:"changed"`
	User    int   `json:"user"`

	Tag  *string `json:"tag"`  // -> Tag.ID, nil = uncategorized, "00000000-..." = total
	Date string  `json:"date"` // yyyy-MM-dd (month start)

	Income      float64 `json:"income"`
	IncomeLock  bool    `json:"incomeLock"`
	Outcome     float64 `json:"outcome"`
	OutcomeLock bool    `json:"outcomeLock"`
}

// Deletion represents a hard-delete record sent via the diff endpoint.
type Deletion struct {
	ID     string `json:"id"`
	Object string `json:"object"` // entity type: "transaction", "account", "tag", etc.
	User   int    `json:"user"`
	Stamp  int64  `json:"stamp"`
}

// DiffRequest is the payload sent to POST /v8/diff/.
type DiffRequest struct {
	CurrentClientTimestamp int64    `json:"currentClientTimestamp"`
	ServerTimestamp        int64    `json:"serverTimestamp"`
	ForceFetch            []string `json:"forceFetch,omitempty"`

	Instrument     []Instrument     `json:"instrument,omitempty"`
	Company        []Company        `json:"company,omitempty"`
	User           []User           `json:"user,omitempty"`
	Account        []Account        `json:"account,omitempty"`
	Tag            []Tag            `json:"tag,omitempty"`
	Merchant       []Merchant       `json:"merchant,omitempty"`
	Reminder       []Reminder       `json:"reminder,omitempty"`
	ReminderMarker []ReminderMarker `json:"reminderMarker,omitempty"`
	Transaction    []Transaction    `json:"transaction,omitempty"`
	Budget         []Budget         `json:"budget,omitempty"`
	Deletion       []Deletion       `json:"deletion,omitempty"`
}

// DiffResponse is the payload returned from POST /v8/diff/.
type DiffResponse struct {
	ServerTimestamp int64 `json:"serverTimestamp"`

	Instrument     []Instrument     `json:"instrument,omitempty"`
	Company        []Company        `json:"company,omitempty"`
	User           []User           `json:"user,omitempty"`
	Account        []Account        `json:"account,omitempty"`
	Tag            []Tag            `json:"tag,omitempty"`
	Merchant       []Merchant       `json:"merchant,omitempty"`
	Reminder       []Reminder       `json:"reminder,omitempty"`
	ReminderMarker []ReminderMarker `json:"reminderMarker,omitempty"`
	Transaction    []Transaction    `json:"transaction,omitempty"`
	Budget         []Budget         `json:"budget,omitempty"`
	Deletion       []Deletion       `json:"deletion,omitempty"`
}

// SuggestRequest is a partial transaction sent to POST /v8/suggest/.
type SuggestRequest struct {
	Payee    *string  `json:"payee,omitempty"`
	Merchant *string  `json:"merchant,omitempty"`
	Tag      []string `json:"tag,omitempty"`
	Comment  *string  `json:"comment,omitempty"`
	MCC      *int     `json:"mcc,omitempty"`
}

// SuggestResponse contains category/merchant suggestions.
type SuggestResponse struct {
	Payee    *string  `json:"payee,omitempty"`
	Merchant *string  `json:"merchant,omitempty"`
	Tag      []string `json:"tag,omitempty"`
}
