package commands

import "fmt"

func requireAuth() error {
	token, err := app.Keyring.Get()
	if err != nil {
		return fmt.Errorf("not logged in (use 'register' or 'login' first): %w", err)
	}
	app.API.SetToken(token)
	return nil
}
