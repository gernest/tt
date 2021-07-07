package tseries

import (
	"net/http"
	"strconv"

	"github.com/gernest/tt/pkg/meta"
	"github.com/gernest/tt/pkg/xhttp/xlabels"
	ua "github.com/mileusna/useragent"
	"github.com/prometheus/client_golang/prometheus"
)

func Labels(
	r *http.Request,
	code int,
	info *meta.Metrics,
) prometheus.Labels {
	agent := ua.Parse(r.UserAgent())
	return prometheus.Labels{
		xlabels.Route:   info.Route,
		xlabels.Service: info.Service,
		xlabels.Target:  info.Target,
		xlabels.Host:    info.VirtualHost,
		xlabels.Code:    sanitizeCode(code),
		xlabels.Method:  sanitizeMethod(r.Method),
		xlabels.Path:    r.URL.Path,

		// user agent labels
		xlabels.UserAgentName:      agent.Name,
		xlabels.UserAgentVersion:   agent.Version,
		xlabels.UserAgentOs:        agent.OS,
		xlabels.UserAgentOsVersion: agent.OSVersion,
		xlabels.UserAgentDevice:    agent.Device,
		xlabels.UserAgentMobile:    strconv.FormatBool(agent.Mobile),
		xlabels.UserAgentTablet:    strconv.FormatBool(agent.Tablet),
		xlabels.UserAgentDesktop:   strconv.FormatBool(agent.Desktop),
	}
}
