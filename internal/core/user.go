package core

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/goslu/sid"
	"github.com/SimonSchneider/pefigo/internal/pdb"
	"net/http"
	"time"
)

type User pdb.User

func EmptyUser() User {
	return User{}
}

func GetUser(ctx context.Context, db *sql.DB, id string) (User, error) {
	u, err := pdb.New(db).GetUser(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("failed to get user: %w", err)
	}
	return User(u), nil
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
	return User(u), nil
}

func DeleteUser(ctx context.Context, db *sql.DB, id string) error {
	_, err := pdb.New(db).DeleteUser(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

func userListFromDB(dbUsers []pdb.User) []User {
	users := make([]User, len(dbUsers))
	for i := range dbUsers {
		users[i] = User(dbUsers[i])
	}
	return users
}

func ListUsers(ctx context.Context, db *sql.DB) ([]User, error) {
	us, err := pdb.New(db).ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	return userListFromDB(us), nil
}
