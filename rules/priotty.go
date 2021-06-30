package rules

import "github.com/gernest/tt/api"

type mask int

const (
	port mask = iota
	portRange
	sni
	method
	exact
	prefix
	regex
)

// Priority calculates the priority of the rule by scoring it and multiplying by
// priorityScore
func Priority(rule *api.Rule, priorityScore int) int {
	return score(rule) * priorityScore
}

func score(r *api.Rule) (total int) {
	switch e := r.Match.(type) {
	case *api.Rule_All:
		for _, v := range e.All.Rules {
			total += score(v)
		}
	case *api.Rule_Any:
		for _, v := range e.Any.Rules {
			total += score(v)
		}
	case *api.Rule_Not:
		total -= score(r)

	case *api.Rule_Tcp:
		switch e.Tcp.Match.(type) {
		case *api.Rule_TCP_Port:
			total += int(port)
		case *api.Rule_TCP_Ports:
			total += int(portRange)
		case *api.Rule_TCP_Sni:
			total += int(sni)
		}
	case *api.Rule_Http:
		switch m := e.Http.Match.(type) {
		case *api.Rule_HTTP_Method_:
			total += int(method)
		case *api.Rule_HTTP_Path:
			switch m.Path.Match.(type) {
			case *api.Rule_StringMatch_Exact:
				total += int(exact)
			case *api.Rule_StringMatch_Prefix:
				total += int(prefix)
			case *api.Rule_StringMatch_Regexp:
				total += int(regex)
			}
		}
	}
	return
}
