package commands

import (
	"bytes"
	"io"
	"os"
	"regexp"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func init() {
	rootCmd.AddCommand(docsCmd)
}

var docsCmd = &cobra.Command{
	Use:          "docs",
	Short:        "Generate markdown documentation for all commands to stdout",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return genDocs(rootCmd, os.Stdout)
	},
}

func genDocs(cmd *cobra.Command, w io.Writer) error {
	if cmd.Name() == "completion" {
		return nil
	}
	b := bytes.NewBuffer(nil)
	if err := doc.GenMarkdown(cmd, b); err != nil {
		return err
	}
	re := regexp.MustCompile(`(?s)### (SEE ALSO|Options inherited from parent commands).*`)
	t := re.ReplaceAll(b.Bytes(), nil)
	if _, err := w.Write(t); err != nil {
		return err
	}

	for _, c := range cmd.Commands() {
		//if _, err := fmt.Fprintf(w, "\n\n---\n\n"); err != nil {
		//	return err
		//}
		if err := genDocs(c, w); err != nil {
			return err
		}
	}
	return nil
}
