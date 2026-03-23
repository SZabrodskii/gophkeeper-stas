package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/SZabrodskii/gophkeeper-stas/internal/client/api"
)

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an entry",
}

var updateCredentialCmd = &cobra.Command{
	Use:   "credential <id>",
	Short: "Update credential entry",
	Args:  cobra.ExactArgs(1),
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

		resp, err := app.API.UpdateEntry(cmd.Context(), args[0], api.CreateEntryRequest{
			EntryType: "credential",
			Name:      name,
			Metadata:  meta,
			Data:      data,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Updated credential %s at %s\n", resp.ID, resp.UpdatedAt)
		return nil
	},
}

var updateTextCmd = &cobra.Command{
	Use:   "text <id>",
	Short: "Update text entry",
	Args:  cobra.ExactArgs(1),
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

		resp, err := app.API.UpdateEntry(cmd.Context(), args[0], api.CreateEntryRequest{
			EntryType: "text",
			Name:      name,
			Metadata:  meta,
			Data:      data,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Updated text %s at %s\n", resp.ID, resp.UpdatedAt)
		return nil
	},
}

var updateCardCmd = &cobra.Command{
	Use:   "card <id>",
	Short: "Update card entry",
	Args:  cobra.ExactArgs(1),
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

		resp, err := app.API.UpdateEntry(cmd.Context(), args[0], api.CreateEntryRequest{
			EntryType: "card",
			Name:      name,
			Metadata:  meta,
			Data:      data,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Updated card %s at %s\n", resp.ID, resp.UpdatedAt)
		return nil
	},
}

var updateBinaryCmd = &cobra.Command{
	Use:   "binary <id>",
	Short: "Update binary entry",
	Args:  cobra.ExactArgs(1),
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

		resp, err := app.API.UpdateEntry(cmd.Context(), args[0], api.CreateEntryRequest{
			EntryType: "binary",
			Name:      name,
			Metadata:  meta,
			Data:      data,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Updated binary %s at %s\n", resp.ID, resp.UpdatedAt)
		return nil
	},
}

func init() {
	updateCredentialCmd.Flags().String("name", "", "entry name")
	updateCredentialCmd.Flags().StringP("login", "l", "", "login")
	updateCredentialCmd.Flags().StringP("password", "p", "", "password")
	_ = updateCredentialCmd.MarkFlagRequired("name")
	_ = updateCredentialCmd.MarkFlagRequired("login")
	_ = updateCredentialCmd.MarkFlagRequired("password")
	addMetadataFlag(updateCredentialCmd)
	updateTextCmd.Flags().String("name", "", "entry name")
	updateTextCmd.Flags().StringP("content", "c", "", "text content")
	_ = updateTextCmd.MarkFlagRequired("name")
	_ = updateTextCmd.MarkFlagRequired("content")
	addMetadataFlag(updateTextCmd)
	updateCardCmd.Flags().String("name", "", "entry name")
	updateCardCmd.Flags().String("number", "", "card number")
	updateCardCmd.Flags().String("expiry", "", "expiry MM/YY")
	updateCardCmd.Flags().String("holder", "", "holder name")
	updateCardCmd.Flags().String("cvv", "", "CVV")
	_ = updateCardCmd.MarkFlagRequired("name")
	_ = updateCardCmd.MarkFlagRequired("number")
	_ = updateCardCmd.MarkFlagRequired("expiry")
	_ = updateCardCmd.MarkFlagRequired("holder")
	_ = updateCardCmd.MarkFlagRequired("cvv")
	addMetadataFlag(updateCardCmd)
	updateBinaryCmd.Flags().String("name", "", "entry name")
	updateBinaryCmd.Flags().StringP("file", "f", "", "path to file")
	_ = updateBinaryCmd.MarkFlagRequired("name")
	_ = updateBinaryCmd.MarkFlagRequired("file")
	addMetadataFlag(updateBinaryCmd)

	updateCmd.AddCommand(updateCredentialCmd, updateTextCmd, updateCardCmd, updateBinaryCmd)
	rootCmd.AddCommand(updateCmd)
}
