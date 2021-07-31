package main

// https://github.com/crahles/bolt-cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/boltdb/bolt"
	"github.com/peterh/liner"
)

const goodbyeMsg = "Toto, I have a feeling we're not in Kansas anymore."

var (
	verbose    = kingpin.Flag("verbose", "Enable verbose mode.").Short('v').Bool()
	dbFile     = kingpin.Flag("db", "Path to db file.").Short('d').Required().String()
	bucketName = kingpin.Flag("bucket", "Bucket name.").Short('b').String()

	historyFile = kingpin.Flag("history-file", "History file.").Short('h').Default("/tmp/.liner_history").String()
	commands    = []string{"keys ", "get ", "bucket "}

	db *bolt.DB

	// Version will be set during compile to reflect compiled git sha state
	Version = "dev"
)

func main() {
	kingpin.Version("Version: " + Version)
	kingpin.Parse()

	var err error
	if _, err = os.Stat(*dbFile); os.IsNotExist(err) {
		fmt.Printf("Couldn't open BoltDB file: %s\n", *dbFile)
		os.Exit(1)
	}
	db, err = bolt.Open(*dbFile, 0600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer db.Close()

	line := liner.NewLiner()
	defer line.Close()

	setAutoComplete(line)
	readHistory(line)
	defer saveHistory(line)

	for {
		cmd, err := line.Prompt("> ")
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("quit")
				fmt.Println(goodbyeMsg)
				break
			}
			fmt.Println("Unknown error: ", err)
		}

		switch {
		case regexp.MustCompile(`^(?i)bucket.*$`).MatchString(cmd):
			changeBucket(cmd)
		case regexp.MustCompile(`^(?i)keys.*$`).MatchString(cmd):
			showKeys(cmd)
		case regexp.MustCompile(`^(?i)get .*$`).MatchString(cmd):
			getKey(cmd)
		case regexp.MustCompile(`^(?i)quit$`).MatchString(cmd):
			fmt.Println(goodbyeMsg)
			return
		default:
			fmt.Println("> Unknown command:", cmd)
		}

		line.AppendHistory(cmd)
	}

}

func changeBucket(cmd string) {
	arg := regexp.MustCompile(" ").Split(cmd, 2)
	if len(arg) == 2 {
		fmt.Printf("> Bucket was \"%s\", changed to \"%s\"\n", *bucketName, arg[1])
		*bucketName = arg[1]
	} else {
		if *bucketName == "" {
			fmt.Println("> You are at the root bucket.")
		} else {
			fmt.Println("> You are at the", *bucketName, "bucket.")
		}

	}

}

func showKeys(cmd string) {
	arg := regexp.MustCompile(" ").Split(cmd, 2)
	if len(arg) == 2 {
		for _, v := range queryKeys(arg[1]) {
			fmt.Println(">", v)
		}
	} else {
		for _, v := range queryKeys("*") {
			fmt.Println(">", v)
		}
	}
}

func getKey(cmd string) {
	arg := regexp.MustCompile(" ").Split(cmd, 2)
	if len(arg) == 2 {
		queryKeyValue(arg[1])
	} else {
		fmt.Println("> Key must be given.")
	}
}

func locateBucket(tx *bolt.Tx, bucketName string) (b *bolt.Bucket) {
	arr := strings.Split(bucketName, ".")
	for index, bu := range arr {
		if index == 0 {
			b = tx.Bucket([]byte(bu))
			continue
		}

		if b == nil {
			break
		}

		b = b.Bucket([]byte(bu))
	}

	return
}

func queryKeys(keyName string) []string {
	var keys []string
	db.View(func(tx *bolt.Tx) error {
		b := locateBucket(tx, *bucketName)
		if b == nil {
			fmt.Println("> Bucket does not exist or no bucket selected.")
			return nil
		}
		b.ForEach(func(k, v []byte) error {
			if keyName == "*" || strings.Contains(keyName, string(k)) {
				keys = append(keys, string(k))
			}
			return nil
		})
		return nil
	})
	return keys
}

func queryKeyValue(keyName string) {
	db.View(func(tx *bolt.Tx) error {
		b := locateBucket(tx, *bucketName)
		if b == nil {
			fmt.Println("> Bucket does not exist or no bucket selected.")
			return nil
		}

		v := b.Get([]byte(keyName))
		if v == nil {
			fmt.Println("> Key does not exist or key is a nested bucket.")
			return nil
		}
		var out bytes.Buffer
		err := json.Indent(&out, v, "", "  ")
		if err == nil {
			fmt.Println("> JSON Pretty Printed:")
			fmt.Print(out.String())
			fmt.Println()
		} else {
			fmt.Println(">", string(v))
		}

		return nil
	})
}

func sayGoodbye() {
	fmt.Println("quit")
	fmt.Println("Toto, I have a feeling we're not in Kansas anymore.")
}

func setAutoComplete(line *liner.State) {
	line.SetCompleter(func(line string) (c []string) {
		for _, cmd := range commands {
			if strings.HasPrefix(cmd, strings.ToLower(line)) {
				c = append(c, cmd)
			}
		}
		return
	})
}

func readHistory(line *liner.State) {
	if f, err := os.Open(*historyFile); err == nil {
		line.ReadHistory(f)
		f.Close()
	}
}

func saveHistory(line *liner.State) {
	if f, err := os.Create(*historyFile); err != nil {
		log.Print("Error writing history file: ", err)
	} else {
		line.WriteHistory(f)
		f.Close()
	}
}
