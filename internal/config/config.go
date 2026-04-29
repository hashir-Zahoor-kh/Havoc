// Package config loads component configuration from environment variables.
// Each binary calls LoadX() for the pieces it needs. Keeping config loading
// central avoids scattering getenv calls across every file.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Control holds the settings required by the havoc-control binary.
type Control struct {
	HTTPAddr       string
	KafkaBrokers   []string
	PostgresDSN    string
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	KubeconfigPath string
	InCluster      bool

	BlastRadiusPercent int
	LockBufferSeconds  int
}

// Agent holds the settings required by the havoc-agent binary.
type Agent struct {
	NodeName       string
	KafkaBrokers   []string
	KafkaGroupID   string
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	KubeconfigPath string
	InCluster      bool

	// HealthAddr is the bind address for the readiness/liveness HTTP
	// server. Kept distinct from the control-plane API port so this
	// can run on the same network namespace without clashing in
	// non-cluster contexts (e.g. a developer running both binaries
	// locally).
	HealthAddr string
}

// Recorder holds the settings required by the havoc-recorder binary.
type Recorder struct {
	KafkaBrokers  []string
	KafkaGroupID  string
	PostgresDSN   string
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// HealthAddr is the bind address for the readiness/liveness HTTP
	// server. See Agent.HealthAddr.
	HealthAddr string
}

// Client holds the settings required by the havoc CLI when talking to the
// control-plane HTTP API.
type Client struct {
	APIURL  string
	Timeout time.Duration
}

// LoadControl returns control-plane config pulled from the environment.
func LoadControl() (Control, error) {
	c := Control{
		HTTPAddr:           getEnv("HAVOC_HTTP_ADDR", ":8080"),
		KafkaBrokers:       splitCSV(getEnv("HAVOC_KAFKA_BROKERS", "localhost:9092")),
		PostgresDSN:        getEnv("HAVOC_POSTGRES_DSN", "postgres://havoc:havoc@localhost:5432/havoc?sslmode=disable"),
		RedisAddr:          getEnv("HAVOC_REDIS_ADDR", "localhost:6379"),
		RedisPassword:      os.Getenv("HAVOC_REDIS_PASSWORD"),
		KubeconfigPath:     os.Getenv("HAVOC_KUBECONFIG"),
		InCluster:          getEnvBool("HAVOC_IN_CLUSTER", false),
		BlastRadiusPercent: getEnvInt("HAVOC_BLAST_RADIUS_PCT", 25),
		LockBufferSeconds:  getEnvInt("HAVOC_LOCK_BUFFER_SECONDS", 30),
	}
	db, err := parseInt(getEnv("HAVOC_REDIS_DB", "0"))
	if err != nil {
		return c, fmt.Errorf("HAVOC_REDIS_DB: %w", err)
	}
	c.RedisDB = db
	if len(c.KafkaBrokers) == 0 {
		return c, errors.New("HAVOC_KAFKA_BROKERS is empty")
	}
	if c.BlastRadiusPercent < 1 || c.BlastRadiusPercent > 100 {
		return c, fmt.Errorf("HAVOC_BLAST_RADIUS_PCT must be 1..100, got %d", c.BlastRadiusPercent)
	}
	return c, nil
}

// LoadAgent returns agent config pulled from the environment.
func LoadAgent() (Agent, error) {
	a := Agent{
		NodeName:       os.Getenv("HAVOC_NODE_NAME"),
		KafkaBrokers:   splitCSV(getEnv("HAVOC_KAFKA_BROKERS", "localhost:9092")),
		KafkaGroupID:   getEnv("HAVOC_KAFKA_GROUP", "havoc-agent"),
		RedisAddr:      getEnv("HAVOC_REDIS_ADDR", "localhost:6379"),
		RedisPassword:  os.Getenv("HAVOC_REDIS_PASSWORD"),
		KubeconfigPath: os.Getenv("HAVOC_KUBECONFIG"),
		InCluster:      getEnvBool("HAVOC_IN_CLUSTER", true),
		HealthAddr:     getEnv("HAVOC_HEALTH_ADDR", ":8081"),
	}
	db, err := parseInt(getEnv("HAVOC_REDIS_DB", "0"))
	if err != nil {
		return a, fmt.Errorf("HAVOC_REDIS_DB: %w", err)
	}
	a.RedisDB = db
	if a.NodeName == "" {
		return a, errors.New("HAVOC_NODE_NAME is required (set via downward API in the DaemonSet)")
	}
	return a, nil
}

// LoadRecorder returns recorder config pulled from the environment.
func LoadRecorder() (Recorder, error) {
	r := Recorder{
		KafkaBrokers:  splitCSV(getEnv("HAVOC_KAFKA_BROKERS", "localhost:9092")),
		KafkaGroupID:  getEnv("HAVOC_KAFKA_GROUP", "havoc-recorder"),
		PostgresDSN:   getEnv("HAVOC_POSTGRES_DSN", "postgres://havoc:havoc@localhost:5432/havoc?sslmode=disable"),
		RedisAddr:     getEnv("HAVOC_REDIS_ADDR", "localhost:6379"),
		RedisPassword: os.Getenv("HAVOC_REDIS_PASSWORD"),
		HealthAddr:    getEnv("HAVOC_HEALTH_ADDR", ":8081"),
	}
	db, err := parseInt(getEnv("HAVOC_REDIS_DB", "0"))
	if err != nil {
		return r, fmt.Errorf("HAVOC_REDIS_DB: %w", err)
	}
	r.RedisDB = db
	return r, nil
}

// LoadClient returns CLI-client config.
func LoadClient() Client {
	return Client{
		APIURL:  getEnv("HAVOC_API_URL", "http://localhost:8080"),
		Timeout: 15 * time.Second,
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getEnvBool(key string, def bool) bool {
	v := strings.ToLower(os.Getenv(key))
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "yes", "y":
		return true
	case "0", "false", "no", "n":
		return false
	}
	return def
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}
