package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/go-envparse"
	"github.com/ochinchina/go-ini"
	log "github.com/sirupsen/logrus"
)

// Config memory representation of supervisor configuration file
type Config struct {
	configFile string
	// mapping between the section name and configuration entry
	entries map[string]*Entry
}

// NewEntry creates configuration entry
func NewEntry(configDir string) *Entry {
	return &Entry{configDir, "", "", make(map[string]string)}
}

// NewConfig creates Config object
func NewConfig(configFile string) *Config {
	return &Config{configFile, make(map[string]*Entry)}
}

// create a new entry or return the already-exist entry
func (c *Config) createEntry(name string, configDir string) *Entry {
	entry, ok := c.entries[name]

	if !ok {
		entry = NewEntry(configDir)
		c.entries[name] = entry
	}
	return entry
}

// Load the configuration and return loaded programs
func (c *Config) Load() ([]string, error) {
	myini := ini.NewIni()
	log.WithFields(log.Fields{"file": c.configFile}).Info("load configuration from file")
	myini.LoadFile(c.configFile)

	includeFiles := c.getIncludeFiles(myini)
	for _, f := range includeFiles {
		log.WithFields(log.Fields{"file": f}).Info("load configuration from file")
		myini.LoadFile(f)
	}
	return c.parse(myini), nil
}

// GetConfigFileDir returns directory of zssld configuration file
func (c *Config) GetConfigFileDir() string {
	return filepath.Dir(c.configFile)
}

// GetUnixHTTPServer returns unix_http_server configuration section
func (c *Config) GetUnixHTTPServer() (*Entry, bool) {
	entry, ok := c.entries["unix_http_server"]

	return entry, ok
}

// GetZssld returns "zssld" configuration section
func (c *Config) GetZssld() (*Entry, bool) {
	entry, ok := c.entries["zssld"]
	return entry, ok
}

// GetInetHTTPServer returns inet_http_server configuration section
func (c *Config) GetInetHTTPServer() (*Entry, bool) {
	entry, ok := c.entries["inet_http_server"]
	return entry, ok
}

// GetZsslctl returns "zsslctl" configuration section
func (c *Config) GetZsslctl() (*Entry, bool) {
	entry, ok := c.entries["zsslctl"]
	return entry, ok
}

// GetZsslServer
func (c *Config) GetZsslServer() (*Entry, bool) {
	entry, ok := c.entries["zssl-server"]
	return entry, ok
}

// GetEntries returns configuration entries by filter
func (c *Config) GetEntries(filterFunc func(entry *Entry) bool) []*Entry {
	result := make([]*Entry, 0)
	for _, entry := range c.entries {
		if filterFunc(entry) {
			result = append(result, entry)
		}
	}
	return result
}

// String converts configuration to the string
func (c *Config) String() string {
	buf := bytes.NewBuffer(make([]byte, 0))
	for _, v := range c.entries {
		fmt.Fprintf(buf, "[%s]\n", v.Name)
		fmt.Fprintf(buf, "%s\n", v.String())
	}
	return buf.String()
}

// GetPrograms returns configuration entries of all programs
func (c *Config) GetPrograms() []*Entry {
	programs := c.GetEntries(func(entry *Entry) bool {
		return entry.IsProgram()
	})

	return programs
}

// GetEventListeners returns configuration entries of event listeners
func (c *Config) GetEventListeners() []*Entry {
	eventListeners := c.GetEntries(func(entry *Entry) bool {
		return entry.IsEventListener()
	})

	return eventListeners
}

// GetProgramNames returns slice with all program names
func (c *Config) GetProgramNames() []string {
	result := make([]string, 0)
	programs := c.GetPrograms()

	// programs = sortProgram(programs)
	for _, entry := range programs {
		result = append(result, entry.GetProgramName())
	}
	return result
}

// GetProgram returns the program configuration entry or nil
func (c *Config) GetProgram(name string) *Entry {
	for _, entry := range c.entries {
		if entry.IsProgram() && entry.GetProgramName() == name {
			return entry
		}
	}
	return nil
}

func (c *Config) getIncludeFiles(cfg *ini.Ini) []string {
	result := make([]string, 0)
	if includeSection, err := cfg.GetSection("include"); err == nil {
		key, err := includeSection.GetValue("files")
		if err == nil {
			env := NewStringExpression("here", c.GetConfigFileDir())
			files := strings.Fields(key)
			for _, fRaw := range files {
				dir := c.GetConfigFileDir()
				f, err := env.Eval(fRaw)
				if err != nil {
					continue
				}
				if filepath.IsAbs(f) {
					dir = filepath.Dir(f)
				} else {
					dir = filepath.Join(c.GetConfigFileDir(), filepath.Dir(f))
				}
				fileInfos, err := ioutil.ReadDir(dir)
				if err == nil {
					goPattern := toRegexp(filepath.Base(f))
					for _, fileInfo := range fileInfos {
						if matched, err := regexp.MatchString(goPattern, fileInfo.Name()); matched && err == nil {
							result = append(result, filepath.Join(dir, fileInfo.Name()))
						}
					}
				}

			}
		}
	}
	return result
}

func (c *Config) parse(cfg *ini.Ini) []string {
	c.setProgramDefaultParams(cfg)
	loadedPrograms := c.parseProgram(cfg)

	// parse non-group, non-program and non-eventlistener sections
	for _, section := range cfg.Sections() {
		// 过滤组，程序，和监听
		if !strings.HasPrefix(section.Name, "group:") && !strings.HasPrefix(section.Name, "program:") && !strings.HasPrefix(section.Name, "eventlistener:") {
			entry := c.createEntry(section.Name, c.GetConfigFileDir())
			c.entries[section.Name] = entry
			entry.parse(section)
		}
	}
	return loadedPrograms
}

// set the default parameters of programs
func (c *Config) setProgramDefaultParams(cfg *ini.Ini) {
	programDefaultSection, err := cfg.GetSection("program-default")
	if err == nil {
		for _, section := range cfg.Sections() {
			if section.Name == "program-default" || !strings.HasPrefix(section.Name, "program:") {
				continue
			}
			for _, key := range programDefaultSection.Keys() {
				if !section.HasKey(key.Name()) {
					section.Add(key.Name(), key.ValueWithDefault(""))
				}
			}

		}
	}
}

func (c *Config) isProgramOrEventListener(section *ini.Section) (bool, string) {
	// check if it is a program or event listener section
	isProgram := strings.HasPrefix(section.Name, "program:")
	isEventListener := strings.HasPrefix(section.Name, "eventlistener:")
	prefix := ""
	if isProgram {
		prefix = "program:"
	} else if isEventListener {
		prefix = "eventlistener:"
	}
	return isProgram || isEventListener, prefix
}

// parse the sections starts with "program:" prefix.
//
// Return all the parsed program names in the ini
func (c *Config) parseProgram(cfg *ini.Ini) []string {
	loadedPrograms := make([]string, 0)
	for _, section := range cfg.Sections() {
		programOrEventListener, prefix := c.isProgramOrEventListener(section)

		// if it is program or event listener
		if programOrEventListener {
			// get the number of processes
			numProcs, err := section.GetInt("numprocs")
			programName := section.Name[len(prefix):]
			if err != nil {
				numProcs = 1
			}
			procName, err := section.GetValue("process_name")
			if numProcs > 1 {
				if err != nil || strings.Index(procName, "%(process_num)") == -1 {
					log.WithFields(log.Fields{
						"numprocs":     numProcs,
						"process_name": procName,
					}).Error("no process_num in process name")
				}
			}
			originalProcName := programName
			if err == nil {
				originalProcName = procName
			}

			originalCmd := section.GetValueWithDefault("command", "")

			for i := 1; i <= numProcs; i++ {
				envs := NewStringExpression("program_name", programName,
					"process_num", fmt.Sprintf("%d", i),
					"here", c.GetConfigFileDir())
				envValue, err := section.GetValue("environment")
				if err == nil {
					for k, v := range *parseEnv(envValue) {
						envs.Add(fmt.Sprintf("ENV_%s", k), v)
					}
				}
				cmd, err := envs.Eval(originalCmd)
				if err != nil {
					log.WithFields(log.Fields{
						log.ErrorKey: err,
						"program":    programName,
					}).Error("get envs failed")
					continue
				}
				section.Add("command", cmd)

				procName, err := envs.Eval(originalProcName)
				if err != nil {
					log.WithFields(log.Fields{
						log.ErrorKey: err,
						"program":    programName,
					}).Error("get envs failed")
					continue
				}

				section.Add("process_name", procName)
				section.Add("numprocs_start", fmt.Sprintf("%d", i-1))
				section.Add("process_num", fmt.Sprintf("%d", i))
				entry := c.createEntry(procName, c.GetConfigFileDir())
				entry.parse(section)
				entry.Name = prefix + procName
				loadedPrograms = append(loadedPrograms, procName)
			}
		}
	}
	return loadedPrograms
}

func parseEnv(s string) *map[string]string {
	result := make(map[string]string)
	start := 0
	n := len(s)
	var i int
	for {
		// find the '='
		for i = start; i < n && s[i] != '='; {
			i++
		}
		key := s[start:i]
		start = i + 1
		if s[start] == '"' {
			for i = start + 1; i < n && s[i] != '"'; {
				i++
			}
			if i < n {
				result[strings.TrimSpace(key)] = strings.TrimSpace(s[start+1 : i])
			}
			if i+1 < n && s[i+1] == ',' {
				start = i + 2
			} else {
				break
			}
		} else {
			for i = start; i < n && s[i] != ','; {
				i++
			}
			if i < n {
				result[strings.TrimSpace(key)] = strings.TrimSpace(s[start:i])
				start = i + 1
			} else {
				result[strings.TrimSpace(key)] = strings.TrimSpace(s[start:])
				break
			}
		}
	}

	return &result
}

func parseEnvFiles(s string) *map[string]string {
	result := make(map[string]string)
	for _, envFilePath := range strings.Split(s, ",") {
		envFilePath = strings.TrimSpace(envFilePath)
		f, err := os.Open(envFilePath)
		if err != nil {
			log.WithFields(log.Fields{
				log.ErrorKey: err,
				"file":       envFilePath,
			}).Error("Read file failed: " + envFilePath)
			continue
		}
		r, err := envparse.Parse(f)
		if err != nil {
			log.WithFields(log.Fields{
				log.ErrorKey: err,
				"file":       envFilePath,
			}).Error("Parse env file failed: " + envFilePath)
			continue
		}
		for k, v := range r {
			result[k] = v
		}
	}
	return &result
}

// convert supervisor file pattern to the go regrexp
func toRegexp(pattern string) string {
	tmp := strings.Split(pattern, ".")
	for i, t := range tmp {
		s := strings.Replace(t, "*", ".*", -1)
		tmp[i] = strings.Replace(s, "?", ".", -1)
	}
	return strings.Join(tmp, "\\.")
}
