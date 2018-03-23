package main

import (
  "fmt"
  "time"
  "net/url"
  "net/http"
  "io/ioutil"
  "encoding/json"

  log "github.com/sirupsen/logrus"
)

const (
  HeaderName string = "X-PrQL-Token"
)

type Config struct {
  Port int16
}

var (
  IpLogger log.FieldLogger
  host string
)

func StartServer(config *Config) {
  mux := http.NewServeMux()
  port := fmt.Sprintf(":%d", config.Port)
  host = fmt.Sprintf("127.0.0.1%s", port)

  mux.HandleFunc("/", handler)
  mux.HandleFunc("/check", func(w http.ResponseWriter, req *http.Request) { 
    w.WriteHeader(http.StatusOK)
  })

  go checkServerStatus()

  log.Info("Starting server")
  http.ListenAndServe(port, mux)
}


/**
* Private
*/

type postRequestBody struct {
  Query string
}

func checkServerStatus() {
  running := false

  for i := 0; i < 10; i += 1 {
    time.Sleep(time.Second) 
    endpoint := url.URL{Scheme: "http", Host: host, Path: "check"}
    res, err := http.Get(endpoint.String())

    if err != nil {
      fmt.Println("error: ", err)
      continue
    }

    res.Body.Close()

    if res.StatusCode != http.StatusOK {
      fmt.Println(res.StatusCode)
      continue 
    }

    running = true
    break
  } 

  if running {
    log.Info(fmt.Sprintf("Server listening at %s", host))
  } else {
    log.Panic(fmt.Sprintf("Cannot connect to server at %s.\nExiting...", host)) 
  }
}

func handler(w http.ResponseWriter, r *http.Request) {
  IpLogger = log.WithFields(log.Fields{"IP": r.RemoteAddr})

  var token string

  token = r.Header.Get(HeaderName)
  if token == "" {
    tokenParam := r.URL.Query()["token"]

    if len(tokenParam) != 0 {
      token = tokenParam[0] 
    } else {
      fail(w, "No token")
      return
    }
  }
  IpLogger.Info(fmt.Sprintf("Received request using token %s", token))

  query := getQuery(r)
  if query == "" {
    fail(w, "No query")
    return
  }
  IpLogger.Info(fmt.Sprintf("Token %s was used for the following query: \"%s\"", token, query))

  data, err := PerformQuery(query, token)
  if err != nil {
    fail(w, "Query failed") 
  }

  json.NewEncoder(w).Encode(data)
}


func getQuery(r *http.Request) string {
  var query string

  if r.Method == "GET" {
    q := r.URL.Query()

    if len(q["query"]) != 0 {
      query = q["query"][0]
    }
  }

  if r.Method == "POST" {
    var body postRequestBody

    bodyBytes, err := ioutil.ReadAll(r.Body)
    if err != nil {
      IpLogger.Panic(err) 
    }

    if len(bodyBytes) != 0 {
      err = json.Unmarshal(bodyBytes, &body)
      if err != nil {
        IpLogger.Panic(err) 
      }

      if body.Query != "" {
        query = body.Query
      }
    }
  }

  return query
} 

func fail(w http.ResponseWriter, message string) {
  IpLogger.Error(message)
  http.Error(w, message, http.StatusBadRequest)
}