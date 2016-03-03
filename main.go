package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Whitelist map[string]struct{}

func (w Whitelist) Add(username string) {
	whitelist[username] = struct{}{}
}

func (w Whitelist) Contains(username string) bool {
	_, ok := whitelist[username]
	return ok
}

const (
	slackTokenEnv = "SLACK_TOKEN"
	finTokenEnv   = "FIN_TOKEN"
	whitelistEnv  = "WHITELIST"
)

var (
	requiredEnv = []string{slackTokenEnv, finTokenEnv, whitelistEnv}
	slackToken  = os.Getenv(slackTokenEnv)
	finToken    = os.Getenv(finTokenEnv)
	whitelist   = Whitelist{}
)

func makeWhitelist(usernames ...string) Whitelist {
	w := Whitelist{}

	for _, username := range usernames {
		w.Add(username)
	}

	return w
}

func checkEnvKeys(env ...string) (errs []error) {
	for _, key := range env {
		if os.Getenv(key) == "" {
			err := fmt.Errorf("%s must be set", key)
			errs = append(errs, err)
		}
	}

	return errs
}

func doFinRequest(text string) (*http.Response, error) {
	form := url.Values{}
	form.Add("abbreviated", "0")
	form.Add("local_time", "todo")
	form.Add("text", text)
	form.Add("thread_id", "0")
	form.Add("token", finToken)

	return http.PostForm("https://www.fin.com/api/messages", form)
}

func handler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	givenToken := r.FormValue("token")
	username := r.FormValue("user_name")
	text := r.FormValue("text")

	if givenToken != slackToken {
		err := fmt.Errorf("Invalid Slack token")
		http.Error(w, err.Error(), 400)
		return
	}

	if !whitelist.Contains(username) {
		err := fmt.Errorf("%s is not a whitelisted user", username)
		http.Error(w, err.Error(), 400)
		return
	}

	_, err := doFinRequest(text)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	fmt.Fprintf(w, "Successfully sent to Fin.\n\n> %s", text)
}

func main() {
	if errs := checkEnvKeys(requiredEnv...); len(errs) != 0 {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err.Error())
		}
		os.Exit(1)
	}

	parsedWhitelistEnv := strings.Split(os.Getenv(whitelistEnv), ",")
	whitelist = makeWhitelist(parsedWhitelistEnv...)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	http.HandleFunc("/", handler)

	fmt.Printf("Started server on port %s...\n", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}
