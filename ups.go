package main

// User-Provided Service

import (
	"errors"
	"reflect"
)

// Credentials we care about to be loaded from the UPS.
type Credentials struct {
	APIEndpoint       string `json:"api_endpoint"`
	Debug             bool   `json:"debug"`
	Password          string `json:"password"`
	SkipSslValidation bool   `json:"skip_ssl_validation"`
	StatsdEndpoint    string `json:"statsd_endpoint"`
	StatsdPrefix      string `json:"statsd_prefix"`
	Username          string `json:"username"`
	PrefixJob         bool   `json:"prefix_job"`
	UpdateFrequency   int64  `json:"update_frequency"`
}

// UPSBind may exist in many appearances.
type UPSBind struct {
	Credentials Credentials `json:"credentials"`
	Generated   bool        `json:"-"`
	Name        string      `json:"name"`

	// Label          string      `json:"label"`
	// SyslogDrainURL string      `json:"syslog_drain_url"`
	// Tags           []string    `json:"tags"`
	// VolumeMounts   []string    `json:"volume_mounts"`
}

// UPS should consist of the CF Custom User Provided Services.
type UPS struct {
	UserProvided []UPSBind `json:"user-provided"`
}

// First acquired entry in our UPS slice will be returned.
func (c UPS) First() UPSBind {
	if len(c.UserProvided) == 0 {
		c.UserProvided = append(c.UserProvided, UPSBind{Generated: true})
	}

	return c.UserProvided[0]
}

// GetStringValue will attempt to obtain the Credential from our struct.
// It should fallback into default value if the one in the struct is empty.
func (c Credentials) GetStringValue(param string, defaultVal string) string {
	stored, err := c.getField(&c, param)

	if err != nil {
		return defaultVal
	}

	return stored.(string)
}

// GetBoolValue will attempt to obtain the Credential from our struct.
// It should fallback into default value if the one in the struct is empty.
func (c Credentials) GetBoolValue(param string, defaultVal bool) bool {
	stored, err := c.getField(&c, param)
	if err != nil {
		return defaultVal
	}

	return stored.(bool)
}

func (c Credentials) getField(v *Credentials, field string) (interface{}, error) {
	r := reflect.ValueOf(v)
	f := reflect.Indirect(r).FieldByName(field)

	if f.IsValid() {
		return f.Interface(), nil
	}

	return nil, errors.New("Unassigned parameter has been called.")
}
