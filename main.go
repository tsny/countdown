package main

import (
	"flag"
	"fmt"
	"github.com/nsf/termbox-go"
	"os"
	"strconv"
	"time"
)

const (
	usage = `
 Examples:
    timer 25s
    timer 1m50s
    timer 3h40m50s`
	tick = time.Second
)

var (
	timer          *time.Timer
	ticker         *time.Ticker
	queues         chan termbox.Event
	startDone      bool
	startX, startY int

	message string
	countUp bool
)

func init() {
	// This doesn't work, maybe fix it some day
	//flag.StringVar(&message, "m", "", "Message to display next to the timer")
	flag.BoolVar(&countUp, "up", false, "Whether to count up to the time")
	flag.Usage = func() {
		println("\n Flags")
		flag.PrintDefaults()
		println(usage)
	}
	flag.Parse()
}

func main() {
	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(0)
	}
	// Try parsing kitchen sink duration
	dur, err := getKitchenTimeDuration(os.Args[1])
	if err != nil {
		// Try just seconds if there's no letters
		i, err := strconv.Atoi(os.Args[1])
		if err == nil {
			dur = time.Duration(i) * time.Second
		} else {
			// Fall back to parsing as default time format
			dur, err = time.ParseDuration(os.Args[1])
			if err != nil {
				stderr("error: invalid duration or kitchen time: %v\n", os.Args[1])
				os.Exit(2)
			}
		}
	}

	err = termbox.Init()
	if err != nil {
		panic(err)
	}

	queues = make(chan termbox.Event)
	go func() {
		for {
			queues <- termbox.PollEvent()
		}
	}()
	countdown(dur, countUp)
}

func draw(d time.Duration) {
	w, h := termbox.Size()
	clear()

	str := format(d)
	if message != "" {
		str += message
	}
	text := toText(str)

	if !startDone {
		startDone = true
		startX, startY = w/2-text.width()/2, h/2-text.height()/2
	}

	x, y := startX, startY
	for _, s := range text {
		echo(s, x, y)
		x += s.width()
	}

	flush()
}

func format(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h < 1 {
		return fmt.Sprintf("%02d:%02d", m, s)
	}
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func start(d time.Duration) {
	timer = time.NewTimer(d)
	ticker = time.NewTicker(tick)
}

func stop() {
	timer.Stop()
	ticker.Stop()
}

func countdown(dur time.Duration, countUp bool) {
	var exitCode int

	start(dur)
	if countUp {
		dur = 0
	}
	draw(dur)

loop:
	for {
		select {
		case ev := <-queues:
			if ev.Type == termbox.EventKey && (ev.Key == termbox.KeyEsc || ev.Key == termbox.KeyCtrlC) {
				exitCode = 1
				break loop
			}
			if ev.Ch == 'p' || ev.Ch == 'P' {
				stop()
			}
			if ev.Ch == 'c' || ev.Ch == 'C' {
				start(dur)
			}
		case <-ticker.C:
			if countUp {
				dur += tick
			} else {
				dur -= tick
			}
			draw(dur)
		case <-timer.C:
			break loop
		}
	}

	termbox.Close()
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	fmt.Printf("\a")
}

func getKitchenTimeDuration(date string) (time.Duration, error) {

	targetTime, err := time.Parse(time.Kitchen, date)
	if err != nil {
		return time.Duration(0), err
	}

	now := time.Now()
	originTime := time.Date(0, time.January, 1, now.Hour(), now.Minute(), now.Second(), 0, time.UTC)

	// the time of day has already passed, so target tomorrow
	if targetTime.Before(originTime) {
		targetTime = targetTime.AddDate(0, 0, 1)
	}

	duration := targetTime.Sub(originTime)

	return duration, err
}
