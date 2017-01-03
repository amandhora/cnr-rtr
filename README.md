# Real Time Recommendation Engine (RTR)

# To compile

cmd> go build

# Config File
Create following in same directory (sample in github)
rtr_conf.json

# To Run

Start Redis on 10000 (or change default config in rtr_conf.json)

cmd> redis-server --daemonize yes --port 10000

Run RTR

cmd> ./rtr



# Stats

RTR Stats

(CLI port : 11000)
cnr_mon> show rtr stats

REST API listen on localhost:8080

http://localhost:8080/rtr/stats

