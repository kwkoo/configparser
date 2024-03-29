package configparser

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
)

type configFile struct {
	subDirs  string
	contents string
}

func ExampleParse() {
	type Config struct {
		Hostname string `env:"HOST" flag:"host" usage:"hostname of the server" mandatory:"true"`
		Port     int    `default:"8080"`
		Async    bool
	}

	c := Config{}

	// Directory for config files are defined in the CONFIGDIR environment
	// variable, the -configdir command line argument, or the /config
	// directory, in that order.
	err := ParseWithDir(&c, RetrieveConfigDirectory("CONFIGDIR", "configdir", "/config"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Hostname: %v\n", c.Hostname)

	// Needed because we are calling flag.Parse() each time we run a test.
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

func TestBasic(t *testing.T) {
	type Config struct {
		Hostname string `env:"HOST" flag:"host" default:"localhost"`
		Port     int    `default:"8080"`
		Async    bool
	}

	tables := []struct {
		flags    []string
		env      []string
		expected Config
		isErr    bool
		stderr   bool
	}{
		{[]string{"-host", "abc", "-port", "8000", "-async"}, []string{"", "", ""}, Config{"abc", 8000, true}, false, false},        // flags set, env not set
		{[]string{}, []string{"def", "7000", "true"}, Config{"def", 7000, true}, false, false},                                      // flags not set, env set
		{[]string{"-host", "ghi"}, []string{"", "6000", ""}, Config{"ghi", 6000, false}, false, false},                              // some flags set, some env set
		{[]string{"-host", "abc", "-port", "8000", "-async"}, []string{"def", "7000", ""}, Config{"def", 7000, true}, false, false}, // both flags and env set, env should override flags
		{[]string{"-host", "abc", "-port", "text", "-async"}, []string{"", "", ""}, Config{}, false, true},                          // integer command line flag parsing error
		{[]string{}, []string{"def", "text", "true"}, Config{"def", 7000, true}, true, false},                                       // environment variable for int field set to non-integer
		{[]string{}, []string{"", "", ""}, Config{"localhost", 8080, false}, false, false},                                          // nothing set, so defaults shoud kick in
		{[]string{"-async"}, []string{"", "", "0"}, Config{"localhost", 8080, false}, false, false},                                 // async flag set, but should be overridden by env var
	}

	for index, table := range tables {
		t.Logf("Testing table %d", index)
		setFlags(table.flags)
		setConfigEnv(table.env)

		// Needed because we are calling flag.Parse() each time we run a test.
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		stderr := new(bytes.Buffer)
		flag.CommandLine.SetOutput(stderr)

		result := Config{}
		err := Parse(&result)

		if table.isErr {
			if err == nil {
				t.Error("Expected an error but did not get it")
			} else {
				t.Logf("Expected an error - got: %v", err)
			}
		}

		if stderr.Len() > 0 {
			if !table.stderr {
				t.Errorf("Test produced unexpected output to stderr: %v", stderr.String())
			}
			continue
		} else {
			if table.stderr {
				t.Error("Test was expected to output to stderr but it did not")
			}
		}

		if err != nil {
			continue
		}

		if result.Hostname != table.expected.Hostname {
			t.Errorf("Expected hostname %v but got %v instead", table.expected.Hostname, result.Hostname)
		}
		if result.Port != table.expected.Port {
			t.Errorf("Expected port %v but got %v instead", table.expected.Port, result.Port)
		}
		if result.Async != table.expected.Async {
			t.Errorf("Expected async %v but got %v instead", table.expected.Async, result.Async)
		}
	}

	// Needed because we are calling flag.Parse() each time we run a test.
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}
func TestMandatory(t *testing.T) {
	type User struct {
		Name    string `mandatory:"true"`
		Age     int    `mandatory:"true"`
		Adult   bool   `mandatory:"true"`
		Address string
	}

	tables := []struct {
		flags    []string
		env      []string
		expected User
		isErr    bool
		stderr   bool
	}{
		{[]string{"-name", "abc"}, []string{"", "", "", ""}, User{}, true, true},                                                // should fail because Age is missing
		{[]string{"-name", "abc", "-adult=true"}, []string{"", "20", "", "def"}, User{"abc", 20, true, "def"}, false, false},    // a mandatory parameter is in the env
		{[]string{"-name", "abc", "-adult=false"}, []string{"", "20", "", "def"}, User{"abc", 20, false, "def"}, false, false},  // a mandatory parameter is in the env, bool is false                                             // should fail because Age is missing
		{[]string{"-name", "abc", "-adult=true"}, []string{"ghi", "20", "", "def"}, User{"ghi", 20, true, "def"}, false, false}, // env should override flag
	}

	for index, table := range tables {
		t.Logf("Testing table %d", index)
		setFlags(table.flags)
		setUserEnv(table.env)

		// Needed because we are calling flag.Parse() each time we run a test.
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		stderr := new(bytes.Buffer)
		flag.CommandLine.SetOutput(stderr)

		result := User{}
		err := Parse(&result)
		if table.isErr {
			if err == nil {
				t.Error("Expected an error but did not get it")
			} else {
				t.Logf("Expected an error - got: %v", err)
			}
		}

		if stderr.Len() > 0 {
			if !table.stderr {
				t.Errorf("Test produced unexpected output to stderr: %v", stderr.String())
			} else {
				t.Logf("Expected output to stderr - got: %v", stderr.String())
			}
			continue
		} else {
			if table.stderr {
				t.Error("Test was expected to output to stderr but it did not")
			}
		}

		if err != nil {
			continue
		}

		if result.Name != table.expected.Name {
			t.Errorf("Expected name %v but got %v instead", table.expected.Name, result.Name)
		}
		if result.Age != table.expected.Age {
			t.Errorf("Expected age %v but got %v instead", table.expected.Age, result.Age)
		}
		if result.Adult != table.expected.Adult {
			t.Errorf("Expected adult %v but got %v instead", table.expected.Adult, result.Adult)
		}
		if result.Address != table.expected.Address {
			t.Errorf("Expected address %v but got %v instead", table.expected.Address, result.Address)
		}
	}

	// Needed because we are calling flag.Parse() each time we run a test.
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

func TestFilesSimple(t *testing.T) {
	filevalues := make(map[string]configFile)
	filevalues["username"] = configFile{
		subDirs:  "",
		contents: "admin",
	}
	filevalues["password"] = configFile{
		subDirs:  "",
		contents: "mypassword",
	}
	filevalues["maxretries"] = configFile{
		subDirs:  "",
		contents: "5",
	}
	filevalues["locked"] = configFile{
		subDirs:  "",
		contents: "true",
	}
	filevalues["param"] = configFile{
		subDirs:  "",
		contents: "abc",
	}

	dir, err := createFilesInTempDir(filevalues)
	if err != nil {
		t.Errorf("Could not create files in temp dir: %v", err)
		return
	}

	defer os.RemoveAll(dir)

	config := struct {
		Username   string
		Password   string
		MaxRetries int
		Locked     bool
		RealParam  string `file:"param"`
	}{}

	setFlags([]string{})
	if err := ParseWithDir(&config, dir); err != nil {
		t.Errorf("Unexpected error while parsing config directory: %v", err)
		return
	}

	if config.Username != "admin" {
		t.Errorf("username was an unexpected value: %v", config.Username)
	}

	if config.Password != "mypassword" {
		t.Errorf("password was an unexpected value: %v", config.Password)
	}

	if config.MaxRetries != 5 {
		t.Errorf("maxretries was an unexpected value: %v", config.MaxRetries)
	}

	if !config.Locked {
		t.Errorf("locked was an unexpected value: %v", config.Locked)
	}

	if config.RealParam != "abc" {
		t.Errorf("realparam was an unexpected value: %v", config.RealParam)
	}

	// Needed because we are calling flag.Parse() each time we run a test.
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

func TestFilesNestedDirectories(t *testing.T) {
	filevalues := make(map[string]configFile)
	filevalues["username"] = configFile{
		subDirs:  "",
		contents: "admin",
	}
	filevalues["password"] = configFile{
		subDirs:  "0",
		contents: "mypassword",
	}
	filevalues["maxretries"] = configFile{
		subDirs:  "1",
		contents: "5",
	}
	filevalues["locked"] = configFile{
		subDirs:  "1/2",
		contents: "true",
	}
	filevalues["param"] = configFile{
		subDirs:  "3",
		contents: "abc",
	}

	dir, err := createFilesInTempDir(filevalues)
	if err != nil {
		t.Errorf("Could not create files in temp dir: %v", err)
		return
	}

	defer os.RemoveAll(dir)

	config := struct {
		Username   string
		Password   string
		MaxRetries int
		Locked     bool
		RealParam  string `file:"param"`
	}{}

	setFlags([]string{})
	if err := ParseWithDir(&config, dir); err != nil {
		t.Errorf("Unexpected error while parsing config directory: %v", err)
		return
	}

	if config.Username != "admin" {
		t.Errorf("username was an unexpected value: %v", config.Username)
	}

	if config.Password != "mypassword" {
		t.Errorf("password was an unexpected value: %v", config.Password)
	}

	if config.MaxRetries != 5 {
		t.Errorf("maxretries was an unexpected value: %v", config.MaxRetries)
	}

	if !config.Locked {
		t.Errorf("locked was an unexpected value: %v", config.Locked)
	}

	if config.RealParam != "abc" {
		t.Errorf("realparam was an unexpected value: %v", config.RealParam)
	}

	// Needed because we are calling flag.Parse() each time we run a test.
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

func setFlags(args []string) {
	myargs := []string{"test"}
	myargs = append(myargs, args...)
	os.Args = myargs
}

func setConfigEnv(values []string) {
	hostname := values[0]
	port := values[1]
	async := values[2]
	if len(hostname) == 0 {
		os.Unsetenv("HOST")
	} else {
		os.Setenv("HOST", hostname)
	}
	if len(port) == 0 {
		os.Unsetenv("PORT")
	} else {
		os.Setenv("PORT", port)
	}
	if len(async) == 0 {
		os.Unsetenv("ASYNC")
	} else {
		os.Setenv("ASYNC", async)
	}
}

func setUserEnv(values []string) {
	name := values[0]
	age := values[1]
	adult := values[2]
	address := values[3]
	if len(name) == 0 {
		os.Unsetenv("NAME")
	} else {
		os.Setenv("NAME", name)
	}
	if len(age) == 0 {
		os.Unsetenv("AGE")
	} else {
		os.Setenv("AGE", age)
	}
	if len(adult) == 0 {
		os.Unsetenv("ADULT")
	} else {
		os.Setenv("ADULT", adult)
	}
	if len(address) == 0 {
		os.Unsetenv("ADDRESS")
	} else {
		os.Setenv("ADDRESS", address)
	}
}

func createFilesInTempDir(values map[string]configFile) (string, error) {
	dir, err := os.MkdirTemp("", "configparser-test")
	if err != nil {
		return "", err
	}
	for k, v := range values {
		if v.subDirs != "" {
			fullSubDirPath := filepath.Join(dir, v.subDirs)
			if _, err := os.Stat(fullSubDirPath); os.IsNotExist(err) {
				if err := os.MkdirAll(fullSubDirPath, os.ModePerm); err != nil {
					log.Fatalf("could not create directory %s: %v", fullSubDirPath, err)
				}
			}
		}
		if err := os.WriteFile(filepath.Join(dir, v.subDirs, k), []byte(v.contents), 0644); err != nil {
			os.RemoveAll(dir)
			return "", err
		}
	}
	return dir, nil
}
