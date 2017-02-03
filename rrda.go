/*****************************************************************************/
/*                                                                           */
/* RRDA (RRDA REST DNS API) 1.02                                             */
/* Copyright (c) 2012-2016, Frederic Cambus                                  */
/* http://www.statdns.com                                                    */
/*                                                                           */
/* Created: 2012-03-11                                                       */
/* Last Updated: 2016-07-18                                                  */
/*                                                                           */
/* RRDA is released under the BSD 2-Clause license.                          */
/* See LICENSE file for details.                                             */
/*                                                                           */
/*****************************************************************************/

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bmizerany/pat"
	"github.com/miekg/dns"
	"golang.org/x/net/idna"
	"io"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"strings"
)

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

Redo:
	in, _, err := c.Exchange(m, server) // Second return value is RTT, not used for now

	if err == nil {
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
	} else if err == dns.ErrTruncated {
		c.Net = "tcp"
		goto Redo
	} else {
		error(w, 500, 501, "DNS server could not be reached")
	}
}

// Handler for DNS queries
func query(w http.ResponseWriter, r *http.Request) {
	server := r.URL.Query().Get(":server")
	domain := dns.Fqdn(r.URL.Query().Get(":domain"))
	querytype := r.URL.Query().Get(":querytype")

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
	server := r.URL.Query().Get(":server")
	ip := r.URL.Query().Get(":ip")

	if arpa, err := dns.ReverseAddr(ip); err == nil { // Valid IP address (IPv4 or IPv6)
		resolve(w, r, server, arpa, dns.TypePTR)
	} else {
		error(w, 400, 403, "Input string is not a valid IP address")
	}
}

func main() {
	header := "-------------------------------------------------------------------------------\n        RRDA (RRDA REST DNS API) 1.02 (c) by Frederic Cambus 2012-2016\n-------------------------------------------------------------------------------"

	fastcgi := flag.Bool("fastcgi", false, "Enable FastCGI mode")
	host := flag.String("host", "127.0.0.1", "Set the server host")
	port := flag.String("port", "8080", "Set the server port")

	flag.Usage = func() {
		fmt.Println(header)
		fmt.Println("\nUSAGE:")
		flag.PrintDefaults()
	}
	flag.Parse()

	fmt.Println(header)

	fmt.Println("\nListening on :", *host+":"+*port)

	m := pat.New()
	m.Get("/:server/x/:ip", http.HandlerFunc(ptr))
	m.Get("/:server/:domain/:querytype", http.HandlerFunc(query))

	if *fastcgi {
		listener, _ := net.Listen("tcp", *host+":"+*port)

		if err := fcgi.Serve(listener, m); err != nil {
			fmt.Println("\nERROR:", err)
			os.Exit(1)
		}
	} else {
		if err := http.ListenAndServe(*host+":"+*port, m); err != nil {
			fmt.Println("\nERROR:", err)
			os.Exit(1)
		}
	}
}
