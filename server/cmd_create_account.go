package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/KyAnhVo/brainlog-website/server/internal/account"
	"github.com/KyAnhVo/brainlog-website/server/internal/apitime"
	"github.com/KyAnhVo/brainlog-website/server/internal/config"
	"github.com/KyAnhVo/brainlog-website/server/internal/store"
)

// createAccount implements:
//
//	echo 'password' | APP_ENV=production go run . create-account -name "…" -email … [-role admin|user]
//
// It is the way to create the first admin on an empty database (the createAccount endpoint
// needs an admin session). The password is read from the first line of stdin, never from a
// flag, so it does not end up in shell history or the process list.
func createAccount(args []string) error {
	fs := flag.NewFlagSet("create-account", flag.ContinueOnError)
	name := fs.String("name", "", "display name (required)")
	email := fs.String("email", "", "login email (required)")
	role := fs.String("role", "admin", `"admin" or "user"`)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected arguments: %v (the password is read from stdin)", fs.Args())
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Store != config.StorePostgres {
		return errors.New("create-account needs a database (set DB_ADDR, e.g. APP_ENV=production); the memory store would lose the account on exit")
	}

	if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
		fmt.Fprint(os.Stderr, "Password (input is visible; prefer piping it in): ")
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return errors.New("reading password from stdin: nothing received")
	}
	pw := strings.TrimRight(line, "\r\n")

	in, msg := account.Validate(account.Input{Email: *email, Password: pw, Name: *name, Role: *role})
	if msg != "" {
		return errors.New(msg)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	st, closeStore, err := openStore(ctx, cfg)
	defer closeStore()
	if err != nil {
		return err
	}

	acc := account.New(in, apitime.Normalize(time.Now()))
	if err := st.CreateAccount(ctx, acc); errors.Is(err, store.ErrConflict) {
		return fmt.Errorf("an account with email %s already exists", in.Email)
	} else if err != nil {
		return err
	}
	fmt.Printf("created %s account %q <%s> id=%s\n", acc.Role, acc.Name, acc.Email, acc.ID)
	return nil
}
