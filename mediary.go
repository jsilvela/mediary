package main

import (
	"bufio"
	"fmt"
	"github.com/jsilvela/diary"
	"github.com/jsilvela/diary/filters"
	"github.com/jsilvela/diary/reports"
	"log"
	"os"
	"strings"
	"time"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type Parser_state int

const (
	Text Parser_state = iota
	Tags
	Time
	Null
)

func main() {

	var reqs diary.Diary
	var filename string

	if len(os.Args) < 2 {
		filename = "./mediary.txt"
	} else {
		filename = os.Args[1]
		diar, err := diary.Read(filename)
		if err != nil {
			log.Fatal(err)
		}
		reqs = *diar
	}

	if len(reqs) > 0 {
		w, h := reqs.Latest_written(), reqs.Latest_happened()

		if w != h {
			fmt.Println("\n**** Latest written:")
			fmt.Println(w)
			fmt.Println("\n\n**** Latest happened:")
			fmt.Println(h)
		} else {
			fmt.Println("\n**** Latest written:")
			fmt.Println(w)
		}
	}

	scanner := bufio.NewScanner(os.Stdin)
	state := Null
	var record *diary.Record
	inRecord := false
	dirty := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "}" && inRecord == true {
			reqs.Add_entry(record)
			inRecord = false
			dirty = true
		} else if line == "}" && inRecord == false {
			fmt.Printf("Closing brace closes nothing\n")
			continue
		} else if line == "new {" {
			if inRecord {
				fmt.Println("Can't open a new record while we're in a record")
				continue
			}
			record = new(diary.Record)
			inRecord = true
		} else if line == "exit" {
			if inRecord && record.Text != "" {
				reqs.Add_entry(record)
				dirty = true
				fmt.Println("Stored last unfinished record")
			}
			if !dirty {
				fmt.Println("Unmodified, not saving")
			} else {
				err := diary.Write(filename, reqs)
				check(err)
				fmt.Println("Wrote records")
			}
			return
		} else if inRecord {
			frag := strings.SplitN(line, ":", 2)
			if len(frag) == 2 {
				switch {
				case frag[0] == "time":
					process_time(frag[1], record, &state)
				case frag[0] == "tags":
					process_tags(frag[1], record, &state)
				case frag[0] == "text":
					process_text(frag[1], record, &state)
				default:
					if state == Text {
						process_text(line, record, &state)
					} else {
						fmt.Printf("Invalid tag: %s\n", frag[0])
						continue
					}
				}
			} else {
				if state == Text {
					process_text(line, record, &state)
				} else {
					fmt.Println("Invalid. Key must be supplied")
					continue
				}
			}
		} else {
			tempD := reqs
			frags := strings.Split(line, " ")
			for _, cm := range frags {
				execute(cm, &tempD)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}

// Execute single line of strung commands
func execute(command string, d *diary.Diary) {
	switch command {
	case "week":
		*d = *filters.By_week(*d)
	case "tags":
		fmt.Println(reports.Tags(*d))
	case "latest":
		fmt.Println("Latest events of each tag:")
		lt := reports.Latest(*d)
		for k, v := range lt {
			fmt.Printf("%s\t=> %v\n", k, v.Format("Mon 2 Jan 2006"))
		}
	default:
		fmt.Printf("Unrecognized command: %s\n", command)
	}
}

// Parse text declaration when writing new entry
func process_text(line string, record *diary.Record, state *Parser_state) {
	if line == "===" {
		*state = Null
	} else {
		if *state != Text {
			record.Text = strings.Join([]string{record.Text, line}, "")
			*state = Text
		} else {
			record.Text = strings.Join([]string{record.Text, "\n", line}, "")
		}
	}
}

func process_time(line string, record *diary.Record, state *Parser_state) {
	const shortForm = "2006-01-02"
	if strings.TrimSpace(line) == "today" {
		record.Event_time = time.Now()
	} else {
		t, _ := time.Parse(shortForm, strings.TrimSpace(line))
		record.Event_time = t
	}
}

func process_tags(line string, record *diary.Record, state *Parser_state) {
	frags := strings.Split(line, ",")
	tags := make([]string, len(frags))
	for i := 0; i < len(frags); i++ {
		tags[i] = strings.TrimSpace(frags[i])
	}
	record.Tags = tags
}
