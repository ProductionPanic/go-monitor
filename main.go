package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ProductionPanic/go-cursor"
	"github.com/ProductionPanic/go-pretty"
	terminal "golang.org/x/crypto/ssh/terminal"
)

var box_v = '│'
var box_h = '─'
var box_tl = '┌'
var box_tr = '┐'
var box_bl = '└'
var box_br = '┘'

func main() {
	cursor.Hide()
	defer cursor.Show()
	cursor.ClearScreen()
	cursor.Top()
	refreshInterval := flag.Int("r", 10, "Refresh interval in seconds")
	lines := flag.Int("l", 10, "Number of lines to print")
	filepath := os.Args[1]

	flag.CommandLine.Parse(os.Args[2:])

	box := NewLogBox(filepath, *refreshInterval, *lines)
	box.Setup()
	box.Run()

	fmt.Println("filepath", filepath)
	fmt.Println("refreshInterval", refreshInterval)
	fmt.Println("lines", lines)
}

type LogBox struct {
	filePath        string
	refreshInterval int
	lines           int
	cols            int
	offset          int
}

func getTermSize() (int, int, error) {
	fd := int(os.Stdout.Fd())
	return terminal.GetSize(fd)
}

func NewLogBox(url string, refreshInterval int, lines int) *LogBox {
	width, _, err := getTermSize()
	if err != nil {
		log.Fatal(err)
	}
	return &LogBox{
		filePath:        url,
		refreshInterval: refreshInterval,
		lines:           lines,
		cols:            width - 6,
		offset:          3,
	}
}

func (l *LogBox) Setup() {
	// query the width and height of the terminal using tput
	max_width := l.cols
	max_height := l.lines + 2
	cursor.ClearScreen()
	cursor.Top()

	offsetSpace := strings.Repeat(" ", int(l.offset))
	white_space := strings.Repeat(" ", max_width-2)
	top_line := string(box_tl) + strings.Repeat(string(box_h), max_width-2) + string(box_tr)
	bottom_line := string(box_bl) + strings.Repeat(string(box_h), max_width-2) + string(box_br)
	middle_line := string(box_v) + white_space + string(box_v)

	for i := 0; i < max_height; i++ {
		if i == 0 {
			pretty.Println(offsetSpace + top_line)
		} else if i == max_height-1 {
			pretty.Println(offsetSpace + bottom_line)
		} else {
			pretty.Println(offsetSpace + middle_line)
		}
	}
}

func (l *LogBox) Update() {
	content := l.GetContent()
	lines := strings.Split(content, "\n")
	if len(lines) > l.lines {
		// remove n from the beginning of the slice
		lines = lines[len(lines)-(l.lines+1):]
	}

	cursor.Top()
	cursor.Down(1)
	for i := range lines {
		cursor.LineStart()
		if i == l.lines {
			break
		}
		cursor.ClearLine()
		content := lines[i]
		content_len := len(content)
		if content_len > l.cols-2 {
			content = content[:l.cols-2]
		}
		pretty.Print(fmt.Sprintf("%s%s%s", strings.Repeat(" ", l.offset), string(box_v), content))
		cursor.LineEnd()
		cursor.Left(l.offset)
		pretty.Print(string(box_v))
		cursor.Down(1)
	}
	cursor.Top()
	cursor.Down(l.lines)
	pretty.Println("\n")
	cursor.LineStart()
	timestampStr := time.Now().Format("2006-01-02 15:04:05")
	pretty.Print(fmt.Sprintf("%s%s", strings.Repeat(" ", l.offset), timestampStr))

}

func (l *LogBox) Run() {
	var running = true
	var wg sync.WaitGroup
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-c
		running = false
		cursor.Show()
		cursor.ClearScreen()
		cursor.Top()
		os.Exit(0)
	}()
	wg.Add(1)
	go (func() {
		defer wg.Done()
		duration, err := time.ParseDuration(strconv.Itoa(l.refreshInterval) + "s")
		if err != nil {
			log.Fatal(err)
		}
		for {
			if !running {
				break
			}
			l.Update()
			time.Sleep(duration)
		}
	})()

	wg.Wait()
}

func (l *LogBox) GetContent() string {
	is_file_a_url := strings.HasPrefix(l.filePath, "http")
	if !is_file_a_url {
		return l.GetContentFromFile()
	}

	return l.GetContentFromUrl()
}

func (l *LogBox) GetContentFromFile() string {
	content, err := os.ReadFile(l.filePath)
	if err != nil {
		log.Fatal(err)
	}
	return string(content)
}

func (l *LogBox) GetContentFromUrl() string {
	resp, err := http.Get(l.filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(body)
}
