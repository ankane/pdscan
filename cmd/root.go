package cmd

import (
	"fmt"
	"os"

	"github.com/ankane/pdscan/internal"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "pdscan [connection-uri]",
		Short:        "Scan your data stores for unencrypted personal data (PII)",
		Long:         "Scan your data stores for unencrypted personal data (PII)",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			showData, err := cmd.Flags().GetBool("show-data")
			if err != nil {
				return err
			}

			showAll, err := cmd.Flags().GetBool("show-all")
			if err != nil {
				return err
			}

			limit, err := cmd.Flags().GetInt("sample-size")
			if err != nil {
				return err
			}
			if limit < 1 {
				return fmt.Errorf("sample-size must be positive")
			}

			processes, err := cmd.Flags().GetInt("processes")
			if err != nil {
				return err
			}

			only, err := cmd.Flags().GetString("only")
			if err != nil {
				return err
			}

			except, err := cmd.Flags().GetString("except")
			if err != nil {
				return err
			}

			minCount, err := cmd.Flags().GetInt("min-count")
			if err != nil {
				return err
			}
			if minCount < 1 {
				return fmt.Errorf("min-count must be positive")
			}

			pattern, err := cmd.Flags().GetString("pattern")
			if err != nil {
				return err
			}

			if len(args) == 0 {
				cmd.Help()
				os.Exit(1)
			}
			return internal.Main(args[0], showData, showAll, limit, processes, only, except, minCount, pattern)
		},
	}
	cmd.PersistentFlags().Bool("show-data", false, "Show data")
	cmd.PersistentFlags().Bool("show-all", false, "Show all matches")
	cmd.PersistentFlags().Int("sample-size", 10000, "Sample size")
	cmd.PersistentFlags().Int("processes", 1, "Processes")
	cmd.PersistentFlags().String("only", "", "Only certain rules")
	cmd.PersistentFlags().String("except", "", "Except certain rules")
	cmd.PersistentFlags().Int("min-count", 1, "Minimum rows/documents/lines for a match (experimental)")
	cmd.PersistentFlags().String("pattern", "", "Custom pattern")
	return cmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cmd := NewRootCmd()
	cmd.Execute()
}
