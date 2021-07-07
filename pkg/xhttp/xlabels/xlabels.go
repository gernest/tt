package xlabels

const (
	Route            = "route"
	Service          = "service"
	Host             = "host"
	Path             = "req_url_path"
	Method           = "req_method"
	UserAgentName    = "req_ua_name"
	UserAgentVersion = "req_ua_version"
	UserAgentOs      = "req_ua_os"
	UserAgentDevice  = "req_ua_device"
	UserAgentMobile  = "req_ua_mobile"
	UserAgentTablet  = "req_ua_tablet"
	UserAgentDesktop = "req_ua_desktop"
	UserAgentBot     = "req_ua_bot"
	Duration         = "req_duration"
	Error            = "error"
)

var All = []string{
	Route,
	Service,
	Host,
	Path,
	Method,
	UserAgentName,
	UserAgentVersion,
	UserAgentOs,
	UserAgentDevice,
	UserAgentMobile,
	UserAgentTablet,
	UserAgentDesktop,
	UserAgentBot,
	Duration,
	Error,
}
