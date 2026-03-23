package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func parseMetadata(cmd *cobra.Command) (*json.RawMessage, error) {
	vals, _ := cmd.Flags().GetStringArray("metadata")
	if len(vals) == 0 {
		return nil, nil
	}

	m := make(map[string]string, len(vals))
	for _, v := range vals {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid metadata format %q, expected key=value", v)
		}
		m[parts[0]] = parts[1]
	}

	raw, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}
	msg := json.RawMessage(raw)
	return &msg, nil
}

func addMetadataFlag(cmd *cobra.Command) {
	cmd.Flags().StringArray("metadata", nil, "metadata key=value (repeatable)")
}
