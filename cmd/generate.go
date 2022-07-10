/*
Copyright © 2022 Kebin Liu <lkebin@gmail.com>

*/
package cmd

import (
	"anki-card-maker/tts"
	"bufio"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var errNotFound = errors.New("word not found")

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Commands for generate",
}

var generateSoundCmd = &cobra.Command{
	Use:   "sound",
	Short: "Generate sound",
	Run: func(cmd *cobra.Command, args []string) {
		wordsPath, err := cmd.Flags().GetString("word")
		if err != nil {
			log.Fatal(err)
		}

		wordsLang, err := cmd.Flags().GetString("language")
		if err != nil {
			log.Fatal(err)
		}

		prefix, err := cmd.Flags().GetString("prefix")
		if err != nil {
			log.Fatal(err)
		}

		output, err := cmd.Flags().GetString("output")
		if err != nil {
			log.Fatal(err)
		}

		key, err := cmd.Flags().GetString("key")
		if err != nil {
			log.Fatal(err)
		}

		region, err := cmd.Flags().GetString("region")
		if err != nil {
			log.Fatal(err)
		}

		wf, err := os.Open(wordsPath)
		if err != nil {
			log.Fatal(err)
		}
		defer wf.Close()

		scanner := bufio.NewScanner(wf)
		ttser := tts.New(key, region)

		re, replace := stripRegexForLanguage(wordsLang)

		for scanner.Scan() {
			words := strings.Split(scanner.Text(), ",")
			word := strings.TrimSpace(words[0])
			sw := word
			if re != nil {
				sw = re.ReplaceAllString(word, replace)
			}

			if word != sw {
				log.Println(word, " => ", sw)
			} else {
				log.Println(word)
			}

			sound := filepath.Join(output, fmt.Sprintf("%s%s.mp3", prefix, word))
			if isFileExists(sound) {
				// Skip
				continue
			}

			buf, err := ttser.TTS(sw, tts.Lang(wordsLang))
			if err != nil {
				log.Fatalf("TTS error: %v", err)
			}

			if err := writeToFile(filepath.Join(output, fmt.Sprintf("%s.mp3", word)), buf); err != nil {
				log.Fatalf("write sound to file error: %v", err)
			}

			time.Sleep(100 * time.Millisecond)
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}

		fmt.Println("done")
	},
}

var generateDefinitionCmd = &cobra.Command{
	Use:   "definition",
	Short: "Generate definitions",
	Run: func(cmd *cobra.Command, args []string) {
		wordsPath, err := cmd.Flags().GetString("word")
		if err != nil {
			log.Fatal(err)
		}

		output, err := cmd.Flags().GetString("output")
		if err != nil {
			log.Fatal(err)
		}

		dbFile, err := cmd.Flags().GetString("db")
		if err != nil {
			log.Fatal(err)
		}

		db, err := sql.Open("sqlite3", dbFile)
		if err != nil {
			log.Fatal(err)
		}

		wf, err := os.Open(wordsPath)
		if err != nil {
			log.Fatal(err)
		}
		defer wf.Close()

		scanner := bufio.NewScanner(wf)

		for scanner.Scan() {
			words := strings.Split(scanner.Text(), ",")
			word := strings.TrimSpace(words[0])

			log.Println(word)

			definition := filepath.Join(output, fmt.Sprintf("%s.txt", word))
			if isFileExists(definition) {
				// Skip
				continue
			}

			var (
				definitions []string
			)

			for _, v := range words {
				definitions, err = queryDefinition(db, strings.TrimSpace(v))
				if err != nil {
					if errors.Is(err, errNotFound) {
						continue
					}
					log.Fatal(err)
				}

				if definitions != nil {
					break
				}
			}

			if definitions == nil {
				log.Printf("words [%s] not found\n", word)
				continue
			}

			buf := []byte(strings.Join(definitions, "\n"))

			if err := writeToFile(filepath.Join(output, fmt.Sprintf("%s.txt", word)), buf); err != nil {
				log.Fatalf("write definition to file error: %v", err)
			}
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}

		fmt.Println("done")
	},
}

var generateAnkiCmd = &cobra.Command{
	Use:   "anki",
	Short: "Generate Anki card",
	Run: func(cmd *cobra.Command, args []string) {
		output, err := cmd.Flags().GetString("output")
		if err != nil {
			log.Fatal(err)
		}

		definitionDir, err := cmd.Flags().GetString("definition")
		if err != nil {
			log.Fatal(err)
		}

		prefix, err := cmd.Flags().GetString("prefix")
		if err != nil {
			log.Fatal(err)
		}

		f, err := os.Create(filepath.Join(output, "anki.txt"))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		df, err := os.ReadDir(definitionDir)
		if err != nil {
			log.Fatal(err)
		}

		cf := csv.NewWriter(f)
		cf.Comma = '\t'
		defer cf.Flush()

		for _, v := range df {
			fi, err := v.Info()
			if err != nil {
				log.Fatal(err)
			}

			fname := fi.Name()
			word := fname[:len(fname)-4]

			buf, err := os.ReadFile(filepath.Join(definitionDir, fi.Name()))
			if err != nil {
				log.Fatal(err)
			}

			data := [5]string{
				word, //Words
				fmt.Sprintf("[sound:%s%s.mp3]", prefix, word), // Sound
				// Definition 0,
				// Definition 1,
				// Definition 2,
			}

			rr := ankiFieldLimitSplit(buf)

			bi := 2
			for kk, vv := range rr {
				data[bi+kk] = vv
			}

			cf.Write(data[:])
		}

		fmt.Println("done")
	},
}

func ankiFieldLimitSplit(buf []byte) (s []string) {
	runes := []rune(string(buf))

	length := len(runes)

	var section strings.Builder
	for i := 0; i < length; i++ {
		if section.Len() < 131071 {
			section.WriteRune(runes[i])
		} else {
			s = append(s, section.String())
			section.Reset()
		}
	}

	if section.Len() > 0 {
		s = append(s, section.String())
		section.Reset()
	}

	return
}

func isFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func writeToFile(path string, buf []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(buf)
	return err
}

func queryDefinition(db *sql.DB, word string) (definitions []string, err error) {
	rows, err := db.Query(`select definition from dict where word = ? collate nocase;`, word)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}

		definitions = append(definitions, d)
	}

	if definitions == nil {
		err = errNotFound
		return
	}

	return
}

func init() {
	rootCmd.AddCommand(generateCmd)

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	generateCmd.PersistentFlags().String("word", "./words.txt", "A words list")
	generateCmd.PersistentFlags().String("db", "./dict.db", "A sqlite database file")
	generateCmd.PersistentFlags().String("language", "zh-CN", "Language of word")
	generateCmd.PersistentFlags().String("output", "./", "Output directory of resources")
	generateCmd.PersistentFlags().String("prefix", "", "Prefix for media file name")

	generateAnkiCmd.Flags().String("definition", "./definition", "Path to definition directory")
	generateSoundCmd.Flags().String("key", "", "Key of Azure TTS service")
	generateSoundCmd.MarkFlagRequired("key")
	generateSoundCmd.Flags().String("region", "southeastasia", "Key of Azure TTS service")

	generateCmd.AddCommand(generateDefinitionCmd)
	generateCmd.AddCommand(generateSoundCmd)
	generateCmd.AddCommand(generateAnkiCmd)
}

func stripRegexForLanguage(lang string) (*regexp.Regexp, string) {
	switch lang {
	case "zh-CN":
		return regexp.MustCompile("(（.*?）)"), ""
	case "en-US":
		return nil, ""
	}

	return nil, ""
}
