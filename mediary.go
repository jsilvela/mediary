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
			fmt.Println("\n**** Latest written:\n%s\n\n**** Latest happened:\n%\n", w, h)
		} else {
			fmt.Println("\n**** Latest written:\n%s\n", w)
		}
	}

	scanner := bufio.NewScanner(os.Stdin)
	in_text := false
	var record *diary.Record
	out_of_sync := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		} else if line == "}" && record != nil {
			reqs.Add_entry(record)
			record = nil
			in_text = false
			out_of_sync = true
		} else if line == "}" && record == nil {
			fmt.Printf("Closing brace closes nothing\n")
			continue
		} else if line == "new {" || line == "new{" {
			if record != nil {
				fmt.Println("Can't open a new record while we're" +
					" in a record")
				continue
			}
			record = new(diary.Record)
		} else if line == "exit" {
			if record != nil && record.Text != "" {
				reqs.Add_entry(record)
				out_of_sync = true
				fmt.Println("Adding last unfinished record")
			}
			if out_of_sync {
				err := diary.Write(filename, reqs)
				check(err)
				fmt.Println("Wrote records")
			} else {
				fmt.Println("Unmodified, not saving")
			}
			return
		} else if record != nil {
			frag := strings.SplitN(line, ":", 2)
			if len(frag) == 2 {
				sel, content := frag[0], frag[1]
				switch {
				case sel == "time":
					process_time(content, record)
				case sel == "tags":
					process_tags(content, record)
				case sel == "text":
					in_text = process_text(content, record)
				default:
					if in_text {
						in_text = process_text(line, record)
					} else {
						fmt.Printf("Invalid tag: %s\n", sel)
						continue
					}
				}
			} else {
				if in_text {
					in_text = process_text(line, record)
				} else {
					fmt.Println("Invalid. Key must be supplied")
					continue
				}
			}
		} else {
			diary_copy := reqs
			frags := strings.Split(line, " ")
			for _, command := range frags {
				execute(command, &diary_copy)
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
	case "month":
		*d = *filters.By_month(*d)
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
func process_text(line string, record *diary.Record) bool {
	if line == "===" {
		return false
	} else {
		if record.Text == "" {
			record.Text = strings.TrimSpace(line)
		} else {
			record.Text = strings.Join([]string{record.Text, "\n", line}, "")
		}
		return true
	}
}

func process_time(line string, record *diary.Record) {
	const shortForm = "2006-01-02"
	if strings.TrimSpace(line) == "today" {
		record.Event_time = time.Now()
	} else {
		t, _ := time.Parse(shortForm, strings.TrimSpace(line))
		record.Event_time = t
	}
}

func process_tags(line string, record *diary.Record) {
	frags := strings.Split(line, ",")
	tags := make([]string, len(frags))
	for i := 0; i < len(frags); i++ {
		tags[i] = strings.TrimSpace(frags[i])
	}
	record.Tags = tags
}
