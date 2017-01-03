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


# Output

```
cinarra@cinarra:~/go/src/github.com/cinarra/cnr-atca-repo/rtr$ ./rtr 
*********************************
CONFIG: {11000 {localhost 10000} {localhost 8000} {localhost 9001} {localhost 9002}}
*********************************
2016/12/16 20:01:18 
2016/12/16 20:01:18 DSP: Sending Req for User [ TheAppuid ]
2016/12/16 20:01:18 RTR: *** cache MISS ***
2016/12/16 20:01:18 MGW: Sending REP 
2016/12/16 20:01:18 RTR: stored in cache:  TheAppuid
2016/12/16 20:01:18 DSP: Recv Recommendation for [ TheAppuid ]
2016/12/16 20:01:18    :  [50000000001 50000000002]
2016/12/16 20:01:21 
2016/12/16 20:01:21 DSP: Sending Req for User [ TheAppuid ]
2016/12/16 20:01:21 RTR: Cache HIT 
2016/12/16 20:01:21 DSP: Recv Recommendation for [ TheAppuid ]
2016/12/16 20:01:21    :  [50000000001 50000000002]
2016/12/16 20:01:24 
2016/12/16 20:01:24 DSP: Sending Req for User [ TheAppuid ]
2016/12/16 20:01:24 RTR: Cache HIT 
2016/12/16 20:01:24 DSP: Recv Recommendation for [ TheAppuid ]
2016/12/16 20:01:24    :  [50000000001 50000000002]
2016/12/16 20:01:27 
2016/12/16 20:01:27 DSP: Sending Req for User [ TheAppuid ]
2016/12/16 20:01:27 RTR: Cache HIT 
2016/12/16 20:01:27 DSP: Recv Recommendation for [ TheAppuid ]
2016/12/16 20:01:27    :  [50000000001 50000000002]
2016/12/16 20:01:30 
2016/12/16 20:01:30 DSP: Sending Req for User [ TheAppuid ]
2016/12/16 20:01:30 RTR: *** cache MISS ***
2016/12/16 20:01:30 MGW: Lets DROP 
2016/12/16 20:01:30 Life is to short to wait that long. Expire request after 10 Msec
2016/12/16 20:01:30 DSP: NO-REC for [ TheAppuid ]
2016/12/16 20:01:33 
2016/12/16 20:01:33 DSP: Sending Req for User [ TheAppuid ]
2016/12/16 20:01:33 RTR: *** cache MISS ***
2016/12/16 20:01:33 MGW: Sending REP 
2016/12/16 20:01:33 DSP: Recv Recommendation for [ TheAppuid ]
2016/12/16 20:01:33    :  [50000000001 50000000002]
2016/12/16 20:01:33 RTR: stored in cache:  TheAppuid
*********************************
CLI CMD:  show rtr stats
CLI RSP:  
"DSP Recv Req": 6
"DSP Send NoRec": 1
"DSP Send Rec": 5
"MGW Recv": 2
"MGW Send": 3
"RTR Cache Hit": 3
"RTR Cache Miss": 3
"StartTime": 2016-12-16 20:01:15.790237556 -0800 PST
*********************************
2016/12/16 20:01:36 
2016/12/16 20:01:36 DSP: Sending Req for User [ TheAppuid ]
2016/12/16 20:01:36 RTR: Cache HIT 
2016/12/16 20:01:36 DSP: Recv Recommendation for [ TheAppuid ]
2016/12/16 20:01:36    :  [50000000001 50000000002]

```

# REST API

<p align="center">
  <img style="float: right;" src="assets/Rest_output.jpeg" alt="REST output"/>
</p>
