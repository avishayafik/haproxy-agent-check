package main

import (
	"context"
    "fmt"
    "log"
    "github.com/firstrow/tcp_server"
    "strconv"
//   "runtime"
    //"github.com/shirou/gopsutil/load"
    "time"
    "io/ioutil"
//    "encoding/json"
    //"net/http"
	yaml "gopkg.in/yaml.v2"
	"os"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"math"
	"strings"
)


const (
	prometheus    = "http://i-was-stg-web1.stapp.me:9090/"
	//prometheus = "http://prometheus.hosts-app.com:9090/"
	prometheusURI = "/api/v1/query"
	low_weight_threshold = 60
	high_weight_threshold = 120

)




type conf struct {
        Weight int64 `yaml:"weight"`
}



type rejects struct {
	Name         string      `json:"name"`
	Description  interface{} `json:"description"`
	BaseUnit     interface{} `json:"baseUnit"`
	Measurements []struct {
		Statistic string  `json:"statistic"`
		Value     float64 `json:"value"`
	} `json:"measurements"`
	AvailableTags []struct {
		Tag    string   `json:"tag"`
		Values []string `json:"values"`
	} `json:"availableTags"`
}



func readConf(filename string) (*conf, error) {
        buf, err := ioutil.ReadFile(filename)
        if err != nil {
                return nil, err
        }
        c := &conf{}
        err = yaml.Unmarshal(buf, c)
        if err != nil {
                return nil, fmt.Errorf("in file %q: %v", filename, err)
        }
        return c, nil
}


// wright weight to file if not exist
func check(e error) {
    if e != nil {
        //panic(e)
        ioutil.WriteFile("/tmp/weight.txt", []byte("60"), 0655)
    }
}


func writeWeightConf(filename string,Weight int64 ) (*conf, error) {
        buf, err := ioutil.ReadFile(filename)
        if err != nil {
                return nil, err
        }
        c := &conf{}
        err = yaml.Unmarshal(buf, c)
        if err != nil {
                return nil, fmt.Errorf("in file %q: %v", filename, err)
        }

        f, err := os.Create("/tmp/dat2")
        if err != nil {
                log.Fatal(err)
        }

        c.Weight = Weight
        d, err := yaml.Marshal(&c)
        if err != nil {
                log.Fatalf("error: %v", err)
        }

        err = ioutil.WriteFile( "/tmp/conf.yaml", d, 0644)
        if err != nil {
                log.Fatal(err)
        }
        f.Close()
        return c, nil
}






func main() {

    server := tcp_server.New(":9999")
////// get weight from disk
    c, err := readConf("/tmp/conf.yaml")
    if err != nil {
              log.Fatal(err)
    }
    log.Println(c.Weight)
    //fmt.Println(c.Weight)
    var start_weight = int(c.Weight)
//////
    var last_weight = &start_weight // Pointer to an `int` type
    go  weight(last_weight)
    server.OnNewClient(func(c *tcp_server.Client) {
       //fmt.Println(*last_weight)
       //fmt.Println("Client connected")
       last_weight := int(*last_weight)
       last_weight_str := strconv.Itoa(last_weight)

       c.Send(last_weight_str+"%\n")
       c.Close()
    })
    server.Listen()
}


func validate_weight() {

}


func weight(last_weight *int)  {

   hostname, err := os.Hostname()
   if err != nil {
            panic(err)
   }

   prometheus_query := "100-((avg(avg_over_time(node_load1{instance=~'.*"+hostname+".stapp.me.*'}[30d:1h])))/avg(avg_over_time(node_load1{instance=~'.*was-prd-web.*'}[30d:1h]))-1)*avg(avg_over_time(haproxy_server_weight{server=~'.*"+hostname+".stapp.me',proxy='rtb'}[30d:1h]))"

   rejects_diff(prometheus_query)
   //var cpu_count = float64(runtime.NumCPU())
   for {
   log.Println("weight is : " ,*last_weight)
   rejects_weight  := rejects_diff(prometheus_query)
   log.Println(rejects_weight)

   if  rejects_weight < low_weight_threshold  {
         rejects_weight = low_weight_threshold
   } else if rejects_weight > high_weight_threshold {
         rejects_weight = weight_avg()
   }

   *last_weight = int(rejects_weight)

    last_weight := int(*last_weight)
    write, err := writeWeightConf("/tmp/conf.yaml",int64(last_weight))
    if err != nil {
                log.Fatal(err)
     }
   log.Println("write to disk: ", write.Weight)
   time.Sleep(60000000000)

   }
}



func rejects_diff(prometheus_query string) float64 {
	client, err := api.NewClient(api.Config{
		Address: prometheus,
	})
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}



	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r := v1.Range{
		Start: time.Now().Add(-time.Minute),
		End:   time.Now(),
		Step:  time.Minute,
	}
    //hostname := "was-prd-web3"

    log.Println(prometheus_query)
	result, warnings, err := v1api.QueryRange(ctx, prometheus_query , r)
	if err != nil {
		fmt.Printf("Error querying Prometheus: %v\n", err)
		os.Exit(1)
	}
	if len(warnings) > 0 {
		fmt.Printf("Warnings: %v\n", warnings)
	}
	res := strings.Split(result.String(), " ")[1]
	res = strings.Split(res, "=>")[1]
	res = strings.Replace(res, "\n", "", -1)
	resfloat, err := strconv.ParseFloat(res, 64)
	load_diff := math.Round(resfloat)
	return load_diff
}



func weight_avg() float64 {
	client, err := api.NewClient(api.Config{
		Address: prometheus,
	})
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r := v1.Range{
		Start: time.Now().Add(-time.Minute),
		End:   time.Now(),
		Step:  time.Minute,
	}
    // hostname := "was-prd-web1"

    log.Println("avg(haproxy_server_weight{server=~'was-prd-web.*',proxy='rtb'})")

	result, warnings, err := v1api.QueryRange(ctx, "avg(haproxy_server_weight{server=~'was-prd-web.*',proxy='rtb'})" , r)
	if err != nil {
		fmt.Printf("Error querying Prometheus: %v\n", err)
		os.Exit(1)
	}
	if len(warnings) > 0 {
		fmt.Printf("Warnings: %v\n", warnings)
	}
	res := strings.Split(result.String(), " ")[1]
	res = strings.Split(res, "=>")[1]
	res = strings.Replace(res, "\n", "", -1)
	resfloat, err := strconv.ParseFloat(res, 64)
	weight_avg := math.Round(resfloat)
	return weight_avg
}