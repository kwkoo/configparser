package configparser

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

var params []*param

type param struct {
	filename     string
	envKey       string
	flagKey      string
	fieldKind    reflect.Kind
	paramPointer unsafe.Pointer
	mandatory    bool
	isSet        bool
}

func (p param) String() string {
	if p.fieldKind == reflect.String {
		return *((*string)(p.paramPointer))
	}
	if p.fieldKind == reflect.Int {
		i := *((*int)(p.paramPointer))
		return strconv.Itoa(i)
	}
	if p.fieldKind == reflect.Bool {
		if *((*bool)(p.paramPointer)) {
			return "true"
		}
		return "false"
	}
	return ""
}

func (p *param) setParam(val, configType, keyName string) error {
	if p.fieldKind == reflect.String {
		p.isSet = true
		*(*string)(p.paramPointer) = val
		return nil
	}
	if p.fieldKind == reflect.Int {
		p.isSet = true
		i, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("%s %s must be an integer - instead it is: %v", configType, keyName, val)
		}
		*(*int)(p.paramPointer) = i
		return nil
	}
	if p.fieldKind == reflect.Bool {
		p.isSet = true
		l := strings.ToLower(val)
		bval := true
		if l == "0" || l == "f" || l == "false" || l == "n" || l == "no" {
			bval = false
		}
		*(*bool)(p.paramPointer) = bval
		return nil
	}

	return fmt.Errorf("%s %s is of an unknown type: %v", configType, keyName, val)
}

func (p *param) Set(s string) error {
	return p.setParam(s, "command line flag", p.flagKey)
}

func (p param) IsBoolFlag() bool {
	return p.fieldKind == reflect.Bool
}

// Parse will take in a pointer to a struct and set each field to an
// environment variable or a flag from the command line. The environment
// variable will take precedence over the command line flag.
//
// Parse will invoke ParseWithDir with dir set to an empty string.
//
func Parse(ptrtostruct interface{}) error {
	return ParseWithDir(ptrtostruct, "")
}

// ParseWithDir will take in a pointer to a struct and set each field to a
// value in the a file, environment variable, or a flag from the command line.
// The file will take precedence over the environment variable and the
// environment variable will take precedence over the command line flag.
//
// If a field is of type bool, it will be set to true as long as the
// corresponding environment variable is set, irrespective of the environment
// variable's value.
//
// Set the appropriate tag in each field to tell ParseWithDir how to handle the
// field. ParseWithDir accepts the following tags: env, flag, default, usage,
// mandatory.
//
// The env tag specifies the environment variable name which corresponds to
// the field. If this is not specified, ParseWithDir uses the uppercase version
// of the field name.
//
// The flag tag specifies the command line flag name which corresponds to the
// field. If this is not specified, ParseWithDir uses the lowercase version of
// the field name.
//
// The default tag specifies a default value for the field. This value is used
// if the corresponding environment variable and command line flag do not
// exist.
//
// The mandatory tag marks the field as mandatory. If the corresponding
// environment variable and command line flag do not exist, ParseWithDir will
// print an error message and the usage to stderr and return with an error.
// ParseWithDir will assume that the field is mandatory as long as the tag
// exists - it doesn't matter what value the tag is set to.
//
// The usage tag specifies the usage text for the command line flag.
//
func ParseWithDir(ptrtostruct interface{}, dir string) error {
	ptrtostructval := reflect.ValueOf(ptrtostruct)
	if ptrtostructval.Kind() != reflect.Ptr {
		return fmt.Errorf("argument must be a pointer to struct - got %v instead", ptrtostructval.Kind())
	}

	structval := ptrtostructval.Elem()
	if structval.Kind() != reflect.Struct {
		return fmt.Errorf("argument must be a pointer to struct - got a pointer to %v instead", structval.Kind())
	}

	configFiles := allFilesInDirectory(dir)

	params = []*param{}
	structtype := structval.Type()
	fieldcount := structtype.NumField()

	// We'll loop through the parameters twice - once for the command line
	// flags, and another for the files and environment variables. This is
	// because the files and environment variables take precedence over
	// command line flags.
	for i := 0; i < fieldcount; i++ {
		structfield := structtype.FieldByIndex([]int{i})
		structfieldkind := structfield.Type.Kind()

		// We only support fields of type string, int, and bool.
		if structfieldkind != reflect.String && structfieldkind != reflect.Int && structfieldkind != reflect.Bool {
			log.Printf("skipping field %v because it is not of a supported type", structfield.Name)
			continue
		}

		// Skip invalid fields and fields that cannot be set.
		field := structval.FieldByIndex([]int{i})
		if !field.IsValid() || !field.CanSet() {
			log.Printf("skipping field %v because it is not valid or cannot be set", structfield.Name)
			continue
		}

		// Skip field if this field cannot be converted to a pointer (necessary
		// for flag call).
		if !field.CanAddr() {
			log.Printf("skipping field %v because it cannot be converted to a pointer", structfield.Name)
			continue
		}

		filename := structfield.Tag.Get("file")
		if dir != "" {
			if filename == "" {
				filename = strings.ToLower(structfield.Name)
			}
		} else {
			filename = ""
		}

		envkey := structfield.Tag.Get("env")
		if len(envkey) == 0 {
			envkey = strings.ToUpper(structfield.Name)
		}
		flagkey := structfield.Tag.Get("flag")
		if len(flagkey) == 0 {
			flagkey = strings.ToLower(structfield.Name)
		}

		usage := structfield.Tag.Get("usage")
		_, ismandatory := structfield.Tag.Lookup("mandatory")

		p := param{
			filename:     filename,
			envKey:       envkey,
			flagKey:      flagkey,
			fieldKind:    structfieldkind,
			paramPointer: unsafe.Pointer(field.Addr().Pointer()),
			mandatory:    ismandatory,
			isSet:        false,
		}
		params = append(params, &p)

		if defaultval, defaultexists := structfield.Tag.Lookup("default"); defaultexists {
			p.Set(defaultval)
		}
		flag.Var(&p, flagkey, usage)
	}

	flag.Parse()

	// Loop through parameters a second time for the files and environment
	// variables.
	for _, p := range params {
		if p.filename != "" {
			configFilePath, ok := configFiles[p.filename]
			if ok {
				filecontents, err := getFileContents(configFilePath)
				if err == nil {
					err := p.setParam(filecontents, "file", p.filename)
					if err != nil {
						return err
					}
					// no errors setting param to file contents
					continue
				} else {
					if !os.IsNotExist(err) {
						// error is not file not found - i.e. the file exists
						// and the error is something else
						return err
					}
					// file does not exist, fall through and check if it's set as
					// an environment variable
				}
			}
		}

		envval, envkeyexists := os.LookupEnv(p.envKey)
		if !envkeyexists {
			continue
		}

		if err := p.setParam(envval, "environment variable", p.envKey); err != nil {
			return err
		}
	}

	// Loop through parameters again to pick up missing mandatory parameters.
	missingCount := 0
	for _, p := range params {
		if !p.mandatory || p.isSet {
			continue
		}
		missingCount++
		fmt.Fprintf(flag.CommandLine.Output(), "Mandatory flag -%s (or environment variable %s) does not exist.\n", p.flagKey, p.envKey)
	}

	params = []*param{}
	if missingCount > 0 {
		flag.Usage()
		return fmt.Errorf("%d mandatory parameters missing", missingCount)
	}

	return nil
}

func getFileContents(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func allFilesInDirectory(dir string) map[string]string {
	files := make(map[string]string)

	if dir == "" {
		return files
	}

	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if !entry.Type().IsRegular() {
			return nil
		}
		files[entry.Name()] = path
		return nil
	})

	if err != nil {
		log.Fatalf("error traversing config directory %s: %v", dir, err)
	}

	return files
}

// Retrieves file config directory from an environment variable or command
// line flag. The environment variable takes precedence.
// This function is only used to retrieve the configuration directory name.
func RetrieveConfigDirectory(envKey, flagKey, defaultval string) string {
	var val string
	if len(envKey) > 0 {
		val = os.Getenv(envKey)
		if len(val) == 0 {
			return defaultval
		}
		return val
	}

	if len(flagKey) > 0 {
		flag.StringVar(&val, flagKey, defaultval, "")
		flag.Parse()

		// reset flag variables
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		return val
	}

	return defaultval
}
