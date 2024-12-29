package pefi

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/goslu/sid"
	"github.com/SimonSchneider/goslu/sqlu"
	"github.com/SimonSchneider/pefigo/internal/pdb"
	"net/http"
	"time"
)

type AccountInput struct {
	ID      string
	Name    string
	OwnerID string
	Type    string
}

func (a *AccountInput) FromForm(r *http.Request) error {
	a.ID = r.FormValue("id")
	a.Name = r.FormValue("name")
	a.Type = r.FormValue("type")
	a.OwnerID = r.FormValue("owner_id")
	return nil
}

type Account struct {
	ID        string
	Name      string
	Type      string
	OwnerID   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func accountFromDB(a pdb.Account) Account {
	return Account{
		ID:        a.ID,
		Name:      a.Name,
		Type:      a.Type,
		OwnerID:   a.OwnerID.String,
		CreatedAt: time.UnixMilli(a.CreatedAt),
		UpdatedAt: time.UnixMilli(a.UpdatedAt),
	}
}

func GetAccount(ctx context.Context, db *sql.DB, id string) (Account, error) {
	acc, err := pdb.New(db).GetAccount(ctx, id)
	if err != nil {
		return Account{}, fmt.Errorf("failed to get account: %w", err)
	}
	return accountFromDB(acc), nil
}

var accountTypes = []string{"personal", "shared"}

type AccountModal struct {
	Account Account

	AccountTypes []string
	Users        *UserList
}

func NewAccountModal(ctx context.Context, db *sql.DB, account Account) (*AccountModal, error) {
	users, err := ListUsers(ctx, db)
	if err != nil {
		return nil, err
	}
	return &AccountModal{
		Account:      account,
		AccountTypes: accountTypes,
		Users:        users,
	}, nil
}

func GetAccountModal(ctx context.Context, db *sql.DB, accountId string) (*AccountModal, error) {
	acc, err := GetAccount(ctx, db, accountId)
	if err != nil {
		return nil, err
	}
	return NewAccountModal(ctx, db, acc)
}

func NewAccount(ctx context.Context, db *sql.DB) (*AccountModal, error) {
	return NewAccountModal(ctx, db, Account{})
}

func UpsertAccount(ctx context.Context, db *sql.DB, inp AccountInput) (Account, error) {
	var (
		q   = pdb.New(db)
		acc pdb.Account
		err error
	)
	if inp.Type == "shared" && inp.OwnerID != "" {
		return Account{}, fmt.Errorf("shared account can't have owner")
	} else if inp.Type == "personal" && inp.OwnerID == "" {
		return Account{}, fmt.Errorf("personal account must have owner")
	}
	if inp.ID != "" {
		acc, err = q.UpdateAccount(ctx, pdb.UpdateAccountParams{
			ID:        inp.ID,
			Name:      inp.Name,
			Type:      inp.Type,
			OwnerID:   sqlu.NullString(inp.OwnerID),
			UpdatedAt: time.Now().UnixMilli(),
		})
	} else {
		acc, err = q.CreateAccount(ctx, pdb.CreateAccountParams{
			ID:        sid.MustNewString(15),
			Name:      inp.Name,
			Type:      inp.Type,
			OwnerID:   sqlu.NullString(inp.OwnerID),
			CreatedAt: time.Now().UnixMilli(),
			UpdatedAt: time.Now().UnixMilli(),
		})
	}
	if err != nil {
		return Account{}, fmt.Errorf("failed to upsert account: %w", err)
	}
	return accountFromDB(acc), nil
}

func DeleteAccount(ctx context.Context, db *sql.DB, id string) error {
	_, err := pdb.New(db).DeleteAccount(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}
	return nil
}

type AccountsList struct {
	Accounts []Account
}

func accountsListFromDB(dbAccs []pdb.Account) *AccountsList {
	accs := make([]Account, len(dbAccs))
	for i := range dbAccs {
		accs[i] = accountFromDB(dbAccs[i])
	}
	return &AccountsList{accs}
}

func ListAccounts(ctx context.Context, db *sql.DB) (*AccountsList, error) {
	accs, err := pdb.New(db).ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	return accountsListFromDB(accs), nil
}

type Index struct {
	Accounts *AccountsList
	Users    *UserList
}

func IndexPage(ctx context.Context, db *sql.DB) (*Index, error) {
	accs, err := ListAccounts(ctx, db)
	if err != nil {
		return nil, err
	}
	users, err := ListUsers(ctx, db)
	if err != nil {
		return nil, err
	}
	return &Index{Accounts: accs, Users: users}, nil
}

type User struct {
	ID   string
	Name string
}

func EmptyUser() User {
	return User{}
}

func userFromDB(u pdb.User) User {
	return User{ID: u.ID, Name: u.Name}
}

func GetUser(ctx context.Context, db *sql.DB, id string) (User, error) {
	u, err := pdb.New(db).GetUser(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("failed to get user: %w", err)
	}
	return userFromDB(u), nil
}

type UserInput struct {
	ID   string
	Name string
}

func (u *UserInput) FromForm(r *http.Request) error {
	u.ID = r.FormValue("id")
	u.Name = r.FormValue("name")
	return nil
}

func UpsertUser(ctx context.Context, db *sql.DB, inp UserInput) (User, error) {
	var (
		q   = pdb.New(db)
		u   pdb.User
		err error
	)
	if inp.ID != "" {
		u, err = q.UpdateUser(ctx, pdb.UpdateUserParams{
			ID:        inp.ID,
			Name:      inp.Name,
			UpdatedAt: time.Now().UnixMilli(),
		})
	} else {
		u, err = q.CreateUser(ctx, pdb.CreateUserParams{
			ID:        sid.MustNewString(15),
			Name:      inp.Name,
			CreatedAt: time.Now().UnixMilli(),
			UpdatedAt: time.Now().UnixMilli(),
		})
	}
	if err != nil {
		return User{}, fmt.Errorf("failed to upsert user: %w", err)
	}
	return userFromDB(u), nil
}

func DeleteUser(ctx context.Context, db *sql.DB, id string) error {
	_, err := pdb.New(db).DeleteUser(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

type UserList struct {
	Users []User
}

func userListFromDB(dbUsers []pdb.User) *UserList {
	users := make([]User, len(dbUsers))
	for i := range dbUsers {
		users[i] = userFromDB(dbUsers[i])
	}
	return &UserList{Users: users}
}

func ListUsers(ctx context.Context, db *sql.DB) (*UserList, error) {
	us, err := pdb.New(db).ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	return userListFromDB(us), nil
}
