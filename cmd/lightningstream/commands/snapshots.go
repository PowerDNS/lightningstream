package commands

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/PowerDNS/simpleblob"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	"powerdns.com/platform/lightningstream/lmdbenv/dbiflags"
	"powerdns.com/platform/lightningstream/lmdbenv/header"
	"powerdns.com/platform/lightningstream/snapshot"
	"powerdns.com/platform/lightningstream/utils"
)

func init() {
	rootCmd.AddCommand(snapshotsCmd)

	snapshotsCmd.AddCommand(snapshotsListCmd)
	snapshotsListCmd.Flags().StringP("prefix", "p", "", "Prefix filter")
	snapshotsListCmd.Flags().BoolP("long", "l", false, "Add extra information, like size")
	snapshotsListCmd.Flags().BoolP("time", "t", false, "Sort by snapshot time")

	snapshotsCmd.AddCommand(snapshotsRemoveCmd)

	snapshotsCmd.AddCommand(snapshotsDumpCmd)
	snapshotsDumpCmd.Flags().StringP("format", "f", "debug",
		"Output format, one of: 'debug' (default), 'text' (same)")
	snapshotsDumpCmd.Flags().StringP("dbi", "d", "", "Only output DBI with this exact name")
	snapshotsDumpCmd.Flags().BoolP("local", "l", false,
		"Dump a local file instead of a remote snapshot")

	snapshotsCmd.AddCommand(snapshotsGetCmd)
	snapshotsGetCmd.Flags().StringP("output", "o", "",
		"Output filename, if not the same as the remote name")

	snapshotsCmd.AddCommand(snapshotsPutCmd)
	snapshotsPutCmd.Flags().StringP("name", "n", "",
		"Name to store the snapshot as, if different from the local name")
	snapshotsPutCmd.Flags().Bool("force", false, "Force the use of an invalid snapshot name")
}

var snapshotsCmd = &cobra.Command{
	Use:   "snapshots",
	Short: "Remote snapshot operations (list, dump, remove, etc)",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var snapshotsListCmd = &cobra.Command{
	Use:          "list",
	Short:        "List snapshots",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(rootCtx, time.Minute)
		defer cancel()

		st, err := simpleblob.GetBackend(ctx, conf.Storage.Type, conf.Storage.Options)
		if err != nil {
			return err
		}

		prefix, err := cmd.Flags().GetString("prefix")
		if err != nil {
			return err
		}
		long, err := cmd.Flags().GetBool("long")
		if err != nil {
			return err
		}
		byTime, err := cmd.Flags().GetBool("time")
		if err != nil {
			return err
		}

		list, err := st.List(ctx, prefix)
		if err != nil {
			return err
		}
		if byTime {
			sortByTime(list)
		}

		for _, blob := range list {
			if long {
				fmt.Printf("%12d\t%s\n", blob.Size, blob.Name)
			} else {
				fmt.Printf("%s\n", blob.Name)
			}
		}
		return nil
	},
}

var snapshotsRemoveCmd = &cobra.Command{
	Use:          "remove",
	Short:        "Remove snapshot",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(rootCtx, time.Minute)
		defer cancel()

		st, err := simpleblob.GetBackend(ctx, conf.Storage.Type, conf.Storage.Options)
		if err != nil {
			return err
		}

		return st.Delete(ctx, args[0])
	},
}

var snapshotsDumpCmd = &cobra.Command{
	Use:          "dump",
	Short:        "Dump snapshot contents for debugging",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(rootCtx, time.Minute)
		defer cancel()

		format, err := cmd.Flags().GetString("format")
		if err != nil {
			return err
		}
		if format != "debug" && format != "text" {
			return fmt.Errorf("output format not supported: %s", format)
		}
		dbiName, err := cmd.Flags().GetString("dbi")
		if err != nil {
			return err
		}
		local, err := cmd.Flags().GetBool("local")
		if err != nil {
			return err
		}

		// Load snapshot
		var data []byte
		if local {
			data, err = os.ReadFile(args[0])
			if err != nil {
				return err
			}
		} else {
			st, err := simpleblob.GetBackend(ctx, conf.Storage.Type, conf.Storage.Options)
			if err != nil {
				return err
			}
			data, err = st.Load(ctx, args[0])
			if err != nil {
				return err
			}
		}
		snap, err := snapshot.LoadData(data)
		if err != nil {
			return err
		}

		// Filter DBIs if needed
		if dbiName != "" {
			snap.Databases = lo.Filter(snap.Databases, func(item *snapshot.DBI, index int) bool {
				return item.Name() == dbiName
			})
		}

		// Buffered output speeds things up
		out := bufio.NewWriter(os.Stdout)
		defer out.Flush()
		outf := func(sfmt string, args ...any) {
			_, _ = fmt.Fprintf(out, sfmt, args...)
		}

		switch format {
		case "debug", "text":
			databases := snap.Databases
			snap.Databases = nil
			// FIXME: Ensure this is not printing private fields?
			outf("%+v", snap)

			// Print DBI contents
			now := time.Now()
			for _, dbi := range databases {
				outf("\n### %s (transform=%q, flags=%q)\n\n",
					dbi.Name, dbi.Transform, dbiflags.Flags(dbi.Flags()))
				dbi.ResetCursor()
				for {
					e, err := dbi.Next()
					if err != nil {
						if err != io.EOF {
							return err
						}
						break
					}
					t := header.Timestamp(e.TimestampNano).Time()
					outf("%s  =  %s  (%s, %s ago; flags=%02x)\n",
						utils.DisplayASCII(e.Key),
						utils.DisplayASCII(e.Value),
						t,
						now.Sub(t).Round(time.Second),
						e.Flags,
					)
				}
			}
			return nil
		default:
			panic("unhandled output format: " + format)
		}
	},
}

var snapshotsGetCmd = &cobra.Command{
	Use:          "get",
	Short:        "Download a snapshot",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(rootCtx, time.Minute)
		defer cancel()

		outName, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}
		if outName == "" {
			outName = args[0]
		}

		st, err := simpleblob.GetBackend(ctx, conf.Storage.Type, conf.Storage.Options)
		if err != nil {
			return err
		}
		data, err := st.Load(ctx, args[0])
		if err != nil {
			return err
		}

		return os.WriteFile(outName, data, 0666)
	},
}

var snapshotsPutCmd = &cobra.Command{
	Use:          "put",
	Short:        "Upload a snapshot",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(rootCtx, time.Minute)
		defer cancel()

		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return err
		}
		if name == "" {
			name = filepath.Base(args[0])
		}
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			return err
		}

		if _, err = snapshot.ParseName(name); err != nil {
			if !force {
				return fmt.Errorf(
					"invalid snapshot name (use -n to specify a different one, or "+
						"--force to skip this check): %v", err)
			}
			logrus.WithError(err).Warn("Invalid snapshot name forced")
		}

		st, err := simpleblob.GetBackend(ctx, conf.Storage.Type, conf.Storage.Options)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		return st.Store(ctx, name, data)
	},
}

func sortByTime(list simpleblob.BlobList) {
	slices.SortFunc(list, func(a, b simpleblob.Blob) bool {
		na, errA := snapshot.ParseName(a.Name)
		nb, errB := snapshot.ParseName(b.Name)
		// Invalid names are sorted by name
		if errA != nil && errB != nil {
			return a.Name < b.Name
		}
		// Invalid names come before valid names
		if errA != nil {
			return true
		}
		if errB != nil {
			return false
		}
		// Valid names are sorted by timestamp
		return na.Timestamp.Before(nb.Timestamp)
	})
}
