# Get slack history

I wrote this to import training data from slack channels for some NLP-like stuff

expects `SLACK_TOKEN` environment variable

go run slack-history.go -start 2018-04-18T00:00:00Z -end 2018-08-02T00:00:00Z -channel "general"

this will search in past and dump everything as an output.csv file
