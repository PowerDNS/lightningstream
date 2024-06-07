package commands

import (
	"encoding/binary"
	"fmt"

	"github.com/PowerDNS/lightningstream/lmdbenv"
	"github.com/PowerDNS/lightningstream/lmdbenv/header"
	"github.com/PowerDNS/lightningstream/utils"
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/PowerDNS/lmdb-go/lmdbscan"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(experimentalCmd)

	experimentalCmd.AddCommand(migrateTimestampsCmd)
	migrateTimestampsCmd.Flags().StringP("database", "d", "",
		"Named database to operate on")
	migrateTimestampsCmd.Flags().String("src-dbi", "", "Source DBI")
	migrateTimestampsCmd.Flags().String("dst-dbi", "", "Destination DBI")
	migrateTimestampsCmd.Flags().Bool("add-delete-entries", false,
		"Create 24-byte deletion headers for entries marked as deleted in the "+
			"source database.")
	migrateTimestampsCmd.Flags().Bool("ignore-src-dbi-not-present", false,
		"Do not exit with an error if the source DBI does not exist, ignore it.")
	_ = migrateTimestampsCmd.MarkFlagRequired("database")
	_ = migrateTimestampsCmd.MarkFlagRequired("src-dbi")
	_ = migrateTimestampsCmd.MarkFlagRequired("dst-dbi")
}

var experimentalCmd = &cobra.Command{
	Use:   "experimental",
	Short: "Experimental commands, may disappear or change in any future version",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

const migrateTimestampsLong = `
Migrate timestamps from one DBI to a target DBI by key.

Requirements: 

- The keys must match between source and destination without any transformation.
- The DBIs must not use special DBI flags.
- The destination DBI must use the new 24+ byte native headers.
- The source DBI must use the new header, or the old (v0.2.0) 8 byte
  timestamp-only headers.

This command can be used for both native DBIs, and for shadow DBIs.
If the target timestamp is higher than the source, it will not be updated.

The actual values are not compared, it blindly copies the timestamps. This is
only useful during a migration where no new data is written.

This command will abort the transaction and exit with an error if any value
shorter than expected is encountered.

Example to migrate shard records when moving from PowerDNS Auth 4.7 to 4.8:

    ... migrate-timestamps --add-delete-entries --database shard --src-dbi _sync_shadow_records --dst-dbi records_v5

`

var migrateTimestampsCmd = &cobra.Command{
	Use:          "migrate-timestamps",
	Short:        "Migrate timestamps from one DBI to another DBI",
	Long:         migrateTimestampsLong,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbName, err := cmd.Flags().GetString("database")
		if err != nil {
			return err
		}
		dbConf, exist := conf.LMDBs[dbName]
		if !exist {
			return fmt.Errorf("no LMDB with name %q configured", dbName)
		}
		srcDBIName, err := cmd.Flags().GetString("src-dbi")
		if err != nil {
			return err
		}
		dstDBIName, err := cmd.Flags().GetString("dst-dbi")
		if err != nil {
			return err
		}
		if srcDBIName == dstDBIName {
			return fmt.Errorf("source and destination DBI cannot be the same")
		}
		ignoreNotPresent, err := cmd.Flags().GetBool("ignore-src-dbi-not-present")
		if err != nil {
			return err
		}
		addDeleted, err := cmd.Flags().GetBool("add-delete-entries")
		if err != nil {
			return err
		}

		env, err := lmdbenv.NewWithOptions(dbConf.Path, dbConf.Options)
		if err != nil {
			return err
		}

		// Stats
		var (
			nUpdated      int64
			nAddedDeletes int64
		)

		// Actual migration
		err = env.Update(func(txn *lmdb.Txn) error {
			txnID := header.TxnID(txn.ID())

			srcDBI, err := txn.OpenDBI(srcDBIName, 0)
			if err != nil {
				if lmdb.IsNotFound(err) && ignoreNotPresent {
					logrus.Warn("Source DBI not found, no migration performed")
					return nil
				}
				return err
			}
			dstDBI, err := txn.OpenDBI(dstDBIName, 0)
			if err != nil {
				return err
			}

			// Update the timestamps in dst from src
			dstScan := lmdbscan.New(txn, dstDBI)
			defer dstScan.Close()
			for dstScan.Scan() {
				key := dstScan.Key()

				// Destination value
				dstVal := dstScan.Val()
				if len(dstVal) < header.MinHeaderSize {
					return fmt.Errorf("dst: value too short for header for key %s: %w",
						utils.DisplayASCII(key), err)
				}
				dstTS, err := header.ParseTimestamp(dstVal)
				if err != nil {
					return fmt.Errorf("dst: value too short for key %s: %w",
						utils.DisplayASCII(key), err)
				}

				// Source value
				srcVal, err := txn.Get(srcDBI, key)
				if err != nil {
					if lmdb.IsNotFound(err) {
						continue
					}
					return err
				}
				srcTS, err := header.ParseTimestamp(srcVal)
				if err != nil {
					return fmt.Errorf("src: value too short for key %s: %w",
						utils.DisplayASCII(key), err)
				}

				// Do not touch if newer
				if dstTS >= srcTS {
					continue
				}

				// Update the timestamp and txnID
				nUpdated++
				copy(dstVal, srcVal[:8])
				binary.BigEndian.PutUint64(dstVal[8:16], uint64(txnID))
				if err := txn.Put(dstDBI, key, dstVal, 0); err != nil {
					return err
				}
			}
			if err := dstScan.Err(); err != nil {
				return err
			}

			if !addDeleted {
				return nil // skip the rest
			}
			// Any key present in src and missing in dst is marked as deleted
			// with the timestamp in src, if the src entry is marked as deleted.
			srcScan := lmdbscan.New(txn, srcDBI)
			defer srcScan.Close()
			for srcScan.Scan() {
				key := srcScan.Key()

				// Source value
				srcVal := srcScan.Val()
				srcTS, err := header.ParseTimestamp(srcVal)
				if err != nil {
					return fmt.Errorf("src: value too short for key %s: %w",
						utils.DisplayASCII(key), err)
				}
				srcIsDeleted := false
				if len(srcVal) == 8 {
					srcIsDeleted = true // old style (0.2.0) DBI
				} else {
					h, appVal, err := header.Parse(srcVal)
					if err != nil {
						return fmt.Errorf("src: invalid header for key %s: %w",
							utils.DisplayASCII(key), err)
					}
					srcIsDeleted = h.Flags.IsDeleted()
					if srcIsDeleted && len(appVal) > 0 {
						return fmt.Errorf("src: deleted but non-empty app value: key %s",
							utils.DisplayASCII(key))
					}
				}
				if !srcIsDeleted {
					continue
				}

				// Only continue if the key does not exist in the destination
				_, err = txn.Get(dstDBI, key)
				if err != nil && !lmdb.IsNotFound(err) {
					return err
				}
				if err == nil {
					continue
				}

				// Create a deletion marker
				nAddedDeletes++
				delVal := make([]byte, header.MinHeaderSize)
				header.PutBasic(delVal, srcTS, txnID, header.FlagDeleted)
				if err := txn.Put(dstDBI, key, delVal, 0); err != nil {
					return err
				}
			}
			if err := srcScan.Err(); err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return err
		}

		logrus.WithFields(logrus.Fields{
			"n_updated":       nUpdated,
			"n_added_deletes": nAddedDeletes,
		}).Info("Done")

		return nil
	},
}
