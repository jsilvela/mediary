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
			fs, rep := parse_script_line(frags)
			eval_script(diary_copy, fs, rep)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}

type func_cells struct {
	f    func(diary.Diary) *diary.Diary
	next *func_cells
}

type report_cell struct {
	f   func(diary.Diary) []string
	arg string
}

func parse_script_line(frags []string) (*func_cells, *report_cell) {
	var funcs *func_cells
	var rep *report_cell

	for len(frags) > 0 {
		switch frags[0] {
		case "week":
			funcs = &func_cells{filters.By_week, funcs}
			frags = frags[1:]
		case "month":
			funcs = &func_cells{filters.By_month, funcs}
			frags = frags[1:]
		case "by-tag":
			tag := frags[1]
			by_tag := func(d diary.Diary) *diary.Diary {
				return filters.By_tag(d, tag)
			}
			funcs = &func_cells{by_tag, funcs}
			frags = frags[2:]
		case "tags":
			rep = &report_cell{reports.Tags, ""}
			frags = frags[1:]
		case "latest":
			latest := func(d diary.Diary) []string {
				rep := reports.Latest(d)
				out := make([]string, len(rep))
				i := 0
				for tag, t := range rep {
					out[i] = fmt.Sprintf("%s\t=> %v\n",
						tag, (*t).Format("Mon 2 Jan 2006"))
					i++
				}
				return out
			}
			rep = &report_cell{latest, ""}
			frags = frags[1:]
		default:
			fmt.Printf("Unrecognized command: %s\n", frags[0])
			return nil, nil
		}
	}

	return funcs, rep
}

func eval_script(d diary.Diary, fs *func_cells, rep *report_cell) {
	if fs != nil {
		eval_script(*fs.f(d), fs.next, rep)
	} else if rep != nil {
		fmt.Println(rep.f(d))
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
