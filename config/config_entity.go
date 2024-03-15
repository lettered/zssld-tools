package config

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ochinchina/go-ini"
	log "github.com/sirupsen/logrus"
)

// Entry standards for a configuration section in supervisor configuration file
type Entry struct {
	ConfigDir string
	Group     string
	Name      string
	keyValues map[string]string
}

// GetName returns true if this is a section
func (c *Entry) GetName() string {
	return c.Name
}

// IsProgram returns true if this is a program section
func (c *Entry) IsProgram() bool {
	return strings.HasPrefix(c.Name, "program:")
}

// GetProgramName returns program name
func (c *Entry) GetProgramName() string {
	if strings.HasPrefix(c.Name, "program:") {
		return c.Name[len("program:"):]
	}
	return ""
}

// IsEventListener returns true if this section is for event listener
func (c *Entry) IsEventListener() bool {
	return strings.HasPrefix(c.Name, "eventlistener:")
}

// GetEventListenerName returns event listener name
func (c *Entry) GetEventListenerName() string {
	if strings.HasPrefix(c.Name, "eventlistener:") {
		return c.Name[len("eventlistener:"):]
	}
	return ""
}

// IsGroup returns true if it is group section
func (c *Entry) IsGroup() bool {
	return strings.HasPrefix(c.Name, "group:")
}

// GetGroupName returns group name if entry is a group
func (c *Entry) GetGroupName() string {
	if strings.HasPrefix(c.Name, "group:") {
		return c.Name[len("group:"):]
	}
	return ""
}

// GetPrograms returns slice with programs from the group
func (c *Entry) GetPrograms() []string {
	if c.IsGroup() {
		r := c.GetStringArray("programs", ",")
		for i, p := range r {
			r[i] = strings.TrimSpace(p)
		}
		return r
	}
	return make([]string, 0)
}

func (c *Entry) setGroup(group string) {
	c.Group = group
}

// String dumps configuration as a string
func (c *Entry) String() string {
	buf := bytes.NewBuffer(make([]byte, 0))
	for k, v := range c.keyValues {
		fmt.Fprintf(buf, "%s=%s\n", k, v)
	}
	return buf.String()
}

// GetBool gets value of key as bool
func (c *Entry) GetBool(key string, defValue bool) bool {
	value, ok := c.keyValues[key]

	if ok {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
	}
	return defValue
}

// HasParameter checks if key (parameter) has value
func (c *Entry) HasParameter(key string) bool {
	_, ok := c.keyValues[key]
	return ok
}

func toInt(s string, factor int, defValue int) int {
	i, err := strconv.Atoi(s)
	if err == nil {
		return i * factor
	}
	return defValue
}

// GetInt gets value of the key as int
func (c *Entry) GetInt(key string, defValue int) int {
	value, ok := c.keyValues[key]

	if ok {
		return toInt(value, 1, defValue)
	}
	return defValue
}

// GetEnv returns slice of strings with keys separated from values by single "=". An environment string example:
//
//	environment = A="env 1",B="this is a test"
func (c *Entry) GetEnv(key string) []string {
	value, ok := c.keyValues[key]
	result := make([]string, 0)

	if ok {
		for k, v := range *parseEnv(value) {
			tmp, err := NewStringExpression("program_name", c.GetProgramName(),
				"process_num", c.GetString("process_num", "0"),
				"group_name", c.GetGroupName(),
				"here", c.ConfigDir).Eval(fmt.Sprintf("%s=%s", k, v))
			if err == nil {
				result = append(result, tmp)
			}
		}
	}

	return result
}

// GetEnvFromFiles returns slice of strings with keys separated from values by single "=". An envFile example:
//
//	envFiles = global.env,prod.env
//
// cat global.env
// varA=valueA
func (c *Entry) GetEnvFromFiles(key string) []string {
	value, ok := c.keyValues[key]
	result := make([]string, 0)

	if ok {
		for k, v := range *parseEnvFiles(value) {
			tmp, err := NewStringExpression("program_name", c.GetProgramName(),
				"process_num", c.GetString("process_num", "0"),
				"group_name", c.GetGroupName(),
				"here", c.ConfigDir).Eval(fmt.Sprintf("%s=%s", k, v))
			if err == nil {
				result = append(result, tmp)
			}
		}
	}

	return result
}

// GetString returns value of the key as a string
func (c *Entry) GetString(key string, defValue string) string {
	s, ok := c.keyValues[key]

	if ok {
		env := NewStringExpression("here", c.ConfigDir)
		repS, err := env.Eval(s)
		if err == nil {
			return repS
		}
		log.WithFields(log.Fields{
			log.ErrorKey: err,
			"program":    c.GetProgramName(),
			"key":        key,
		}).Warn("Unable to parse expression")
	}
	return defValue
}

func (c *Entry) SetString(key string, value string) {
	c.keyValues[key] = strings.TrimSpace(value)
}

// GetStringExpression returns value of key as a string and attempts to parse it with StringExpression
func (c *Entry) GetStringExpression(key string, defValue string) string {
	s, ok := c.keyValues[key]
	if !ok || s == "" {
		return ""
	}

	hostName, err := os.Hostname()
	if err != nil {
		hostName = "Unknown"
	}
	result, err := NewStringExpression("program_name", c.GetProgramName(),
		"process_num", c.GetString("process_num", "0"),
		"group_name", c.GetGroupName(),
		"here", c.ConfigDir,
		"host_node_name", hostName).Eval(s)

	if err != nil {
		log.WithFields(log.Fields{
			log.ErrorKey: err,
			"program":    c.GetProgramName(),
			"key":        key,
		}).Warn("unable to parse expression")
		return s
	}

	return result
}

// GetStringArray gets string value and split it with "sep" to slice
func (c *Entry) GetStringArray(key string, sep string) []string {
	s, ok := c.keyValues[key]

	if ok {
		return strings.Split(s, sep)
	}
	return make([]string, 0)
}

// GetBytes returns value of the key as bytes setting.
//
//	logSize=1MB
//	logSize=1GB
//	logSize=1KB
//	logSize=1024
func (c *Entry) GetBytes(key string, defValue int) int {
	v, ok := c.keyValues[key]

	if ok {
		if len(v) > 2 {
			lastTwoBytes := v[len(v)-2:]
			if lastTwoBytes == "MB" {
				return toInt(v[:len(v)-2], 1024*1024, defValue)
			} else if lastTwoBytes == "GB" {
				return toInt(v[:len(v)-2], 1024*1024*1024, defValue)
			} else if lastTwoBytes == "KB" {
				return toInt(v[:len(v)-2], 1024, defValue)
			}
		}
		return toInt(v, 1, defValue)
	}
	return defValue
}

func (c *Entry) parse(section *ini.Section) {
	c.Name = section.Name
	for _, key := range section.Keys() {
		c.keyValues[key.Name()] = strings.TrimSpace(key.ValueWithDefault(""))
	}
}
