# Haproxy agent check 

## This project aims to who ever runs haproxy on premise , and his hardware backend is not always the same in all backends  

### this project purpose is to make haproxy load equals among all servers based on cpu .

checkout what is haproxy agent check: 
https://cbonte.github.io/haproxy-dconv/1.8/configuration.html


this project expose tcp connection  in order to cahnge haproxy weight


how to run: 
```
go run cpu.go

telnet 127.0.0.1 9999
Trying 127.0.0.1...
Connected to 127.0.0.1.
Escape character is '^]'.
50%
Connection closed by foreign host.


```

how to change thresholds:

go to cpu.go and change the below and commit  :

````
// weight percentage , lower threshold 
low_weight_threshold  := 50
// weight percentage , higher threshold 
high_weight_threshold := 120
// we want it to scale up only if rejects lower than this threshold
low_rejects_threshold := 50
// we want it to scale down only if rejects higher than this threshold
high_rejects_threshold := 100
````





haproxy backend config example:

````
server i-was-prd-web18.stapp.me i-was-prd-web18.stapp.me:8480 weight 100 check minconn 100 maxconn 500 maxqueue 1 slowstart 120s  agent-check agent-inter 5s agent-addr i-was-prd-web18.stapp.me agent-port 9999


```` 

the above check will test weigh percentage every 5 seconds and will adjust accordingly . 

for example if agent returns 60% and weight  was set to 100  , weight now is 60 .

another example if agent return 110 and weight 100 , than weight set to 110 

## statefull vs stateless agent


there are two approaches with this statefull and stateless 

stateless: 
means whenever haproxy checks agent it will return weight based on current cpu usage :

for example if cpu usage is 75% it will return 80% weight (55+cpu idle)

````
        if cpuIdle  > 25 &&  cpuIdle <  35 {
            // Set server weight to 10
            cpuIdle =  55 + cpuIdle
            cpuIdle := int(cpuIdle)
            cpuIdle_str := strconv.Itoa(cpuIdle)
            c.Send(cpuIdle_str+"%\n")
            fmt.Println("weight percentage: ",cpuIdle_str)
       } else if cpuIdle  > 5 &&  cpuIdle <  25 {
            cpuIdle =  55 + (2 * cpuIdle - 20)
            cpuIdle := int(cpuIdle)
            cpuIdle_str := strconv.Itoa(cpuIdle)
            c.Send(cpuIdle_str+"%\n")
            fmt.Println("percentage: ",cpuIdle_str)
        } else if  cpuIdle  > 0 &&  cpuIdle <  5   {
            //cpuIdle = cpu_threshold *
            fmt.Println("percentage: ",10)
            c.Send("10%\n")
        } else {
           fmt.Println("percentage: ",100)
           c.Send("100%\n")
        }


````

statefull:

when ever cpuload is higher than threshold which is 38 (cpucount * 0.95) , or rejects higher than 100 agent will remember last weigh before change and will decrease weight by -2

for example if weight was 70 , after change it will be 68 ,  

for scaling up the same thing for scale up   but with 1 threshold should be lower rejects than min threshold and lower cpu load than 38 
````
          if Load_now  > float64(cpu_count) && float64(*last_weight) >  float64(low_weight_threshold)  ||  float64(*current_rejects)  >  float64(high_rejects_threshold) && float64(*last_weight) > float64(low_weight_threshold)  {
                *last_weight = *last_weight-2
                fmt.Println ("scaling down")
                fmt.Println("weight percentage: ",*last_weight)
                // sleep for 60 sec
                time.Sleep(30000000000)
          } else if Load_now*0.95  < cpu_count*0.8 &&  float64(*last_weight) <  float64(high_weight_threshold)  &&  float64(*current_rejects)  <  float64(low_rejects_threshold) {
                *last_weight = *last_weight+1
                fmt.Println ("scaling up")
                fmt.Println("weight percentage: ",*last_weight)
                 // sleep for 60 sec
                time.Sleep(90000000000)
           } else {
                fmt.Println ("nothing to do")
                fmt.Println("weight percentage now: ",*last_weight)
           }

````

we saw statefull approach was much more stable , and give much better results cpuload is much more stable and rejects are much lower 


last stable option  , this is production option is take measurements from prometheus . 

this approcah is more stable weight doesn't change often .

prometheus query :

query compares server and rejects to all cluster and base on that decide what should be the weight of the server comparing to all the cluster.

this  approcah is more gracefull because it will calc last 24h and will learn about it .

```


load_average:

100-((avg(avg_over_time(node_load1{instance=~"server_name.*"}[24h:1h])))/avg(avg_over_time(node_load1{instance=~"<total_servers_prefix>.*"}[24h:1h]))-1)*avg(avg_over_time(haproxy_server_weight{server=~"<server_name>",proxy="<backend_service>"}[24h:1h]))

```
this code assume the below 

 
* need to be running on backend machine , or in pod if using haproxy ingress controler 

* need prometheus server who expose node exporter metrics on all backend machines .

* need to use haproxy 2.1 or higher and monitor it with haproxy exporter , who being exposed with edded in haproxy  

to configure code just change the following  and build docker images:

```
const (
	prometheus    = "http://i-was-stg-web1.stapp.me:9090/"
	prometheusURI = "/api/v1/query"
    // low weight precentage
	low_weight_threshold = 60
    // high weight precentage
	high_weight_threshold = 120

)


// prometheus query , this query compares between server cpu load  , to total backend servers in the cluster  

prometheus_query := "100-((avg(avg_over_time(node_load1{instance=~'.*"+hostname+".stapp.me.*'}[30d:1h])))/avg(avg_over_time(node_load1{instance=~'.*was-prd-web.*'}[30d:1h]))-1)*avg(avg_over_time(haproxy_server_weight{server=~'.*"+hostname+".stapp.me',proxy='rtb'}[30d:1h]))"



```