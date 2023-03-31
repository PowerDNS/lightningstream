package commands

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/PowerDNS/lmdb-go/lmdbscan"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"powerdns.com/platform/lightningstream/lmdbenv"
	"powerdns.com/platform/lightningstream/lmdbenv/header"
	"powerdns.com/platform/lightningstream/utils"
)

func init() {
	experimentalCmd.AddCommand(pdnsV5FixDuplicateDomainsCmd)
	pdnsV5FixDuplicateDomainsCmd.Flags().StringP("database", "d", "",
		"Named database to operate on (must me the main database for pdns auth)")
	pdnsV5FixDuplicateDomainsCmd.Flags().Bool("dangerous-do-rename", false,
		"Automatically rename the newer duplicate domain name to fix")
	_ = pdnsV5FixDuplicateDomainsCmd.MarkFlagRequired("database")
}

const pdnsV5FixDuplicateDomainsLong = `
The PowerDNS Auth 4.8 schema version 5 makes is possible to create duplicate
domain entries on different instances, which can cause an error in early version
of Auth. This command allows you to remove those duplicate entries.
`

var pdnsV5FixDuplicateDomainsCmd = &cobra.Command{
	Use:          "pdns-v5-fix-duplicate-domains",
	Short:        "Fix duplicate domain entries for PowerDNS Auth 4.8 with schema version 5",
	Long:         pdnsV5FixDuplicateDomainsLong,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbName, err := cmd.Flags().GetString("database")
		if err != nil {
			return err
		}
		doRename, err := cmd.Flags().GetBool("dangerous-do-rename")
		if err != nil {
			return err
		}
		dbConf, exist := conf.LMDBs[dbName]
		if !exist {
			return fmt.Errorf("no LMDB with name %q configured", dbName)
		}
		env, err := lmdbenv.NewWithOptions(dbConf.Path, dbConf.Options)
		if err != nil {
			return err
		}

		var domainsToDelete []string

		err = env.Update(func(txn *lmdb.Txn) error {
			// Check the schemaversion. We only support this for v5 (auth 4.8)
			dbi, err := txn.OpenDBI("pdns", 0)
			if err != nil {
				if lmdb.IsNotFound(err) {
					return fmt.Errorf("'pdns' DBI not found, this does not look like a main DBI")
				}
				return err
			}
			val, err := txn.Get(dbi, []byte("schemaversion"))
			if err != nil {
				return fmt.Errorf("cannot check schemaversion: %v", err)
			}
			v, err := header.Skip(val)
			if err != nil {
				return fmt.Errorf("schemaversion: %v", err)
			}
			if len(v) != 4 {
				return fmt.Errorf("schemaversion wrong length: %v", v)
			}
			version := binary.BigEndian.Uint32(v)
			if version != 5 {
				return fmt.Errorf("schemaversion incorrect: expected 5, got %d", version)
			}

			// We will fix duplicate domains in the domains_v5_0 DBI
			dbi, err = txn.OpenDBI("domains_v5_0", 0)
			if err != nil {
				if lmdb.IsNotFound(err) {
					return fmt.Errorf("'domains_v5_0' DBI not found, this does not look like a main DBI")
				}
				return err
			}

			displayDomain := func(domain []byte) string {
				p := bytes.Split(domain, []byte{0})
				var labels []string
				for i := len(p) - 1; i >= 0; i-- {
					if len(p[i]) == 0 {
						continue
					}
					labels = append(labels, string(p[i]))
				}
				return strings.Join(labels, ".")
			}

			patchDomain := func(domain []byte, id uint32, flags header.Flags) error {
				// Construct key
				key := make([]byte, 2+len(domain)+4)
				binary.BigEndian.PutUint16(key[:2], uint16(len(domain)))
				off := 2 + len(domain)
				copy(key[2:off], domain)
				binary.BigEndian.PutUint32(key[off:off+4], id)

				// Construct LS header as value
				val := make([]byte, header.MinHeaderSize)
				ts := header.TimestampFromTime(time.Now())
				header.PutBasic(val, ts, header.TxnID(txn.ID()), flags)

				// Put
				logrus.WithFields(logrus.Fields{
					"key":       utils.DisplayASCII(key),
					"val":       utils.DisplayASCII(val),
					"domain":    displayDomain(domain),
					"domain_id": id,
					"flags":     flags,
				}).Warn("PATCHING")
				err := txn.Put(dbi, key, val, 0)
				if err != nil {
					return err
				}
				return nil
			}

			// Scan for duplicate domains
			var prevDomain []byte
			var prevHeader header.Header
			var prevID uint32
			scan := lmdbscan.New(txn, dbi)
			defer scan.Close()
			for scan.Scan() {
				key := scan.Key()
				hdrVal := scan.Val() // only an LS header
				hdr, _, err := header.Parse(hdrVal)
				if err != nil {
					return fmt.Errorf("key: %s: invalid header: %v",
						utils.DisplayASCII(key), err)
				}
				if hdr.Flags.IsDeleted() {
					continue
				}

				// At least a 2 byte domain key length and 4 byte ID
				if len(key) < 6 {
					return fmt.Errorf("too short key: %v", key)
				}
				// Length header + domain + 32-bit ID
				n := int(key[0])<<8 + int(key[1])
				expectedLen := 2 + n + 4
				if len(key) != expectedLen {
					return fmt.Errorf("unexpected key length (expected %d, got %d) for key: %v",
						expectedLen, len(key), key)
				}
				domain := key[2 : 2+n]
				domainDisplay := displayDomain(domain)
				id := binary.BigEndian.Uint32(key[2+n:])
				logrus.WithFields(logrus.Fields{
					"domain":    domainDisplay,
					"domain_id": id,
				}).Debug("Scan")

				if strings.HasSuffix(domainDisplay, ".invalid") && strings.Contains(domainDisplay, ".dup-") {
					domainsToDelete = append(domainsToDelete, domainDisplay)
				}

				if bytes.Equal(prevDomain, domain) {
					// Oldest entry wins
					oldest := prevID
					newest := id
					swapPrev := true
					if hdr.Timestamp < prevHeader.Timestamp {
						oldest = id
						newest = prevID
						swapPrev = false
					}
					logrus.WithFields(logrus.Fields{
						"domain":             domainDisplay,
						"domain_id":          id,
						"header_ts":          hdr.Timestamp.Time(),
						"prev_domain_id":     prevID,
						"prev_header_ts":     prevHeader.Timestamp.Time(),
						"keeping_oldest_id":  oldest,
						"renaming_newest_id": newest,
					}).Error("Duplicate domain entry!")

					if doRename {
						if err := patchDomain(domain, newest, header.FlagDeleted); err != nil {
							return err
						}
						// Note that the domain is reversed
						newDomain := []byte(
							fmt.Sprintf("invalid\x00dup-%d\x00%s", newest, string(domain)),
						)
						if err := patchDomain(newDomain, newest, header.NoFlags); err != nil {
							return err
						}
					} else {
						logrus.Info("Not patching, because --dangerous-do-rename not set")
					}

					if swapPrev {
						// To make sure we compare the next entry with the one that remained untouched
						domain = prevDomain
						hdr = prevHeader
						id = prevID
					}
				}

				prevDomain = domain // this was a copy, safe to keep a reference
				prevHeader = hdr
				prevID = id
			}
			if err := scan.Err(); err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return err
		}

		logrus.WithFields(logrus.Fields{
			//"n_updated": -1,
		}).Info("Done")

		if len(domainsToDelete) > 0 {
			fmt.Printf("\n\nThe following zones need to be removed with `pdnsutil delete-zone ZONE`:\n\n")
			for _, d := range domainsToDelete {
				fmt.Printf("- %s\n", d)
			}
			fmt.Printf("\nNote that these will NOT show up in list-all-zones. Removing the zones " +
				"will also not remove it from this list.\n\n")
		}

		return nil
	},
}
