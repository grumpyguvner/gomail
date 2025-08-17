package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// SPF metrics
	SPFPass = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_spf_pass_total",
		Help: "Total number of SPF pass results",
	})

	SPFFail = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_spf_fail_total",
		Help: "Total number of SPF fail results",
	})

	SPFSoftFail = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_spf_softfail_total",
		Help: "Total number of SPF softfail results",
	})

	SPFNeutral = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_spf_neutral_total",
		Help: "Total number of SPF neutral results",
	})

	SPFNone = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_spf_none_total",
		Help: "Total number of SPF none results",
	})

	SPFLookupErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_spf_lookup_errors_total",
		Help: "Total number of SPF lookup errors",
	})

	// DKIM metrics
	DKIMPass = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dkim_pass_total",
		Help: "Total number of DKIM pass results",
	})

	DKIMFail = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dkim_fail_total",
		Help: "Total number of DKIM fail results",
	})

	DKIMNone = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dkim_none_total",
		Help: "Total number of messages without DKIM signatures",
	})

	DKIMPermError = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dkim_permerror_total",
		Help: "Total number of DKIM permanent errors",
	})

	DKIMTempError = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dkim_temperror_total",
		Help: "Total number of DKIM temporary errors",
	})

	DKIMVerifyErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dkim_verify_errors_total",
		Help: "Total number of DKIM verification errors",
	})

	DKIMSigned = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dkim_signed_total",
		Help: "Total number of messages signed with DKIM",
	})

	DKIMSignErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dkim_sign_errors_total",
		Help: "Total number of DKIM signing errors",
	})

	// DMARC metrics
	DMARCPass = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dmarc_pass_total",
		Help: "Total number of DMARC pass results",
	})

	DMARCFail = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dmarc_fail_total",
		Help: "Total number of DMARC fail results",
	})

	DMARCNone = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dmarc_none_total",
		Help: "Total number of DMARC none results",
	})

	DMARCLookupErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dmarc_lookup_errors_total",
		Help: "Total number of DMARC lookup errors",
	})

	DMARCReportPass = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dmarc_report_pass_total",
		Help: "Total number of DMARC pass results recorded for reporting",
	})

	DMARCReportFail = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dmarc_report_fail_total",
		Help: "Total number of DMARC fail results recorded for reporting",
	})

	// Email action metrics
	EmailsQuarantined = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gomail_emails_quarantined_total",
			Help: "Total number of emails quarantined",
		},
		[]string{"reason"},
	)

	EmailsRejected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gomail_emails_rejected_total",
			Help: "Total number of emails rejected",
		},
		[]string{"reason"},
	)
)

// initAuthMetrics initializes authentication metrics
func initAuthMetrics() {
	// SPF metrics
	prometheus.MustRegister(SPFPass)
	prometheus.MustRegister(SPFFail)
	prometheus.MustRegister(SPFSoftFail)
	prometheus.MustRegister(SPFNeutral)
	prometheus.MustRegister(SPFNone)
	prometheus.MustRegister(SPFLookupErrors)

	// DKIM metrics
	prometheus.MustRegister(DKIMPass)
	prometheus.MustRegister(DKIMFail)
	prometheus.MustRegister(DKIMNone)
	prometheus.MustRegister(DKIMPermError)
	prometheus.MustRegister(DKIMTempError)
	prometheus.MustRegister(DKIMVerifyErrors)
	prometheus.MustRegister(DKIMSigned)
	prometheus.MustRegister(DKIMSignErrors)

	// DMARC metrics
	prometheus.MustRegister(DMARCPass)
	prometheus.MustRegister(DMARCFail)
	prometheus.MustRegister(DMARCNone)
	prometheus.MustRegister(DMARCLookupErrors)
	prometheus.MustRegister(DMARCReportPass)
	prometheus.MustRegister(DMARCReportFail)

	// Email action metrics
	prometheus.MustRegister(EmailsQuarantined)
	prometheus.MustRegister(EmailsRejected)
}

// resetAuthMetrics resets authentication metrics
func resetAuthMetrics() {
	// SPF metrics
	SPFPass = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_spf_pass_total",
		Help: "Total number of SPF pass results",
	})
	SPFFail = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_spf_fail_total",
		Help: "Total number of SPF fail results",
	})
	SPFSoftFail = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_spf_softfail_total",
		Help: "Total number of SPF softfail results",
	})
	SPFNeutral = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_spf_neutral_total",
		Help: "Total number of SPF neutral results",
	})
	SPFNone = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_spf_none_total",
		Help: "Total number of SPF none results",
	})
	SPFLookupErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_spf_lookup_errors_total",
		Help: "Total number of SPF lookup errors",
	})

	// DKIM metrics
	DKIMPass = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dkim_pass_total",
		Help: "Total number of DKIM pass results",
	})
	DKIMFail = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dkim_fail_total",
		Help: "Total number of DKIM fail results",
	})
	DKIMNone = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dkim_none_total",
		Help: "Total number of messages without DKIM signatures",
	})
	DKIMPermError = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dkim_permerror_total",
		Help: "Total number of DKIM permanent errors",
	})
	DKIMTempError = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dkim_temperror_total",
		Help: "Total number of DKIM temporary errors",
	})
	DKIMVerifyErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dkim_verify_errors_total",
		Help: "Total number of DKIM verification errors",
	})
	DKIMSigned = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dkim_signed_total",
		Help: "Total number of messages signed with DKIM",
	})
	DKIMSignErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dkim_sign_errors_total",
		Help: "Total number of DKIM signing errors",
	})

	// DMARC metrics
	DMARCPass = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dmarc_pass_total",
		Help: "Total number of DMARC pass results",
	})
	DMARCFail = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dmarc_fail_total",
		Help: "Total number of DMARC fail results",
	})
	DMARCNone = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dmarc_none_total",
		Help: "Total number of DMARC none results",
	})
	DMARCLookupErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dmarc_lookup_errors_total",
		Help: "Total number of DMARC lookup errors",
	})
	DMARCReportPass = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dmarc_report_pass_total",
		Help: "Total number of DMARC pass results recorded for reporting",
	})
	DMARCReportFail = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_dmarc_report_fail_total",
		Help: "Total number of DMARC fail results recorded for reporting",
	})

	// Email action metrics
	EmailsQuarantined = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gomail_emails_quarantined_total",
			Help: "Total number of emails quarantined",
		},
		[]string{"reason"},
	)
	EmailsRejected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gomail_emails_rejected_total",
			Help: "Total number of emails rejected",
		},
		[]string{"reason"},
	)
}
