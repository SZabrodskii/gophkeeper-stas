package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/SZabrodskii/gophkeeper-stas/internal/client/api"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new entry",
}

var createCredentialCmd = &cobra.Command{
	Use:   "credential",
	Short: "Create credential entry",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		login, _ := cmd.Flags().GetString("login")
		password, _ := cmd.Flags().GetString("password")
		meta, err := parseMetadata(cmd)
		if err != nil {
			return err
		}

		data, _ := json.Marshal(map[string]string{
			"login":    login,
			"password": password,
		})

		resp, err := app.API.CreateEntry(cmd.Context(), api.CreateEntryRequest{
			EntryType: "credential",
			Name:      name,
			Metadata:  meta,
			Data:      data,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Created credential %s\n", resp.ID)
		return nil
	},
}

var createTextCmd = &cobra.Command{
	Use:   "text",
	Short: "Create text entry",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		content, _ := cmd.Flags().GetString("content")
		meta, err := parseMetadata(cmd)
		if err != nil {
			return err
		}

		data, _ := json.Marshal(map[string]string{
			"content": content,
		})

		resp, err := app.API.CreateEntry(cmd.Context(), api.CreateEntryRequest{
			EntryType: "text",
			Name:      name,
			Metadata:  meta,
			Data:      data,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Created text %s\n", resp.ID)
		return nil
	},
}

var createCardCmd = &cobra.Command{
	Use:   "card",
	Short: "Create card entry",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		number, _ := cmd.Flags().GetString("number")
		expiry, _ := cmd.Flags().GetString("expiry")
		holder, _ := cmd.Flags().GetString("holder")
		cvv, _ := cmd.Flags().GetString("cvv")
		meta, err := parseMetadata(cmd)
		if err != nil {
			return err
		}

		data, _ := json.Marshal(map[string]string{
			"number":      number,
			"expiry":      expiry,
			"holder_name": holder,
			"cvv":         cvv,
		})

		resp, err := app.API.CreateEntry(cmd.Context(), api.CreateEntryRequest{
			EntryType: "card",
			Name:      name,
			Metadata:  meta,
			Data:      data,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Created card %s\n", resp.ID)
		return nil
	},
}

var createBinaryCmd = &cobra.Command{
	Use:   "binary",
	Short: "Create binary entry",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		filePath, _ := cmd.Flags().GetString("file")
		meta, err := parseMetadata(cmd)
		if err != nil {
			return err
		}

		fileData, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}

		data, _ := json.Marshal(map[string]string{
			"data":              base64.StdEncoding.EncodeToString(fileData),
			"original_filename": filePath,
		})

		resp, err := app.API.CreateEntry(cmd.Context(), api.CreateEntryRequest{
			EntryType: "binary",
			Name:      name,
			Metadata:  meta,
			Data:      data,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Created binary %s\n", resp.ID)
		return nil
	},
}

func init() {
	createCredentialCmd.Flags().String("name", "", "entry name")
	createCredentialCmd.Flags().StringP("login", "l", "", "login")
	createCredentialCmd.Flags().StringP("password", "p", "", "password")
	_ = createCredentialCmd.MarkFlagRequired("name")
	_ = createCredentialCmd.MarkFlagRequired("login")
	_ = createCredentialCmd.MarkFlagRequired("password")
	addMetadataFlag(createCredentialCmd)
	createTextCmd.Flags().String("name", "", "entry name")
	createTextCmd.Flags().StringP("content", "c", "", "text content")
	_ = createTextCmd.MarkFlagRequired("name")
	_ = createTextCmd.MarkFlagRequired("content")
	addMetadataFlag(createTextCmd)
	createCardCmd.Flags().String("name", "", "entry name")
	createCardCmd.Flags().String("number", "", "card number")
	createCardCmd.Flags().String("expiry", "", "expiry MM/YY")
	createCardCmd.Flags().String("holder", "", "holder name")
	createCardCmd.Flags().String("cvv", "", "CVV")
	_ = createCardCmd.MarkFlagRequired("name")
	_ = createCardCmd.MarkFlagRequired("number")
	_ = createCardCmd.MarkFlagRequired("expiry")
	_ = createCardCmd.MarkFlagRequired("holder")
	_ = createCardCmd.MarkFlagRequired("cvv")
	addMetadataFlag(createCardCmd)
	createBinaryCmd.Flags().String("name", "", "entry name")
	createBinaryCmd.Flags().StringP("file", "f", "", "path to file")
	_ = createBinaryCmd.MarkFlagRequired("name")
	_ = createBinaryCmd.MarkFlagRequired("file")
	addMetadataFlag(createBinaryCmd)

	createCmd.AddCommand(createCredentialCmd, createTextCmd, createCardCmd, createBinaryCmd)
	rootCmd.AddCommand(createCmd)
}
