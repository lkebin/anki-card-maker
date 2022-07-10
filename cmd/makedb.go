/*
Copyright © 2022 Kebin Liu <lkebin@gmail.com>

*/
package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/antchfx/xmlquery"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

// makedbCmd represents the makedb command
var makedbCmd = &cobra.Command{
	Use:   "makedb dictinary_path",
	Short: "Create sqlite database from dictinary",
	Run: func(cmd *cobra.Command, args []string) {
		dictPath := args[0]

		dictDir := filepath.Dir(dictPath)
		dictName := filepath.Base(dictPath)

		dbName := dictName[:len(dictName)-len(filepath.Ext(dictPath))] + ".db"
		dbPath := filepath.Join(dictDir, dbName)

		if _, err := os.Stat(dbPath); err == nil {
			// exists
			if err := os.Remove(dbPath); err != nil {
				log.Fatal(err)
			}
		}

		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			log.Fatal(err)
		}

		createTable := `
create table if not exists dictionary(id integer not null primary key, word varchar(64) not null, definition text not null);
create index if not exists "dictionary_word" on dictionary (word);
`

		if _, err := db.Exec(createTable); err != nil {
			log.Fatal(err)
		}

		d, err := os.Open(dictPath)
		if err != nil {
			log.Fatal(err)
		}
		defer d.Close()

		doc, err := xmlquery.Parse(d)
		if err != nil {
			log.Fatal(err)
		}

		entries := xmlquery.Find(doc, "//d:entry")
		for k, v := range entries {
			xmlquery.AddAttr(v, "xml:space", "preserve")
			title := v.SelectAttr("d:title")
			definition := v.OutputXML(true)

			if _, err := db.Exec(`insert into dictionary(id, word, definition) values(?, ?, ?)`, k+1, title, definition); err != nil {
				log.Fatal(err)
			}
		}

		fmt.Println("done")
	},
}

func init() {
	rootCmd.AddCommand(makedbCmd)
}
