/*
 * RRDA (RRDA REST DNS API) 1.2.0
 * Copyright (c) 2012-2022, Frederic Cambus
 * https://www.statdns.com
 *
 * Created: 2012-03-11
 * Last Updated: 2022-11-21
 *
 * RRDA is released under the BSD 2-Clause license.
 * See LICENSE file for details.
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"
	"golang.org/x/net/idna"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var timeout_ms int

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Question struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Class string `json:"class"`
}

type Section struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Class    string `json:"class"`
	Ttl      uint32 `json:"ttl"`
	Rdlength uint16 `json:"rdlength"`
	Rdata    string `json:"rdata"`
}

type Message struct {
	Question   []*Question `json:"question"`
	Answer     []*Section  `json:"answer"`
	Authority  []*Section  `json:"authority,omitempty"`
	Additional []*Section  `json:"additional,omitempty"`
}

// Return rdata
func rdata(RR dns.RR) string {
	return strings.Replace(RR.String(), RR.Header().String(), "", -1)
}

// Return an HTTP Error along with a JSON-encoded error message
func error(w http.ResponseWriter, status int, code int, message string) {
	if output, err := json.Marshal(Error{Code: code, Message: message}); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprintln(w, string(output))
	}
}

// Generate JSON output
func jsonify(w http.ResponseWriter, r *http.Request, question []dns.Question, answer []dns.RR, authority []dns.RR, additional []dns.RR) {
	var answerArray, authorityArray, additionalArray []*Section

	callback := r.URL.Query().Get("callback")

	for _, answer := range answer {
		answerArray = append(answerArray, &Section{answer.Header().Name, dns.TypeToString[answer.Header().Rrtype], dns.ClassToString[answer.Header().Class], answer.Header().Ttl, answer.Header().Rdlength, rdata(answer)})
	}

	for _, authority := range authority {
		authorityArray = append(authorityArray, &Section{authority.Header().Name, dns.TypeToString[authority.Header().Rrtype], dns.ClassToString[authority.Header().Class], authority.Header().Ttl, authority.Header().Rdlength, rdata(authority)})
	}

	for _, additional := range additional {
		additionalArray = append(additionalArray, &Section{additional.Header().Name, dns.TypeToString[additional.Header().Rrtype], dns.ClassToString[additional.Header().Class], additional.Header().Ttl, additional.Header().Rdlength, rdata(additional)})
	}

	if json, err := json.MarshalIndent(Message{[]*Question{&Question{question[0].Name, dns.TypeToString[question[0].Qtype], dns.ClassToString[question[0].Qclass]}}, answerArray, authorityArray, additionalArray}, "", "    "); err == nil {
		if callback != "" {
			io.WriteString(w, callback+"("+string(json)+");")
		} else {
			io.WriteString(w, string(json))
		}
	}
}

// Perform DNS resolution
func resolve(w http.ResponseWriter, r *http.Request, server string, domain string, querytype uint16) {
	m := new(dns.Msg)
	m.SetQuestion(domain, querytype)
	m.MsgHdr.RecursionDesired = true

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	c := new(dns.Client)
	c.Dialer = &net.Dialer{
		Timeout: time.Duration(timeout_ms) * time.Millisecond,
	}

Redo:
	if in, _, err := c.Exchange(m, server); err == nil { // Second return value is RTT, not used for now
		if in.MsgHdr.Truncated {
			c.Net = "tcp"
			goto Redo
		}

		switch in.MsgHdr.Rcode {
		case dns.RcodeServerFailure:
			error(w, 500, 502, "The name server encountered an internal failure while processing this request (SERVFAIL)")
		case dns.RcodeNameError:
			error(w, 500, 503, "Some name that ought to exist, does not exist (NXDOMAIN)")
		case dns.RcodeRefused:
			error(w, 500, 505, "The name server refuses to perform the specified operation for policy or security reasons (REFUSED)")
		default:
			jsonify(w, r, in.Question, in.Answer, in.Ns, in.Extra)
		}
	} else {
		error(w, 500, 501, "DNS server could not be reached")
	}
}

// Handler for DNS queries
func query(w http.ResponseWriter, r *http.Request) {
	server := chi.URLParam(r, "server")
	domain := dns.Fqdn(chi.URLParam(r, "domain"))
	querytype := chi.URLParam(r, "querytype")

	if domain, err := idna.ToASCII(domain); err == nil { // Valid domain name (ASCII or IDN)
		if _, isDomain := dns.IsDomainName(domain); isDomain { // Well-formed domain name
			if querytype, ok := dns.StringToType[strings.ToUpper(querytype)]; ok { // Valid DNS query type
				resolve(w, r, server, domain, querytype)
			} else {
				error(w, 400, 404, "Invalid DNS query type")
			}
		} else {
			error(w, 400, 402, "Input string is not a well-formed domain name")
		}
	} else {
		error(w, 400, 401, "Input string could not be parsed")
	}
}

// Handler for reverse DNS queries
func ptr(w http.ResponseWriter, r *http.Request) {
	server := chi.URLParam(r, "server")
	ip := chi.URLParam(r, "ip")

	if arpa, err := dns.ReverseAddr(ip); err == nil { // Valid IP address (IPv4 or IPv6)
		resolve(w, r, server, arpa, dns.TypePTR)
	} else {
		error(w, 400, 403, "Input string is not a valid IP address")
	}
}

func main() {
	host := flag.String("host", "127.0.0.1", "Set the server host")
	port := flag.String("port", "8080", "Set the server port")
	flag.IntVar(&timeout_ms, "timeout", 2000, "Set the query timeout in ms")
	version := flag.Bool("version", false, "Display version")

	flag.Usage = func() {
		fmt.Println("\nUSAGE:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *version {
		fmt.Println("RRDA 1.2.0")
		os.Exit(0)
	}

	address := *host + ":" + *port

	r := chi.NewRouter()
	r.Get("/{server}/x/{ip}", ptr)
	r.Get("/{server}/{domain}/{querytype}", query)

	log.Fatal(http.ListenAndServe(address, r))

	fmt.Println("Listening on:", address)
}
