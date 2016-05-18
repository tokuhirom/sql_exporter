package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
)

var version string

const namespace = "sql"

var (
	addr       = flag.String("listen-address", ":9012", "The address to listen on for HTTP requests.")
	configPath = flag.String("config", "config.yml", "Config file path")
	versionFlg = flag.Bool("version", false, "Show version number")
)

type Query struct {
	SQL  string `yaml:"sql"`
	Name string `yaml:"name"`
	Help string `yaml:"help"`
}

type Config struct {
	DriverName     string  `yaml:"driver_name"`
	DataSourceName string  `yaml:"data_source_name"`
	Queries        []Query `yaml:"queries"`
}

type Exporter struct {
	mutex          sync.RWMutex
	config         Config
	db             sql.DB
	scrapeFailures prometheus.Counter
	counters       []*prometheus.CounterVec
}

func loadConfig(configPath string) (*Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config;path:<%s>,err:<%s>", configPath, err)
	}

	var config *Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse yaml;err:<%s>", err)
	}
	return config, nil
}

func NewExporter(config *Config, db *sql.DB) (*Exporter, error) {
	counters, err := GetCounters(config, db)
	if err != nil {
		return nil, err
	}

	return &Exporter{
		config: *config,
		db:     *db,
		scrapeFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "exporter_scrape_failures_total",
			Help:      "Number of errors while scraping apache.",
		}),
		counters: counters,
	}, nil
}

func GetCounters(config *Config, db *sql.DB) ([]*prometheus.CounterVec, error) {
	err := db.Ping()
	if err != nil {
		return nil, err
	}

	retval := make([]*prometheus.CounterVec, len(config.Queries))

	for i, query := range config.Queries {
		rows, err := db.Query(query.SQL)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		log.Debugf("Running query: %s", query.SQL)

		cols, err := rows.Columns()
		if err != nil {
			return nil, err
		}

		log.Debugf("Columns: %s => %s", query.SQL, cols[1:])
		counter := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      query.Name,
				Help:      query.Help,
			},
			cols[1:],
		)
		retval[i] = counter
	}
	return retval, nil
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.scrapeFailures.Describe(ch)
	for _, counter := range e.counters {
		counter.Describe(ch)
	}
}

func (e *Exporter) collect(ch chan<- prometheus.Metric) error {
	err := e.db.Ping()
	if err != nil {
		return err
	}

	for i, query := range e.config.Queries {
		log.Debugf("Running query: %s", query.SQL)

		rows, err := e.db.Query(query.SQL)
		if err != nil {
			return err
		}
		defer rows.Close()

		cols, err := rows.Columns()
		if err != nil {
			return err
		}

		pointers := make([]interface{}, len(cols))
		container := make([]string, len(cols)-1)
		var value float64
		pointers[0] = &value
		for j, _ := range container {
			pointers[j+1] = &container[j]
		}

		for rows.Next() {
			err = rows.Scan(pointers...)
			if err != nil {
				return err
			}
			log.Debugf("Result[%d]: %s, %s", i, container, value)
			e.counters[i].WithLabelValues(container...).Set(value)
		}
		e.counters[i].Collect(ch)
	}

	return nil
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()
	if err := e.collect(ch); err != nil {
		log.Debugf("Cannot query sql: %s", err)
		e.scrapeFailures.Inc()
		e.scrapeFailures.Collect(ch)
	}
	return
}

func main() {
	flag.Parse()
	if *versionFlg {
		fmt.Fprintf(os.Stderr, "%s version %s\n", os.Args[0], version)
		os.Exit(0)
	}

	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open(config.DriverName, config.DataSourceName)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Listen: %s, Pid: %d", *addr, os.Getpid())

	exporter, err := NewExporter(config, db)
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(exporter)

	http.Handle("/metrics", prometheus.Handler())
	log.Fatal(http.ListenAndServe(*addr, nil))
}
