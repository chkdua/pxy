package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	u "github.com/jolav/codetabs/_utils"
)

type proxy struct {
	quest string
}

func Router(w http.ResponseWriter, r *http.Request) {
	params := strings.Split(strings.ToLower(r.URL.Path), "/")
	path := params[1:len(params)]
	if path[len(path)-1] == "" { // remove last empty slot after /
		path = path[:len(path)-1]
	}
	if len(path) < 2 || path[0] != "v1" {
		u.BadRequest(w, r)
		return
	}

	p := newProxy(false)
	r.ParseForm()
	p.quest = r.Form.Get("quest")
	if p.quest == "" || len(path) != 2 {
		u.BadRequest(w, r)
		return
	}
	p.doProxyRequest(w, r)
}

func (p *proxy) doProxyRequest(w http.ResponseWriter, r *http.Request) {
	p.quest = "http://" + u.RemoveProtocolFromURL(p.quest)
	var data interface{}
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := netClient.Get(p.quest)
	if err != nil {
		log.Printf("Error PROXY => %s", err)
		msg := fmt.Sprintf("%s is not a valid resource", p.quest)
		u.ErrorResponse(w, msg)
		return
	}
	defer resp.Body.Close()

	contentType := ""
	if len(resp.Header["Content-Type"]) > 0 {
		contentType = resp.Header["Content-Type"][0]
	}

	switch {
	case strings.Contains(contentType, "application/json"):
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			u.ErrorResponse(w, "Failed to decode JSON response")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		u.SendJSONToClient(w, data, 200)
		return

	case strings.Contains(contentType, "application/xml"):
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			u.ErrorResponse(w, "Failed to read XML response")
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.Write(bodyBytes)
		return

	case strings.Contains(contentType, "text/"):
		w.Header().Set("Content-Type", "text/plain")
		io.Copy(w, resp.Body)
		return

	default:
		w.Header().Set("Content-Type", "text/plain")
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			w.Write([]byte(fmt.Sprintf("%v", string(line))))
			if err != nil {
				break
			}
		}
	}
}

func newProxy(test bool) proxy {
	return proxy{
		quest: "",
	}
}
