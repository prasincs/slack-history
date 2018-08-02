package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

func sanitizeNewLines(str string) string {
	noNewLine := strings.Replace(str, "\n", `\n`, -1)
	noNewLine = strings.Replace(noNewLine, "\r", `\r`, -1)
	return noNewLine
}

func writeHistory(history *slack.History, csvWriter *csv.Writer, onlyBots bool) error {
	//log.Printf("%#v\n", history.Messages[0:5])
	for _, m := range history.Messages {
		if onlyBots && m.BotID != "" {
			continue
		}

		text := m.Text
		if m.File != nil {
			//log.Printf("%#v", m.File)
			//out, err := ioutil.ReadAll(*m.File)
			//if err != nil {
			//	log.Fatalf("Failed to read attached file. %s", err)
			//}

			text += `\n`
			text += m.File.Name
			text += `\n`
			text += sanitizeNewLines(m.File.Preview)
		}

		if len(m.Attachments) > 0 {
			for _, attachment := range m.Attachments {
				text += `\n`
				text += sanitizeNewLines(attachment.Fallback)
			}
		}

		floatTs, err := strconv.ParseFloat(m.Timestamp, 64)
		if err != nil {
			log.Printf("timestamp %s not valid, cannot parse. Err: %s", m.Timestamp, err)
		}
		// ignore milliseconds for now
		intTs := int64(floatTs)
		goTime := time.Unix(intTs, 0)

		text = strings.Replace(text, "\n", "\\n", -1)

		err = csvWriter.Write([]string{goTime.Format(time.RFC3339), m.User, text})
		if err != nil {
			log.Printf(`Failed to write "%s"`, text)
		}
		csvWriter.Flush()
		//fmt.Println(m.Timestamp, m.Username, m.Text)

	}
	return nil
}

func main() {
	start := flag.String("start", "2017-06-01T00:00:00-07:00", "Start Time in ISO8601")
	end := flag.String("end", "", "End Time in ISO8601 (default is current time)")
	channel := flag.String("channel", "devops", "Channel Name to get logs for")
	filePath := flag.String("write", "output.csv", "where to output the file")
	bots := flag.Bool("bots", false, "only print bot messages")
	flag.Parse()
	token := os.Getenv("SLACK_TOKEN")
	if token == "" {
		log.Fatal("Need SLACK_TOKEN env variable")
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	api := slack.New(token)
	// If you set debugging, it will log all requests to the console
	// Useful when encountering issues
	// api.SetDebug(true)
	channels, err := api.GetChannels(true)
	if err != nil {
		log.Fatalf("Failed to get channels. %s", err)
	}
	channelID := ""
	for _, ch := range channels {
		if ch.Name == *channel {
			channelID = ch.ID
		}
		//fmt.Printf("Channel: %#v\n", channel)
	}

	if channelID == "" {
		log.Fatalf("Cannot find a channel with name %s", *channel)
	}
	file, err := os.Create(*filePath)
	if err != nil {
		log.Fatalf("cannot create file %s", filePath)
	}

	defer file.Close()

	csvWriter := csv.NewWriter(file)
	//defer csvWriter.Close()

	columns := []string{"timestamp", "user", "message"}

	csvWriter.Write(columns)
	csvWriter.Flush()

	//log.Printf("Channels: %v", channels)
	var endTime time.Time
	historyParams := slack.NewHistoryParameters()
	startTime, err := time.Parse(time.RFC3339, *start)
	if err != nil {
		log.Fatalf("Failed to parse start timestamp %s. Err: %s", *start, err)
	}
	startTimeTs := startTime.Unix() //* 1000
	if *end != "" {
		endTime, err = time.Parse(time.RFC3339, *end)
		if err != nil {
			log.Fatalf("Failed to parse end timestamp %s. Err: %s", *end, err)
		}
	} else {
		endTime = time.Now()
	}
	endTimeTs := endTime.Unix() //* 1000
	historyParams.Latest = strconv.Itoa(int(endTimeTs))
	historyParams.Oldest = strconv.Itoa(int(startTimeTs))
	//historyParams.Count = 1000
	history, err := api.GetChannelHistory(channelID, historyParams)
	if err != nil {
		log.Fatalf("Failed to get history for channel. %s", err)
	}

	//fmt.Println("latest", endTimeTs)

	//log.Println(len(history.Messages))
	writeHistory(history, csvWriter, *bots)
	lastMessageTs := history.Messages[len(history.Messages)-1].Timestamp
	log.Println("lastMessageTs", lastMessageTs)
	if history.HasMore {
		for {
			log.Println("Querying more from", lastMessageTs, "startTimeTs", startTimeTs)
			historyParams.Latest = lastMessageTs
			history, err := api.GetChannelHistory(channelID, historyParams)
			if err != nil {
				log.Fatalf("Failed to get history for channel. %s", err)
			}
			writeHistory(history, csvWriter, *bots)
			lastMessageTs = history.Messages[len(history.Messages)-1].Timestamp
			//log.Printf("history %+v", history.Messages)
			//log.Println("lastMessageTs", lastMessageTs)
			if !history.HasMore {
				break
			}
			time.Sleep(1 * time.Second)
		}
	}

	fmt.Println("Wrote", *filePath)

	//fmt.Printf("%#v\n", history)
}
