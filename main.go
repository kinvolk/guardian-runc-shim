package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Binary  string `required:"true"`
	LogFile string `default:"/var/log/runc-wrapper"`
}

func modifyConfig(c map[string]interface{}) (map[string]interface{}, error) {
	// Copy map so we don't alter the original.
	res := make(map[string]interface{})
	for k, v := range c {
		res[k] = v
	}

	// Perform required modifications.
	// res["ociVersion"] = "1.0.1"
	// TODO Implement actual spec changes.

	return res, nil
}

func main() {
	var c Config
	err := envconfig.Process("runc_wrapper", &c)
	if err != nil {
		log.Fatal(err)
	}

	// Set up logging.
	f, err := os.OpenFile(c.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// Verify runc binary exists.
	binPath, err := exec.LookPath(c.Binary)
	if err != nil {
		log.Fatalf("Binary '%s' does not exist\n", binPath)
	}
	log.Printf("Using runc binary at %s\n", binPath)

	sArgs := strings.Join(os.Args[1:], " ")
	log.Printf("Wrapper called with arguments: %s\n", sArgs)

	// When true we need to modify config.json.
	var modify bool

	// Check if `runc run` was called.
	for _, a := range os.Args[1:] {
		if a == "run" {
			modify = true
		}
	}

	if modify {
		// Extract container ID and bundle path.
		var cid, bundle string
		for i, v := range os.Args[1:] {
			if v == "--bundle" {
				// Take the element right after "--bundle". i+2 because we are iterating over
				// os.Args[1:], not os.Args.
				bundle = os.Args[i+2]
			}
		}
		cid = os.Args[len(os.Args)-1]

		if bundle == "" {
			log.Fatal("Could not read bundle path")
		}

		if cid == "" {
			log.Fatal("Could not read container ID")
		}

		log.Printf("Bundle path: %s\n", bundle)
		log.Printf("Container ID: %s\n", cid)

		log.Println("Modifying config.json")
		cf := filepath.Join(bundle, "config.json")
		b, err := ioutil.ReadFile(cf)
		if err != nil {
			log.Fatalf("Reading config file: %v\n", err)
		}

		var j map[string]interface{}
		json.Unmarshal(b, &j)
	}

	log.Printf("Executing %s %s\n", binPath, sArgs)
	err = syscall.Exec(binPath, append([]string{binPath}, os.Args[1:]...), os.Environ())
	if err != nil {
		log.Fatal(err)
	}
}
