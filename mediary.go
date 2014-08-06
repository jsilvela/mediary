package main

import (
	"bufio"
	"fmt"
	"github.com/jsilvela/diary"
	"github.com/jsilvela/diary/filters"
	"github.com/jsilvela/diary/reports"
	"os"
	"sort"
	"strings"
	"time"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type ParserState int

const (
	Text ParserState = iota
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
		check(err)
		reqs = *diar
	}

	if len(reqs) > 0 {
		sort.Stable(reqs)
		fmt.Println("\n**** Latest entry")
		fmt.Println(reqs[len(reqs)-1])
	}

	scanner := bufio.NewScanner(os.Stdin)
	state := Null
	var record *diary.Record
	inRecord := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "}" && inRecord == true {
			reqs.AddEntry(record)
			inRecord = false
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
				reqs.AddEntry(record)
				fmt.Println("Stored last unfinished record")
			}
			fmt.Println("Wrote records")
			err := diary.Write(filename, reqs)
			check(err)
			return
		} else if inRecord {
			frag := strings.SplitN(line, ":", 2)
			if len(frag) == 2 {
				switch {
				case frag[0] == "time":
					processTime(frag[1], record, &state)
				case frag[0] == "tags":
					processTags(frag[1], record, &state)
				case frag[0] == "text":
					processText(frag[1], record, &state)
				default:
					if state == Text {
						processText(line, record, &state)
					} else {
						fmt.Printf("Invalid tag: %s\n", frag[0])
						continue
					}
				}
			} else {
				if state == Text {
					processText(line, record, &state)
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
		*d = *filters.ByWeek(*d)
	case "tags":
		fmt.Println(reports.Tags(*d))
	case "latest":
		fmt.Println(reports.Latest(*d))
	default:
		fmt.Printf("Unrecognized command: %s\n", command)
	}
}

// Parse text declaration when writing new entry
func processText(line string, record *diary.Record, state *ParserState) {
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

func processTime(line string, record *diary.Record, state *ParserState) {
	const shortForm = "2006-01-02"
	if strings.TrimSpace(line) == "today" {
		record.EventTime = time.Now()
	} else {
		t, _ := time.Parse(shortForm, strings.TrimSpace(line))
		record.EventTime = t
	}
}

func processTags(line string, record *diary.Record, state *ParserState) {
	frags := strings.Split(line, ",")
	tags := make([]string, len(frags))
	for i := 0; i < len(frags); i++ {
		tags[i] = strings.TrimSpace(frags[i])
	}
	record.Tags = tags
}
