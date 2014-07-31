package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"time"
)

type Record struct {
	EventTime   time.Time
	WrittenTime time.Time
	Tags        []string
	Text        string
}

type Diary []*Record

func (a Diary) Len() int           { return len(a) }
func (a Diary) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Diary) Less(i, j int) bool { return a[i].WrittenTime.Before(a[j].WrittenTime) }

func (r *Record) String() string {
	y, m, d := r.EventTime.Date()
	return fmt.Sprintf("t: %d-%d-%d\ntags: %s\ntext: %s\n", y, m, d, r.Tags, r.Text)
}

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

	var reqs Diary
	var filename string

	if len(os.Args) < 2 {
		filename = "./mediary.txt"
	} else {
		filename = os.Args[1]
		bytes, errfile := ioutil.ReadFile(filename)
		check(errfile)
		err := json.Unmarshal(bytes, &reqs)
		check(err)
	}


	if len(reqs) > 0 {
		sort.Stable(reqs)
		fmt.Println("\n**** Latest entry")
		fmt.Println(reqs[len(reqs)-1])
	}

	scanner := bufio.NewScanner(os.Stdin)
	state := Null
	record := new(Record)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "***" {
			addToDiary(&reqs, record)
			record = new(Record)
		} else if line == "====" {
			if record.Text != "" {
				addToDiary(&reqs, record)
				fmt.Println("Stored last record")
			}
			fmt.Println("Wrote records")
			mar, err := json.MarshalIndent(reqs, "", "\t")
			check(err)
			e := ioutil.WriteFile(filename, mar, 0644)
			check(e)
			return
		} else {
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
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}

func addToDiary(a *Diary, r *Record) {
	r.WrittenTime = time.Now()
	*a = append(*a, r)
}

func processText(line string, record *Record, state *ParserState) {
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

func processTime(line string, record *Record, state *ParserState) {
	const shortForm = "2006-01-02"
	if strings.TrimSpace(line) == "today" {
		record.EventTime = time.Now()
	} else {
		t, _ := time.Parse(shortForm, strings.TrimSpace(line))
		record.EventTime = t
	}
}

func processTags(line string, record *Record, state *ParserState) {
	frags := strings.Split(line, ",")
	tags := make([]string, len(frags))
	for i := 0; i < len(frags); i++ {
		tags[i] = strings.TrimSpace(frags[i])
	}
	record.Tags = tags
}
