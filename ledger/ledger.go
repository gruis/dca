package ledger

import (
	"errors"
	"sort"
	"time"

	"github.com/Rhymond/go-money"
)

var NegativeBalanceTransaction = errors.New("transaction would result in a negative balance")

type Transaction struct {
	ID          string
	FromAccount *Account
	ToAccount   *Account
	FromAmount  *money.Money
	ToAmount    *money.Money
	CreatedAt   time.Time
	Note        string
}

type Transactions []*Transaction

func (ts Transactions) Len() int           { return len(ts) }
func (ts Transactions) Swap(i, j int)      { ts[i], ts[j] = ts[j], ts[i] }
func (ts Transactions) Less(i, j int) bool { return ts[i].CreatedAt.Before(ts[j].CreatedAt) }

// TODO: provide, max, min, avg balance, etc.
type Account struct {
	Name          string
	Currency      *money.Currency
	Balance       *money.Money
	LastUpdateAt  time.Time
	Transactions  []*Transaction
	AllowNegative bool
}

func NewAccount(a Account) *Account {
	if a.Currency == nil {
		a.Currency = money.GetCurrency("USD")
	}
	if a.Balance == nil {
		a.Balance = money.New(0, a.Currency.Code)
	}
	sort.Sort(Transactions(a.Transactions))
	return &a
}

func (a *Account) Subtract(amount *money.Money) (err error) {
	b, err := a.Balance.Subtract(amount)
	if b.IsNegative() && !a.AllowNegative {
		return NegativeBalanceTransaction
	}
	a.Balance = b
	return err
}

func (a *Account) Add(amount *money.Money) (err error) {
	a.Balance, err = a.Balance.Add(amount)
	return err
}

type DoubleBook struct {
	Accounts     map[string]*Account
	Transactions []*Transaction
	UpdatedAt    time.Time
}

func NewDoubleBook(accounts ...*Account) (db *DoubleBook) {
	db.Accounts = map[string]*Account{}
	transactionIndex := map[string]bool{}
	for _, a := range accounts {
		db.Accounts[a.Name] = a
		for _, t := range a.Transactions {
			if _, includes := transactionIndex[t.ID]; !includes {
				transactionIndex[t.ID] = true
				db.Transactions = append(db.Transactions, t)
			}
		}
	}
	sort.Sort(Transactions(db.Transactions))
	return db
}

func (db *DoubleBook) RecordTransaction(t Transaction) error {
	if err := db.Accounts[t.FromAccount.Name].Subtract(t.FromAmount); err != nil {
		if !errors.Is(err, NegativeBalanceTransaction) {
			// TODO: rollback
			return err
		}
		// no rollback necessary, reduction was not recorded
	}

	if err := db.Accounts[t.ToAccount.Name].Add(t.FromAmount); err != nil {
		// TODO: check for negative balance transaction, i.e., FromAmount was a negative number
		// TODO: rollback
		return err
	}
	db.Transactions = append(db.Transactions, &t)
	db.UpdatedAt = time.Now()
	return nil
}
